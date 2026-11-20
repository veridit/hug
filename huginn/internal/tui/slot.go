package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/huginn/internal/task"
)

// The open task keeps its slot in the pane it was opened from (24403z,
// Jørgen 2026-11-20 10:37Z: "if you had a position and this one is moved,
// then the next one is the one after this one"). When a write (an
// answer, a tick, a take, done, wait) or a change on disk moves it out of
// that pane, the page stays on it, and the pane remembers where it stood:
// n opens the task that came after it there, p the one before, Esc lands
// on that neighbour. Before, the reload put the cursor on the pane's
// first row (clamp), so n opened the second row and Esc followed the task
// into its new section.

// slotGap is where the open task stood in its pane before it left: the
// pane's task ids then, in order, and its index among them.
type slotGap struct {
	id, view string
	pane     int
	rows     []string
	at       int
}

// paneIDs are the current pane's task ids, in their order.
func (m model) paneIDs() []string {
	indices := m.indices()
	ids := make([]string, 0, len(indices))
	for _, i := range indices {
		ids = append(ids, m.visible[i].ID)
	}
	return ids
}

// slotBefore is the open task's id and its slot in the pane, read before a
// snapshot replaces the rows: the gap it already left, else its row now.
// at is -1 when the open task has no row in the pane.
func (m model) slotBefore(sn snapshot) (id string, rows []string, at int) {
	if !m.detail || m.detailPath == "" || sn.err != nil {
		return "", nil, -1
	}
	for _, t := range sn.tasks {
		if t.Path == m.detailPath {
			id = t.ID
			break
		}
	}
	if id == "" {
		return "", nil, -1
	}
	if g := m.gap(); g != nil && g.id == id {
		return id, g.rows, g.at
	}
	rows = m.paneIDs()
	return id, rows, slices.Index(rows, id)
}

// keepSlot runs after a snapshot is shown: when the open task id has left
// the pane it stood in at rows[at], the gap is remembered and the cursor
// goes to the neighbour after it (else the one before), so Esc lands
// there. A task still in its pane keeps its row and makes no gap.
func (m *model) keepSlot(id string, rows []string, at int) {
	if id == "" || at < 0 || slices.Contains(m.paneIDs(), id) {
		m.slot = nil
		return
	}
	m.slot = &slotGap{id: id, view: m.view, pane: m.pane, rows: rows, at: at}
	if i, ok := m.slotNeighbour(1); ok {
		m.cursor = i
	} else if i, ok := m.slotNeighbour(-1); ok {
		m.cursor = i
	}
}

// gap is the slot the open task left in this pane, nil when it has none:
// it is still in the pane, or the page or the pane changed since.
func (m model) gap() *slotGap {
	if !m.detail || m.slot == nil || m.slot.view != m.view || m.slot.pane != m.pane {
		return nil
	}
	if t, ok := m.taskAt(m.detailPath); !ok || t.ID != m.slot.id {
		return nil
	}
	return m.slot
}

// slotNeighbour is the visible index of the nearest task after the gap
// (step 1) or before it (step -1) that is still in the pane.
func (m model) slotNeighbour(step int) (int, bool) {
	g := m.slot
	if g == nil {
		return 0, false
	}
	at := map[string]int{}
	for _, i := range m.indices() {
		at[m.visible[i].ID] = i
	}
	for j := g.at + step; j >= 0 && j < len(g.rows); j += step {
		if i, ok := at[g.rows[j]]; ok && g.rows[j] != g.id {
			return i, true
		}
	}
	return 0, false
}

// stepFromGap is n or p on a task that left its pane: the neighbour
// after or before the gap opens; with none, the page stays and the
// message says so. False when there is no gap.
func (m *model) stepFromGap(step int) bool {
	g := m.gap()
	if g == nil {
		return false
	}
	i, ok := m.slotNeighbour(step)
	if !ok {
		which, side := "last", "after"
		if step < 0 {
			which, side = "first", "before"
		}
		m.message = fmt.Sprintf("%s was the %s in %s; nothing %s it", g.id, which, m.sectionName(), side)
		return true
	}
	m.slot = nil
	m.detailStack = nil
	m.cursor = i
	m.detailPath = m.visible[i].Path
	m.startFocus()
	m.syncDetail()
	m.openTop()
	return true
}

// inSection says whether t is in overview section p (the rule scanPane
// lists the sections by, and placeName names them by).
func (m model) inSection(t task.Task, p int) bool {
	switch p {
	case sectionAsked:
		return m.forYou(t)
	case sectionDoing:
		return t.State == "doing" && !m.waits(t) && !m.forYou(t)
	case sectionWaiting:
		return m.waits(t) && !m.forYou(t)
	case sectionNext:
		// Unassigned todo, never an epic (it finishes through its
		// children); one that depends shows its stage (Jørgen 11:43).
		return t.State == "todo" && !m.waits(t) && !m.forYou(t) && !m.epics[strings.ToLower(t.ID)] && len(t.Assignee) == 0
	case sectionRecent:
		return t.State == "done" || t.State == "dropped"
	case sectionLint:
		return m.linted[t.Path]
	}
	return false
}

// placeName is where t is, in the terms of the view: its board column on
// the board, else its overview section (LINT aside: a task there is in
// another section too). "" when it is in none.
func (m model) placeName(t task.Task) (string, lipgloss.Style) {
	if m.view == "board" {
		p := m.place(t)
		return strings.ToUpper(p), m.styles.status[p]
	}
	for _, p := range []int{sectionAsked, sectionDoing, sectionWaiting, sectionNext, sectionRecent} {
		if m.inSection(t, p) {
			title, _, _ := strings.Cut(overviewSections[p], " · ")
			return title, m.sectionStyle(p).UnsetBold()
		}
	}
	return "", m.styles.dim
}

// taskByID is the task with id on the board as shown.
func (m model) taskByID(id string) (task.Task, bool) {
	for _, t := range m.all {
		if t.ID == id {
			return t, true
		}
	}
	return task.Task{}, false
}

// shown says whether t has a row among the tasks the filters keep.
func (m model) shown(t task.Task) bool {
	return slices.ContainsFunc(m.visible, func(v task.Task) bool { return v.Path == t.Path })
}

// movedNote is the message for a write that moved task id from the place
// named before (24403z check 4, Jørgen 08:44Z): "<ID> moved to <SECTION>",
// with "(outside this filter)" when the filters no longer show it (crab
// 20:07Z). "" when it stayed, or was in no place before (a new task).
func (m model) movedNote(id, before string) string {
	t, ok := m.taskByID(id)
	if !ok || before == "" {
		return ""
	}
	after, _ := m.placeName(t)
	if after == "" || after == before {
		return ""
	}
	note := id + " moved to " + after
	if (m.query != "" || m.filter != (task.Filter{})) && !m.shown(t) {
		note += " (outside this filter)"
	}
	return note
}

// placeBefore is the place of task id as shown now, before a write.
func (m model) placeBefore(id string) string {
	if t, ok := m.taskByID(id); ok {
		name, _ := m.placeName(t)
		return name
	}
	return ""
}

// movedOr is the message after a write: the moved note when the write
// moved the task, else plain; extra follows either (" · " after a note,
// "; " after plain).
func (m model) movedOr(plain, extra string) string {
	if m.movedTo != "" {
		if extra != "" {
			return m.movedTo + " · " + extra
		}
		return m.movedTo
	}
	if extra != "" {
		return plain + "; " + extra
	}
	return plain
}

// paneCrumb is the breadcrumb's name for the pane the open task was
// opened from (24403z check 5, Jørgen 10:02Z): the overview's section,
// the board's column, a list's filter; "" for a list without one.
func (m model) paneCrumb() string {
	switch m.view {
	case "overview":
		title, _, _ := strings.Cut(overviewSections[min(max(m.pane, 0), len(overviewSections)-1)], " · ")
		return title
	case "board":
		return strings.ToUpper(boardStates[min(max(m.pane, 0), len(boardStates)-1)])
	}
	return m.chipText()
}

// nowCrumb is where the open task is now when it has left the pane the
// breadcrumb names (24403z check 6, option A): " · now NEXT UP", the
// place in its section's colour; "" while it is still there.
func (m model) nowCrumb(t task.Task) string {
	if len(m.detailStack) > 0 {
		return "" // a followed task was not opened from the pane
	}
	if m.view == "list" || m.view == "lint" {
		if m.query == "" && m.filter == (task.Filter{}) || m.shown(t) {
			return ""
		}
	} else if slices.Contains(m.paneIDs(), t.ID) {
		return ""
	}
	name, style := m.placeName(t)
	if name == "" {
		return ""
	}
	return m.styles.dim.Render(" · now ") + style.Render(name)
}
