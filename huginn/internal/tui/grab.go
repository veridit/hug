package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

// Moving a task far in one go (3xcesq). Two ways, one commit each:
//
//   - grab: m grabs the selected task; j/k (with a count), g/G move it on
//     screen, nothing written; Enter drops it with one commit; Esc puts it
//     back.
//   - yank and paste: y marks the selected task; P puts it above the
//     selected row, p below, in the same section.
//
// Both move within the rows shown in one section, as J and K do (ksv323),
// and both end in Store.Move, so the rank is one sparse insert.
type grabState struct {
	active        bool
	path, id      string
	visible       []task.Task // the order before the grab, for Esc
	cursor, start int         // cursor and the task's row when grabbed
	count         string      // a count typed before j/k, as in vim
}

// moveRefusal says why the current section cannot be reordered, or "".
func (m model) moveRefusal() string {
	if len(m.sort) > 0 {
		return sortRefusal // a move changes rank, which these rows are not in (mkzcqr)
	}
	if m.view == "overview" && m.pane == sectionRecent {
		return "RECENTLY DONE is ordered by close time: move tasks in NEXT UP or the list"
	}
	return ""
}

// rowOf is the task's row among rows, or -1.
func (m model) rowOf(rows []int, path string) int {
	for p, i := range rows {
		if m.visible[i].Path == path {
			return p
		}
	}
	return -1
}

// startGrab grabs the selected task (m).
func (m *model) startGrab() {
	t, ok := m.selected()
	if !ok {
		return
	}
	if why := m.moveRefusal(); why != "" {
		m.message = why
		return
	}
	rows := m.indices()
	if len(rows) < 2 {
		m.message = fmt.Sprintf("%s is the only row in %s: nothing to move past", t.ID, m.sectionName())
		return
	}
	m.grab = grabState{active: true, path: t.Path, id: t.ID, visible: append([]task.Task(nil), m.visible...), cursor: m.cursor, start: m.rowOf(rows, t.Path)}
	m.message = ""
}

// grabMove moves the grabbed task to row to (clamped) on screen only.
func (m *model) grabMove(to int) {
	rows := m.indices()
	from := m.rowOf(rows, m.grab.path)
	to = max(0, min(to, len(rows)-1))
	if from < 0 || to == from {
		return
	}
	t := m.visible[rows[from]]
	target := m.visible[rows[to]].Path
	var out []task.Task
	for _, v := range m.visible {
		if v.Path == t.Path {
			continue
		}
		if v.Path == target && to < from {
			out = append(out, t)
		}
		out = append(out, v)
		if v.Path == target && to > from {
			out = append(out, t)
		}
	}
	m.visible = out
	m.visibleGen = nextVisibleGen()
	for i, v := range m.visible {
		if v.Path == t.Path {
			m.cursor = i
		}
	}
}

// cancelGrab puts the grabbed task back where it was.
func (m *model) cancelGrab() {
	m.visible, m.cursor = m.grab.visible, m.grab.cursor
	m.visibleGen = nextVisibleGen()
	m.grab = grabState{}
	m.message = "move cancelled; nothing changed"
}

// dropGrab writes the move with one commit, or nothing when the task is
// where it started.
func (m *model) dropGrab() {
	rows := m.indices()
	at := m.rowOf(rows, m.grab.path)
	g := m.grab
	if at < 0 || at == g.start {
		m.visible, m.cursor = g.visible, g.cursor
		m.visibleGen = nextVisibleGen()
		m.grab = grabState{}
		m.message = "dropped where it was; nothing changed"
		return
	}
	// Moving up lands above the row now below it; moving down lands below
	// the row now above it. Either way the neighbour is a row on screen.
	var o task.MoveOptions
	var where string
	var target task.Task
	if at < g.start {
		target = m.visible[rows[at+1]]
		o, where = task.MoveOptions{Before: target.ID}, "above"
	} else {
		target = m.visible[rows[at-1]]
		o, where = task.MoveOptions{After: target.ID}, "below"
	}
	m.grab = grabState{}
	m.moveTo(g.id, o, where, target.ID)
}

// moveTo runs one move with one commit and says where the task went.
func (m *model) moveTo(id string, o task.MoveOptions, where, target string) {
	release := m.hold()
	r, err := m.store.Move(id, o)
	m.finish(r, err, "move")
	release()
	if err == nil {
		// Say where it went, in the words of the screen; keep warnings and a
		// failed commit, which finish put after or instead of "moved ID".
		done := fmt.Sprintf("moved %s %s %s%s", id, where, target, m.rowPlace(r.Task.Path))
		m.message = strings.Replace(m.message, "moved "+r.Task.ID, done, 1)
		m.result = m.message
	}
}

func (m model) updateGrab(k string) (tea.Model, tea.Cmd) {
	count := 1
	if n, err := strconv.Atoi(m.grab.count); err == nil && n > 0 {
		count = n
	}
	if len(k) == 1 && k[0] >= '0' && k[0] <= '9' && (k != "0" || m.grab.count != "") {
		m.grab.count += k
		return m, nil
	}
	rows := m.indices()
	at := m.rowOf(rows, m.grab.path)
	switch k {
	case "j", "down":
		m.grabMove(at + count)
	case "k", "up":
		m.grabMove(at - count)
	case "<", "home":
		m.grabMove(0)
	case ">", "end":
		m.grabMove(len(rows) - 1)
	case "enter":
		m.dropGrab()
	case "esc":
		m.cancelGrab()
	case "h", "l", "left", "right", "tab", "shift+tab":
		m.message = fmt.Sprintf("%s moves within %s: Enter drops it, Esc cancels", m.grab.id, m.sectionName())
	default:
		m.message = "moving " + m.grab.id + ": j/k move · < > top/bottom · Enter drop · Esc cancel"
	}
	m.grab.count = ""
	return m, nil
}

// yank marks the selected task for paste (y).
func (m *model) yank() {
	t, ok := m.selected()
	if !ok {
		return
	}
	if why := m.moveRefusal(); why != "" {
		m.message = why
		return
	}
	m.yanked = t.ID
	m.message = fmt.Sprintf("yanked %s: P puts it above the selected row, p below · Esc forgets", t.ID)
}

// paste moves the yanked task above (P) or below (p) the selected row, in
// the same section, with one commit.
func (m *model) paste(k string) {
	target, ok := m.selected()
	if !ok {
		return
	}
	if why := m.moveRefusal(); why != "" {
		m.message = why
		return
	}
	id := m.yanked
	rows := m.indices()
	shown := false
	for _, i := range rows {
		if m.visible[i].ID == id {
			shown = true
		}
	}
	switch {
	case !shown:
		m.message = fmt.Sprintf("%s is not shown in %s: paste it where it is shown, or Esc forgets it", id, m.sectionName())
		return
	case target.ID == id:
		m.message = fmt.Sprintf("%s is the selected row: select the row to paste it next to", id)
		return
	}
	o, where := task.MoveOptions{Before: target.ID}, "above"
	if k == "p" {
		o, where = task.MoveOptions{After: target.ID}, "below"
	}
	m.yanked = ""
	m.moveTo(id, o, where, target.ID)
}
