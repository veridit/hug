package task

import "strings"

// Children lists the tasks whose parent resolves to t, in store order.
func Children(t Task, ts []Task) []Task {
	return ChildIndex(ts)[strings.ToLower(t.ID)]
}

// ChildIndex maps each epic id (lowercase) to its children, resolving every
// parent once. Callers with many tasks should build it once and reuse it
// (2026-11-16: per-task Children made hug task ls take 19 s on 2,500 tasks).
func ChildIndex(ts []Task) map[string][]Task {
	idx := map[string][]Task{}
	var ids map[string]string
	for _, c := range ts {
		if c.Parent == "" {
			continue
		}
		if ids == nil {
			ids = map[string]string{}
			for _, x := range ts {
				ids[strings.ToLower(x.ID)] = x.ID
				for _, a := range x.Aliases {
					if _, taken := ids[strings.ToLower(a)]; !taken {
						ids[strings.ToLower(a)] = x.ID
					}
				}
			}
		}
		if p, ok := ids[strings.ToLower(c.Parent)]; ok {
			idx[strings.ToLower(p)] = append(idx[strings.ToLower(p)], c)
		}
	}
	return idx
}

// OpenChildren are children that are neither done nor dropped.
func OpenChildren(t Task, ts []Task) []Task {
	var out []Task
	for _, c := range Children(t, ts) {
		if c.Status != "done" && c.Status != "dropped" {
			out = append(out, c)
		}
	}
	return out
}

// IsEpic: a task with at least one child.
func IsEpic(t Task, ts []Task) bool {
	for _, c := range ts {
		if c.Parent != "" && (strings.EqualFold(c.Parent, t.ID) || containsFold(t.Aliases, c.Parent)) {
			return true
		}
	}
	return false
}

func containsFold(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

// parentFor validates a new parent for t and returns the value to store: the
// parent's id, or "" to clear ("", "none", "-"). Three fixed levels: the
// parent may not have a parent, and t may not have children.
func parentFor(t Task, ts []Task, ref string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(ref)) {
	case "", "none", "-":
		return "", nil
	}
	p, e := Resolve(ts, ref)
	if e != nil {
		return "", e
	}
	if p.ID == t.ID {
		return "", &Error{Kind: "invalid-parent", ID: t.ID, Value: "a task cannot be its own parent", Fix: "hug task set " + t.ID + " --parent EPIC"}
	}
	if p.Parent != "" {
		return "", &Error{Kind: "invalid-parent", ID: t.ID, Value: p.ID + " already has parent " + p.Parent + "; an epic has no parent (epic > task > acceptance item)", Fix: "hug task set " + t.ID + " --parent " + p.Parent}
	}
	if kids := Children(t, ts); len(kids) > 0 {
		return "", &Error{Kind: "invalid-parent", ID: t.ID, Value: t.ID + " is an epic with " + plural(len(kids), "child", "children") + "; an epic has no parent", Fix: "hug task show " + t.ID}
	}
	return p.ID, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return itoa(n) + " " + many
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// parentFindings: a parent that does not resolve, a parent that itself has a
// parent (too deep), or a self-parent.
func parentFindings(tasks []Task) []Finding {
	var out []Finding
	for _, t := range tasks {
		if t.Parent == "" {
			continue
		}
		p, e := ResolveDependency(tasks, t.Parent)
		switch {
		case e != nil:
			out = append(out, Finding{Path: t.Path, Line: 1, Problem: "missing parent " + t.Parent})
		case p.ID == t.ID:
			out = append(out, Finding{Path: t.Path, Line: 1, Problem: "task is its own parent"})
		case p.Parent != "":
			out = append(out, Finding{Path: t.Path, Line: 1, Problem: "parent " + p.ID + " has parent " + p.Parent + ": only epic > task (fix: hug task set " + t.ID + " --parent " + p.Parent + ")"})
		}
	}
	return out
}
