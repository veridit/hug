package app

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/board"
	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/muninn/internal/holds"
	"github.com/veridit/hug/muninn/internal/ingest"
)

// Conditions (ez50rb). A memory with a stated condition (holds) whose
// condition has ended is left out of recall (holds_ended) and becomes one
// dream item of kind "ended". A memory with only a recorded branch
// (x1t1t1) is never hidden: many lessons learned on a branch are general;
// when its branch is merged or deleted it becomes one dream item of kind
// "branch". Nothing is deleted automatically.

// EndedPerPass bounds new condition items per dream pass.
var EndedPerPass = 20

const (
	endedInstruction = "This memory was stated to hold %s, and %s. It no longer shows in recall. " +
		"If it is still true without that condition, memory_remember it again without holds, then memory_forget this one. " +
		"If it is no longer true, memory_forget it."
	branchInstruction = "This memory was written on branch %s, which %s. Is it still true now that the branch is gone? " +
		"If yes, leave it (resolve with note kept). If it was only true on that branch, memory_forget it; " +
		"if part of it holds, memory_remember the part that does, then memory_forget this one."
)

// repoState answers task and branch questions for one repository, with at
// most one board read and two git calls, cached for the pass.
type repoState struct {
	tasks         map[string]string // id -> status; nil: no board
	refs, merged  map[string]bool   // nil: not a git checkout here
	defaultBranch string
}

func loadRepoState(ctx context.Context, repo string) *repoState {
	st := &repoState{}
	if snap, err := board.Open(ctx, repo); err == nil && snap != nil {
		st.tasks = map[string]string{}
		for _, t := range snap.Tasks {
			st.tasks[strings.ToLower(t.ID)] = t.Status
		}
		snap.Close()
	}
	st.defaultBranch = gitbranch.DefaultBranch(repo)
	if refs, err := gitLines(ctx, repo, "for-each-ref", "--format=%(refname:short)", "refs/heads"); err == nil {
		st.refs = map[string]bool{}
		for _, r := range refs {
			st.refs[r] = true
		}
		st.merged = map[string]bool{}
		if st.defaultBranch != "" {
			if lines, err := gitLines(ctx, repo, "branch", "--format=%(refname:short)", "--merged", st.defaultBranch); err == nil {
				for _, b := range lines {
					st.merged[b] = true
				}
			}
		}
	}
	return st
}

func gitLines(ctx context.Context, dir string, args ...string) ([]string, error) {
	c := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	c.Env = append(c.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := c.Output()
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines, nil
}

func (st *repoState) taskOpen(id string) (bool, bool) {
	if st.tasks == nil {
		return false, false
	}
	s, ok := st.tasks[strings.ToLower(id)]
	if !ok {
		// Not on the board: a typo or an alias (TASK-1960) must not end a
		// memory; tasks are closed with done or dropped, not removed.
		return false, false
	}
	return s != "done" && s != "dropped", true
}

// branchGone: known and gone (deleted, or merged into the default branch).
// The default branch itself never counts as gone.
func (st *repoState) branchGone(name string) (gone bool, why string, known bool) {
	if st.refs == nil || name == "" || name == st.defaultBranch {
		return false, "", false
	}
	switch {
	case !st.refs[name]:
		return true, "is deleted", true
	case st.merged[name]:
		return true, "is merged into " + st.defaultBranch, true
	}
	return false, "", true
}

// queueEnded checks every memory's condition and recorded branch, records
// ended conditions for recall, and queues dream items. With dry it only
// counts.
func (a *App) queueEnded(ctx context.Context, dry bool, now time.Time) (int, error) {
	mems, err := ingest.LoadNative(NativePath())
	if err != nil {
		return 0, err
	}
	repos := map[string]*repoState{}
	state := func(repo string) *repoState {
		if repo == "" {
			return &repoState{}
		}
		repo = filepath.Clean(repo)
		if st, ok := repos[repo]; ok {
			return st
		}
		st := loadRepoState(ctx, repo)
		repos[repo] = st
		return st
	}
	n := 0
	for _, m := range mems {
		if !m.Active || m.SID == "" || (m.Holds == "" && m.Branch == "") {
			continue
		}
		st := state(m.Repo)
		if m.Holds != "" {
			c, err := holds.Parse(m.Holds)
			if err != nil || c.Kind == "" {
				continue
			}
			ended, why := c.Ended(holds.State{Now: now, TaskOpen: st.taskOpen,
				BranchLive: func(b string) (bool, bool) { g, _, k := st.branchGone(b); return !g, k }})
			if !ended {
				continue
			}
			if dry {
				n++
				continue
			}
			// Recall stops showing it at once; only the dream items are
			// paced (EndedPerPass), the rest wait for the next pass.
			if _, err := a.DB.Exec(`INSERT OR IGNORE INTO holds_ended(sid, holds, reason, at_ns) VALUES(?,?,?,?)`, m.SID, c.Text, why, now.UnixNano()); err != nil {
				return n, err
			}
			if n >= EndedPerPass {
				continue
			}
			q, err := a.queueItem("ended", "ended:"+m.SID+":"+c.Text, m.SID, m.Scope, fmt.Sprintf(endedInstruction, c.Text, why), now)
			if err != nil {
				return n, err
			}
			if q {
				n++
			}
			continue
		}
		gone, why, known := st.branchGone(m.Branch)
		if !known || !gone || (!dry && n >= EndedPerPass) {
			continue
		}
		if dry {
			n++
			continue
		}
		q, err := a.queueItem("branch", "branch:"+m.SID+":"+m.Branch, m.SID, m.Scope, fmt.Sprintf(branchInstruction, m.Branch, why), now)
		if err != nil {
			return n, err
		}
		if q {
			n++
		}
	}
	return n, nil
}

func (a *App) queueItem(kind, key, sid, scope, instruction string, now time.Time) (bool, error) {
	r, err := a.DB.Exec(`INSERT OR IGNORE INTO dream_items(kind,key,sids,scope,instruction,created_ns) VALUES(?,?,?,?,?,?)`,
		kind, key, sid, scope, instruction, now.UnixNano())
	if err != nil {
		return false, err
	}
	k, _ := r.RowsAffected()
	return k > 0, nil
}
