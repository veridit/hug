package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// The column cursor (bejd89, Jørgen 2026-11-17 11:43: "the board as a
// spreadsheet"; e3sevr, 23:20). H and L mark one column; o on it cycles
// ascending ▲, descending ▼, off (rank order). One column at a time: o on
// another column replaces the order. The sort builder (: sort) stays for
// orders of more than one field. h/l, and Tab/Shift-Tab, move between
// sections. The mark is drawn in the rows: > before the column's cell on
// the cursor row (911j70: no band down the column, no underline), and the
// column's heading on the selection tint.

// column is one header cell the cursor can mark: its label, the sort
// field it orders by, and whether a row layout shows it.
type column struct {
	label, field string
	shown        func(c rowCols) bool
}

// gridColumns are the columns rows show, left to right, as a spreadsheet
// (911j70, order B, Jørgen 2026-11-20 08:26Z): the id, the row header;
// the title, with the tags as its dim prefix; the stage; the pie (checks
// done); type; priority; and in the overview's owner sections the owner.
// A column the width dropped is skipped by the cursor.
var gridColumns = []column{
	{"id", "id", func(rowCols) bool { return true }},
	{"title", "title", func(rowCols) bool { return true }},
	{"stage", "stage", func(c rowCols) bool { return c.status }},
	{"✓", "checks", func(c rowCols) bool { return c.checks }},
	{"type", "type", func(c rowCols) bool { return c.kind }},
	{"priority", "priority", func(c rowCols) bool { return c.priority }},
	{"owner", "assignee", func(c rowCols) bool { return c.owner > 0 }},
}

// columnLayout is the layout the cursor moves over in the current view:
// the focused section's rows, with the owner cell in IN PROGRESS.
func (m model) columnLayout() rowCols {
	switch m.view {
	case "board":
		count := min(len(boardStates), max(1, (m.width+2)/40))
		return m.rowLayout(max(1, (m.width-2*(count-1))/count), true, 0, 0)
	case "overview", "list", "lint":
		return m.sectionLayout(m.pane) // without the columns it shares (1bbe96)
	}
	return m.rowLayout(m.width, false, 0, 0)
}

// markedColumn is the column the marker stands on: m.column when the
// focused section draws it, else the next column it draws (1bbe96: a
// column the section's tasks share is not drawn, so the marker and o
// move past it); -1 with no column marked.
func (m model) markedColumn() int {
	if m.column < 0 || m.column >= len(gridColumns) {
		return -1
	}
	c := m.columnLayout()
	for n := 0; n < len(gridColumns); n++ {
		i := (m.column + n) % len(gridColumns)
		if gridColumns[i].shown(c) {
			return i
		}
	}
	return -1
}

// moveColumn moves the column cursor by delta over the columns shown.
// The ring is (no column), col1 ... colN (hktdgs, Jørgen 2026-11-20
// 12:07Z): from no column, as the view opens, L marks the first column
// and H the last; L from the last and H from the first land on no
// column again, the marker gone, before the next press goes on.
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
		if i == m.markedColumn() {
			at = n
		}
	}
	switch {
	case at < 0 && delta > 0:
		at = 0
	case at < 0:
		at = len(shown) - 1
	default:
		at += delta
	}
	if at < 0 || at >= len(shown) {
		m.column = -1 // the no-column stop between the last and the first
		m.showColumnHint()
		return
	}
	m.column = shown[at]
	m.showColumnHint()
}

// showColumnHint puts what o will do in the footer, on rows only: the
// hint is drawn from the column, never kept as a message, so it cannot
// follow into a task, where H/L and o have no column (qtj177).
func (m *model) showColumnHint() {
	m.message = ""
	m.columnHinted = true
}

// columnHint is what o will do on the marked column, for the footer.
func (m model) columnHint() string {
	if m.markedColumn() < 0 {
		return "H/L pick a column, then o orders by it"
	}
	col := gridColumns[m.markedColumn()]
	next := "▲"
	if k, ok := m.columnOrder(); ok && k.field == col.field {
		next = "▼"
		if k.desc {
			next = "off"
		}
	}
	return "H/L column · " + mk("o") + "rder by " + col.label + " (" + next + ")"
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
	if m.markedColumn() < 0 {
		m.showColumnHint()
		return
	}
	field := gridColumns[m.markedColumn()].field // never a column the section does not draw (1bbe96)
	k, ok := m.columnOrder()
	switch {
	case !ok || k.field != field:
		m.applySort([]sortKey{{field, false}})
	case !k.desc:
		m.applySort([]sortKey{{field, true}})
	default:
		m.applySort(nil)
	}
	m.showColumnHint()
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

// headerCell is one column's header, width cells wide: its label and the
// order arrow, ▲ or ▼, when the rows are ordered by it. The marked
// column's heading is on the selection tint, its text only (911j70); the
// > is drawn in the rows. A label that does not fit beside the arrow gives
// way to it: the arrow says more than the label's last letters.
func (m model) headerCell(i int, label string, width int, marked bool) string {
	arrow := strings.TrimSpace(m.arrow(gridColumns[i].field))
	if arrow != "" {
		arrow = " " + arrow
	}
	room := width - lipgloss.Width(arrow)
	text := label
	if room < 1 {
		text, arrow = "", strings.TrimSpace(arrow)
	} else if lipgloss.Width(text) > room {
		text = ansi.Truncate(text, room, "")
	}
	cell := fit(text+arrow, width)
	if !marked {
		return m.styles.colHead.Render(cell)
	}
	bare := strings.TrimRight(cell, " ")
	return m.styles.colHeadSelected.Render(bare) + strings.Repeat(" ", width-lipgloss.Width(bare))
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

// gridCell is one cell of a row: its column (an index into gridColumns),
// where its text starts and how wide the text is, the space after it not
// counted.
type gridCell struct{ col, at, width int }

// rowCells are the cells of a row laid out as c, left to right in
// gridColumns' order. Every row and its header put their cells here, so
// the > lands before the cell it names and the headings stand over their
// cells.
func (m model) rowCells(compact bool, c rowCols) []gridCell {
	at := rowHead
	var out []gridCell
	add := func(col, width, step int) {
		if width > 0 {
			out = append(out, gridCell{col, at, width})
			at += step
		}
	}
	kind, priority := kindCell, priorityCell
	if compact {
		kind, priority = letterCell, letterCell
	}
	add(0, idCell-1, idCell)
	add(1, c.title-1, c.title)
	add(2, on(c.status, stageCell-1), stageCell)
	add(3, on(c.checks, m.progressWidth()-1), m.progressWidth())
	add(4, on(c.kind, kind-1), kind)
	add(5, on(c.priority, priority-1), priority)
	if c.owner > 0 {
		at += ownerGap
		add(6, c.owner, c.owner)
	}
	return out
}

// cursorColumn is the column the > stands before on the cursor row of a
// row laid out as c: the marked column, or -1 when none is marked or the
// row does not draw it; then the > stays in the gutter, before the id.
func (m model) cursorColumn(compact bool, c rowCols) int {
	col := m.markedColumn() // the board's columnLayout is its compact one
	if col <= 0 || col >= len(gridColumns) || !gridColumns[col].shown(c) {
		return -1 // the id is the row header: its > is the gutter's
	}
	return col
}
