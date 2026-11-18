package tui

// PROTOTYPE (mock/grid, ported for the 911j70 compare sheets on
// 2026-11-18 by kikazaru): the sectioned spreadsheet over current main:
// tags right after the id (txjkrm), main's quiet view line (qh9ep1
// check 9) and the indigo depends (zhvkmj). Nothing here lands; it exists
// so Jørgen can choose A or B on today's screen.
//
//   - The id is the leftmost column and the row header.
//   - Selection as in a spreadsheet: only the selected row's id cell and
//     the marked column's heading are highlighted; their intersection, the
//     current cell, is underlined. No row bar, no column band.
//   - Ordinary cells are normal weight on the plain background.
//   - MOCKGRID_ORDER=A (default) or B picks one of two column orders;
//     tags sit right after the id in both (Jørgen 19:35Z).

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// mockOrders are the two column orders to choose from. The owner cell
// stays last in both: it only exists in the overview's owner sections.
var mockOrders = map[string][]string{
	// A: attributes first, then the text. ID TAGS ✓ TYPE PRIORITY STAGE TITLE OWNER.
	"A": {"id", "tags", "checks", "type", "priority", "stage", "title", "assignee"},
	// B: read first, then the attributes. ID TAGS TITLE STAGE ✓ TYPE PRIORITY OWNER.
	"B": {"id", "tags", "title", "stage", "checks", "type", "priority", "assignee"},
}

func mockOrder() string {
	if os.Getenv("MOCKGRID_ORDER") == "B" {
		return "B"
	}
	return "A"
}

func buildGridColumns() []column {
	if !mockOn() {
		return mainColumns
	}
	defs := map[string]column{}
	for _, c := range mainColumns {
		defs[c.field] = c
	}
	var cols []column
	for _, f := range mockOrders[mockOrder()] {
		cols = append(cols, defs[f])
	}
	return cols
}

func columnLabel(field string) string {
	for _, c := range mainColumns {
		if c.field == field {
			return c.label
		}
	}
	return field
}

// gridCell is one cell of a row in display order: its field, its first
// cell, the cells it takes with the space after it, and where its content
// (what the current-cell underline and the heading tint cover) stands.
type gridCell struct {
	field                string
	at, width, from, len int
}

// gridCells lays out the cells of a row laid out as c, in the mock order.
func (m model) gridCells(compact bool, c rowCols) []gridCell {
	at := rowHead
	var out []gridCell
	for _, f := range mockOrders[mockOrder()] {
		w, shown := 0, false
		switch f {
		case "id":
			w, shown = idCell, true
		case "tags":
			w, shown = c.tags+1, c.tags > 0
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

// currentField is the field the column cursor marks, or "".
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
		cell := func(style lipgloss.Style, plain string, w int) string {
			if closed && g.field != "stage" {
				style = m.styles.dim
			}
			if cur {
				text := strings.TrimRight(fit(plain, w), " ")
				if text == "" {
					text = "·" // an empty current cell still shows where it is
				}
				pad := strings.Repeat(" ", max(0, w-lipgloss.Width(text)))
				return style.Underline(true).Render(text) + pad
			}
			return style.Render(fit(plain, w))
		}
		switch g.field {
		case "id":
			id := " " + fit(oneLine(t.ID), idCell-3) + " "
			if selected {
				row += m.styles.rowHeader.Render(id) + " "
			} else {
				row += m.styles.idCell.Render(id) + " "
			}
		case "tags":
			row += cell(m.styles.dim, m.tagText(t), g.len) + " "
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
// and normal weight; the marked column's heading in the selection tint.
func (m model) gridHeader(width int, compact bool, c rowCols) string {
	marked := m.currentField(compact)
	s := strings.Repeat(" ", rowHead)
	for _, g := range m.gridCells(compact, c) {
		label := strings.ToUpper(columnLabel(g.field))
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

// mockOn says whether the prototype draws: only with MOCKGRID=1, so
// main's tests see main's design.
func mockOn() bool { return os.Getenv("MOCKGRID") == "1" }
