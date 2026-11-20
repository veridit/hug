package tui

import (
	"fmt"
	"os/exec"
	"strings"

	bkey "github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/editor"
	"github.com/veridit/hug/huginn/internal/task"
)

// Marks (rxr4z8, Jørgen 23:42): in the overview, the list and the board,
// Space marks the task under the cursor and steps to the next row. While
// tasks are marked the footer offers what can be done to every one of
// them, and the action asks once, naming them.
//
// A mark is one sign in the row's gutter and nothing else (crab 20:31, the
// calm rule: one cue per meaning, no background fills). The cursor on a
// marked row shows cursorOnMark, one sign for both (check 7).
const (
	markSign     = "◆"
	cursorOnMark = "⋗" // Jørgen is asked: ⋗ or ▸ (cursor-on-mark.md)
	// narrowMarks is the width under which the footer names no actions,
	// only where to find them (check 8: never bare letters).
	narrowMarks = 70
)

// markedActions are the keys that act on the marked tasks, in the order
// the footer offers them, with their words.
var markedActions = []struct{ key, word, help string }{
	{"m", "move", "move the marked as one block: j/k, Enter places them"},
	{"t", "take", "take the marked task"},
	{"d", "done", "mark every marked task done"},
	{"x", "drop", "drop every marked task, with one reason"},
	{"w", "wait", "park every marked task on what it waits on"},
	{"e", "edit", "edit every marked task in the editor"},
	// One request for exactly the marked set (ec6z6j check 5, Jørgen
	// 2026-11-20: option A, the box at the bottom of the list).
	{"r", "request", "write one request for the marked tasks, the box under the list"},
}

func (m model) isMarked(id string) bool { return contains(m.marks, id) }

// markedTasks are the marked tasks that still exist, in the order marked.
func (m model) markedTasks() []task.Task {
	var out []task.Task
	for _, id := range m.marks {
		for _, t := range m.all {
			if t.ID == id {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

// toggleMark marks the task under the cursor, or unmarks it, and steps to
// the next row (check 1).
func (m *model) toggleMark() {
	t, ok := m.selected()
	if !ok {
		return
	}
	if m.isMarked(t.ID) {
		m.unmark([]string{t.ID})
	} else {
		m.marks = append(m.marks, t.ID)
	}
	m.message = ""
	m.move(1, false)
}

// unmark drops ids from the marks.
func (m *model) unmark(ids []string) {
	var keep []string
	for _, id := range m.marks {
		if !contains(ids, id) {
			keep = append(keep, id)
		}
	}
	m.marks = keep
}

// marksIn counts the marked tasks in section p, shown or scrolled away.
func (m model) marksIn(p int) int {
	n := 0
	for _, i := range m.rawPane(p) {
		if m.isMarked(m.visible[i].ID) {
			n++
		}
	}
	return n
}

// markedRefusal says why key k cannot act on every marked task, or "".
// Only actions with no refusal are offered (rxr4z8 principle).
func (m model) markedRefusal(k string, ts []task.Task) string {
	if len(ts) == 0 {
		return "the marked tasks are gone: Esc unmarks"
	}
	for _, t := range ts {
		switch k {
		case "t":
			// Several are taken together (byj479): the first plainly, the
			// rest with it, so each must be a todo task. One may also be
			// someone else's to take for review, as t does on a row.
			if len(ts) > 1 && t.State != "todo" {
				return t.ID + " is " + t.State + ": t takes several todo tasks together"
			}
			if t.State == "done" || t.State == "dropped" || (t.State == "doing" && contains(t.Assignee, m.me())) {
				return t.ID + " is " + t.State + ": t takes a task that is not yours yet"
			}
		case "d":
			if t.State != "doing" {
				return t.ID + " is " + t.State + ": d needs every marked task doing (t takes it first)"
			}
			if m.epics[strings.ToLower(t.ID)] {
				return t.ID + " is an epic: it finishes through its children"
			}
		case "x":
			if t.State == "dropped" {
				return t.ID + " is dropped already"
			}
		case "w":
			if t.State != "doing" {
				return t.ID + " is " + t.State + ": only a doing task waits"
			}
		case "m":
			if why := m.moveRefusal(); why != "" {
				return why
			}
			if !m.inPane(t.Path) {
				return "m moves marked tasks within one section: " + t.ID + " is not in " + m.sectionName()
			}
		}
	}
	if k == "m" && len(m.indices()) <= len(ts) {
		return "every row is marked: nothing to move past"
	}
	return ""
}

// inPane says whether the task at path is a row of the current section.
func (m model) inPane(path string) bool {
	for _, i := range m.indices() {
		if m.visible[i].Path == path {
			return true
		}
	}
	return false
}

// markedHints are the actions valid for every marked task.
func (m model) markedHints() []bkey.Binding {
	ts := m.markedTasks()
	var out []bkey.Binding
	for _, a := range markedActions {
		if m.markedRefusal(a.key, ts) == "" {
			out = append(out, hint(a.key, a.word))
		}
	}
	return out
}

// markedFooter is the footer while tasks are marked (check 2): "N marked
// · " the actions, " · Esc unmark"; narrow, "? actions" in their place
// (check 8).
func (m model) markedFooter() string {
	head := fmt.Sprintf("%d marked · ", len(m.marks))
	tail := " · Esc unmark"
	short := head + "? actions" + tail
	if m.width < narrowMarks {
		return short
	}
	var parts []string
	for _, it := range itemsOf(m.markedHints()) {
		if it.label != "" {
			parts = append(parts, it.label+" "+it.words)
		} else {
			parts = append(parts, it.words)
		}
	}
	if len(parts) == 0 {
		return head + "nothing fits all of them" + tail
	}
	full := head + strings.Join(parts, " · ") + tail
	if ansi.StringWidth(plainMarks(full)) > m.width {
		return short
	}
	return full
}

// markedHelp is the ? section while tasks are marked: what each action
// does to them, in words (check 8).
func (m model) markedHelp() helpSection {
	ts := m.markedTasks()
	rows := []helpRow{
		row("Space", "mark or unmark the task, then the next row", " "),
		row("j/k", "move; Space marks as it goes", "j", "k"),
	}
	for _, a := range markedActions {
		if m.markedRefusal(a.key, ts) == "" {
			rows = append(rows, row(a.key, a.help, a.key))
		}
	}
	rows = append(rows, row("Esc", "unmark all", "esc"))
	return helpSection{fmt.Sprintf("%d MARKED", len(m.marks)), rows}
}

// markedKey runs action k on the marked tasks: refused with why, or the
// prompt or confirm that asks once.
func (m *model) markedKey(k string) tea.Cmd {
	ts := m.markedTasks()
	if why := m.markedRefusal(k, ts); why != "" {
		m.message = why
		return nil
	}
	ids := make([]string, len(ts))
	for i, t := range ts {
		ids[i] = t.ID
	}
	switch k {
	case "m":
		m.grabMarked(ts)
		return nil
	case "t":
		return m.takeMarked(ids)
	case "r":
		return m.requestMarked(ids)
	}
	m.bulk = ids
	switch k {
	case "x":
		cmd := m.openInput(dropPrompt, "")
		m.bulk = ids
		return cmd
	case "w":
		cmd := m.openInput(waitPrompt, "")
		m.bulk = ids
		return cmd
	case "e":
		m.confirm, m.pendingValue = "e", strings.Join(editor.Command(), " ")
	default:
		m.confirm, m.pendingValue = k, ""
	}
	return nil
}

// takeMarked asks to take the marked tasks: one plainly; several, the
// first plainly and the rest with it (byj479: tasks that ship together are
// both doing), after one confirm naming them.
func (m *model) takeMarked(ids []string) tea.Cmd {
	m.bulk = ids
	m.confirm, m.pendingValue = "t", ""
	return nil
}

// bulkQuestion is the confirm for the marked: the action, the count and
// the ids.
func (m model) bulkQuestion() string {
	verb := map[string]string{"t": "Take", "d": "Done", "x": "Drop", "w": "Park", "e": "Edit"}[m.confirm]
	noun := "tasks"
	if len(m.bulk) == 1 {
		noun = "task"
	}
	q := fmt.Sprintf("%s %d %s: %s", verb, len(m.bulk), noun, strings.Join(m.bulk, ", "))
	if len(m.bulk) == 1 {
		for _, t := range m.all {
			if t.ID == m.bulk[0] {
				q += " " + oneLine(t.Title)
				break
			}
		}
	}
	switch m.confirm {
	case "t":
		if len(m.bulk) > 1 {
			q += ", " + strings.Join(m.bulk[1:], ", ") + " with " + m.bulk[0]
		}
	case "x":
		q += ": " + oneLine(m.pendingValue)
	case "w":
		q += ", waiting on " + oneLine(m.pendingValue)
	case "e":
		q += " in " + oneLine(m.pendingValue)
	}
	return q
}

// runBulk does the confirmed action to each marked task, one write and
// one commit each, in the board's turn. A refused task is named with its
// first fix: line and stays marked; the others go ahead and are unmarked
// (checks 3 and 9).
func (m *model) runBulk() tea.Cmd {
	k, value, ids := m.confirm, m.pendingValue, m.bulk
	m.confirm, m.pendingValue, m.bulk = "", "", nil
	if k == "e" {
		return m.editMarked(ids)
	}
	action := map[string]string{"t": "take", "d": "done", "x": "drop", "w": "wait"}[k]
	var done, refused, fixes []string
	for n, id := range ids {
		with := ""
		if k == "t" && n > 0 {
			if len(done) == 0 || done[0] != ids[0] {
				// The first was refused: the rest have nothing to go with.
				refused = append(refused, id+" not taken: "+ids[0]+" was not")
				continue
			}
			with = ids[0]
		}
		if err := m.bulkOne(k, action, id, value, with); err != nil {
			first, _, _ := strings.Cut(err.Error(), "\n")
			refused = append(refused, id+" refused: "+strings.Replace(first, " "+id, "", 1))
			for _, line := range strings.Split(err.Error(), "\n") {
				if strings.HasPrefix(line, "fix: ") {
					fixes = append(fixes, line)
					break
				}
			}
			continue
		}
		done = append(done, id)
	}
	path := ""
	if t, ok := m.selected(); ok {
		path = t.Path
	}
	m.refresh(path)
	m.unmark(done)
	msg := fmt.Sprintf("%s %d of %d", pastTense(action), len(done), len(ids))
	for _, r := range refused {
		msg += " · " + r
	}
	for _, f := range fixes {
		msg += "\n" + f
	}
	m.message = msg
	return nil
}

// bulkOne is one write of a bulk action and its commit; with is the task
// a take goes with (byj479), "" for a plain take.
func (m *model) bulkOne(k, action, id, value, with string) error {
	viaTake := k == "t" && with == "" && m.opts.Take != nil // it takes the turn and commits itself
	release := func() {}
	if !viaTake {
		release = m.hold()
	}
	defer release()
	var r task.Result
	var err error
	switch k {
	case "t":
		switch {
		case with != "":
			r, err = m.store.TakeWith(id, with, m.me())
		case viaTake:
			r, err = m.opts.Take(id)
		default:
			r, err = m.store.Take(id, m.me())
		}
	case "d":
		r, err = m.store.Done(id)
	case "x":
		r, err = m.store.Drop(id, value)
	case "w":
		r, err = m.store.Wait(id, value, m.me())
	}
	if err != nil {
		return err
	}
	if r.Changed && m.opts.Commit != nil && !viaTake {
		if with != "" {
			return m.opts.Commit("hug task take " + id + " --with " + with)
		}
		return m.opts.Commit("hug task " + action + " " + id)
	}
	return nil
}

// editMarked opens every marked task in one editor; when it closes, the
// edit is committed once and the marks go.
func (m *model) editMarked(ids []string) tea.Cmd {
	var files []string
	first := ""
	for _, t := range m.all {
		if !contains(ids, t.ID) {
			continue
		}
		f, e := m.editorPath(t)
		if e != nil {
			m.message = t.ID + ": " + e.Error()
			return nil
		}
		if first == "" {
			first = t.Path
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		m.message = "the marked tasks are gone: Esc unmarks"
		return nil
	}
	args := editor.Command()
	cmd := exec.Command(args[0], append(args[1:], files...)...)
	m.editBefore = map[string]bool{}
	for _, f := range m.editFindings(first) {
		m.editBefore[f.Problem] = true
	}
	m.editing = true
	return tea.ExecProcess(cmd, func(e error) tea.Msg { return editorMsg{path: first, err: e, ids: ids} })
}

// grabMarked grabs the marked tasks as one block (check 4): they gather at
// the first marked row in their order on screen; j/k move the block,
// Enter writes it, Esc puts every row back.
func (m *model) grabMarked(ts []task.Task) {
	rows := m.indices()
	var block []string
	for _, i := range rows {
		if m.isMarked(m.visible[i].ID) {
			block = append(block, m.visible[i].Path)
		}
	}
	before := append([]task.Task(nil), m.visible...)
	// Gather: the block's rows in screen order from the first one, the
	// other rows in theirs after.
	at := m.rowOf(rows, block[0])
	var seq []string
	for _, i := range rows {
		if p := m.visible[i].Path; !contains(block, p) {
			seq = append(seq, p)
		}
	}
	seq = append(seq[:at], append(append([]string(nil), block...), seq[at:]...)...)
	m.placeRows(rows, seq)
	cursor := 0
	for i, v := range m.visible {
		if v.Path == block[0] {
			cursor = i
		}
	}
	id := ts[0].ID
	for _, t := range ts {
		if t.Path == block[0] {
			id = t.ID
		}
	}
	m.grab = grabState{active: true, path: block[0], id: id, block: block, visible: before, cursor: m.cursor, start: at}
	m.cursor = cursor
	m.message = ""
}

// placeRows puts the tasks at paths seq into the slots of rows, in order,
// on screen only.
func (m *model) placeRows(rows []int, seq []string) {
	byPath := map[string]task.Task{}
	for _, i := range rows {
		byPath[m.visible[i].Path] = m.visible[i]
	}
	slots := append([]int(nil), rows...)
	sortInts(slots)
	out := append([]task.Task(nil), m.visible...)
	for n, slot := range slots {
		out[slot] = byPath[seq[n]]
	}
	m.visible = out
	m.visibleGen = nextVisibleGen()
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

// blockMove moves the grabbed block so its first row is row to (clamped),
// on screen only.
func (m *model) blockMove(to int) {
	rows := m.indices()
	block := m.grab.block
	to = max(0, min(to, len(rows)-len(block)))
	var seq []string
	for _, i := range rows {
		if p := m.visible[i].Path; !contains(block, p) {
			seq = append(seq, p)
		}
	}
	seq = append(seq[:to], append(append([]string(nil), block...), seq[to:]...)...)
	m.placeRows(rows, seq)
	for i, v := range m.visible {
		if v.Path == block[0] {
			m.cursor = i
		}
	}
}

// dropBlock writes the block where it is on screen, in its order, with one
// commit: the first after the row above it (or before the row below, at
// the top), each next one after the one before. The moved tasks are
// unmarked.
func (m *model) dropBlock() {
	g := m.grab
	rows := m.indices()
	at := m.rowOf(rows, g.block[0])
	m.grab = grabState{}
	ids := map[string]string{}
	for _, i := range rows {
		ids[m.visible[i].Path] = m.visible[i].ID
	}
	var order []string
	for _, p := range g.block {
		order = append(order, ids[p])
	}
	release := m.hold()
	defer release()
	var o task.MoveOptions
	where := ""
	if at > 0 {
		o, where = task.MoveOptions{After: m.visible[rows[at-1]].ID}, "below "+m.visible[rows[at-1]].ID
	} else {
		o, where = task.MoveOptions{Before: m.visible[rows[len(g.block)]].ID}, "above "+m.visible[rows[len(g.block)]].ID
	}
	changed := false
	for n, id := range order {
		if n > 0 {
			o = task.MoveOptions{After: order[n-1]}
		}
		r, err := m.store.Move(id, o)
		if err != nil {
			m.refresh(g.path)
			m.message = err.Error()
			return
		}
		changed = changed || r.Changed
	}
	m.message = fmt.Sprintf("moved %s %s", strings.Join(order, ", "), where)
	if changed && m.opts.Commit != nil {
		if e := m.opts.Commit("hug task move " + strings.Join(order, " ")); e != nil {
			m.message = e.Error()
		}
	}
	m.unmark(order)
	m.refresh(g.path)
}
