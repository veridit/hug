package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

// The / search (5cjy88, Jørgen 2026-11-21: '/ k5q120' picked a task that
// only mentions k5q120, the arrows did nothing and Enter jumped back to the
// board on it). Typing narrows, Enter commits to the results, Esc steps back
// one level:
//
//   - while typing, a task whose id (or alias) is typed whole is selected,
//     in whatever section it is; else the first match, the current
//     section first; ↓ and ctrl-n/p pick among the matches across
//     sections, and so does ↑ once a match was picked (↑ in the empty
//     prompt recalls earlier searches, xfv9pw);
//   - Enter closes the prompt and leaves the results on screen
//     (m.results): j/k walk every match in order across sections, Enter
//     opens one and Esc from it comes back to them; with no match the
//     prompt stays open;
//   - Esc on the results clears the search and keeps the picked task
//     selected and in view (mcvfbz: the selection follows the task, not
//     the row number); a view switch ends them the same way, and m does
//     before it grabs, so a move is among the real neighbours;
//   - Esc while typing cancels: the search, the results or the chip, the
//     section and the task selected before / come back;
//   - Tab keeps the search as a filter chip, which lasts across opening a
//     task and switching view until F (or ⌫) clears it; Esc clears no chip
//     (crab 2026-11-21 09:16Z: a chip lasts until F or ⌫).
//
// A task the view or a filter leaves out is never swapped for another: no
// row is selected and the footer says why (hiddenWhy).

// searchReturn is where / was pressed, for Esc while typing.
type searchReturn struct {
	query            string
	results          bool
	pane, cursor     int
	offset           int
	id, head, hidden string
	memo             [2][maxSections]sectionMemo
}

// openSearch opens the / prompt with value, remembering where it was
// opened so Esc can come back.
func (m *model) openSearch(value string) tea.Cmd {
	r := searchReturn{query: m.query, results: m.results, pane: m.pane, cursor: m.cursor, offset: m.offset, head: m.head, hidden: m.hidden, memo: m.memo}
	if t, ok := m.selected(); ok && !m.detail {
		r.id = t.ID
	}
	m.searchFrom = &r
	m.results = false
	m.searchPicked = false
	m.message = "" // the footer says where the pick is among the matches
	return m.openInput("/", value)
}

// updateSearch handles the keys of the / prompt that are not typing. It
// reports false for a key the text input should take.
func (m *model) updateSearch(k string) bool {
	switch k {
	case "down", "ctrl+n":
		m.searchPicked = true
		m.move(1, false)
	case "up", "ctrl+p":
		m.searchPicked = true
		m.move(-1, false)
	case "enter":
		m.showResults()
	case "tab":
		m.leaveSearch()
	default:
		return false
	}
	return true
}

// searchTyped applies what is typed in the / prompt. A task whose id is
// typed whole is selected wherever it is; else, in the overview and the
// board, a section without a match gives way to the first one with a
// match, so the match is selected without leaving the prompt.
func (m *model) searchTyped() {
	m.query = m.input.Value()
	m.message = "" // a 'no match' said before the edit is old
	m.applyFilter()
	if id := m.exactID(); id != "" && m.follow(id) {
		return
	}
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

// exactID is the id of the shown task whose id or alias is the searched
// word, whole and in any case, or "".
func (m model) exactID() string {
	_, words := parseSearch(m.query)
	words = strings.TrimSpace(words)
	if words == "" || strings.ContainsAny(words, " \t") {
		return ""
	}
	for _, t := range m.visible {
		if strings.EqualFold(t.ID, words) || slices.ContainsFunc(t.Aliases, func(a string) bool { return strings.EqualFold(a, words) }) {
			return t.ID
		}
	}
	return ""
}

// showResults is Enter in the / prompt: the prompt closes and the matches
// stay on screen, the picked one selected. Nothing typed is no search: the
// prompt closes and the task stays. No match keeps the prompt open.
func (m *model) showResults() {
	if strings.TrimSpace(m.query) == "" {
		had := m.searchFrom != nil && m.searchFrom.query != ""
		m.cancelSearch()
		if had {
			// The text cleared and Enter: the search ends, the task stays.
			m.clearKeeping(false, m.dropSearch)
		}
		return
	}
	if len(m.visible) == 0 {
		m.message = "no match for " + oneLine(m.query) + ": edit it, or Esc"
		return
	}
	m.history().add(searchHistory, m.query)
	m.closeSearch()
	m.results = true
	m.message = ""
}

// leaveSearch is Tab in the / prompt: the search stays on as a filter chip.
func (m *model) leaveSearch() {
	m.history().add(searchHistory, m.query)
	m.closeSearch()
}

// cancelSearch is Esc while typing: back where / was pressed, its search,
// results or chip, section and task. What was typed is kept for Up to
// recall (xfv9pw).
func (m *model) cancelSearch() {
	m.history().add(searchHistory, m.input.Value())
	r := m.searchFrom
	m.closeSearch()
	m.message = ""
	if r == nil {
		return
	}
	m.query, m.results = r.query, r.results
	m.applyFilter()
	m.pane, m.memo, m.head = r.pane, r.memo, r.head
	switch {
	case r.id != "" && m.follow(r.id):
	case r.hidden != "":
		m.cursor, m.hidden = -1, r.hidden
	default:
		m.cursor = r.cursor
		m.clamp()
	}
	m.offset = r.offset
}

// closeSearch closes the / prompt.
func (m *model) closeSearch() {
	m.prompt = ""
	m.input.Blur()
	m.searchFrom = nil
	m.searchPicked = false
}

// matchPlace is where the selected task is among the matches shown, in
// walk order: "match 2 of 4 in LINT"; "" with no search or none selected.
func (m model) matchPlace() string {
	if m.query == "" {
		return ""
	}
	n, at := 0, 0
	for p := 0; p < m.paneCount(); p++ {
		for _, i := range m.paneIndices(p) {
			n++
			if p == m.pane && i == m.cursor && m.head == "" {
				at = n
			}
		}
	}
	if at == 0 {
		return ""
	}
	where := ""
	switch m.view {
	case "overview", "board":
		where = " in " + strings.TrimSuffix(m.sectionName(), " (filtered)")
	}
	return fmt.Sprintf("match %d of %d%s", at, n, where)
}

// resultsShown says the rows are the results of a search Enter left on
// screen, not a kept filter.
func (m model) resultsShown() bool { return m.results && m.query != "" }

// leaveResults clears the results, keeping the selected task; say reports
// where it is (Esc), or not (m, a view switch: what follows says).
func (m *model) leaveResults(say bool) {
	m.clearKeeping(say, m.dropSearch)
}

// dropSearch removes the search, results or chip.
func (m *model) dropSearch() {
	m.query = ""
	m.results = false
}

// walk is move whenever a search is shown, in the overview and the board:
// every match in order, section after section, so no match needs h/l to
// reach (5cjy88 check 6). Headings of grouped rows are stops too.
func (m *model) walk(n int, absolute bool) {
	type stop struct{ pane, row int }
	var stops []stop
	at, first := -1, -1
	for p := 0; p < m.paneCount(); p++ {
		rows := m.paneRows(p)
		if m.detail {
			rows = m.paneIndices(p) // n and p in a task step tasks, no headings
		}
		for _, r := range rows {
			if p == m.pane {
				if first < 0 {
					first = len(stops)
				}
				if at < 0 && (r == m.rowCursor(p) || m.detail && r == m.cursor) {
					at = len(stops)
				}
			}
			stops = append(stops, stop{p, r})
		}
	}
	if len(stops) == 0 {
		return
	}
	switch {
	case absolute:
		at = n
	case at < 0:
		// Nothing selected (a task kept hidden, mcvfbz): a step selects
		// the section's first row, or the first match.
		at = max(first, 0)
	default:
		at += n
	}
	s := stops[max(0, min(at, len(stops)-1))]
	if s.pane != m.pane {
		m.remember()
		m.pane = s.pane
	}
	if isHead(s.row) {
		m.head = m.grouped(s.pane).heads[headOf(s.row)].key
		m.cursor = -1
	} else {
		m.head, m.cursor = "", s.row
	}
	m.clamp()
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
	return fmt.Sprintf("%s has no row in %s: vl lists every task, # %s opens it", t.ID, where, t.ID)
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
