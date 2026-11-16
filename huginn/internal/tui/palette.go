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

// The palette is a tree: views and actions at the top, every filter under
// "filter", then the field, then its values (2026-11-16, user).
var commands = []command{
	{"overview", "", "At a glance", false}, {"list", "", "Ranked task list", false}, {"board", "", "Status columns", false},
	{"lint", "", "Tasks with findings", false},
	{"filter", "field", "Narrow the tasks by a field", false},
	{"goto", "id", "Open a task by id", false},
	{"sort", "", "Order rows by fields: p priority, t type, ... then Enter", false},
	{"sort rank", "", "Back to rank order, the order moves change", true},
	{"clear", "", "Clear every filter", false}, {"help", "", "All keys and commands", false},
	{"todo", "", "Filter todo", true}, {"doing", "", "Filter in progress", true}, {"done", "", "Filter completed", true},
	{"blocked", "", "Filter unmet dependencies", true}, {"dropped", "", "Filter dropped", true},
	{"status", "status", "todo, doing, done, blocked, dropped", true}, {"type", "type", "feature, bug, chore, spike, docs", true},
	{"priority", "priority", "critical, high, medium, low", true}, {"group", "group", "Folder the task lives in", true},
	{"label", "label", "Free labels", true}, {"epic", "epic", "An epic and its children", true},
	{"assignee", "name", "Who is assigned", true},
}

// paletteFields are the branches under "filter", in display order.
var paletteFields = []string{"status", "type", "priority", "group", "label", "epic", "assignee"}

// completion is one palette row. branch rows open a further level; field is
// set on filter values so the row can show how many tasks it would leave.
type completion struct {
	value, description string
	branch             bool
	field, fieldValue  string
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
	case "status":
		for _, s := range append(append([]string{}, task.Statuses...), "blocked") {
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
		case "status":
			set[t.Status] = true
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
		case "label":
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
func (m model) completions() []completion {
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
	// values lists a field's values that start with arg, as filter rows.
	values := func(field, arg string) {
		for _, v := range m.values(field) {
			if strings.HasPrefix(strings.ToLower(v), strings.ToLower(arg)) {
				out = append(out, completion{value: "filter " + field + " " + v, field: field, fieldValue: v})
			}
		}
	}
	fields := func() {
		for _, f := range paletteFields {
			if strings.HasPrefix(f, name) {
				out = append(out, completion{value: "filter " + f + " ", description: describe(f), branch: true})
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
				out = append(out, completion{value: v, description: c.description, branch: c.argument != ""})
			}
		}
		// Typing a field name finds it without spelling out "filter".
		if name != "" && !strings.HasPrefix("filter", name) {
			fields()
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
// one field starts with it; anything else comes back unchanged.
func expandField(name string) string {
	if name == "" {
		return name
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

// unfilter drops the leading "filter", so "filter status todo" runs as
// "status todo", and "f s todo" does too.
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
	case "overview", "list", "board", "lint":
		m.setView(name)
	case "todo", "doing", "done", "dropped", "blocked":
		m.setFilter("status", name)
		m.setView("list")
	case "status", "type", "priority", "group", "label", "assignee", "epic":
		if name == "epic" {
			// Accept the bare id (or prefix) as well as "ID title".
			for _, v := range m.values("epic") {
				id, _, _ := strings.Cut(v, " ")
				if strings.HasPrefix(id, strings.ToLower(arg)) && arg != "" {
					arg = v
					break
				}
			}
		}
		if !contains(m.values(name), arg) {
			m.message = "unknown " + name + ": " + arg
			return false
		}
		m.setFilter(name, arg)
		m.setView("list")
	case "goto":
		t, err := task.Resolve(m.all, arg)
		if err != nil {
			m.message = err.Error()
			return false
		}
		m.openTask(t)
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
