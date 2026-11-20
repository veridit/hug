package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// One request for the marked tasks (ec6z6j check 5, Jørgen 2026-11-20
// 08:31Z, option A of board-request-mock.md): with tasks marked, r opens
// the request box at the bottom of the list, under a heading that counts
// the set and names every id, so the marked rows stay in view while the
// reader writes. Enter sends one request for exactly that set, as
// hug task request a,b,c does (task.RequestFor): the text once on the
// first, a link on each other, one commit, one delivery. Esc writes
// nothing and keeps the marks. It is the same box r opens in a task, with
// the same keys.

// requestMarked opens the box for the marked ids, fixed now: what is sent
// is what was marked when r was pressed.
func (m *model) requestMarked(ids []string) tea.Cmd {
	m.requestIDs = append([]string(nil), ids...)
	m.confirmPath = ""
	return m.openBox(requestPrompt)
}

// markedBox says whether the open box is the one for the marked tasks,
// drawn under the list rather than in a task.
func (m model) markedBox() bool {
	return m.prompt == requestPrompt && len(m.requestIDs) > 0 && !m.detail && !m.help
}

// markedBoxLines are the lines the box takes at the bottom of the list,
// over the key bar: a blank line, the heading naming the set, the box.
// More ids than fit end in "…"; the count is always whole.
func (m model) markedBoxLines() []string {
	head := m.styles.accent.Render(fmt.Sprintf("▸ REQUEST FOR %d MARKED", len(m.requestIDs)))
	ids := m.styles.dim.Render(" · " + strings.Join(m.requestIDs, " "))
	return append([]string{"", ansi.Truncate(head+ids, m.width, "…")}, m.answerLines()...)
}

// markedBoxExtra is how many rows the box takes from the list beyond the
// message and input lines: the blank and heading over the box, its
// lines, and the message line under it, in place of two.
func (m model) markedBoxExtra() int {
	if !m.markedBox() {
		return 0
	}
	return 2 + m.answerHeight() + 1 - 2 // the box is answerHeight lines
}

// sendMarkedRequest writes text as one request for the marked ids and
// commits it once, then delivers it once (gmzd87, through the home). The
// marks go when it is written; a refusal keeps them and says why.
func (m *model) sendMarkedRequest(text string) {
	ids := m.requestIDs
	m.requestIDs = nil
	who := m.me()
	if who == "" {
		who = "human"
	}
	path := ""
	if t, ok := m.selected(); ok {
		path = t.Path
	}
	release := m.hold()
	r, err := m.store.RequestFor(ids, text, who)
	if err == nil && r.Changed && m.opts.Commit != nil {
		err = m.opts.Commit("hug task request " + strings.Join(ids, ","))
	}
	release()
	if err != nil {
		m.refresh(path)
		m.message = err.Error()
		return
	}
	m.message = "requested " + r.Task.ID
	if len(ids) > 1 {
		n := 0
		for _, q := range r.Task.Requests {
			n = max(n, q.N)
		}
		m.message = fmt.Sprintf("request %d for %d tasks, on %s", n, len(ids), r.Task.ID)
	}
	m.unmark(ids)
	m.refresh(path)
	m.deliver(r, nil, "request", who)
}
