package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/huginn/internal/task"
)

// Rows fit one line at any width (102bc0, Jørgen 2026-11-16: "handle all
// of the things at the same line without a wrap"). Every task row, in every
// view, takes its cells from rowLayout, so a new column cannot reintroduce
// a wrap. The cells read left to right as a spreadsheet (911j70, order B,
// Jørgen 2026-11-20 08:26Z): ID TITLE STAGE ✓ TYPE PRIORITY OWNER. When the
// width runs short, what gives way, in order:
//
//  1. the title is cut, down to titleMin;
//  2. the owner cell (IN PROGRESS) is cut: the branch first, then the
//     owner down to ownerMin;
//  3. whole columns drop: type, then priority, then stage.
//
// Tags are no column (911j70): they are a dim prefix of the TITLE cell,
// so they take no width of their own (titleTags).
//
// The gutter, the id and the progress pie always stay. Widths are display
// cells (wide runes and emoji count 2), never bytes.

// titleMin is the title width the other cells give way to; board
// columns, narrow by design, give way at titleMinCompact.
const (
	titleMin        = 20
	titleMinCompact = 12
)

// ownerMin is the least an owner cell is cut to before columns drop.
const ownerMin = 8

// tagMax is the most the tags prefix of a title takes (Jørgen 19:35Z: at
// most 22 columns).
const tagMax = 22

// Cell widths, each with the space before the next cell.
const (
	rowHead      = 3 // the gutter, a space, and the lint mark's cell before the id (e3sevr)
	idCell       = 9 // fit(id, 8) and a space
	kindCell     = 8
	priorityCell = 9
	letterCell   = 2 // a board's one-letter type or priority and a space
	ownerGap     = 2 // the space before the owner cell
)

// rowCols is the cells of a row at one width.
type rowCols struct {
	status, kind, priority bool
	checks                 bool // the ✓ cell: some task on the board has checks
	title                  int  // the title cell's width, with the space after it
	owner                  int  // the owner cell's width; 0: none
}

// rowLayout lays out a row width cells wide. compact rows (board columns)
// have letters for type and priority and no status. owner is the owner
// cell wanted (0 for none); names is the widest owner name without its
// branch, which the owner cell is cut to first.
func (m model) rowLayout(width int, compact bool, owner, names int) rowCols {
	return m.rowLayoutSame(width, compact, owner, names, sameCols{})
}

// rowLayoutSame is rowLayout for a section whose tasks all share the
// columns s names (1bbe96): those are not drawn, the heading says them
// once, and the title takes their width.
func (m model) rowLayoutSame(width int, compact bool, owner, names int, s sameCols) rowCols {
	c := rowCols{status: !compact && s.stage == "", checks: m.progressWidth() > 0, kind: s.kind == "", priority: s.priority == "", owner: max(0, owner)}
	if s.owner != "" {
		c.owner = 0
	}
	head := rowHead + m.progressWidth() + idCell
	title := func() int {
		w := width - head
		if c.owner > 0 {
			w -= c.owner + ownerGap
		}
		if compact {
			return w - on(c.kind, letterCell) - on(c.priority, letterCell)
		}
		return w - on(c.status, stageCell) - on(c.kind, kindCell) - on(c.priority, priorityCell)
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

// on is w when b, else 0: a cell's width when it is shown.
func on(b bool, w int) int {
	if b {
		return w
	}
	return 0
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

// sameCols are the values every task of a section shares, one per row
// column; "" where the tasks differ (1bbe96).
type sameCols struct{ stage, kind, priority, owner string }

func (s sameCols) any() bool { return s != sameCols{} }

// sectionSame is what every task of section p shares, decided on all of
// its tasks after filters (rawPane: folded ones too), never on the rows
// on screen, so scrolling cannot make a column come or go. A section of
// one task keeps its columns, and an empty value is no reason to hide
// one. The board's columns are states: it is left as is. The owner cell
// is the overview's owner sections' only.
func (m model) sectionSame(p int) sameCols {
	if m.view != "overview" && m.view != "list" && m.view != "lint" {
		return sameCols{}
	}
	rows := m.rawPane(p)
	if len(rows) < 2 {
		return sameCols{}
	}
	ownerCell := m.view == "overview" && ownerSection(p)
	first := m.visible[rows[0]]
	s := sameCols{stage: m.stageCell(first), kind: oneLine(first.Type), priority: oneLine(first.Priority)}
	if ownerCell {
		s.owner = m.rowOwner(first)
	}
	for _, i := range rows[1:] {
		t := m.visible[i]
		if s.stage != m.stageCell(t) {
			s.stage = ""
		}
		if s.kind != oneLine(t.Type) {
			s.kind = ""
		}
		if s.priority != oneLine(t.Priority) {
			s.priority = ""
		}
		if s.owner != "" && s.owner != m.rowOwner(t) {
			s.owner = ""
		}
		if !s.any() {
			break
		}
	}
	return s
}

// stageCell is the word t's STAGE cell shows: its stage, or for a closed
// task its state.
func (m model) stageCell(t task.Task) string {
	word, _ := m.rowStage(t)
	if word == "" {
		word = t.State
	}
	return word
}

// sectionLayout is section p's rows at full width: the owner cell in the
// overview's owner sections, without the columns its tasks all share.
func (m model) sectionLayout(p int) rowCols {
	s := m.sectionSame(p)
	if m.view == "overview" && ownerSection(p) {
		return m.rowLayoutSame(m.width, false, m.ownerWidth(), m.ownerNames(m.paneIndices(p)), s)
	}
	return m.rowLayoutSame(m.width, false, 0, 0, s)
}

// sameLabel is the heading's '· all active · all feature' for what s
// names, each value in its own colour (1bbe96); t is one of the
// section's tasks, for the stage's colour.
func (m model) sameLabel(s sameCols, t task.Task) string {
	out := ""
	piece := func(value string, style lipgloss.Style) {
		if value != "" {
			out += m.styles.dim.Render(" · all ") + style.Render(value)
		}
	}
	piece(s.stage, m.stageStyle(t, s.stage))
	piece(s.kind, m.styles.kind[s.kind])
	piece(s.priority, m.styles.priority[s.priority])
	piece(s.owner, m.styles.accent)
	return out
}
