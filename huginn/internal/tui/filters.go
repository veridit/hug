package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

type filterChip struct{ field, value string }

var filterFields = []struct{ letter, name string }{{"s", "status"}, {"t", "type"}, {"p", "priority"}, {"g", "group"}, {"l", "label"}, {"o", "owner"}}

func ownerMatches(t task.Task, value string) bool {
	return value == "(unassigned)" && len(t.Assignee) == 0 || contains(t.Assignee, value)
}
func groupMatches(value, filter string) bool {
	return value == filter || value == "" && filter == "(root)"
}
func optionalMatches(value, filter string) bool {
	return value == filter || value == "" && filter == "(none)"
}
func (m *model) writeFilter(name, value string) {
	switch name {
	case "status":
		m.filter.Status = value
	case "type":
		m.filter.Type = value
	case "priority":
		m.filter.Priority = value
	case "group":
		m.filter.Group = value
	case "label":
		m.filter.Label = value
	case "owner":
		m.filter.Mine = value
	}
}
func (m *model) setFilter(name, value string) {
	for i := 0; i < len(m.chips); i++ {
		if m.chips[i].field == name {
			m.chips = append(m.chips[:i], m.chips[i+1:]...)
			break
		}
	}
	m.writeFilter(name, value)
	if value != "" {
		m.chips = append(m.chips, filterChip{name, value})
	}
	m.applyFilter()
}
func (m *model) removeChip() {
	if len(m.chips) == 0 {
		if m.query != "" {
			m.query = ""
			m.applyFilter()
		}
		return
	}
	c := m.chips[len(m.chips)-1]
	m.chips = m.chips[:len(m.chips)-1]
	m.writeFilter(c.field, "")
	m.applyFilter()
}
func (m model) filterBar() string {
	var chips []string
	for _, c := range m.chips {
		style := m.styles.accent
		switch c.field {
		case "status":
			style = m.styles.status[c.value]
		case "type":
			style = m.styles.kind[c.value]
		case "priority":
			style = m.styles.priority[c.value]
		}
		chips = append(chips, style.Bold(true).Render("["+c.field+":"+oneLine(c.value)+" ×]"))
	}
	if m.query != "" {
		chips = append(chips, m.styles.accent.Render("[/ "+oneLine(m.query)+"]"))
	}
	if len(chips) == 0 {
		return m.styles.dim.Render("No filters · f picker · 1 todo  2 doing  3 done  4 blocked")
	}
	return strings.Join(chips, " ") + m.styles.dim.Render(" · ⌫ last · F clear")
}
func (m *model) openFilter() tea.Cmd { m.pickerField = ""; return m.openInput("filter", "") }
func (m model) filterChoices() []string {
	var values []string
	for _, v := range m.values(m.pickerField) {
		if strings.Contains(strings.ToLower(v), strings.ToLower(m.input.Value())) {
			values = append(values, v)
		}
	}
	return values
}
func (m model) valueCount(field, value string) int {
	copy := m
	copy.writeFilter(field, value)
	copy.visible = nil
	// applyFilter writes only copy-owned slice state and never mutates task data.
	copy.applyFilter()
	return len(copy.visible)
}
func (m model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.pickerField == "" {
		for _, f := range filterFields {
			if k == f.letter {
				m.pickerField = f.name
				m.input.SetValue("")
				m.input.Prompt = f.name + " "
				m.completion = 0
				return m, nil
			}
		}
		switch k {
		case "j", "down", "ctrl+n":
			m.completion = (m.completion + 1) % len(filterFields)
		case "k", "up", "ctrl+p":
			m.completion = (m.completion + len(filterFields) - 1) % len(filterFields)
		case "enter":
			m.pickerField = filterFields[m.completion].name
			m.completion = 0
			m.input.Prompt = m.pickerField + " "
		}
		return m, nil
	}
	choices := m.filterChoices()
	switch k {
	case "down", "ctrl+n":
		if len(choices) > 0 {
			m.completion = (m.completion + 1) % len(choices)
		}
		return m, nil
	case "up", "ctrl+p":
		if len(choices) > 0 {
			m.completion = (m.completion + len(choices) - 1) % len(choices)
		}
		return m, nil
	case "j":
		if m.input.Value() == "" {
			if len(choices) > 0 {
				m.completion = (m.completion + 1) % len(choices)
			}
			return m, nil
		}
	case "k":
		if m.input.Value() == "" {
			if len(choices) > 0 {
				m.completion = (m.completion + len(choices) - 1) % len(choices)
			}
			return m, nil
		}
	case "enter":
		if len(choices) > 0 {
			m.setFilter(m.pickerField, choices[min(m.completion, len(choices)-1)])
			m.prompt = ""
			m.input.Blur()
			m.setView("list")
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.completion = 0
	return m, cmd
}
func (m model) filterText() string {
	lines := []string{m.styles.accent.Render("QUICK FILTER · choose a field")}
	if m.pickerField == "" {
		for i, f := range filterFields {
			line := fmt.Sprintf("  %s  %s", f.letter, f.name)
			if i == m.completion {
				line = m.styles.selected.Render("> " + line[2:])
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n")
	}
	lines[0] = m.styles.accent.Render("QUICK FILTER · " + m.pickerField + " · counts include your other filters")
	lines = append(lines, m.input.View(), "")
	choices := m.filterChoices()
	rows := max(1, m.bodyHeight()-4)
	start := max(0, m.completion-rows+1)
	for i := start; i < min(len(choices), start+rows); i++ {
		line := fmt.Sprintf("  %-24s %5d", oneLine(choices[i]), m.valueCount(m.pickerField, choices[i]))
		if i == m.completion {
			line = m.styles.selected.Render("> " + line[2:])
		}
		lines = append(lines, line)
	}
	if len(choices) == 0 {
		lines = append(lines, m.styles.dim.Render("No matching values"))
	}
	return strings.Join(lines, "\n")
}
func (m model) jumpChoices() []task.Task {
	query := strings.ToLower(strings.TrimSpace(m.input.Value()))
	var choices []task.Task
	for _, t := range m.all {
		match := strings.HasPrefix(strings.ToLower(t.ID), query)
		for _, alias := range t.Aliases {
			alias = strings.ToLower(alias)
			match = match || strings.HasPrefix(alias, query) || strings.HasPrefix(strings.TrimPrefix(alias, "task-"), query)
		}
		if match {
			choices = append(choices, t)
		}
	}
	return choices
}
func (m *model) openTask(t task.Task) {
	// Jump is global, but returning from detail retains the user's filter context.
	m.setView("list")
	for i, v := range m.visible {
		if v.Path == t.Path {
			m.cursor = i
			break
		}
	}
	m.detailPath = t.Path
	m.detail = true
	m.syncDetail()
	m.viewport.GotoTop()
}
func (m model) updateJump(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	choices := m.jumpChoices()
	switch msg.String() {
	case "down", "ctrl+n":
		if len(choices) > 0 {
			m.completion = (m.completion + 1) % len(choices)
		}
		return m, nil
	case "up", "ctrl+p":
		if len(choices) > 0 {
			m.completion = (m.completion + len(choices) - 1) % len(choices)
		}
		return m, nil
	case "tab", "enter":
		if len(choices) == 0 {
			m.message = "no matching id or alias"
			return m, nil
		}
		t, err := task.Resolve(m.all, choices[min(m.completion, len(choices)-1)].ID)
		if err != nil {
			m.message = err.Error()
			return m, nil
		}
		if msg.String() == "tab" {
			m.input.SetValue(t.ID)
			m.input.CursorEnd()
			m.completion = 0
		} else {
			m.prompt = ""
			m.input.Blur()
			m.openTask(t)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.completion = 0
	return m, cmd
}
func (m model) jumpText() string {
	lines := []string{m.styles.accent.Render("JUMP TO ID · prefixes and TASK-number aliases"), m.input.View(), ""}
	choices := m.jumpChoices()
	rows := max(1, m.bodyHeight()-4)
	start := max(0, m.completion-rows+1)
	for i := start; i < min(len(choices), start+rows); i++ {
		t := choices[i]
		line := "  " + fit(t.ID, 6) + "  " + fit(strings.Join(t.Aliases, ", "), 16) + "  " + oneLine(t.Title)
		if i == m.completion {
			line = m.styles.selected.Render("> " + line[2:])
		}
		lines = append(lines, line)
	}
	if len(choices) == 0 {
		lines = append(lines, "No matching id or alias")
	}
	return strings.Join(lines, "\n")
}
