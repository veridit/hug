package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// Leaving a search keeps the found task (mcvfbz, Jørgen 2026-11-16: he
// found the task, pressed Esc, and the selection jumped away, so he could not
// grab it and move it). The selection follows the task, not the row number:
//
//   - while typing, the first match is selected and ↑↓ pick another;
//   - Enter goes to the match: the search ends and the full view shows with
//     that task selected and in view, ready for m, J and K;
//   - Esc clears the search the same way, keeping the task;
//   - Tab keeps the search on as a filter (Enter did that before);
//   - clearing a kept search or filters (Esc, c, F) keeps the task too.
//
// A task the view or a filter leaves out is never swapped for another: no
// row is selected and the footer says why (hiddenWhy).

// updateSearch handles the keys of the / prompt that are not typing. It
// reports false for a key the text input should take.
func (m *model) updateSearch(k string) bool {
	switch k {
	case "down", "ctrl+n":
		m.move(1, false)
	case "up", "ctrl+p":
		m.move(-1, false)
	case "enter":
		m.leaveSearch(false)
	case "tab":
		m.leaveSearch(true)
	default:
		return false
	}
	return true
}

// searchTyped applies what is typed in the / prompt. In the overview and
// the board a section without a match gives way to the first one with a
// match, so the match is selected without leaving the prompt.
func (m *model) searchTyped() {
	m.query = m.input.Value()
	m.applyFilter()
	if memoView(m.view) < 0 || len(m.indices()) > 0 {
		return
	}
	for p := 0; p < m.paneCount(); p++ {
		if rows := m.paneIndices(p); len(rows) > 0 {
			m.remember()
			m.pane = p
			m.cursor = rows[0]
			m.clamp()
			return
		}
	}
}

// leaveSearch closes the / prompt. keep leaves the search on as a filter;
// otherwise the search is cleared and the task selected in it stays
// selected in the full view.
func (m *model) leaveSearch(keep bool) {
	m.prompt = ""
	m.input.Blur()
	if keep {
		return
	}
	m.clearKeeping(true, func() { m.query = "" })
}

// clearKeeping runs clear (which removes the search, filters or both) and
// keeps the selected task selected where it now shows. say reports where
// it is. A task no longer shown is never swapped for another.
func (m *model) clearKeeping(say bool, clear func()) {
	t, ok := m.rowTask()
	clear()
	m.applyFilter()
	if !ok {
		return
	}
	if !m.follow(t.ID) {
		m.hide(t)
		return
	}
	if say {
		m.message = fmt.Sprintf("found %s%s", t.ID, m.rowPlace(t.Path))
		if len(m.sort) > 0 {
			m.message += " · sorted by " + sortLabel(m.sort) + "; " + sortRefusal
		}
	}
}

// hide leaves no row selected because t, the task to keep, has no row
// here; it is remembered, so clearing the filter selects it again.
func (m *model) hide(t task.Task) {
	m.cursor = -1
	m.hidden = t.ID
	m.message = m.hiddenWhy(t)
}

// hiddenWhy says why t has no row here, with what brings it back.
func (m model) hiddenWhy(t task.Task) string {
	if !slices.ContainsFunc(m.visible, func(v task.Task) bool { return v.Path == t.Path }) {
		if text := m.chipText(); text != "" {
			return fmt.Sprintf("%s is hidden by the filter %s: F clears it", t.ID, text)
		}
	}
	where := map[string]string{"overview": "the overview", "board": "the board", "lint": "the lint view", "list": "the list"}[m.view]
	return fmt.Sprintf("%s has no row in %s: i lists every task, # %s opens it", t.ID, where, t.ID)
}

// follow selects the task with id where the current view shows it: its
// section in the overview, its column on the board, its row in a list.
// False when the filters or the view leave it out.
func (m *model) follow(id string) bool {
	at := slices.IndexFunc(m.visible, func(v task.Task) bool { return v.ID == id })
	if at < 0 {
		return false
	}
	panes := []int{m.pane}
	for p := 0; p < m.paneCount(); p++ {
		if p != m.pane {
			panes = append(panes, p)
		}
	}
	for _, p := range panes {
		if m.grouping != "" && !slices.Contains(m.paneIndices(p), at) && slices.Contains(m.rawPane(p), at) {
			m.unfold(m.visible[at]) // a task found is shown, never left folded (kx2ndm)
		}
		if slices.Contains(m.paneIndices(p), at) {
			if p != m.pane {
				m.remember()
				m.pane = p
			}
			m.head = ""
			m.cursor = at
			m.hidden = ""
			m.clamp()
			return true
		}
	}
	return false
}

// paneCount is how many sections the current view has: overview sections,
// board columns, or the one list.
func (m model) paneCount() int {
	switch m.view {
	case "overview":
		return len(overviewSections)
	case "board":
		return len(boardStates)
	}
	return 1
}

// rowTask is the task to keep: the open task in a detail, else the
// selected row.
func (m model) rowTask() (task.Task, bool) {
	if m.detail && m.detailPath != "" {
		return m.taskAt(m.detailPath)
	}
	if m.cursor == -1 && m.hidden != "" {
		for _, t := range m.all {
			if t.ID == m.hidden {
				return t, true
			}
		}
	}
	if m.cursor < 0 || m.cursor >= len(m.visible) || !slices.Contains(m.indices(), m.cursor) {
		return task.Task{}, false
	}
	return m.visible[m.cursor], true
}

// chipText is the active filters as the chips name them: status:todo type:bug,
// and the search when one is kept.
func (m model) chipText() string {
	var parts []string
	for _, c := range m.chips {
		parts = append(parts, c.field+":"+oneLine(c.value))
	}
	if m.query != "" {
		parts = append(parts, "/"+oneLine(m.query))
	}
	return strings.Join(parts, " ")
}
