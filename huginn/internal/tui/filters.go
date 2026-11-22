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

// keeps is one filter field's rule (7zg9mf): "" keeps every task, "X"
// the tasks has says have X, "!X" every task but those.
func keeps(filter string, has func(string) bool) bool {
	if filter == "" {
		return true
	}
	v, not := task.Not(filter)
	return has(v) != not
}

// listDefault is the state filter the list opens on (7zg9mf, Jørgen
// 2026-11-20): the open work, without the done tasks. A chip like any
// other; F or ⌫ on it clears it for the session. It is the list's own:
// the overview and the board have their own sections and never show it.
const listDefault = "!done"

// showListDefault puts the list's default chip up in the list, while the
// reader has not cleared it and has no state filter of their own, and
// takes it down in every other view.
func (m *model) showListDefault() {
	switch {
	case m.view == "list" && m.defaultOn && !m.defaultShown && m.filter.State == "":
		m.filter.State = listDefault
		// First on the line, as when the list opens before any chip.
		m.chips = append([]filterChip{{"state", listDefault}}, m.chips...)
		m.defaultShown = true
		m.applyFilter()
	case m.view != "list" && m.defaultShown:
		m.dropChip("state")
		m.filter.State = ""
		m.defaultShown = false
		m.applyFilter()
	}
}

// retireDefault ends the list's default for the session: the reader
// cleared it in the list (F, or ⌫ on a state chip) or replaced it with a
// state filter of their own while it was up. Clearing filters in another
// view leaves it.
func (m *model) retireDefault() { m.defaultOn, m.defaultShown = false, false }

// dropChip removes the chip of field, if one is set.
func (m *model) dropChip(field string) {
	for i := 0; i < len(m.chips); i++ {
		if m.chips[i].field == field {
			m.chips = append(m.chips[:i], m.chips[i+1:]...)
			return
		}
	}
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
		// Values are "ID title"; the filter keeps the id, and its !.
		m.filter.Epic, _, _ = strings.Cut(value, " ")
	}
}
func (m *model) setFilter(name, value string) {
	if name == "state" && m.defaultShown {
		m.retireDefault() // the reader's own state filter replaces the list's
	}
	m.dropChip(name)
	m.writeFilter(name, value)
	if value != "" {
		m.chips = append(m.chips, filterChip{name, value})
	}
	m.applyFilter()
}

// filterHere applies a filter in the view the reader is in (ht3cdx,
// Jørgen 2026-11-20: "I think I should just keep the view I'm in"): only
// choosing a view changes it. The selected task stays selected while the
// filter keeps it; a section the filter empties says so on its title.
func (m *model) filterHere(name, value string) {
	t, ok := m.rowTask()
	m.closeToRows()
	m.setFilter(name, value)
	if ok && m.follow(t.ID) {
		return
	}
	// The task left: in this view the focus goes to the first section
	// with rows when its own is empty; the empty ones say "None with"
	// the filter.
	if len(m.indices()) == 0 {
		for p := 0; p < m.paneCount(); p++ {
			if len(m.paneIndices(p)) > 0 {
				m.pane = p
				break
			}
		}
	}
	m.clamp()
}

// closeToRows is what a filter or a jump does in place of choosing the
// list (ht3cdx): an open task closes to its rows, as Esc, and Enter's
// results end, as a view switch ends them (5cjy88). The view and its
// section stay.
func (m *model) closeToRows() {
	if m.resultsShown() {
		m.dropSearch()
		m.applyFilter()
	}
	m.detail = false
	m.glance = false
	m.find = findState{}
	m.detailPath = ""
	m.detailStack = nil
	m.detailPane = 0
	m.head = ""
	m.slot = nil
}
func (m *model) removeChip() {
	if len(m.chips) == 0 {
		if m.query != "" {
			m.clearKeeping(false, func() { m.query = "" })
		}
		return
	}
	c := m.chips[len(m.chips)-1]
	if c.field == "state" && (m.defaultShown || m.view == "list") {
		m.retireDefault() // ⌫ on the list's state chip clears the default for the session
	}
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
		value, _ := task.Not(c.value) // !done is drawn in done's colour, the ! says not
		switch c.field {
		case "state":
			style = m.styles.status[value]
		case "type":
			style = m.styles.kind[value]
		case "priority":
			style = m.styles.priority[value]
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
func (m *model) openFilter() tea.Cmd {
	m.pickerField, m.pickerNot = "", false
	return m.openInput("filter", "")
}

// pickedValue is the value Enter on choice adds: with ! pressed in the
// picker, its negation (7zg9mf).
func (m model) pickedValue(choice string) string {
	if m.pickerNot {
		return "!" + choice
	}
	return choice
}
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
	if k == "!" {
		// ! flips the value to its negation, before the field, before the
		// value or on it; again flips it back (7zg9mf). It is never text.
		m.pickerNot = !m.pickerNot
		return m, nil
	}
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
			m.prompt = ""
			m.input.Blur()
			m.filterHere(m.pickerField, m.pickedValue(choices[min(m.completion, len(choices)-1)]))
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
	not := ""
	if m.pickerNot {
		not = " · not: every task but the value picked"
	}
	lines := []string{m.styles.accent.Render("FILTER · a letter picks a field · j/k and Enter · ! not · Esc back" + not), m.styles.normal.Bold(true).Render("filter: " + now)}
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
	lines = []string{m.styles.accent.Render("FILTER · " + m.pickerField + not + " · counts include your other filters"), m.input.View(), ""}
	choices := m.filterChoices()
	rows := max(1, m.bodyHeight()-4)
	start := max(0, m.completion-rows+1)
	for i := start; i < min(len(choices), start+rows); i++ {
		value := m.pickedValue(choices[i]) // !done, with its count, while ! is on
		line := fmt.Sprintf("  %-24s %5d", oneLine(value), m.valueCount(m.pickerField, value))
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

// openTask opens t over the view the reader is in (ht3cdx): S c, # and
// goto, as Enter on its row. Its section or column there is the pane the
// breadcrumb names and n, p and Esc follow (24403z); a task the view or the
// filters leave out opens all the same, and Esc says why it has no row
// (mcvfbz).
func (m *model) openTask(t task.Task) {
	m.closeToRows()
	m.follow(t.ID)
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
