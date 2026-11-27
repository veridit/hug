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
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/harness"
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
	// Asked: a question on the task is open to someone (72n8m4), so done
	// waits for its answer.
	Asked bool
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
	// Step is the step due on it now (task.StepOf, erv680): the board cue
	// says Step.Cue to Step.For, the same next: the CLI and MCP give.
	Step task.Step
	// Review is the review hug handed to the task's reviewer (5yh7hr),
	// with its receipt; nil when it is not in review.
	Review *task.Item
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
	reflect     func() ([]Reflection, error)
	close       func() error
	common      string // the git common dir, for the worktree roots
}

// Reflection is one open reflection (tb0dra): the task it is on and the
// day it was written (YYYY-MM-DD).
type Reflection struct {
	TaskID, Date string
}

// OpenReflections returns the board's open reflections, as hug task
// reflections lists them (rk5q7r: the coordinator's cue counts the old
// ones). Computed on demand: it reads every task's notes.
func (s *Snapshot) OpenReflections() ([]Reflection, error) {
	if s == nil || s.reflect == nil {
		return nil, nil
	}
	return s.reflect()
}

func openReflections(ts []task.Task) []Reflection {
	var out []Reflection
	for _, t := range ts {
		if t.Texts == nil || !strings.Contains(t.Texts.Notes, "Reflection (") {
			continue
		}
		for _, r := range task.ParseReflections(t, t.Texts.Notes) {
			if r.State == "open" {
				out = append(out, Reflection{TaskID: r.ID, Date: r.Date})
			}
		}
	}
	return out
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

// Owners reads the harness sessions that worked in this repository (its
// main checkout and every linked worktree) from the jcode instance at
// jcodeHome ("" is ~/.jcode) and every instance the zsh registry under
// home names, as hug task lint's stale-owner does (9ahvgt), and returns
// whether a name is gone by the same rule (task.Liveness.Gone): its
// newest session closed over an hour ago, or silent for a day, also
// under a "You are NAME" alias. known is false for a name no session has
// (a person, a model). Read once, on the first call (agh2h7).
func (s *Snapshot) Owners(jcodeHome, home string) func(name string, now time.Time) (gone, known bool) {
	var live *task.Liveness
	read := false
	return func(name string, now time.Time) (bool, bool) {
		if !read {
			read = true
			var more []string
			if home != "" {
				for _, i := range harness.DiscoverJcode(home, nil) {
					more = append(more, i.Home)
				}
			}
			live = task.JcodeLivenessAt(jcodeHome, task.RepoRoots(s.Place.Main, s.common), more...)
		}
		return live.Gone(name, now)
	}
}

// ReplayOwners is Owners for an offline replay (agh2h7): the sessions in
// dirs that worked in the repository at repo (its main checkout and the
// linked worktrees git lists now; "": every session), each name judged
// as its sessions stood at the moment asked (task.Liveness.At).
func ReplayOwners(repo string, dirs ...string) func(name string, at time.Time) (gone, known bool) {
	var roots []string
	if repo != "" {
		roots = task.RepoRoots(repo, filepath.Join(repo, ".git"))
	}
	all := task.LivenessIn(roots, time.Time{}, dirs...)
	var cached *task.Liveness
	var cachedAt time.Time
	return func(name string, at time.Time) (bool, bool) {
		if cached == nil || !at.Equal(cachedAt) {
			cached, cachedAt = all.At(at), at
		}
		return cached.Gone(name, at)
	}
}

// At pins the clock lint reads (a waiting task's age, bg1rd8) for offline
// replays, and returns a func that restores it. Not for concurrent use.
func At(t time.Time) (restore func()) {
	old := task.Now
	task.Now = func() time.Time { return t }
	return func() { task.Now = old }
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
	s := &Snapshot{Tasks: convert(ts, task.StepConfigOf(root)), Place: p, close: c.Close, common: common}
	s.Coordinator, _ = task.Coordinator(root)
	s.lint = func() ([]task.Finding, []task.Task, error) {
		fs, err := c.Lint(ctx)
		return fs, ts, err
	}
	s.reflect = func() ([]Reflection, error) {
		cp := append([]task.Task(nil), ts...)
		if err := c.FillTexts(ctx, cp); err != nil {
			return nil, err
		}
		return openReflections(cp), nil
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
	return &Snapshot{Tasks: convert(ts, task.StepConfig{}), lint: func() ([]task.Finding, []task.Task, error) { return task.Lint(docs), ts, nil },
		reflect: func() ([]Reflection, error) { return openReflections(ts), nil }}
}

func convert(ts []task.Task, c task.StepConfig) []Task {
	out := make([]Task, 0, len(ts))
	for _, t := range ts {
		x := Task{ID: t.ID, Title: t.Title, State: t.State, Path: t.Path, Assignee: t.Assignee, Model: t.Model,
			Branch: t.Branch, Worktree: t.Worktree, Parent: t.Parent, Checks: len(t.Acceptance), Answer: t.Answer, Requests: task.OpenRequests(t), Outcome: t.Outcome, Asked: task.HasOpenQuestion(t), Step: task.StepOf(t, c), Review: t.Review}
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

// DeliveredTo reports whether request r already reached the session named
// name (bbxdd7): hug delivered it to name, or name acknowledged it. Such a
// request is in that session's transcript, so the board cue does not name
// it to that session again.
func DeliveredTo(r task.Request, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || !strings.EqualFold(r.Receipt.To, name) {
		return false
	}
	return r.Receipt.Delivered != "" || r.Receipt.State != task.ReceiptPending
}

// ReviewDeliveredTo is DeliveredTo for the review hug handed to the
// task's reviewer when it entered review (5yh7hr): the board cue does not
// say the review step again to the session that has it.
func ReviewDeliveredTo(r *task.Item, name string) bool {
	name = strings.TrimSpace(name)
	if r == nil || name == "" || !strings.EqualFold(r.Receipt.To, name) {
		return false
	}
	return r.Receipt.Delivered != "" || r.Receipt.State != task.ReceiptPending
}

// OutcomeDeliveredTo is DeliveredTo for an answer's outcome (41hhgn check
// 11): hug handed it to the session named name, or name acknowledged it,
// so the board cue does not say it to that session again.
func OutcomeDeliveredTo(o *task.Outcome, name string) bool {
	name = strings.TrimSpace(name)
	if o == nil || o.Receipt == nil || name == "" || !strings.EqualFold(o.Receipt.To, name) {
		return false
	}
	return o.Receipt.Delivered != "" || o.Receipt.State != task.ReceiptPending
}
