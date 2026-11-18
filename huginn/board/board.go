// Package board is a read-only view of a repository's hug task board for the
// rest of hug: the recall hook reads it to notice when the board and the
// work disagree (qawmnb). It never creates the task worktree, never pulls or
// pushes, and never writes task files; it reads through the same disposable
// cache hug task uses, so an unchanged board costs a stat walk, not a parse.
package board

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/task"
)

// The board's branch and folder (huginn/internal/cli TaskBranch, TaskDir).
const (
	branch = "hug/tasks"
	dir    = ".hug/tasks"
)

// Task is what a reader needs to compare the board with the work.
type Task struct {
	ID, Title, State, Path string
	Assignee               []string
	Model                  string
	// Branch and Worktree are where the task was taken (x1t1t1).
	Branch, Worktree string
	Parent           string
	// Checks counts acceptance items; Open counts the unticked ones.
	Checks, Open int
	// Answer is the newest answer to a question asked on the task that its
	// asker (Answer.For) has not read yet (3jtfw9), for the recall board
	// cue: "jhf answered your question on ID: ...".
	Answer *task.Answer
	// Requests are the task's open requests (hbgybn), oldest first, for
	// the recall cue "jhf asks on ID: ...".
	Requests []task.Request
	// Outcome is what the asker did first after the newest answer (41hhgn
	// check 11), for the coordinator's cue.
	Outcome *task.Outcome
}

// Finding is one open lint finding (findings on done or dropped tasks are
// left out, as hug task lint does) with the command that repairs it.
type Finding struct {
	Path, TaskID, TaskState, Code, Problem, Fix string
}

// Place is where the reader works: its branch and worktree, and whether
// that worktree is a linked one (git worktree add).
type Place struct {
	Branch, Worktree, Main string
	Linked                 bool
}

// Snapshot is the board at one moment.
type Snapshot struct {
	Tasks []Task
	Place Place
	// Coordinator is hug.toml [board] coordinator (hbgybn): who hears a
	// request on a task nobody holds; "" means every session.
	Coordinator string
	lint        func() ([]task.Finding, []task.Task, error)
	close       func() error
}

// Findings returns the open lint findings. It is computed on demand: the
// recall hook asks only at a break.
func (s *Snapshot) Findings() ([]Finding, error) {
	if s == nil || s.lint == nil {
		return nil, nil
	}
	fs, ts, err := s.lint()
	if err != nil {
		return nil, err
	}
	open, _ := task.OpenFindings(fs, ts)
	status := map[string]string{}
	for _, t := range ts {
		status[t.ID] = t.State
	}
	out := make([]Finding, 0, len(open))
	for _, f := range open {
		id, _ := task.IDOf(f.Path)
		out = append(out, Finding{Path: f.Path, TaskID: id, TaskState: status[id], Code: task.FindingCode(f.Problem), Problem: f.Problem, Fix: f.FixLine()})
	}
	return out, nil
}

// Close releases the cache. Safe on nil.
func (s *Snapshot) Close() error {
	if s == nil || s.close == nil {
		return nil
	}
	return s.close()
}

// Open reads the board of the repository that holds cwd. It returns nil and
// no error when cwd is not in git or the repository has no board checked
// out: no board, nothing to say.
func Open(ctx context.Context, cwd string) (*Snapshot, error) {
	if cwd == "" {
		return nil, nil
	}
	var p Place
	common := ""
	if fp, c, ok := fastPlace(cwd); ok {
		p, common = fp, c
	} else {
		gp, err := gitbranch.Where(cwd)
		if err != nil {
			return nil, nil
		}
		p = Place{Branch: gp.Branch, Worktree: gp.Worktree, Main: gp.Main, Linked: gp.Linked}
	}
	root := filepath.Join(p.Main, filepath.FromSlash(dir))
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return nil, nil
	}
	if common == "" {
		out, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
		if err != nil {
			return nil, err
		}
		common = strings.TrimSpace(string(out))
	}
	c, err := cache.Open(root, cache.File(common), repoGit{r: &gitbranch.Repository{Dir: root, Branch: branch}, common: common})
	if err != nil {
		return nil, err
	}
	if _, err := c.Sync(ctx); err != nil {
		c.Close()
		return nil, err
	}
	ts, err := c.List(ctx)
	if err != nil {
		c.Close()
		return nil, err
	}
	s := &Snapshot{Tasks: convert(ts), Place: p, close: c.Close}
	s.Coordinator, _ = task.Coordinator(root)
	s.lint = func() ([]task.Finding, []task.Task, error) {
		fs, err := c.Lint(ctx)
		return fs, ts, err
	}
	return s, nil
}

// Files is a board built from task files, for offline replays that walk a
// board's history commit by commit. Only changed files are parsed again.
type Files struct {
	docs map[string]*task.Document
}

// NewFiles returns an empty board.
func NewFiles() *Files { return &Files{docs: map[string]*task.Document{}} }

// Set stores a task file (path relative to the board root).
func (f *Files) Set(path string, data []byte) {
	if !strings.HasSuffix(path, ".md") {
		return
	}
	if d := task.Parse(path, data); d.Task.ID != "" {
		f.docs[path] = d
	} else {
		delete(f.docs, path)
	}
}

// Delete removes a task file.
func (f *Files) Delete(path string) { delete(f.docs, path) }

// Snapshot is the board as the files are now.
func (f *Files) Snapshot() *Snapshot {
	docs := make([]*task.Document, 0, len(f.docs))
	ts := make([]task.Task, 0, len(f.docs))
	for _, d := range f.docs {
		docs = append(docs, d)
		ts = append(ts, d.Task)
	}
	task.Sort(ts)
	return &Snapshot{Tasks: convert(ts), lint: func() ([]task.Finding, []task.Task, error) { return task.Lint(docs), ts, nil }}
}

func convert(ts []task.Task) []Task {
	out := make([]Task, 0, len(ts))
	for _, t := range ts {
		x := Task{ID: t.ID, Title: t.Title, State: t.State, Path: t.Path, Assignee: t.Assignee, Model: t.Model,
			Branch: t.Branch, Worktree: t.Worktree, Parent: t.Parent, Checks: len(t.Acceptance), Answer: t.Answer, Requests: task.OpenRequests(t), Outcome: t.Outcome}
		for _, a := range t.Acceptance {
			if !a.Settled() {
				x.Open++
			}
		}
		out = append(out, x)
	}
	return out
}

// repoGit is the cache's view of the board branch (as cli.gitCache). Head
// reads the ref from the files when it can (no subprocess per turn).
type repoGit struct {
	r      *gitbranch.Repository
	common string
}

func (g repoGit) Head(context.Context) (string, error) {
	if h, ok := fastRef(g.common, branch); ok {
		return h, nil
	}
	return g.r.Head()
}
func (g repoGit) Diff(_ context.Context, a, b string) ([]cache.Change, error) {
	cs, err := g.r.DiffNameStatus(a, b)
	out := make([]cache.Change, 0, len(cs))
	for _, c := range cs {
		out = append(out, cache.Change{Status: c.Status, OldPath: c.OldPath, Path: c.Path})
	}
	return out, err
}
