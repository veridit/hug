package task

import (
	"encoding/json"
	"fmt"
	"github.com/veridit/hug/huginn/internal/id"
	"github.com/veridit/hug/internal/sharefile"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Result struct {
	Task     Task     `json:"task"`
	Changed  bool     `json:"changed"`
	Next     string   `json:"next"`
	Warnings []string `json:"warnings,omitempty"`
	// Said is one line the command says before next: (resume: what the
	// task was waiting on, 4tycwd), not a warning.
	Said string `json:"said,omitempty"`
	// Guide: what to do now, at this state change (see Guide).
	Guide   []string `json:"guide,omitempty"`
	Renamed []string `json:"renamed,omitempty"`
	Renames []Rename `json:"renames,omitempty"`
	// Answered: this write answered the question the task asked (qsqhpe:
	// the person asked ticked the check it is about), so it goes to the
	// coordinator as an answer does (gmzd87).
	Answered bool `json:"answered,omitempty"`
	// Outcome: this write was the asker's first after an answer, so it is
	// that answer's outcome, which goes to the coordinator too (41hhgn
	// check 11).
	Outcome bool `json:"outcome,omitempty"`
}
type NewOptions struct {
	Title, Goal, Group, Type, Priority string
	// Purpose, Principle and Details become their own sections after Goal
	// (the task description pattern; docs/task-vocabulary-test.md).
	Purpose, Principle, Details string
	Acceptance, Deps, Labels    []string
	Top                         bool
	After                       string
	// Pattern holds the task to the whole description pattern: Purpose and
	// Principle are required too (hug task new, MCP task_new; fkpyg3).
	Pattern bool
}
type Filter struct {
	Type, Priority, Label, Group, Mine string
	State                              string
	// Epic, when set, is an epic id: keep that epic and its children.
	Epic string
	// Me is who asks for next, and Coordinator the board's (hbgybn): an
	// open request that routes to Me comes first, until it is resolved.
	Me, Coordinator string
}

// Not splits a filter value into the value and whether it is negated: a
// leading ! means every task but those with the value (7zg9mf), as in
// 'hug task ls --state !done'. One mark, !, in the CLI, MCP and the TUI.
func Not(value string) (string, bool) {
	if rest, ok := strings.CutPrefix(value, "!"); ok {
		return rest, true
	}
	return value, false
}

// matchValue is one field's filter rule: "" keeps all, "X" keeps X, "!X"
// keeps all but X. has says whether the task has value.
func matchValue(filter string, has func(string) bool) bool {
	if filter == "" {
		return true
	}
	v, not := Not(filter)
	return has(v) != not
}

// Keeps reports whether f keeps t, the rule of ls, search and board: each
// set field must hold, a !value field must not (7zg9mf). ts is the board,
// for the epic.
func (f Filter) Keeps(t Task, ts []Task) bool {
	eq := func(field string) func(string) bool { return func(v string) bool { return field == v } }
	return matchValue(f.State, eq(t.State)) && matchValue(f.Type, eq(t.Type)) && matchValue(f.Priority, eq(t.Priority)) &&
		matchValue(f.Group, eq(t.Group)) && matchValue(f.Label, func(v string) bool { return contains(t.Labels, v) }) &&
		(f.Mine == "" || contains(t.Assignee, f.Mine)) && InEpic(t, ts, f.Epic)
}

// InEpic reports whether t is the epic with id epic or one of its children.
func InEpic(t Task, ts []Task, epic string) bool {
	if epic == "" {
		return true
	}
	if strings.EqualFold(t.ID, epic) {
		return true
	}
	if t.Parent == "" {
		return false
	}
	p, e := ResolveDependency(ts, t.Parent)
	return e == nil && strings.EqualFold(p.ID, epic)
}

type MoveOptions struct {
	Before, After string
	Top           bool
}

// Store exposes filesystem operations. Cache integration is supplied via Refresh,
// which must sync and return the current tree, before and after each operation.
type Store struct {
	Root           string
	DepsMode       string
	AcceptanceMode string
	Refresh        func() ([]Task, error)
	// RefreshWritten, when set, is Refresh after this Store wrote the task
	// files at paths itself: it rereads those paths only (44w1ev step 8).
	// Changes made by anything else still go through Invalidate and Refresh.
	RefreshWritten func(paths []string) ([]Task, error)
	// Snapshot keeps the board Refresh returned until this Store writes a
	// task file or Invalidate is called, so one command reads the board
	// once, not once per helper (44w1ev: ls at 10,000 tasks walked the
	// files six times). Set it only for one short-lived command; the TUI,
	// which waits while others write, leaves it off.
	Snapshot bool
	// Actor is who this Store writes as (the CLI's --as, HUG_USER or the
	// agent session; the TUI's user). A write by the asker of an unseen
	// answer marks it read (3jtfw9).
	Actor string
	// Action is the write being made, as the command reads (hug task note
	// ID "..."): an asker's first write after an answer records it as the
	// answer's outcome (41hhgn check 11).
	Action string
	// waking is set while Wake writes: the wake is hug's, not the
	// reader's, so it marks nothing read or back (z2vmte).
	waking bool
	// quiet is set while hug records a receipt (gmzd87): that write is
	// hug's, not the actor's, so it marks nothing read or in progress.
	quiet bool
	board []Task
	held  bool
}

// Invalidate drops the kept board: the next Tasks reads the files again.
// Call it after anything other than this Store changed task files (an
// editor, a pull, an import).
func (s *Store) Invalidate() { s.board, s.held = nil, false }

// refreshWritten rereads the board after this Store wrote paths.
func (s *Store) refreshWritten(paths ...string) error {
	if s.Refresh == nil {
		return nil
	}
	if s.RefreshWritten == nil {
		s.Invalidate()
		_, e := s.Tasks()
		return e
	}
	ts, e := s.RefreshWritten(paths)
	if e != nil {
		return e
	}
	if s.Snapshot {
		s.board, s.held = cloneTasks(ts), true
	}
	return nil
}
func (s *Store) Tasks() ([]Task, error) {
	if s.Refresh != nil {
		if s.Snapshot && s.held {
			return cloneTasks(s.board), nil
		}
		ts, e := s.Refresh()
		if e != nil || !s.Snapshot {
			return ts, e
		}
		s.board, s.held = cloneTasks(ts), true
		return ts, nil
	}
	var out []Task
	e := filepath.WalkDir(s.Root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != s.Root {
				return filepath.SkipDir
			}
			// A task's files folder holds files, never tasks.
			if rel, _ := filepath.Rel(s.Root, path); path != s.Root && IsTaskFolder(s.Root, filepath.ToSlash(rel)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") || d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		b, e := sharefile.ReadFile(path)
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(s.Root, path)
		if e != nil {
			return e
		}
		out = append(out, Parse(filepath.ToSlash(rel), b).Task)
		return nil
	})
	Sort(out)
	return out, e
}

// cloneTasks copies tasks and their slices, so a caller that edits what
// Tasks returned cannot change the board the Store keeps.
func cloneTasks(ts []Task) []Task {
	out := make([]Task, len(ts))
	for i, t := range ts {
		t.Assignee = append([]string(nil), t.Assignee...)
		t.Labels = append([]string(nil), t.Labels...)
		t.Aliases = append([]string(nil), t.Aliases...)
		t.Sections = append([]string(nil), t.Sections...)
		t.Dependencies = append([]Dependency(nil), t.Dependencies...)
		t.Waits = append([]Wait(nil), t.Waits...)
		t.Acceptance = append([]Acceptance(nil), t.Acceptance...)
		out[i] = t
	}
	return out
}
func (s *Store) load(ref string) (*Document, []Task, error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, nil, e
	}
	t, e := Resolve(ts, ref)
	if e != nil {
		return nil, nil, e
	}
	path, e := s.diskPath(t.Path)
	if e != nil {
		return nil, nil, e
	}
	b, e := sharefile.ReadFile(path)
	if e != nil {
		return nil, nil, e
	}
	return Parse(t.Path, b), ts, nil
}
func hint(t Task) string {
	switch t.State {
	case "todo":
		return "hug task take " + t.ID
	case "doing":
		for n, a := range t.Acceptance {
			if !a.Settled() {
				return fmt.Sprintf("hug task check %s %d", t.ID, n+1)
			}
		}
		// Every check holds but a question is open (72n8m4): done would
		// refuse, so the next thing is other work while the answer comes.
		if HasOpenQuestion(t) {
			return "hug task next"
		}
		return "hug task done " + t.ID
	default:
		return "hug task next"
	}
}
func (s *Store) finish(d *Document, changed bool) (Result, error) {
	if changed && Staged(d.Task) {
		// Every write keeps a staged file's stage right: a check, a note,
		// a dependency can change which stage holds (6ephqk).
		ts, e := s.Tasks()
		if e != nil {
			return Result{}, e
		}
		if e = settle(d, ts); e != nil {
			return Result{}, e
		}
	}
	if changed {
		outcome := false
		if !s.waking && !s.quiet {
			outcome = markRead(d, s.Actor, s.Action)
			markBack(d, s.Actor)
			markProgress(d, s.Actor, s.Action)
		}
		path, e := s.diskPath(d.Task.Path)
		if e != nil {
			return Result{}, e
		}
		s.Invalidate()
		if e := AtomicWrite(path, d.Bytes()); e != nil {
			return Result{}, e
		}
		if e := s.refreshWritten(d.Task.Path); e != nil {
			return Result{}, e
		}
		return Result{Task: d.Task, Changed: changed, Next: hint(d.Task), Outcome: outcome}, nil
	}
	return Result{Task: d.Task, Changed: changed, Next: hint(d.Task)}, nil
}
func safeGroup(g string) error {
	if filepath.IsAbs(g) || g == ".." || strings.HasPrefix(filepath.Clean(g), ".."+string(filepath.Separator)) || strings.Contains(g, "\\") {
		return &Error{Kind: "invalid-group", Value: g, Fix: "hug task mv <id> auth"}
	}
	for _, p := range strings.Split(filepath.ToSlash(g), "/") {
		if strings.HasPrefix(p, ".") && p != "" && p != "." {
			return &Error{Kind: "invalid-group", Value: g, Fix: "hug task mv <id> auth"}
		}
	}
	return nil
}
func (s *Store) New(o NewOptions) (Result, error) {
	// Every check, every problem, one fix (fkpyg3): the same list lint
	// uses, so a task new accepts is lint-clean.
	if e := newProblems(o); e != nil {
		return Result{}, e
	}
	if e := safeGroup(o.Group); e != nil {
		return Result{}, e
	}
	if o.Type == "" {
		o.Type = "feature"
	}
	if !contains(Types, o.Type) {
		return Result{}, Invalid("type", o.Type, Types)
	}
	ts, e := s.Tasks()
	if e != nil {
		return Result{}, e
	}
	known := map[string]bool{}
	for _, t := range ts {
		known[t.ID] = true
	}
	i, e := id.New(known)
	if e != nil {
		return Result{}, e
	}
	deps := []string{}
	for _, r := range o.Deps {
		t, e := Resolve(ts, r)
		if e != nil {
			return Result{}, e
		}
		deps = append(deps, t.ID)
	}
	plan, e := rankPlan(ts, Task{ID: i}, MoveOptions{Top: o.Top, After: o.After})
	if e != nil {
		return Result{}, e
	}
	r := ""
	for _, p := range plan {
		if p.Task.ID == i {
			r = p.Rank
		}
	}
	// Layout 3 (w40cmk): the task lives at C/ID.md from the start, its
	// group and rank in the front matter.
	path := PathFor(i)
	q := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	today := Now().Format("2006-01-02") // the board clock, as done_at (z25pb2)
	body := "---\nid: " + i + "\ntitle: " + q(o.Title) + "\nstate: todo\ntype: " + o.Type + "\npriority: " + q(o.Priority) + "\nassignee: []\nlabels: "
	labels := []string{}
	for _, l := range o.Labels {
		labels = append(labels, q(l))
	}
	body += "[" + strings.Join(labels, ", ") + "]\ngroup: " + q(filepath.ToSlash(o.Group)) + "\nrank: " + q(r) + "\ncreated: " + today + "\nupdated: " + today + "\n---\n\n## Goal\n" + o.Goal + "\n" + patternSections(o) + "\n## Depends on\n"
	for _, dep := range deps {
		body += "- " + dep + "\n"
	}
	body += "\n## Acceptance\n"
	for _, a := range o.Acceptance {
		body += "- [ ] " + a + "\n"
	}
	body += "\n## Notes\n"
	d := Parse(path, []byte(body))
	if len(d.Findings) > 0 {
		return Result{}, &Error{Kind: "invalid-task-content", ID: i, Value: d.Findings[0].Problem, Fix: "hug task new \"title\" --goal \"outcome\" --accept \"observable check\""}
	}
	created, e := s.finish(d, true)
	if e != nil {
		return created, e
	}
	otherPlan := []rankChange{}
	for _, p := range plan {
		if p.Task.ID != i {
			otherPlan = append(otherPlan, p)
		}
	}
	repaired, e := s.applyRanks(otherPlan, d.Task)
	created.Renamed = repaired.Renamed
	created.Renames = repaired.Renames
	return created, e
}
func (s *Store) Set(ref, field, value string) (Result, error) {
	if field == "state" {
		return s.Transition(ref, value, "")
	}
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	valid := []string{"title", "type", "priority", "label", "assignee", "parent"}
	if !contains(valid, field) {
		return Result{}, invalidFor(d.Task.ID, "field", field, valid)
	}
	if field == "type" && !contains(Types, value) {
		return Result{}, invalidFor(d.Task.ID, "type", value, Types)
	}
	before := string(d.Bytes())
	var v any = value
	if field == "parent" {
		ts, e := s.Tasks()
		if e != nil {
			return Result{}, e
		}
		if v, e = parentFor(d.Task, ts, value); e != nil {
			return Result{}, e
		}
	}
	if field == "label" {
		field = "labels"
		v = []string{value}
	}
	if field == "assignee" {
		v = []string{value}
	}
	if e = d.SetField(field, v); e != nil {
		return Result{}, e
	}
	return s.finish(d, before != string(d.Bytes()))
}
func (s *Store) Check(ref string, n int) (Result, error) { return s.CheckWith(ref, n, "") }

// CheckWith ticks item n and, with evidence, notes "check N: EVIDENCE" in
// the same write (pw0t19: the instructions say "hug task check ID N with
// evidence"; the tick and its evidence were two commands and two commits).
// Evidence on an item already ticked is still noted.
func (s *Store) CheckWith(ref string, n int, evidence string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	before := string(d.Bytes())
	if e = d.Check(n); e != nil {
		return Result{}, e
	}
	ticked := before != string(d.Bytes())
	if evidence = strings.TrimSpace(evidence); evidence != "" {
		d.Note(fmt.Sprintf("check %d: %s", n, evidence))
	}
	// The person asked ticking the check the question is about holds it:
	// that tick is their answer (qsqhpe, Jørgen on e3sevr).
	answered := false
	if q, ok := tickAnswers(d.Task, n, s.Actor); ticked && ok {
		if e = answerOn(d, ts, q, "answered by ticking check "+strconv.Itoa(n), strings.TrimSpace(s.Actor)); e != nil {
			return Result{}, e
		}
		answered = true
	}
	r, e := s.finish(d, before != string(d.Bytes()))
	r.Answered = answered
	return r, e
}

// Strike strikes acceptance item n of ref out as not relevant, with the
// reason and who decided, dated today (sfa8w4); undo restores it.
func (s *Store) Strike(ref string, n int, reason, who string, undo bool) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	before := string(d.Bytes())
	if undo {
		e = d.Unstrike(n)
	} else {
		e = d.Strike(n, reason, who, Now().Format("2006-01-02"))
	}
	if e != nil {
		return Result{}, e
	}
	return s.finish(d, before != string(d.Bytes()))
}
func (s *Store) Note(ref, text string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	d.Note(text)
	return s.finish(d, true)
}

// Take sets ref doing for user. Agents and people take through TakeAs, which
// also records the model; Take keeps the assignee only (the TUI's local
// fallback and tests).
func (s *Store) Take(ref, user string) (Result, error) { return s.transition(ref, "doing", user, "") }

// TakeAs takes ref for an agent or person. model is required (et85h1,
// Jørgen 2026-11-16: agents are ephemeral, models are not): a model name
// such as claude-opus-5-5, or human for a person. name (the agent or
// session, such as poodle) is optional; without it the assignee is the
// model. Both are recorded on the task: assignee and model.
func (s *Store) TakeAs(ref, name, model string) (Result, error) {
	name, model = strings.TrimSpace(name), strings.TrimSpace(model)
	if model == "" {
		return Result{}, &Error{Kind: "no-model", ID: ref, Fix: "hug task take " + ref + " --model MODEL"}
	}
	if name == "" {
		name = model
	}
	return s.TakeAt(ref, name, model, Place{})
}

// Place is where a task is taken: the git branch and worktree folder
// (x1t1t1). Empty fields are not recorded.
type Place struct{ Branch, Worktree string }

// TakeAt is TakeAs that also records where the work happens. A take from a
// new place updates branch and worktree; one with no place leaves them.
func (s *Store) TakeAt(ref, name, model string, at Place) (Result, error) {
	name, model = strings.TrimSpace(name), strings.TrimSpace(model)
	if model == "" {
		return Result{}, &Error{Kind: "no-model", ID: ref, Fix: "hug task take " + ref + " --model MODEL"}
	}
	if name == "" {
		name, _ = SplitModel(model) // the model is the assignee; the effort is not part of a name
	}
	return s.transitionFull(ref, "doing", name, model, at, "", "")
}

// TakeWith takes ref for user alongside with, user's active task (byj479):
// both are doing, ref marked doing_with: with. For tasks that ship
// together (one change or release); the TUI calls it per marked task.
func (s *Store) TakeWith(ref, with, user string) (Result, error) {
	return s.transitionFull(ref, "doing", user, "", Place{}, "", with)
}

// TakeAtWith is TakeAt alongside with (TakeWith): the CLI's take --with.
func (s *Store) TakeAtWith(ref, with, name, model string, at Place) (Result, error) {
	name, model = strings.TrimSpace(name), strings.TrimSpace(model)
	if model == "" {
		return Result{}, &Error{Kind: "no-model", ID: ref, Fix: "hug task take " + ref + " --with " + with + " --model MODEL"}
	}
	if name == "" {
		name, _ = SplitModel(model)
	}
	return s.transitionFull(ref, "doing", name, model, at, "", with)
}
func (s *Store) Done(ref string) (Result, error)         { return s.Transition(ref, "done", "") }
func (s *Store) Drop(ref, reason string) (Result, error) { return s.DropBecause(ref, reason) }
func (s *Store) Transition(ref, status, user string) (Result, error) {
	return s.transition(ref, status, user, "")
}
func (s *Store) transition(ref, status, user, model string) (Result, error) {
	return s.transitionAt(ref, status, user, model, Place{})
}

func (s *Store) transitionAt(ref, status, user, model string, at Place) (Result, error) {
	return s.transitionFull(ref, status, user, model, at, "", "")
}

// transitionWhy is a transition with a reason (drop's dropped_reason).
func (s *Store) transitionWhy(ref, status, reason string) (Result, error) {
	return s.transitionFull(ref, status, "", "", Place{}, reason, "")
}

func (s *Store) transitionFull(ref, status, user, model string, at Place, reason, with string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if !contains(StateList, status) {
		return Result{}, invalidFor(d.Task.ID, "state", status, StateList)
	}
	// Only the moves in Transitions (6ephqk check 3). done comes only from
	// doing: take before done (Jørgen 11:37).
	if d.Task.State != status && !allowed(d.Task, status) {
		fix := "hug task show " + d.Task.ID
		if status == "done" && d.Task.State == "todo" {
			fix = "hug task take " + d.Task.ID + " --model MODEL"
		}
		return Result{}, refuse(d.Task, status, fix)
	}
	// A question to a person is work still open (72n8m4, crab 2026-11-20:
	// done on ddpxs8 withdrew jhf's question as a side effect, "doing/asked
	// -> done"). done waits for the answer; withdrawing it is its own act.
	if status == "done" && d.Task.State != "done" {
		if e := openQuestionRefusal(d.Task); e != nil {
			return Result{}, e
		}
	}
	if status == "dropped" && d.Task.State != "dropped" && strings.TrimSpace(reason) == "" {
		return Result{}, usageErr(d.Task.ID, "drop needs a reason", "hug task drop "+d.Task.ID+` "duplicate of OTHER"`)
	}
	if d.Task.State == status && (status != "doing" || user == "" || contains(d.Task.Assignee, user)) && (model == "" || d.Task.ModelLabel() == model) && (at.Branch == "" || d.Task.Branch == at.Branch) && (at.Worktree == "" || d.Task.Worktree == at.Worktree) {
		// Same state: settle it (3nhp67). This is the fix lint gives for
		// stage-disagrees and stray-node-field, so it must make them go:
		// one stage stays, the others' fields go to History.
		if g := settleStage(d.Task); g != "" {
			before := string(d.Bytes())
			if e = applyMove(d, ts, move{state: status, stage: g, why: "settle: one stage, " + g}); e != nil {
				return Result{}, e
			}
			return s.finish(d, before != string(d.Bytes()))
		}
		if Staged(d.Task) && Stage(d.Task, ts) != d.Task.Stage {
			if e = applyMove(d, ts, move{state: status, why: "settle"}); e != nil {
				return Result{}, e
			}
			return s.finish(d, true)
		}
		return s.finish(d, false)
	}
	if status == "doing" && user != "" && len(d.Task.Assignee) > 0 && !contains(d.Task.Assignee, user) {
		return Result{}, &Error{Kind: "taken", ID: d.Task.ID, Value: strings.Join(d.Task.Assignee, ", "), Fix: "hug task next"}
	}
	// One ticket in doing per agent (n2hq8p): a take that would give user a
	// second active ticket is refused. Taking a todo/review task makes it
	// doing/review, not active, so it is allowed (the reviewer's own work
	// goes on).
	who := Taker{Name: user, Model: model}
	with = strings.TrimSpace(with)
	if with != "" {
		// Taken with the taker's active task (byj479): only that one, so
		// the link is what the taker is on now, never guessed.
		a, ok := Active(user, d.Task.ID, ts)
		if status != "doing" || user == "" || d.Task.State != "todo" {
			return Result{}, usageErr(d.Task.ID, "--with takes a todo task alongside your active one", "hug task show "+d.Task.ID)
		}
		w, err := Resolve(ts, with)
		if err != nil || !ok || w.ID != a.ID {
			name := with
			if err == nil {
				name = w.ID
			}
			return Result{}, withRefused(d.Task, who, name, a, ok)
		}
		with = w.ID
	} else if status == "doing" && user != "" && d.Task.State != "doing" && d.Task.Stage != "review" {
		if a, ok := Active(user, d.Task.ID, ts); ok {
			return Result{}, busy(d.Task, who, a, true)
		}
	}
	var warnings []string
	if status == "doing" || status == "done" {
		policy := s.DepsMode
		if policy == "" {
			policy, e = DepsPolicy(s.Root)
			if e != nil {
				return Result{}, e
			}
		}
		if !contains([]string{"block", "warn"}, policy) {
			return Result{}, Invalid("deps", policy, []string{"block", "warn"})
		}
		blocked := Blocked(d.Task, ts)
		if len(blocked) > 0 {
			if policy != "warn" {
				// The ways out (r5r8c1): work on the first blocker (take it
				// when it is todo), or drop the dependency.
				first := blocked[0]
				way := "hug task show " + first.ID
				if first.State == "todo" {
					way = "hug task take " + first.ID + " --model MODEL"
				}
				fixes := []string{way, "hug task dep " + d.Task.ID + " --remove " + first.ID}
				return Result{}, &Error{Kind: "blocked", ID: d.Task.ID, Dependencies: blocked, Fix: fixes[0], Fixes: fixes}
			}
			warnings = append(warnings, "open dependencies")
		}
		// A stated condition (16dxsr) holds back starting; one about the
		// board (a task or group closed) also holds back done, as a
		// dependency does. A date, load, branch or tag only says when to
		// begin.
		if unmet := Unmet(d.Task, ts, status == "done"); len(unmet) > 0 {
			if policy != "warn" {
				return Result{}, &Error{Kind: "blocked", ID: d.Task.ID, Field: "waits", Value: WaitSummary(unmet), Waits: unmet, Fix: "hug task show " + d.Task.ID}
			}
			warnings = append(warnings, "unmet conditions")
		}
	}
	if status == "done" {
		policy := s.AcceptanceMode
		if policy == "" {
			policy, e = AcceptancePolicy(s.Root)
			if e != nil {
				return Result{}, e
			}
		}
		if !contains([]string{"block", "warn"}, policy) {
			return Result{}, Invalid("acceptance", policy, []string{"block", "warn"})
		}
		unchecked := []Unchecked{}
		for n, a := range d.Task.Acceptance {
			if !a.Settled() {
				unchecked = append(unchecked, Unchecked{Index: n + 1, Text: a.Text, Fix: fmt.Sprintf("hug task check %s %d", d.Task.ID, n+1)})
			}
		}
		if len(unchecked) > 0 && policy != "warn" {
			return Result{}, &Error{Kind: "unchecked-acceptance", ID: d.Task.ID, Value: fmt.Sprintf("tick each open check that holds, or strike one that no longer applies: hug task strike %s %d --reason \"why\"", d.Task.ID, unchecked[0].Index), Unchecked: unchecked, Fix: unchecked[0].Fix}
		}
		if open := OpenChildren(d.Task, ts); len(open) > 0 {
			if policy != "warn" {
				return Result{}, &Error{Kind: "blocked", ID: d.Task.ID, Field: "children", Value: fmt.Sprintf("%d open child tasks", len(open)), Dependencies: open, Fix: "hug task show " + open[0].ID}
			}
			warnings = append(warnings, "open child tasks")
		}
	}
	if d.Task.State != status {
		if with != "" {
			reason = with
		}
		if e = stagedMove(d, ts, status, user, reason); e != nil {
			return Result{}, e
		}
	}
	if status == "doing" && user != "" {
		if e = d.SetField("assignee", []string{user}); e != nil {
			return Result{}, e
		}
	}
	if status == "doing" && model != "" {
		id, effort := SplitModel(model)
		if e = d.SetField("model", id); e != nil {
			return Result{}, e
		}
		if effort != "" || d.Task.Effort != "" {
			if e = d.SetField("effort", effort); e != nil {
				return Result{}, e
			}
		}
	}
	if status == "doing" {
		for _, f := range [][2]string{{"branch", at.Branch}, {"worktree", at.Worktree}} {
			if f[1] != "" {
				if e = d.SetField(f[0], f[1]); e != nil {
					return Result{}, e
				}
			}
		}
	}
	if status == "done" {
		for _, a := range d.Task.Acceptance {
			if !a.Settled() {
				warnings = append(warnings, "unchecked acceptance: "+a.Text)
			}
		}
	}
	from := ""
	if t, e := Resolve(ts, d.Task.ID); e == nil {
		from = t.State
	}
	r, e := s.finish(d, true)
	r.Warnings = warnings
	if e == nil && from != status {
		// Tasks that depend on this one: their depends stage follows it.
		settled, ce := s.cascade(d.Task.ID)
		for _, t := range settled {
			r.Warnings = append(r.Warnings, "stage of "+t)
		}
		if ce != nil {
			return r, ce
		}
	}
	if e == nil && status == "done" {
		// Items elsewhere that wait on this task ("done when task:<id> is
		// done", from promote) are ticked now.
		ticked, te := s.tickWaiting(d.Task.ID)
		for _, t := range ticked {
			r.Warnings = append(r.Warnings, "ticked "+t)
		}
		if te != nil {
			return r, te
		}
	}
	return r, e
}
func Next(tasks []Task, f Filter) (Result, error) {
	if typ, not := Not(f.Type); f.Type != "" && !contains(Types, typ) {
		e := Invalid("type", f.Type, Types)
		if not {
			e.Closest = "!" + e.Closest
		}
		return Result{}, e
	}
	ts := append([]Task(nil), tasks...)
	Sort(ts)
	epics := ChildIndex(tasks)
	// An open request for me comes before everything (hbgybn, crab 00:58):
	// the cue is not its only path. Oldest first.
	var req *Task
	var reqAt string
	for i := range ts {
		if !RequestFor(ts[i], f.Me, f.Coordinator) {
			continue
		}
		if at := OpenRequests(ts[i])[0].At; req == nil || at < reqAt {
			req, reqAt = &ts[i], at
		}
	}
	if req != nil {
		r := OpenRequests(*req)[0]
		return Result{Task: *req, Next: "hug task show " + req.ID + ", act on request " + strconv.Itoa(r.N) + " by " + r.By + ", then: hug task resolve " + req.ID + " " + strconv.Itoa(r.N) + ` "what changed"`}, nil
	}
	if f.Mine != "" {
		// An answer to my question comes first, whatever its state
		// (3jtfw9): reading it is the next thing to do. Newest first.
		var newest *Task
		for i := range ts {
			if AnswerFor(ts[i], f.Mine) && (newest == nil || ts[i].Answer.At > newest.Answer.At) {
				newest = &ts[i]
			}
		}
		if newest != nil {
			return Result{Task: *newest, Next: "hug task show " + newest.ID}, nil
		}
		// A task its date woke comes next (z2vmte): the owner asked to be
		// brought back to it.
		for _, t := range ts {
			if t.Woken && t.State == "doing" && contains(t.Assignee, f.Mine) && InEpic(t, tasks, f.Epic) {
				return Result{Task: t, Next: hint(t)}, nil
			}
		}
		sort.SliceStable(ts, func(i, j int) bool { return ts[i].State == "doing" && ts[j].State != "doing" })
	}
	// My doing tasks in stage waiting are not work now (4tycwd, crab
	// 17:57): hug wakes a wait (--until, its condition), resume ends it.
	// With nothing else for me, they are listed, one line each.
	var waits []string
	for _, t := range ts {
		if f.Mine != "" && t.State == "doing" && ownHolds(t, "waiting") {
			if contains(t.Assignee, f.Mine) && InEpic(t, tasks, f.Epic) {
				waits = append(waits, WaitingLine(t))
			}
			continue
		}
		if (t.State != "todo" && !(f.Mine != "" && t.State == "doing")) || Waiting(t, tasks) {
			continue
		}
		// An epic is finished through its children, never picked itself.
		if len(epics[strings.ToLower(t.ID)]) > 0 || !InEpic(t, tasks, f.Epic) {
			continue
		}
		if f.Mine != "" {
			if !contains(t.Assignee, f.Mine) {
				continue
			}
		} else if len(t.Assignee) > 0 {
			continue
		}
		if !(Filter{Type: f.Type, Priority: f.Priority, Label: f.Label, Group: f.Group}).Keeps(t, nil) {
			continue
		}
		return Result{Task: t, Next: hint(t)}, nil
	}
	if len(waits) > 0 {
		return Result{}, &Error{Kind: "nothing-matches", Value: "nothing for " + f.Mine + " to work on now; waiting:", Waiting: waits, Fix: "hug task ls"}
	}
	return Result{}, &Error{Kind: "nothing-matches", Fix: "hug task ls"}
}

// WaitingLine is a waiting task in one line: "ID waiting on WHAT", and
// " · until DATE" for a timed wait.
func WaitingLine(t Task) string {
	line := t.ID + " waiting on " + strings.Join(strings.Fields(t.Fields["waiting_on"]), " ")
	if u := strings.TrimSpace(t.Fields["waiting_until"]); u != "" {
		line += " · until " + u
	}
	return line
}
func (s *Store) rename(d *Document, group, r, slug string) (Result, error) {
	oldPath := d.Task.Path
	if e := safeGroup(group); e != nil {
		return Result{}, e
	}
	if _, layout3 := IDPath(d.Task.Path); layout3 {
		return s.reorder(d, group, r, slug)
	}
	if slug == "" || strings.ContainsAny(slug, "/\\.\x00") {
		return Result{}, &Error{Kind: "invalid-slug", Value: slug, Fix: "hug task show " + d.Task.ID}
	}
	path := filepath.ToSlash(filepath.Join(group, BuildName(r, d.Task.ID, slug)))
	if path == d.Task.Path {
		return s.finish(d, false)
	}
	dest, e := s.diskPath(path)
	if e != nil {
		return Result{}, e
	}
	source, e := s.diskPath(d.Task.Path)
	if e != nil {
		return Result{}, e
	}
	available, e := renameDestinationAvailable(source, dest)
	if e != nil {
		return Result{}, e
	}
	if !available {
		return Result{}, &Error{Kind: "path-exists", ID: d.Task.ID, Value: path, Fix: "hug task lint"}
	}
	if e := os.MkdirAll(filepath.Dir(dest), 0755); e != nil {
		return Result{}, e
	}
	s.Invalidate()
	if e := renameFile(source, dest); e != nil {
		return Result{}, e
	}
	// The task's files folder moves with it (group moves only; rank and
	// slug changes keep the same parent folder).
	if oldDir, newDir := filepath.Join(filepath.Dir(source), d.Task.ID), filepath.Join(filepath.Dir(dest), d.Task.ID); oldDir != newDir {
		if st, e := os.Lstat(oldDir); e == nil && st.IsDir() {
			if _, e := os.Lstat(newDir); e == nil {
				return Result{}, &Error{Kind: "path-exists", ID: d.Task.ID, Value: filepath.ToSlash(filepath.Join(group, d.Task.ID)), Fix: "hug task lint"}
			}
			if e := os.Rename(oldDir, newDir); e != nil {
				return Result{}, e
			}
		}
	}
	d.Task.Path = path
	d.Task.Group = filepath.ToSlash(filepath.Dir(path))
	if d.Task.Group == "." {
		d.Task.Group = ""
	}
	d.Task.Rank = r
	d.Task.Slug = slug
	s.Invalidate()
	if e := s.refreshWritten(oldPath, path); e != nil {
		return Result{}, e
	}
	return Result{Task: d.Task, Changed: true, Next: hint(d.Task), Renamed: []string{d.Task.ID}, Renames: []Rename{{ID: d.Task.ID, From: oldPath, To: path}}}, nil
}
func (s *Store) Move(ref string, o MoveOptions) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	plan, e := rankPlan(ts, d.Task, o)
	if e != nil {
		return Result{}, e
	}
	return s.applyRanks(plan, d.Task)
}
func (s *Store) MoveGroup(ref, group string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	return s.rename(d, group, d.Task.Rank, d.Task.Slug)
}
func (s *Store) Reslug(ref, slug string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	return s.rename(d, d.Task.Group, d.Task.Rank, slug)
}

// DescriptionSections are the sections after Goal in the task description
// pattern, in order (2026-11-16 vocabulary test: named sections +0.49 over
// none; "Relevant details" equals "Details", "Constraints" scores worse).
var DescriptionSections = []string{"Purpose", "Principle", "Relevant details"}

func patternSections(o NewOptions) string {
	out := ""
	for i, v := range []string{o.Purpose, o.Principle, o.Details} {
		if strings.TrimSpace(v) != "" {
			out += "\n## " + DescriptionSections[i] + "\n" + strings.TrimSpace(v) + "\n"
		}
	}
	return out
}
