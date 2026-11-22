package tui

import (
	"sort"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

type command struct {
	name, argument, description string
	// hidden commands still run when typed, but the palette lists them
	// under "filter" instead of at the top level.
	hidden bool
}

// The palette's global tree (05r3pg: the group after the context ones,
// palettetree.go): views under "view", every filter under "filter", then
// the field, then its values (2026-11-16, user). Hidden commands still run
// when typed.
var commands = []command{
	{"view", "view", "Overview, list, board, lint", false},
	{"overview", "", "At a glance", true}, {"list", "", "Ranked task list", true}, {"board", "", "State columns", true},
	{"lint", "", "Tasks with findings", true},
	{"filter", "field", "Narrow the tasks by a field", false},
	{"goto", "id", "Open a task by id", false},
	{"sort", "", "Order rows by fields, then Enter", false},
	{"sort rank", "", "Back to rank order, the order moves change", true},
	{"grouping", "", "Rows flat, by group, by epic, by tag: cycles", false},
	{"grouping flat", "", "Rows without headings", true}, {"grouping group", "", "A heading per group", true},
	{"grouping epic", "", "A heading per epic", true}, {"grouping tag", "", "A heading per tag (tag: first listed tag)", true},
	{"clear", "", "Back to every task, nothing narrowed", false},
	{"new", "title", "A new task, titled", false},
	{"rebalance", "", "Rekey rank prefixes; order unchanged", false},
	{"help", "", "All keys and commands", false},
	{"todo", "", "Filter todo", true}, {"doing", "", "Filter in progress", true}, {"done", "", "Filter completed", true},
	{"depends", "", "Filter tasks whose dependency is not done", true}, {"dropped", "", "Filter dropped", true},
	{"state", "state", "todo, doing, done, depends, dropped", true}, {"type", "type", "feature, bug, chore, spike, docs", true},
	{"priority", "priority", "critical, high, medium, low", true}, {"group", "group", "Folder the task lives in", true},
	{"tag", "tag", "One of the board's tags", true}, {"epic", "epic", "An epic and its children", true},
	{"assignee", "name", "Who is assigned", true},
}

// paletteFields are the branches under "filter", in display order.
var paletteFields = []string{"state", "type", "priority", "group", "tag", "epic", "assignee"}

// completion is one palette row. branch rows open a further level; field is
// set on filter values so the row can show how many tasks it would leave.
type completion struct {
	value, description string
	branch             bool
	field, fieldValue  string
	// group is the heading the row is under: "CHECK", "TASK", or "" for
	// GLOBAL; key the key that does the same outside the palette.
	group, key string
}

// globalKey is the key that does what a global command does, here.
func (m model) globalKey(name string) string {
	switch name {
	case "view":
		return "v"
	case "filter":
		return "f"
	case "goto":
		return "#"
	case "sort":
		return "s"
	case "grouping":
		if !m.detail {
			return "g"
		}
	case "clear":
		return "F"
	case "new":
		return "N"
	case "help":
		return "?"
	}
	return ""
}

// filterKey is f and the field's letter in the filter picker.
func filterKey(field string) string {
	for _, f := range filterFields {
		if f.name == field {
			return "f " + f.letter
		}
	}
	return ""
}

func describe(name string) string {
	for _, c := range commands {
		if c.name == name {
			return c.description
		}
	}
	return ""
}

func (m model) values(name string) []string {
	set := map[string]bool{}
	switch name {
	case "state":
		for _, s := range append(append([]string{}, task.StateList...), "depends") {
			set[s] = true
		}
	case "type":
		for _, s := range task.Types {
			set[s] = true
		}
	}
	for _, t := range m.all {
		switch name {
		case "goto":
			set[t.ID] = true
		case "type":
			set[t.Type] = true
			if t.Type == "" {
				set["(none)"] = true
			}
		case "state":
			set[t.State] = true
		case "priority":
			set[t.Priority] = true
			if t.Priority == "" {
				set["(none)"] = true
			}
		case "assignee":
			if len(t.Assignee) == 0 {
				set["(unassigned)"] = true
			}
			for _, owner := range t.Assignee {
				set[owner] = true
			}
		case "group":
			set[t.Group] = true
			if t.Group == "" {
				set["(root)"] = true
			}
		case "tag":
			for _, s := range t.Labels {
				set[s] = true
			}
		case "epic":
			if m.epics[strings.ToLower(t.ID)] {
				set[t.ID+" "+t.Title] = true
			}
		}
	}
	var out []string
	for s := range set {
		if s != "" {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// completions are the palette's rows for what is typed: the context
// groups first (CHECK, TASK), then the global tree (05r3pg).
func (m model) completions() []completion {
	raw := strings.TrimLeft(m.input.Value(), " ")
	var out []completion
	if _, isFilter := cutFilter(raw); !isFilter {
		out = m.contextCompletions(raw)
	}
	return append(out, m.globalCompletions()...)
}

func (m model) globalCompletions() []completion {
	value := strings.TrimLeft(m.input.Value(), " ")
	prefix := ""
	if rest, ok := cutFilter(value); ok {
		prefix, value = "filter ", strings.TrimLeft(rest, " ")
	}
	name, arg, hasArg := strings.Cut(value, " ")
	if prefix != "" && hasArg {
		// "f g " opens group: after filter, a field may be shortened to
		// any prefix that names one field only (2026-11-16, user).
		name = expandField(name)
	}
	var out []completion
	// values lists a field's values that start with arg, as filter rows;
	// arg with ! first lists their ! forms (7zg9mf).
	values := func(field, arg string) {
		base, not := task.Not(arg)
		mark := ""
		if not {
			mark = "!"
		}
		for _, v := range m.values(field) {
			if strings.HasPrefix(strings.ToLower(v), strings.ToLower(base)) {
				out = append(out, completion{value: "filter " + field + " " + mark + v, field: field, fieldValue: mark + v})
			}
		}
	}
	fields := func() {
		for _, f := range paletteFields {
			if strings.HasPrefix(f, name) {
				out = append(out, completion{value: "filter " + f + " ", description: describe(f), branch: true, key: filterKey(f)})
			}
		}
	}
	switch {
	case !hasArg && prefix != "":
		fields()
		// "filter done" finds the value without naming its field.
		if name != "" {
			for _, f := range paletteFields {
				values(f, name)
			}
		}
	case !hasArg:
		for _, c := range commands {
			if !c.hidden && strings.HasPrefix(c.name, name) {
				v := c.name
				if c.argument != "" {
					v += " "
				}
				out = append(out, completion{value: v, description: c.description, branch: c.argument != "" && c.name != "new", key: m.globalKey(c.name)})
			}
		}
		// Typing a field name finds it without spelling out "filter",
		// and a view's without "view" ("bo" is board, 05r3pg).
		if name != "" && !strings.HasPrefix("filter", name) {
			fields()
		}
		if name != "" && !strings.HasPrefix("view", name) {
			for _, v := range views {
				if strings.HasPrefix(v.name, name) {
					out = append(out, completion{value: v.name, description: describe(v.name), key: "v " + v.key})
				}
			}
		}
	case name == "view":
		for _, v := range views {
			if strings.HasPrefix(v.name, strings.TrimLeft(arg, " ")) {
				out = append(out, completion{value: "view " + v.name, description: describe(v.name), key: "v " + v.key})
			}
		}
	default:
		arg = strings.TrimLeft(arg, " ")
		if name != "goto" {
			values(name, arg)
			break
		}
		for _, v := range m.values(name) {
			if strings.HasPrefix(strings.ToLower(v), strings.ToLower(arg)) {
				c := completion{value: "goto " + v}
				for _, t := range m.all {
					if t.ID == v {
						c.description = t.Title
						break
					}
				}
				out = append(out, c)
			}
		}
	}
	return out
}
func (m *model) complete() {
	cs := m.completions()
	if len(cs) == 0 {
		m.message = "no completions; Esc closes palette"
		return
	}
	m.completion = min(m.completion, len(cs)-1)
	m.input.SetValue(cs[m.completion].value)
	m.input.CursorEnd()
	m.completion = 0
}

// cutFilter removes a leading "filter " or its short form "f ".
func cutFilter(value string) (string, bool) {
	for _, p := range []string{"filter ", "f "} {
		if rest, ok := strings.CutPrefix(value, p); ok {
			return rest, true
		}
	}
	return value, false
}

// expandField turns a shortened field name into the full one when exactly
// one field starts with it; anything else comes back unchanged. One letter
// is the filter picker's letter (f t is type, f l is tag), as before tag
// and type shared a first letter (txjkrm).
func expandField(name string) string {
	if name == "" {
		return name
	}
	for _, f := range filterFields {
		if f.letter == name {
			return f.name
		}
	}
	found := ""
	for _, f := range paletteFields {
		if f == name {
			return f
		}
		if strings.HasPrefix(f, name) {
			if found != "" {
				return name
			}
			found = f
		}
	}
	if found != "" {
		return found
	}
	return name
}

// unfilter drops the leading "filter", so "filter state todo" runs as
// "state todo", and "f s todo" does too.
func unfilter(value string) string {
	value = strings.TrimSpace(value)
	if rest, ok := cutFilter(value); ok {
		rest = strings.TrimSpace(rest)
		if name, arg, hasArg := strings.Cut(rest, " "); hasArg {
			return expandField(name) + " " + strings.TrimSpace(arg)
		}
		return rest
	}
	return value
}
func (m model) commandReady() bool {
	value := unfilter(m.input.Value())
	if value == "filter" || value == "f" {
		return false
	}
	name, arg, _ := strings.Cut(value, " ")
	for _, c := range commands {
		if c.name == name {
			return c.argument == "" || strings.TrimSpace(arg) != ""
		}
	}
	return false
}
func (m *model) executeCommand() bool {
	value := unfilter(m.input.Value())
	name, arg, _ := strings.Cut(value, " ")
	arg = strings.TrimSpace(arg)
	switch name {
	case "view":
		if _, ok := pickViewName(arg); !ok {
			m.message = "unknown view: " + arg + "; overview, list, board or lint"
			return false
		}
		m.setView(arg)
	case "new":
		m.requestAction("n", arg)
	case "rebalance":
		m.askRun([]string{"rebalance"}, "")
	case "overview", "list", "board", "lint":
		m.setView(name)
	case "todo", "doing", "done", "dropped", "depends":
		m.filterHere("state", name) // in the view shown (ht3cdx)
	case "state", "type", "priority", "group", "tag", "assignee", "epic":
		// ': state !done' keeps every task but those (7zg9mf): the value
		// is checked without its !, which the chip then shows.
		base, not := task.Not(arg)
		if name == "epic" {
			// Accept the bare id (or prefix) as well as "ID title".
			for _, v := range m.values("epic") {
				id, _, _ := strings.Cut(v, " ")
				if strings.HasPrefix(id, strings.ToLower(base)) && base != "" {
					base = v
					break
				}
			}
		}
		if !contains(m.values(name), base) {
			m.message = "unknown " + name + ": " + arg
			return false
		}
		if not {
			base = "!" + base
		}
		m.filterHere(name, base) // in the view shown (ht3cdx)
	case "goto":
		t, err := task.Resolve(m.all, arg)
		if err != nil {
			m.message = err.Error()
			return false
		}
		m.openTask(t)
	case "grouping":
		// kx2ndm: cycles flat -> group -> epic; an argument picks one.
		if arg == "" {
			arg = "next"
		}
		return m.setGrouping(arg)
	case "clear":
		m.clearFilters()
	case "help":
		m.help = true
		m.helpViewport.GotoTop()
	default:
		m.message = "unknown command: " + name
		return false
	}
	m.message = ""
	return true
}
