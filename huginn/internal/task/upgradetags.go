package task

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// hug upgrade --tags (txjkrm check 5): cleans a board's labels against its
// hug.toml tags list, after a person has read the diff. On open tasks:
//   - a variant of a listed tag (case, - and _, or a [tag_merges] line)
//     becomes the listed tag;
//   - a label the task's type or priority already says goes; one that
//     disagrees with the field stays, the only record of it, and is listed;
//   - parked, paused or deferred on a todo task makes it deferred, the label
//     its reason (approving the diff is the person's decision);
//   - draft and other stage words on a task go (hug works the stage out);
//   - any other label not on the list goes.
// Closed tasks are history (crab, t01kqm): only merges reach them, which
// lose nothing. design is a tag a board may list; it is never a stage here,
// since the label asks no question.

// TagLabelType maps labels that say a type to it.
var TagLabelType = map[string]string{"bug": "bug", "feature": "feature", "chore": "chore", "spike": "spike", "docs": "docs"}

// TagLabelPriority maps labels that say a priority to it.
var TagLabelPriority = map[string]string{"low-pri": "low", "lowpri": "low", "urgent": "high", "high-pri": "high", "highpri": "high"}

// tagLabelDefer are labels that, on a todo task, mean deferred.
var tagLabelDefer = map[string]bool{"parked": true, "paused": true, "deferred": true}

// tagLabelStage are stage words hug works out itself: they go.
var tagLabelStage = map[string]bool{"draft": true}

// TagChange is one task's new labels and what the upgrade does to it.
type TagChange struct {
	ID, Path string
	From, To []string
	Defer    string // the deferred_reason to set, "" for none
}

// TagPlan is what hug upgrade --tags would do.
type TagPlan struct {
	Changes  []TagChange
	List     []string       // the list a board without one gets ("" when it has one)
	Merged   map[string]int // "OLD -> NEW": count
	Field    map[string]int // "label (type bug)": count
	Deferred map[string][]string
	Stage    map[string]int
	Dropped  map[string]int
	Disagree []string // "ID label (type T)"
	Closed   int      // closed tasks left as they are, beyond merges
	Design   int      // open tasks that keep a listed design tag
}

// PlanTags works out the cleanup of ts against list. When list has no tags
// it proposes one: every tag on min or more tasks after variant merges.
func PlanTags(list TagList, ts []Task, min int) TagPlan {
	p := TagPlan{Merged: map[string]int{}, Field: map[string]int{}, Deferred: map[string][]string{}, Stage: map[string]int{}, Dropped: map[string]int{}}
	names := list.List
	if len(names) == 0 {
		names = proposeTags(ts, min)
		p.List = names
	}
	listed := map[string]bool{}
	byKey := map[string]string{}
	for _, n := range names {
		listed[n] = true
		if _, ok := byKey[tagKey(n)]; !ok {
			byKey[tagKey(n)] = n
		}
	}
	canon := func(l string) (string, bool) {
		if listed[l] {
			return l, true
		}
		if w, ok := list.Merges[l]; ok && listed[w] {
			return w, true
		}
		w, ok := byKey[tagKey(l)]
		return w, ok
	}
	closedLeft := map[string]bool{}
	for _, t := range ts {
		open := t.State != "done" && t.State != "dropped"
		var to []string
		seen := map[string]bool{}
		keep := func(l string) {
			if !seen[l] {
				seen[l] = true
				to = append(to, l)
			}
		}
		reason := ""
		for _, l := range t.Labels {
			if c, ok := canon(l); ok {
				if c != l {
					p.Merged[l+" -> "+c]++
				}
				keep(c)
				continue
			}
			if !open {
				keep(l)
				closedLeft[t.ID] = true
				continue
			}
			k := strings.ToLower(l)
			switch {
			case TagLabelType[k] != "":
				if t.Type == TagLabelType[k] {
					p.Field[l+" (type "+t.Type+")"]++
				} else {
					keep(l)
					p.Disagree = append(p.Disagree, fmt.Sprintf("%s %s (type %s)", t.ID, l, t.Type))
				}
			case TagLabelPriority[k] != "":
				if t.Priority == TagLabelPriority[k] {
					p.Field[l+" (priority "+t.Priority+")"]++
				} else {
					keep(l)
					pr := t.Priority
					if pr == "" {
						pr = "none"
					}
					p.Disagree = append(p.Disagree, fmt.Sprintf("%s %s (priority %s)", t.ID, l, pr))
				}
			case tagLabelDefer[k] && t.State == "todo":
				if strings.TrimSpace(t.Fields["deferred_reason"]) == "" && reason == "" {
					reason = "label " + l + " (hug upgrade --tags)"
				}
				p.Deferred[l] = append(p.Deferred[l], t.ID)
			case tagLabelDefer[k] || tagLabelStage[k]:
				p.Stage[l]++
			default:
				p.Dropped[l]++
			}
		}
		if open && seen["design"] {
			p.Design++
		}
		if to == nil {
			to = []string{}
		}
		if !sameStrings(to, t.Labels) || reason != "" {
			p.Changes = append(p.Changes, TagChange{ID: t.ID, Path: t.Path, From: t.Labels, To: to, Defer: reason})
		}
	}
	p.Closed = len(closedLeft)
	return p
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// proposeTags is every label on min or more tasks once variants (case, -
// and _) count as one, under its most used spelling, most used first.
// Labels a field says are never proposed.
func proposeTags(ts []Task, min int) []string {
	tasks := map[string]map[string]bool{}
	spell := map[string]map[string]int{}
	for _, t := range ts {
		for _, l := range t.Labels {
			k := strings.ToLower(l)
			if TagLabelType[k] != "" || TagLabelPriority[k] != "" || tagLabelDefer[k] || tagLabelStage[k] {
				continue
			}
			key := tagKey(l)
			if tasks[key] == nil {
				tasks[key], spell[key] = map[string]bool{}, map[string]int{}
			}
			tasks[key][t.ID] = true
			spell[key][l]++
		}
	}
	type kv struct {
		name string
		n    int
	}
	var out []kv
	for key, ids := range tasks {
		if len(ids) < min {
			continue
		}
		best, bn := "", -1
		for s, n := range spell[key] {
			if n > bn || n == bn && s < best {
				best, bn = s, n
			}
		}
		out = append(out, kv{best, len(ids)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].name < out[j].name
	})
	names := make([]string, len(out))
	for i, x := range out {
		names[i] = x.name
	}
	return names
}

// Empty says the plan changes nothing.
func (p TagPlan) Empty() bool { return len(p.Changes) == 0 && len(p.List) == 0 }

// Lines are the plan in counts, one line per kind, for the preview.
func (p TagPlan) Lines(min int) []string {
	var out []string
	if len(p.List) > 0 {
		q := make([]string, len(p.List))
		for i, n := range p.List {
			q[i] = strconv.Quote(n)
		}
		noun := "tags"
		if len(p.List) == 1 {
			noun = "tag"
		}
		out = append(out, fmt.Sprintf("hug.toml gets tags = [%s] (%d %s on %d+ tasks after merging)", strings.Join(q, ", "), len(p.List), noun, min))
	}
	if n, s := counted(p.Merged); n > 0 {
		out = append(out, fmt.Sprintf("%s into a listed tag: %s", plural(n, "label merges", "labels merge"), s))
	}
	if n, s := counted(p.Field); n > 0 {
		out = append(out, fmt.Sprintf("%s, the task's type or priority already says it: %s", plural(n, "label goes", "labels go"), s))
	}
	if len(p.Deferred) > 0 {
		n, parts := 0, []string{}
		for _, l := range sortedKeys(p.Deferred) {
			n += len(p.Deferred[l])
			parts = append(parts, fmt.Sprintf("%s %d (%s)", l, len(p.Deferred[l]), strings.Join(p.Deferred[l], ", ")))
		}
		out = append(out, fmt.Sprintf("%s: %s", plural(n, "todo task deferred by its label", "todo tasks deferred by their label"), strings.Join(parts, ", ")))
	}
	if n, s := counted(p.Stage); n > 0 {
		out = append(out, fmt.Sprintf("%s, hug works the stage out: %s", plural(n, "stage label goes", "stage labels go"), s))
	}
	if n, s := counted(p.Dropped); n > 0 {
		out = append(out, fmt.Sprintf("%s: %s", plural(n, "label not on the list goes", "labels not on the list go"), s))
	}
	if len(p.Disagree) > 0 {
		n := len(p.Disagree)
		out = append(out, fmt.Sprintf("%d %s the task's field and %s, the only record of it: %s; fix: hug task set ID --type TYPE (or --priority), then hug task set ID --tag LABEL=", n, pick(n, "label disagrees with", "labels disagree with"), pick(n, "stays", "stay"), strings.Join(p.Disagree, ", ")))
	}
	if p.Design > 0 {
		out = append(out, fmt.Sprintf("design: %d open %s the tag; to make one a design stage: hug task design ID \"question\" --for NAME", p.Design, pick(p.Design, "task keeps", "tasks keep")))
	}
	if p.Closed > 0 {
		out = append(out, fmt.Sprintf("%d done or dropped %s: closed tasks are history, only merges reach them", p.Closed, pick(p.Closed, "task keeps its other labels", "tasks keep their other labels")))
	}
	return out
}

func pick(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// Diff is one line per changed task: its labels before and after.
func (p TagPlan) Diff() string {
	var b strings.Builder
	for _, c := range p.Changes {
		fmt.Fprintf(&b, "%s: labels [%s] -> [%s]", c.ID, strings.Join(c.From, ", "), strings.Join(c.To, ", "))
		if c.Defer != "" {
			fmt.Fprintf(&b, ", deferred_reason %q", c.Defer)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func counted(m map[string]int) (int, string) {
	n := 0
	var parts []string
	for _, k := range sortedKeys(m) {
		n += m[k]
		parts = append(parts, fmt.Sprintf("%s %d", k, m[k]))
	}
	return n, strings.Join(parts, ", ")
}

// ApplyTags writes plan's changes: labels, and deferred_reason where the
// plan defers. It returns the changed task paths, for one commit.
func (s *Store) ApplyTags(plan TagPlan) ([]string, error) {
	var paths []string
	for _, c := range plan.Changes {
		d, ts, e := s.load(c.ID)
		if e != nil {
			return paths, e
		}
		if e = d.SetField("labels", c.To); e != nil {
			return paths, e
		}
		if c.Defer != "" {
			if e = d.SetField("deferred_reason", c.Defer); e != nil {
				return paths, e
			}
			d.History(stamp() + " todo -> todo/deferred (" + c.Defer + ")")
			d.Task.Fields["deferred_reason"] = c.Defer
			if e = settle(d, ts); e != nil {
				return paths, e
			}
		}
		path, e := s.diskPath(d.Task.Path)
		if e != nil {
			return paths, e
		}
		s.Invalidate()
		if e = AtomicWrite(path, d.Bytes()); e != nil {
			return paths, e
		}
		paths = append(paths, d.Task.Path)
	}
	return paths, nil
}

// WithTags is hug.toml's text with a top-level tags = [...] for a board
// that has none: before the first [section], since a key under one belongs
// to it.
func WithTags(text string, names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = strconv.Quote(n)
	}
	entry := "tags = [" + strings.Join(q, ", ") + "]"
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if strings.TrimSpace(text) == "" {
		return entry + "\n"
	}
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "[") {
			head := append(append([]string(nil), lines[:i]...), entry, "")
			return strings.Join(append(head, lines[i:]...), "\n") + "\n"
		}
	}
	return strings.Join(append(lines, entry), "\n") + "\n"
}
