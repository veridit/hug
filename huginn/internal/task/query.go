package task

import (
	"fmt"
	"github.com/veridit/hug/huginn/internal/rank"
	"sort"
	"strings"
)

type Match struct {
	Slug string `json:"slug"`
	ID   string `json:"id"`
}
type Unchecked struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
	Fix   string `json:"fix"`
}
type Error struct {
	Kind         string      `json:"kind"`
	ID           string      `json:"id,omitempty"`
	Field        string      `json:"field,omitempty"`
	Value        string      `json:"value,omitempty"`
	Closest      string      `json:"closest,omitempty"`
	Fix          string      `json:"fix,omitempty"`
	Valid        []string    `json:"valid,omitempty"`
	Matches      []Match     `json:"matches,omitempty"`
	Dependencies []Task      `json:"dependencies,omitempty"`
	Unchecked    []Unchecked `json:"unchecked,omitempty"`
	Cycle        []string    `json:"cycle,omitempty"`
	// Waits are the conditions that do not hold (16dxsr).
	Waits []WaitDetail `json:"waits,omitempty"`
	// Problems: every problem a write found, all at once (fkpyg3).
	Problems []Problem `json:"problems,omitempty"`
	// Fixes are alternative ways out, each one fix: line (n2hq8p); Fix
	// is the first.
	Fixes []string `json:"fixes,omitempty"`
	// Waiting: next --mine with nothing to work on lists my waits, one
	// line each (4tycwd).
	Waiting []string `json:"waiting,omitempty"`
}

func (e *Error) Error() string {
	msg := e.Kind
	if e.ID != "" {
		msg += " " + e.ID
	}
	if e.Field != "" {
		msg += " " + e.Field
	}
	if e.Value != "" {
		msg += ": " + e.Value
	}
	if len(e.Valid) > 0 {
		msg += "; valid: " + strings.Join(e.Valid, ", ") + "; closest: " + e.Closest
	}
	for _, m := range e.Matches {
		msg += "\n" + m.Slug + "  " + m.ID
	}
	for _, d := range e.Dependencies {
		msg += "\n" + d.ID + " " + d.Title
	}
	for _, w := range e.Waiting {
		msg += "\n" + w
	}
	for _, a := range e.Unchecked {
		msg += fmt.Sprintf("\n%d. %s\nfix: %s", a.Index, a.Text, a.Fix)
	}
	if len(e.Fixes) > 0 {
		for _, f := range e.Fixes {
			msg += "\nfix: " + f
		}
	} else if e.Fix != "" {
		msg += "\nfix: " + e.Fix
	}
	return msg
}
func (e *Error) ExitCode() int {
	switch e.Kind {
	case "blocked", "unchecked-acceptance", "busy":
		return 3
	case "nothing-matches":
		return 4
	default:
		return 2
	}
}
func Invalid(field, value string, valid []string) *Error {
	closest := ""
	best := int(^uint(0) >> 1)
	for _, s := range valid {
		n := distance(strings.ToLower(value), s)
		if n < best {
			best = n
			closest = s
		}
	}
	return &Error{Kind: "invalid-value", Field: field, Value: value, Valid: valid, Closest: closest, Fix: fmt.Sprintf("hug task set <id> --%s %s", field, closest)}
}
func invalidFor(id, field, value string, valid []string) *Error {
	e := Invalid(field, value, valid)
	e.ID = id
	e.Fix = strings.ReplaceAll(e.Fix, "<id>", id)
	if field == "field" {
		e.Fix = "hug task set " + id + " --type feature"
	}
	return e
}
func distance(a, b string) int {
	p := make([]int, len(b)+1)
	for j := range p {
		p[j] = j
	}
	for i := range a {
		q := make([]int, len(b)+1)
		q[0] = i + 1
		for j := range b {
			cost := 0
			if a[i] != b[j] {
				cost = 1
			}
			q[j+1] = min(q[j]+1, p[j+1]+1, p[j]+cost)
		}
		p = q
	}
	return p[len(b)]
}

// BareRef is ref without a leading "task:": running text writes a
// reference as task:ID, and a command given that form finds the same task
// as the bare id (xxvm6t).
func BareRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if len(ref) > 5 && strings.EqualFold(ref[:5], "task:") {
		ref = ref[5:]
	}
	// A task's file name, in either layout, is a ref to it (w40cmk check
	// 4): an old link to 00i.6ephqk.task-state-stage-hard.md, or to
	// later/6ephqk.slug, finds 6ephqk after the move to 6/6ephqk.md.
	name := ref
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		name += ".md"
	}
	// IDOf needs a dot (rank.id.slug) or a slash (C/ID), so a bare id or
	// slug never gets here.
	if i, ok := IDOf(name); ok {
		return i
	}
	return ref
}

func Resolve(tasks []Task, ref string) (Task, error) {
	ref = strings.ToLower(BareRef(ref))
	var ids []Task
	for _, t := range tasks {
		if strings.EqualFold(t.ID, ref) {
			ids = append(ids, t)
		}
	}
	if len(ids) == 1 {
		return ids[0], nil
	}
	var exact, prefix []Task
	for _, t := range tasks {
		hit := strings.EqualFold(t.ID, ref) || strings.EqualFold(t.Slug, ref)
		for _, a := range t.Aliases {
			hit = hit || strings.EqualFold(a, ref)
		}
		if hit {
			exact = append(exact, t)
		} else if ref != "" && strings.HasPrefix(strings.ToLower(t.ID), ref) {
			prefix = append(prefix, t)
		}
	}
	if len(ids) > 1 {
		exact = ids
	}
	if len(exact) == 0 {
		exact = prefix
	}
	// A bare number is the natural shorthand for imported ids: "2035" finds
	// TASK-2035 when nothing else matched (seid-lang feedback, 2026-11-09).
	if len(exact) == 0 && ref != "" && strings.Trim(ref, "0123456789.") == "" {
		for _, t := range tasks {
			for _, a := range t.Aliases {
				if i := strings.LastIndex(a, "-"); i > 0 && strings.EqualFold(a[i+1:], ref) {
					exact = append(exact, t)
					break
				}
			}
		}
	}
	if len(exact) == 1 {
		return exact[0], nil
	}
	e := &Error{Kind: "not-found", Value: ref, Fix: "hug task ls"}
	if len(exact) > 1 {
		e.Kind = "ambiguous"
		sort.Slice(exact, func(i, j int) bool { return exact[i].ID < exact[j].ID })
		for _, t := range exact {
			e.Matches = append(e.Matches, Match{t.Slug, t.ID})
		}
	}
	return Task{}, e
}

// ResolveDependency deliberately accepts only exact ids and aliases, never slugs or prefixes.
func ResolveDependency(tasks []Task, ref string) (Task, error) {
	ref = BareRef(ref)
	if x := refIndexFor(tasks, false); x != nil {
		t, e, state := x.resolve(tasks, ref)
		if state == refStale {
			t, e, state = refIndexFor(tasks, true).resolve(tasks, ref)
		}
		if state == refHit {
			return t, e
		}
	}
	var ids []Task
	for _, t := range tasks {
		if strings.EqualFold(t.ID, ref) {
			ids = append(ids, t)
		}
	}
	if len(ids) == 1 {
		return ids[0], nil
	}
	if len(ids) > 1 {
		return Task{}, &Error{Kind: "duplicate-id", ID: ref, Fix: "hug task lint"}
	}
	var found []Task
	for _, t := range tasks {
		hit := strings.EqualFold(t.ID, ref)
		for _, a := range t.Aliases {
			hit = hit || strings.EqualFold(a, ref)
		}
		if hit {
			found = append(found, t)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return Task{}, &Error{Kind: "dangling-dependency", Value: "no task has the id " + ref, Fix: "hug task ls"}
}

// missingDependency is the finding for a dependency line whose reference
// matches no task: it names the reference and how to remove it.
func missingDependency(t Task, ref string, line int) Finding {
	who := t.ID
	if who == "" {
		who = "ID"
	}
	return Finding{Path: t.Path, Line: line,
		Problem: "depends on " + ref + ", but no task has that id or alias; remove the line, or write an existing task's id",
		Fix:     "hug task dep " + who + " --remove " + ref}
}
func Blocked(t Task, tasks []Task) []Task {
	var out []Task
	for _, d := range t.Dependencies {
		x, e := ResolveDependency(tasks, d.Ref)
		if e != nil {
			out = append(out, Task{ID: d.Ref, Title: "unresolved dependency"})
		} else if !Closed(x, tasks) {
			out = append(out, x)
		}
	}
	return out
}
func Sort(tasks []Task) {
	sort.Slice(tasks, func(i, j int) bool { return rank.Less(tasks[i].Rank, tasks[i].ID, tasks[j].Rank, tasks[j].ID) })
}

// MaxTitle is the longest open-task title lint accepts. Short titles keep
// lists, boards and file names readable; the detail belongs in the Goal
// (2026-11-16, user approved).
const MaxTitle = 80

func Lint(docs []*Document) []Finding {
	var out []Finding
	var tasks []Task
	counts := map[string]int{}
	paths := map[string][]string{}
	for _, d := range docs {
		counts[d.Task.ID]++
		key := strings.ToLower(d.Task.Path)
		paths[key] = append(paths[key], d.Task.Path)
	}
	for _, matches := range paths {
		sort.Strings(matches)
	}
	for _, d := range docs {
		out = append(out, d.Findings...)
		t := d.Task
		tasks = append(tasks, t)
		if matches := paths[strings.ToLower(t.Path)]; len(matches) > 1 {
			out = append(out, Finding{Path: t.Path, Line: 1, Problem: "case-insensitive filename collision: " + strings.Join(matches, ", ")})
		}
		if t.ID != "" && counts[t.ID] > 1 {
			out = append(out, Finding{Path: t.Path, Line: 1, Problem: "duplicate id " + t.ID})
		}
	}
	adj := map[string][]string{}
	for _, t := range tasks {
		for _, dep := range t.Dependencies {
			v, e := ResolveDependency(tasks, dep.Ref)
			if e != nil {
				out = append(out, missingDependency(t, dep.Ref, dep.Line))
			} else {
				adj[t.ID] = append(adj[t.ID], v.ID)
			}
		}
	}
	for _, t := range tasks {
		visited := map[string]bool{}
		var walk func(string) bool
		walk = func(v string) bool {
			if visited[v] {
				return false
			}
			visited[v] = true
			for _, w := range adj[v] {
				if w == t.ID || walk(w) {
					return true
				}
			}
			return false
		}
		if walk(t.ID) {
			out = append(out, Finding{Path: t.Path, Line: 1, Problem: "dependency cycle involving " + t.ID})
		}
	}
	out = append(out, parentFindings(tasks)...)
	out = append(out, derivedFindings(tasks)...)
	// A waiting task with nothing left is asked to finish (bg1rd8).
	out = append(out, ChecksHoldFindings(tasks, Now())...)
	// The description pattern holds for every open task (fkpyg3, Jørgen
	// 2026-11-17: the old 2026-11-16 cutoff hid it on most of a board).
	// Done and dropped tasks are history. Same check list as hug task new;
	// missing Goal, title and Acceptance come from Parse already.
	for _, d := range docs {
		t := d.Task
		if !Pattern(t) {
			continue
		}
		for _, p := range DescriptionProblems(t.Title, "x", contains(t.Sections, "Purpose"), contains(t.Sections, "Principle"), 1) {
			if p.Flag == "" {
				continue // missing title: Parse reports it as missing field title
			}
			out = append(out, Finding{Path: t.Path, Line: 1, Problem: p.Text, Fix: "hug task set " + t.ID + " " + p.Flag})
		}
		// A Goal over one sentence or 140 characters (0pr03h): a warning
		// with the fix; lint never rewrites it.
		if p, ok := GoalProblem(t.Goal); ok {
			out = append(out, Finding{Path: t.Path, Line: 1, Problem: p.Text, Fix: "hug task set " + t.ID + " " + p.Flag})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Problem < out[j].Problem
	})
	return out
}
