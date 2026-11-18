package tui

// PROTOTYPE (mock/grid, 2026-11-18): the sectioned spreadsheet. Nothing
// here lands; it exists so Jørgen can see the design before we commit.
//
//   - The id is the leftmost column and the row header.
//   - Selection as in a spreadsheet: only the selected row's id cell and
//     the marked column's heading are highlighted; their intersection, the
//     current cell, is underlined. No row bar, no column band.
//   - Ordinary cells are normal weight on the plain background.
//   - MOCKGRID_ORDER=A (default) or B picks one of two column orders.

import (
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// mockOrders are the two column orders to choose from. The owner cell
// stays last in both: it only exists in the overview's owner sections.
var mockOrders = map[string][]string{
	// A: attributes first, then the text. ID ✓ TYPE PRIORITY STAGE TITLE OWNER.
	"A": {"id", "checks", "type", "priority", "stage", "title", "assignee"},
	// B: read first, then the attributes. ID TITLE STAGE ✓ TYPE PRIORITY OWNER.
	"B": {"id", "title", "stage", "checks", "type", "priority", "assignee"},
}

func mockOrder() string {
	if os.Getenv("MOCKGRID_ORDER") == "B" {
		return "B"
	}
	return "A"
}

var columnDefs = map[string]column{
	"checks":   {"✓", "checks", func(rowCols) bool { return true }},
	"id":       {"id", "id", func(rowCols) bool { return true }},
	"type":     {"type", "type", func(c rowCols) bool { return c.kind }},
	"priority": {"priority", "priority", func(c rowCols) bool { return c.priority }},
	"stage":    {"stage", "stage", func(c rowCols) bool { return c.status }},
	"title":    {"title", "title", func(rowCols) bool { return true }},
	"assignee": {"owner", "assignee", func(c rowCols) bool { return c.owner > 0 }},
}

func buildGridColumns() []column {
	var cols []column
	for _, f := range mockOrders[mockOrder()] {
		cols = append(cols, columnDefs[f])
	}
	return cols
}

// gridCell is one cell of a row in display order: its field, its first
// cell, the cells it takes with the space after it, and where its content
// (what the current-cell underline and the heading tint cover) stands.
type gridCell struct {
	field                string
	at, width, from, len int
}

// gridCells lays out the cells of a row laid out as c, in the mock order.
// The gutter (scroll arrow, grab, yank) and the lint mark come first, then
// the id cell " 9taa9d " (8 cells, its pads part the next cell).
func (m model) gridCells(compact bool, c rowCols) []gridCell {
	at := rowHead
	var out []gridCell
	for _, f := range mockOrders[mockOrder()] {
		w, shown := 0, false
		switch f {
		case "id":
			w, shown = idCell, true
		case "checks":
			w, shown = m.progressWidth(), m.progressWidth() > 0
		case "type":
			w, shown = kindCell, c.kind
			if compact {
				w = letterCell
			}
		case "priority":
			w, shown = priorityCell, c.priority
			if compact {
				w = letterCell
			}
		case "stage":
			w, shown = stageCell, !compact && c.status
		case "title":
			w, shown = c.title, c.title > 0
		case "assignee":
			w, shown = c.owner+ownerGap, c.owner > 0
		}
		if !shown || w <= 0 {
			continue
		}
		g := gridCell{field: f, at: at, width: w, from: at, len: max(1, w-1)}
		switch f {
		case "id":
			g.from, g.len = at+1, idCell-2
		case "assignee":
			g.from, g.len = at+ownerGap, c.owner
		}
		out = append(out, g)
		at += w
	}
	return out
}

// currentField is the field the column cursor marks in rows laid out as
// compact, or "".
func (m model) currentField(compact bool) string {
	col := m.column
	if !compact {
		col = m.markedColumn()
	}
	if col < 0 || col >= len(gridColumns) {
		return ""
	}
	return gridColumns[col].field
}

// gridRow draws t as cells: title is the text cell already composed
// (place, files, request marks), word the stage word.
func (m model) gridRow(t task.Task, width int, selected, compact bool, pointer, title, word string, c rowCols) string {
	closed := t.State == "done" || t.State == "dropped"
	current := ""
	if selected {
		current = m.currentField(compact)
	}
	row := pointer + " " + m.marker(t.Path)
	for _, g := range m.gridCells(compact, c) {
		cur := g.field == current
		// cell renders plain text in style, underlined when it is the
		// current cell (the row's id meets the column's heading).
		cell := func(style lipgloss.Style, plain string, w int) string {
			if closed && g.field != "stage" {
				style = m.styles.dim
			}
			if cur {
				// The current cell: its text underlined, not its padding.
				text := strings.TrimRight(fit(plain, w), " ")
				if text == "" {
					text = "·" // an empty current cell still shows where it is
				}
				pad := strings.Repeat(" ", max(0, w-lipgloss.Width(text)))
				if cellBlink() && !m.pendingV {
					// MOCKGRID_CELLBLINK=1: the current cell is the
					// blinking cursor while nothing else waits for a key.
					return m.cursorCell(text) + pad
				}
				return style.Underline(true).Render(text) + pad
			}
			return style.Render(fit(plain, w))
		}
		switch g.field {
		case "id":
			// The row header: " 9taa9d " in the id colour, and on the
			// selected row in the selection tint its column heading takes.
			id := " " + fit(oneLine(t.ID), idCell-3) + " "
			if selected {
				row += m.styles.rowHeader.Render(id) + " "
			} else {
				row += m.styles.idCell.Render(id) + " "
			}
		case "checks":
			if cur || closed {
				row += cell(m.styles.normal, progressText(t), g.width-1) + " "
			} else {
				row += m.progress(t)
			}
		case "type":
			if compact {
				row += cell(m.styles.kind[t.Type], strings.ToUpper(firstLetter(t.Type)), 1) + " "
			} else {
				row += cell(m.styles.kind[t.Type], oneLine(t.Type), g.len) + " "
			}
		case "priority":
			if compact {
				row += cell(m.styles.priority[t.Priority], strings.ToUpper(firstLetter(t.Priority)), 1) + " "
			} else {
				row += cell(m.styles.priority[t.Priority], oneLine(t.Priority), g.len) + " "
			}
		case "stage":
			style := m.stageStyle(t, word)
			if closed {
				style = m.styles.status[t.State]
				word = t.State
			}
			row += cell(style, word, g.len) + " "
		case "title":
			if cur || closed {
				row += cell(m.styles.normal, ansi.Strip(title), g.len) + " "
			} else {
				row += fit(title, g.len) + " "
			}
		case "assignee":
			row += "  " + cell(m.styles.owner, m.rowOwner(t), g.len)
		}
	}
	return fit(row, width)
}

func firstLetter(s string) string {
	s = oneLine(s)
	if s == "" {
		return " "
	}
	return s[:1]
}

// gridHeader is the column header for rows laid out as c: the labels dim
// and normal weight; the marked column's heading in the selection tint,
// the one other place the selection shows. Its cells stand where the
// rows' cells do (gridCells).
func (m model) gridHeader(width int, compact bool, c rowCols) string {
	marked := m.currentField(compact)
	s := strings.Repeat(" ", rowHead)
	for _, g := range m.gridCells(compact, c) {
		label := strings.ToUpper(columnDefs[g.field].label)
		if compact && (g.field == "type" || g.field == "priority") {
			label = label[:1]
		}
		style := m.styles.colHead
		if g.field == marked {
			style = m.styles.colHeadSelected
		}
		text := func(w int) string {
			arrow := strings.TrimSpace(m.arrow(g.field))
			if arrow == "" {
				return fit(label, w)
			}
			return fit(ansi.Truncate(label, max(0, w-2), "")+" "+arrow, w)
		}
		switch g.field {
		case "id":
			s += style.Render(" "+text(idCell-3)+" ") + " "
		case "assignee":
			s += "  " + style.Render(text(g.len))
		default:
			s += style.Render(text(g.len)) + strings.Repeat(" ", max(0, g.width-g.len))
		}
	}
	return fit(s, width)
}

// mockOn says whether the prototype draws; MOCKGRID_OFF=1 shows main's design.
func mockOn() bool { return os.Getenv("MOCKGRID_OFF") == "" }

// vBlinkMsg flips the pending-v cursor (mock/grid). gen ties it to one
// press of v, so an old tick cannot blink a later prefix.
type vBlinkMsg struct{ gen int }

// vBlinkRate is the bubbles cursor's own blink rate (cursor.BlinkSpeed,
// 530 ms), so the v cursor and the answer box cursor (qpp30y) blink as one.
const vBlinkRate = 530 * time.Millisecond

// vBlink schedules the next blink; MOCKGRID_BLINK=static holds the cursor
// on, for deterministic captures.
func (m model) vBlink() tea.Cmd {
	if os.Getenv("MOCKGRID_BLINK") == "static" || (!m.pendingV && !cellBlink()) {
		return nil
	}
	gen := m.vBlinkGen
	return tea.Tick(vBlinkRate, func(time.Time) tea.Msg { return vBlinkMsg{gen} })
}

// cellBlink says whether the grid's current cell blinks as the cursor
// (MOCKGRID_CELLBLINK=1). Off by default: in a list the selection shows on
// the id and the heading, and the cell underline holds still.
func cellBlink() bool { return os.Getenv("MOCKGRID_CELLBLINK") == "1" }
