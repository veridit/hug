package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

type filterChip struct{ field, value string }

var filterFields = []struct{ letter, name string }{{"s", "state"}, {"t", "type"}, {"p", "priority"}, {"g", "group"}, {"l", "tag"}, {"a", "assignee"}, {"e", "epic"}}

// filterHints say what each filter field picks, as the sort picker's
// hints do (ahejza: both panels look alike).
var filterHints = map[string]string{
	"state": "todo, doing, depends, done, dropped", "type": "feature, bug, chore, spike, docs",
	"priority": "critical, high, medium, low", "group": "a group, or (root)", "tag": "one tag",
	"assignee": "a person or agent, or (unassigned)", "epic": "an epic's children",
}

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
	case "state":
		m.filter.State = value
	case "type":
		m.filter.Type = value
	case "priority":
		m.filter.Priority = value
	case "group":
		m.filter.Group = value
	case "tag": // the task field is still labels until hug upgrade renames it (txjkrm)
		m.filter.Label = value
	case "assignee":
		m.filter.Mine = value
	case "epic":
		// Values are "ID title"; the filter keeps the id.
		m.filter.Epic, _, _ = strings.Cut(value, " ")
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
			m.clearKeeping(false, func() { m.query = "" })
		}
		return
	}
	c := m.chips[len(m.chips)-1]
	m.clearKeeping(false, func() {
		m.chips = m.chips[:len(m.chips)-1]
		m.writeFilter(c.field, "")
	})
}
func (m model) filterBar() string {
	// Enter's results (5cjy88): not a chip, so no brackets; Esc ends
	// them, and the chips (f) stay. While typing the search is not kept
	// either: the same line, without Esc, which cancels there.
	found := ""
	typing := m.prompt == "/" && m.query != ""
	if m.resultsShown() || typing {
		noun := "matches"
		if len(m.visible) == 1 {
			noun = "match"
		}
		found = m.styles.dim.Render("found: ") + m.styles.accent.Render("/ "+oneLine(m.query)) + m.styles.dim.Render(fmt.Sprintf(" · %d %s", len(m.visible), noun))
		if !typing {
			found += m.styles.dim.Render(" · Esc all tasks")
		}
		if len(m.chips) == 0 {
			return found
		}
		found = m.styles.dim.Render(" · ") + found
	}
	var chips []string
	for _, c := range m.chips {
		style := m.styles.accent
		switch c.field {
		case "state":
			style = m.styles.status[c.value]
		case "type":
			style = m.styles.kind[c.value]
		case "priority":
			style = m.styles.priority[c.value]
		}
		chips = append(chips, style.Bold(true).Render("["+c.field+":"+oneLine(c.value)+" ×]"))
	}
	if m.query != "" && !m.results && m.prompt != "/" {
		chips = append(chips, m.styles.accent.Render("[/ "+oneLine(m.query)+"]"))
	}
	// The line that says what shapes the rows (ahejza): its letters f
	// and s are the keys that change them, and they open look-alike
	// pickers.
	key := m.styles.accent.Underline(true)
	line := key.Render("f") + m.styles.dim.Render("ilter: ")
	if len(chips) == 0 {
		line += m.styles.dim.Render("none")
	} else {
		line += strings.Join(chips, " ") + m.styles.dim.Render(" ⌫ last · F clear")
	}
	if found != "" {
		return line + found
	}
	return line + m.styles.dim.Render(" · ") + key.Render("s") + m.styles.dim.Render("ort: "+sortLabel(m.sort))
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
				return m, m.input.Focus() // the field takes text: its cursor blinks (q3sh69)
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
			return m, m.input.Focus()
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
	now := "none"
	var set []string
	for _, c := range m.chips {
		set = append(set, c.field+":"+oneLine(c.value))
	}
	if len(set) > 0 {
		now = strings.Join(set, " ")
	}
	lines := []string{m.styles.accent.Render("FILTER · a letter picks a field · j/k and Enter · Esc back"), m.styles.normal.Bold(true).Render("filter: " + now)}
	if m.pickerField == "" {
		for i, f := range filterFields {
			line := "  " + f.letter + "  " + fit(f.name, 10) + m.styles.dim.Render(filterHints[f.name])
			if i == m.completion {
				line = m.styles.selected.Render("> " + line[2:])
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n")
	}
	lines = []string{m.styles.accent.Render("FILTER · " + m.pickerField + " · counts include your other filters"), m.input.View(), ""}
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
	m.startFocus()
	m.syncDetail()
	m.openTop()
}
func (m model) updateJump(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	choices := m.jumpChoices()
	m.jumpPending = false
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
			// Before the start's board lands, the id may be on its way:
			// Enter is answered when it arrives (vnyz8z).
			m.jumpPending = msg.String() == "enter" && m.firstLoad()
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
			if m.picking {
				// Picking a task for the capture (hjbfzd): # finds the row;
				// Enter on it attaches.
				if !m.follow(t.ID) {
					m.message = t.ID + " is not in these rows: Esc, then type it at S"
				}
				return m, nil
			}
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
