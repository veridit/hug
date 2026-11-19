package cli

import (
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// setSnapshot is what hug task set can change on a task, read before and
// after the write so the reply says what changed (zdgw60, reflection
// asr4t2: a set whose reply ended in a lint fix read as a failed write).
type setSnapshot struct {
	title, state, typ, priority, parent, assignee string
	tags                                          []string
	sections                                      map[string]string
	checks                                        int
}

func snapshotForSet(d *task.Document) setSnapshot {
	t := d.Task
	s := setSnapshot{title: t.Title, state: t.State, typ: t.Type, priority: t.Priority, parent: t.Parent,
		assignee: strings.Join(t.Assignee, ", "), tags: append([]string(nil), t.Labels...),
		sections: map[string]string{}, checks: len(t.Acceptance)}
	for _, name := range task.SectionFlags {
		s.sections[name] = strings.TrimSpace(d.SectionText(name))
	}
	return s
}

// setWrote is one line per field set changed, in a fixed order, then each
// check it added; "unchanged: ..." when the write changed nothing.
func setWrote(before, after setSnapshot, acceptance []task.Acceptance) []string {
	var out []string
	pair := func(name, a, b string) {
		if a != b {
			out = append(out, name+": "+orEmpty(a)+" -> "+orEmpty(b))
		}
	}
	pair("title", before.title, after.title)
	pair("state", before.state, after.state)
	pair("type", before.typ, after.typ)
	pair("priority", before.priority, after.priority)
	pair("assignee", before.assignee, after.assignee)
	if before.parent != after.parent {
		if after.parent == "" {
			out = append(out, "parent: none (was "+before.parent+")")
		} else {
			out = append(out, "parent: "+after.parent)
		}
	}
	if add, drop := tagDiff(before.tags, after.tags); len(add)+len(drop) > 0 {
		var parts []string
		for _, x := range add {
			parts = append(parts, "+ "+x)
		}
		for _, x := range drop {
			parts = append(parts, "- "+x)
		}
		out = append(out, "tags: "+strings.Join(parts, ", "))
	}
	for _, flag := range []string{"goal", "purpose", "principle", "details"} {
		name := task.SectionFlags[flag]
		if b, a := before.sections[name], after.sections[name]; a != b {
			out = append(out, strings.ToLower(name)+": "+clip(oneLineText(a), 120))
		}
	}
	for i := before.checks; i < after.checks && i < len(acceptance); i++ {
		out = append(out, "added check "+strconv.Itoa(i+1)+": "+oneLineText(acceptance[i].Text))
	}
	if len(out) == 0 {
		out = []string{"unchanged: the task already had these values"}
	}
	return out
}

func tagDiff(before, after []string) (add, drop []string) {
	in := func(xs []string, x string) bool {
		for _, y := range xs {
			if y == x {
				return true
			}
		}
		return false
	}
	for _, x := range after {
		if !in(before, x) {
			add = append(add, x)
		}
	}
	for _, x := range before {
		if !in(after, x) {
			drop = append(drop, x)
		}
	}
	return add, drop
}

func orEmpty(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func oneLineText(s string) string { return strings.Join(strings.Fields(s), " ") }
