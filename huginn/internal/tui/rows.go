package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/huginn/internal/task"
)

// Rows fit one line at any width (102bc0, Jørgen 2026-11-16: "handle all
// of the things at the same line without a wrap"). Every task row, in every
// view, takes its cells from rowLayout, so a new column cannot reintroduce
// a wrap. When the width runs short, what gives way, in order:
//
//  1. the title is cut, down to titleMin;
//  2. the owner cell (IN PROGRESS) is cut: the branch first, then the
//     owner down to ownerMin;
//  3. whole columns drop: type, then priority, then status.
//
// The gutter, the progress pie and the id always stay. Widths are display
// cells (wide runes and emoji count 2), never bytes.

// titleMin is the title width the other cells give way to; board
// columns, narrow by design, give way at titleMinCompact.
const (
	titleMin        = 20
	titleMinCompact = 12
)

// ownerMin is the least an owner cell is cut to before columns drop.
const ownerMin = 8

// Cell widths, each with the space after it.
const (
	idCell       = 9 // fit(id, 8) and a space
	statusCell   = 8
	kindCell     = 8
	priorityCell = 9
	letterCell   = 2 // a board's one-letter type or priority and a space
	ownerGap     = 2 // the space before the owner cell
)

// rowCols is the cells of a row at one width.
type rowCols struct {
	status, kind, priority bool
	title                  int // the title cell's width
	owner                  int // the owner cell's width; 0: none
}

// rowLayout lays out a row width cells wide. compact rows (board columns)
// have letters for type and priority and no status. owner is the owner
// cell wanted (0 for none); names is the widest owner name without its
// branch, which the owner cell is cut to first.
func (m model) rowLayout(width int, compact bool, owner, names int) rowCols {
	c := rowCols{status: !compact, kind: true, priority: true, owner: max(0, owner)}
	head := 2 + m.progressWidth() + idCell
	on := func(b bool, w int) int {
		if b {
			return w
		}
		return 0
	}
	title := func() int {
		w := width - head
		if c.owner > 0 {
			w -= c.owner + ownerGap
		}
		if compact {
			return w - on(c.kind, letterCell) - on(c.priority, letterCell)
		}
		return w - on(c.status, statusCell) - on(c.kind, kindCell) - on(c.priority, priorityCell)
	}
	least := titleMin
	if compact {
		least = titleMinCompact
	}
	for short := least - title(); short > 0; short = least - title() {
		switch {
		case c.owner > max(names, ownerMin):
			c.owner = max(names, ownerMin, c.owner-short) // the branch first
		case c.owner > ownerMin:
			c.owner = max(ownerMin, c.owner-short)
		case c.kind:
			c.kind = false
		case c.priority:
			c.priority = false
		case c.status:
			c.status = false
		default:
			c.title = max(0, title())
			return c
		}
	}
	c.title = title()
	return c
}

// ownerName is who works on t, without the branch.
func ownerName(t task.Task) string {
	owner := strings.Join(t.Assignee, ", ")
	if owner == "" {
		owner = t.Model
	}
	if owner == "" {
		return "unassigned"
	}
	return oneLine(owner)
}

// ownerNames is the widest owner name among rows, without branches.
func (m model) ownerNames(rows []int) int {
	w := 0
	for _, i := range rows {
		w = max(w, lipgloss.Width(ownerName(m.visible[i])))
	}
	return w
}
