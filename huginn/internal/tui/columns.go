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
// the cursor row, and the column tinted down the section.

// column is one header cell the cursor can mark: its label, the sort
// field it orders by, and whether a row layout shows it.
type column struct {
	label, field string
	shown        func(c rowCols) bool
}

// gridColumns are the columns rows show, left to right: the pie (checks
// done), id, tags (txjkrm), type, priority, stage, title, and in IN
// PROGRESS the owner. A column the width dropped is skipped by the cursor.
// PROTOTYPE (mock/grid, 911j70 sheets): with the mock on, in its order.
var gridColumns = buildGridColumns()

var mainColumns = []column{
	{"✓", "checks", func(rowCols) bool { return true }},
	{"id", "id", func(rowCols) bool { return true }},
	{"tags", "tags", func(c rowCols) bool { return c.tags > 0 }},
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
		at = (at + delta + len(shown)) % len(shown)
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
// order arrow, ▲ or ▼, when the rows are ordered by it. The header names
// the columns only (e3sevr, Jørgen 23:20): the column cursor is drawn in
// the rows (markColumn). A label that does not fit beside the arrow gives
// way to it: the arrow says more than the label's last letters.
func (m model) headerCell(i int, label string, width int) string {
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
	return m.styles.dim.Render(fit(text+arrow, width))
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

// colSpan is where column i's cell stands in a row laid out as c: its
// first cell and its width (the space after it not counted); ok is false
// when the layout does not show it. Every row and the header put their
// cells here (rowLayout), so the marker lands on the cell it names.
func (m model) colSpan(i int, compact bool, c rowCols) (from, width int, ok bool) {
	at := rowHead
	type cell struct {
		w     int
		shown bool
	}
	cells := []cell{{m.progressWidth(), true}, {idCell, true}, {c.tags + 1, c.tags > 0}}
	if compact {
		cells = append(cells, cell{letterCell, c.kind}, cell{letterCell, c.priority}, cell{0, false})
	} else {
		cells = append(cells, cell{kindCell, c.kind}, cell{priorityCell, c.priority}, cell{stageCell, c.status})
	}
	cells = append(cells, cell{c.title, true}, cell{c.owner + ownerGap, c.owner > 0})
	for k, x := range cells {
		if k == i {
			if !x.shown || x.w <= 0 {
				return 0, 0, false
			}
			if k == len(cells)-1 {
				return at + ownerGap, c.owner, true // the owner cell, after its gap
			}
			return at, max(1, x.w-1), true
		}
		if x.shown {
			at += x.w
		}
	}
	return 0, 0, false
}

// markColumn draws the column cursor into a row (e3sevr, Jørgen 23:20): on
// the cursor row a > stands in the space just before the marked column's
// cell, and the cell is highlighted on every row of the section, so the
// row's > sigil, the in-row > and the column tint together show where the
// cursor is. Without a marked column the row is as it was.
func (m model) markColumn(row string, width int, compact bool, c rowCols, selected bool) string {
	col := m.column
	if !compact {
		col = m.markedColumn()
	}
	if col < 0 || col >= len(gridColumns) {
		return row
	}
	from, w, ok := m.colSpan(col, compact, c)
	if !ok || from+w > width {
		return row
	}
	cell := ansi.Strip(ansi.Cut(row, from, from+w))
	before, after := ansi.Cut(row, 0, from-1), ansi.Cut(row, from+w, width)
	gap := ansi.Cut(row, from-1, from)
	style := m.styles.column
	if selected {
		gap = m.styles.selected.Render(">")
		style = m.styles.columnCursor
	}
	return before + gap + style.Render(cell) + after
}
