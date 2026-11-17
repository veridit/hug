package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// The column cursor (bejd89, Jørgen 2026-11-17 11:43: "the board as a
// spreadsheet"). Tab and Shift-Tab mark one column header; o on it cycles
// ascending ▲, descending ▼, off (rank order). One column at a time: o on
// another column replaces the order. The sort builder (: sort) stays for
// orders of more than one field. h/l keep moving between sections.

// column is one header cell the cursor can mark: its label, the sort
// field it orders by, and whether a row layout shows it.
type column struct {
	label, field string
	shown        func(c rowCols) bool
}

// gridColumns are the columns rows show, left to right: the pie (checks
// done), id, type, priority, stage, title, and in IN PROGRESS the owner.
// A column the width dropped is skipped by the cursor.
var gridColumns = []column{
	{"✓", "checks", func(rowCols) bool { return true }},
	{"id", "id", func(rowCols) bool { return true }},
	{"type", "type", func(c rowCols) bool { return c.kind }},
	{"priority", "priority", func(c rowCols) bool { return c.priority }},
	{"stage", "stage", func(c rowCols) bool { return c.status }},
	{"title", "title", func(rowCols) bool { return true }},
	{"owner", "assignee", func(c rowCols) bool { return c.owner > 0 }},
}

// columnLayout is the layout the cursor moves over in the current view:
// the focused section's rows, with the owner cell in IN PROGRESS.
func (m model) columnLayout() rowCols {
	switch m.view {
	case "board":
		count := min(len(boardStates), max(1, (m.width+2)/40))
		return m.rowLayout(max(1, (m.width-2*(count-1))/count), true, 0, 0)
	case "overview":
		if m.pane == sectionDoing || m.pane == sectionWaiting {
			return m.rowLayout(m.width, false, m.ownerWidth(), m.ownerNames(m.paneIndices(m.pane)))
		}
	}
	return m.rowLayout(m.width, false, 0, 0)
}

// moveColumn moves the column cursor by delta over the columns shown,
// wrapping. The first press marks the first column.
func (m *model) moveColumn(delta int) {
	c := m.columnLayout()
	var shown []int
	for i, col := range gridColumns {
		if col.shown(c) {
			shown = append(shown, i)
		}
	}
	if len(shown) == 0 {
		return
	}
	at := -1
	for n, i := range shown {
		if i == m.column {
			at = n
		}
	}
	switch {
	case at < 0 && delta > 0:
		at = 0
	case at < 0:
		at = len(shown) - 1
	default:
		at = (at + delta + len(shown)) % len(shown)
	}
	m.column = shown[at]
	m.message = m.columnHint()
}

// columnHint is what o will do on the marked column, for the footer.
func (m model) columnHint() string {
	if m.column < 0 {
		return "Tab picks a column, then o orders by it"
	}
	col := gridColumns[m.column]
	next := "▲"
	if k, ok := m.columnOrder(); ok && k.field == col.field {
		next = "▼"
		if k.desc {
			next = "off"
		}
	}
	return "Tab next column · o order by " + col.label + " (" + next + ") · h/l section"
}

// columnOrder is the one-column order the cursor set, if that is the
// order in force.
func (m model) columnOrder() (sortKey, bool) {
	if len(m.sort) != 1 {
		return sortKey{}, false
	}
	return m.sort[0], true
}

// orderByColumn is o: on the marked column, ascending, then descending,
// then off (rank order). Another column's order is replaced.
func (m *model) orderByColumn() {
	if m.column < 0 {
		m.message = m.columnHint()
		return
	}
	field := gridColumns[m.column].field
	k, ok := m.columnOrder()
	switch {
	case !ok || k.field != field:
		m.applySort([]sortKey{{field, false}})
	case !k.desc:
		m.applySort([]sortKey{{field, true}})
	default:
		m.applySort(nil)
	}
	m.message = m.columnHint()
}

// arrow is the order mark a header puts after a column: ▲ ascending, ▼
// descending, "" when the rows are not ordered by it.
func (m model) arrow(field string) string {
	k, ok := m.columnOrder()
	if !ok || k.field != field {
		return ""
	}
	if k.desc {
		return " ▼"
	}
	return " ▲"
}

// headerCell is one column's header, width cells wide: its label and
// order arrow; the column the cursor marks starts with ▸ and is
// underlined, so it shows without colour too (NO_COLOR). A label that
// does not fit with its mark and arrow gives way to them: the arrow says
// more than the label's last letters. In a cell too narrow for even one
// letter beside them (the board's one-letter T and P), the ▸ goes first
// (the underline and the hint line still name the column), then the
// label, so the arrow always shows.
func (m model) headerCell(i int, label string, width int) string {
	mark := ""
	if i == m.column {
		mark = "▸"
	}
	arrow := strings.TrimSpace(m.arrow(gridColumns[i].field))
	if arrow != "" {
		arrow = " " + arrow
	}
	room := width - lipgloss.Width(mark) - lipgloss.Width(arrow)
	if room < 1 {
		mark = ""
		room = width - lipgloss.Width(arrow)
	}
	text := label
	if room < 1 {
		text, arrow = "", strings.TrimSpace(arrow)
	} else if lipgloss.Width(text) > room {
		text = ansi.Truncate(text, room, "")
	}
	text = mark + text + arrow
	s := fit(text, width)
	if i == m.column {
		t := strings.TrimRight(s, " ")
		return m.styles.accent.Bold(true).Underline(true).Render(t) + strings.Repeat(" ", max(0, width-lipgloss.Width(t)))
	}
	return m.styles.dim.Render(s)
}

// overviewHeading says whether the overview shows a column header under
// its focused section: while the column cursor marks a column or the rows
// are ordered by one. Without it the overview reads as before.
func (m model) overviewHeading() bool {
	if m.pane == sectionLint {
		return false
	}
	_, ordered := m.columnOrder()
	return m.column >= 0 || ordered
}
