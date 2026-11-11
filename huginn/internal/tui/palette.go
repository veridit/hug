package tui

import (
	"sort"
	"strings"

	"github.com/veridit/huginn/internal/task"
)

type command struct{ name, argument, description string }

var commands = []command{
	{"overview", "", "At a glance"}, {"list", "", "Ranked task list"}, {"board", "", "Status columns"},
	{"todo", "", "Filter todo"}, {"doing", "", "Filter in progress"}, {"done", "", "Filter completed"},
	{"blocked", "", "Filter unmet dependencies"}, {"dropped", "", "Filter dropped"},
	{"status", "status", "Filter status"}, {"type", "type", "Filter task type"}, {"group", "group", "Filter group"},
	{"label", "label", "Filter label"}, {"priority", "priority", "Filter priority"}, {"owner", "owner", "Filter owner"}, {"goto", "id", "Open a task by id"}, {"lint", "", "Tasks with findings"},
	{"clear", "", "Clear every filter"}, {"help", "", "All keys and commands"},
}

type completion struct{ value, description string }

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
		case "owner":
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
	name, arg, hasArg := strings.Cut(value, " ")
	var out []completion
	if !hasArg {
		for _, c := range commands {
			if strings.HasPrefix(c.name, name) {
				v := c.name
				if c.argument != "" {
					v += " "
				}
				out = append(out, completion{v, c.description})
			}
		}
	} else {
		arg = strings.TrimLeft(arg, " ")
		for _, v := range m.values(name) {
			if strings.HasPrefix(strings.ToLower(v), strings.ToLower(arg)) {
				desc := ""
				if name == "goto" {
					for _, t := range m.all {
						if t.ID == v {
							desc = t.Title
							break
						}
					}
				}
				out = append(out, completion{name + " " + v, desc})
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
func (m model) commandReady() bool {
	value := strings.TrimSpace(m.input.Value())
	name, arg, _ := strings.Cut(value, " ")
	for _, c := range commands {
		if c.name == name {
			return c.argument == "" || strings.TrimSpace(arg) != ""
		}
	}
	return false
}
func (m *model) executeCommand() bool {
	value := strings.TrimSpace(m.input.Value())
	name, arg, _ := strings.Cut(value, " ")
	arg = strings.TrimSpace(arg)
	switch name {
	case "overview", "list", "board", "lint":
		m.setView(name)
	case "todo", "doing", "done", "dropped", "blocked":
		m.setFilter("status", name)
		m.setView("list")
	case "status", "type", "priority", "group", "label", "owner":
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
