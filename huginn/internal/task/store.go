package task

import (
	"encoding/json"
	"fmt"
	"github.com/veridit/hug/huginn/internal/id"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Result struct {
	Task     Task     `json:"task"`
	Changed  bool     `json:"changed"`
	Next     string   `json:"next"`
	Warnings []string `json:"warnings,omitempty"`
	// Guide: what to do now, at this state change (see Guide).
	Guide   []string `json:"guide,omitempty"`
	Renamed []string `json:"renamed,omitempty"`
	Renames []Rename `json:"renames,omitempty"`
}
type NewOptions struct {
	Title, Goal, Group, Type, Priority string
	// Purpose, Principle and Details become their own sections after Goal
	// (the task description pattern; docs/task-vocabulary-test.md).
	Purpose, Principle, Details string
	Acceptance, Deps, Labels    []string
	Top                         bool
	After                       string
}
type Filter struct {
	Type, Priority, Label, Group, Mine string
	Status                             string
	// Epic, when set, is an epic id: keep that epic and its children.
	Epic string
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
	// Snapshot keeps the board Refresh returned until this Store writes a
	// task file or Invalidate is called, so one command reads the board
	// once, not once per helper (44w1ev: ls at 10,000 tasks walked the
	// files six times). Set it only for one short-lived command; the TUI,
	// which waits while others write, leaves it off.
	Snapshot bool
	board    []Task
	held     bool
}

// Invalidate drops the kept board: the next Tasks reads the files again.
// Call it after anything other than this Store changed task files (an
// editor, a pull, an import).
func (s *Store) Invalidate() { s.board, s.held = nil, false }

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
		b, e := os.ReadFile(path)
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
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, nil, e
	}
	return Parse(t.Path, b), ts, nil
}
func hint(t Task) string {
	switch t.Status {
	case "todo":
		return "hug task take " + t.ID
	case "doing":
		for n, a := range t.Acceptance {
			if !a.Settled() {
				return fmt.Sprintf("hug task check %s %d", t.ID, n+1)
			}
		}
		return "hug task done " + t.ID
	default:
		return "hug task next"
	}
}
func (s *Store) finish(d *Document, changed bool) (Result, error) {
	if changed {
		path, e := s.diskPath(d.Task.Path)
		if e != nil {
			return Result{}, e
		}
		s.Invalidate()
		if e := AtomicWrite(path, d.Bytes()); e != nil {
			return Result{}, e
		}
		if s.Refresh != nil {
			if _, e := s.Tasks(); e != nil {
				return Result{}, e
			}
		}
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
	if strings.TrimSpace(o.Title) == "" || strings.TrimSpace(o.Goal) == "" || len(o.Acceptance) == 0 {
		return Result{}, &Error{Kind: "missing-required", Fix: "hug task new \"title\" --goal \"outcome\" --accept \"observable check\""}
	}
	for _, a := range o.Acceptance {
		if strings.TrimSpace(a) == "" {
			return Result{}, &Error{Kind: "empty-acceptance", Fix: "hug task new \"title\" --goal \"outcome\" --accept \"observable check\""}
		}
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
	slug := Slug(o.Title)
	path := filepath.ToSlash(filepath.Join(o.Group, BuildName(r, i, slug)))
	q := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	today := time.Now().Format("2006-01-02")
	body := "---\nid: " + i + "\ntitle: " + q(o.Title) + "\nstatus: todo\ntype: " + o.Type + "\npriority: " + q(o.Priority) + "\nassignee: []\nlabels: "
	labels := []string{}
	for _, l := range o.Labels {
		labels = append(labels, q(l))
	}
	body += "[" + strings.Join(labels, ", ") + "]\ncreated: " + today + "\nupdated: " + today + "\n---\n\n## Goal\n" + o.Goal + "\n" + patternSections(o) + "\n## Depends on\n"
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
	if field == "status" {
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
func (s *Store) Check(ref string, n int) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	before := string(d.Bytes())
	if e = d.Check(n); e != nil {
		return Result{}, e
	}
	return s.finish(d, before != string(d.Bytes()))
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
		e = d.Strike(n, reason, who, time.Now().Format("2006-01-02"))
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
		name = model
	}
	return s.transitionAt(ref, "doing", name, model, at)
}
func (s *Store) Done(ref string) (Result, error) { return s.Transition(ref, "done", "") }
func (s *Store) Drop(ref string) (Result, error) { return s.Transition(ref, "dropped", "") }
func (s *Store) Transition(ref, status, user string) (Result, error) {
	return s.transition(ref, status, user, "")
}
func (s *Store) transition(ref, status, user, model string) (Result, error) {
	return s.transitionAt(ref, status, user, model, Place{})
}

func (s *Store) transitionAt(ref, status, user, model string, at Place) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if !contains(Statuses, status) {
		return Result{}, invalidFor(d.Task.ID, "status", status, Statuses)
	}
	if d.Task.Status == status && (status != "doing" || user == "" || contains(d.Task.Assignee, user)) && (model == "" || d.Task.Model == model) && (at.Branch == "" || d.Task.Branch == at.Branch) && (at.Worktree == "" || d.Task.Worktree == at.Worktree) {
		return s.finish(d, false)
	}
	if status == "doing" && user != "" && len(d.Task.Assignee) > 0 && !contains(d.Task.Assignee, user) {
		return Result{}, &Error{Kind: "taken", ID: d.Task.ID, Value: strings.Join(d.Task.Assignee, ", "), Fix: "hug task next"}
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
				return Result{}, &Error{Kind: "blocked", ID: d.Task.ID, Dependencies: blocked, Fix: "hug task show " + blocked[0].ID}
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
	if e = d.SetField("status", status); e != nil {
		return Result{}, e
	}
	if status == "doing" && user != "" {
		if e = d.SetField("assignee", []string{user}); e != nil {
			return Result{}, e
		}
	}
	if status == "doing" && model != "" {
		if e = d.SetField("model", model); e != nil {
			return Result{}, e
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
	r, e := s.finish(d, true)
	r.Warnings = warnings
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
	if f.Type != "" && !contains(Types, f.Type) {
		return Result{}, Invalid("type", f.Type, Types)
	}
	ts := append([]Task(nil), tasks...)
	Sort(ts)
	epics := ChildIndex(tasks)
	if f.Mine != "" {
		sort.SliceStable(ts, func(i, j int) bool { return ts[i].Status == "doing" && ts[j].Status != "doing" })
	}
	for _, t := range ts {
		if (t.Status != "todo" && !(f.Mine != "" && t.Status == "doing")) || Waiting(t, tasks) {
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
		if f.Type != "" && t.Type != f.Type || f.Priority != "" && t.Priority != f.Priority || f.Label != "" && !contains(t.Labels, f.Label) || f.Group != "" && t.Group != f.Group {
			continue
		}
		return Result{Task: t, Next: hint(t)}, nil
	}
	return Result{}, &Error{Kind: "nothing-matches", Fix: "hug task ls"}
}
func (s *Store) rename(d *Document, group, r, slug string) (Result, error) {
	oldPath := d.Task.Path
	if e := safeGroup(group); e != nil {
		return Result{}, e
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
	if s.Refresh != nil {
		if _, e := s.Tasks(); e != nil {
			return Result{}, e
		}
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
