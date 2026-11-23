package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// The answer box (41hhgn check 8, Jørgen 23:50: "my answers can be
// multiple lines"): a textarea under the question that grows with what is
// typed, up to half the screen. Enter sends; Alt-Enter (Shift-Enter where
// the terminal reports it, ctrl-j everywhere) starts a new line; a paste
// keeps its lines. As every text field, ? is text in it and help is F1
// (t64v7d).

// newAnswerBox is the empty, unfocused answer box. Its cursor blinks
// (qpp30y); styleAnswerCursor gives it the theme's look.
func newAnswerBox() textarea.Model {
	a := textarea.New()
	a.ShowLineNumbers = false
	a.Prompt = ""
	a.CharLimit = 0
	a.MaxHeight = 0 // the box grows; answerHeight caps what shows
	a.Cursor.SetMode(cursor.CursorBlink)
	// Enter sends: only these keys start a new line.
	a.KeyMap.InsertNewline = bkey.NewBinding(bkey.WithKeys("alt+enter", "shift+enter", "ctrl+j"))
	a.FocusedStyle.CursorLine = lipgloss.NewStyle()
	a.FocusedStyle.Base = lipgloss.NewStyle()
	a.BlurredStyle.Base = lipgloss.NewStyle()
	a.SetHeight(1)
	return a
}

// styleAnswerCursor draws the box's cursor in theme.cursor, an
// underline, not bubbles' reversed block (qpp30y). bubbles reverses the
// cursor's cell itself, so the style is a transform that draws the cell
// whole: its own codes then follow the reverse and a reset, leaving only
// the underlined accent. Blinked off, the cell is plain text.
func styleAnswerCursor(a *textarea.Model, cur lipgloss.Style) {
	a.Cursor.Style = lipgloss.NewStyle().Transform(func(cell string) string {
		return "\x1b[0m" + cur.Render(ansi.Strip(cell))
	})
}

// answerIndent is the margin of the box under the question: "  answer: ".
const answerIndent = "  "

// requestPrompt is the box r opens in a task or on its row (hbgybn, crab
// 01:30): the reader asks the agents for a change, in their own words.
const requestPrompt = "request:"

// boxPrompt says whether the open prompt is the multi-line box: the
// answer or a request.
func (m model) boxPrompt() bool { return m.prompt == answerPrompt || m.prompt == requestPrompt }

// boxLabel is the width the box's label takes, the same for both.
const boxLabel = len(requestPrompt) + 1

// answerHeight is how many lines the box shows for its value: every line
// it holds, wrapped, up to half the screen.
func (m model) answerHeight() int {
	w := max(1, m.width-len(answerIndent)-boxLabel)
	lines := 0
	for _, l := range strings.Split(m.answerBox.Value(), "\n") {
		lines += max(1, (lipgloss.Width(l)+w)/w)
	}
	return max(1, min(lines, max(1, m.height/2)))
}

// fitAnswerBox sizes the box to the screen and to what it holds.
func (m *model) fitAnswerBox() {
	m.answerBox.SetWidth(max(1, m.width-len(answerIndent)-boxLabel))
	m.answerBox.SetHeight(m.answerHeight())
}

// openAnswer opens the empty box under the question.
func (m *model) openAnswer() tea.Cmd { return m.openBox(answerPrompt) }

// openBox opens the empty multi-line box for prompt, drawn at once
// (qpp30y: it showed only after the first key), its cursor blinking.
func (m *model) openBox(prompt string) tea.Cmd {
	m.prompt = prompt
	m.answerBox.Reset()
	m.fitAnswerBox()
	cmd := m.answerBox.Focus()
	m.syncDetail()
	return cmd
}

// blinkBox passes a cursor blink to the open box and draws the page
// again: the cursor is the only thing on screen that moves (qpp30y). A
// closed box takes no blink and asks for none.
func (m *model) blinkBox(msg tea.Msg) tea.Cmd {
	if !m.boxPrompt() || !m.answerBox.Focused() {
		return nil
	}
	var cmd tea.Cmd
	m.answerBox, cmd = m.answerBox.Update(msg)
	m.syncDetail()
	return cmd
}

// answerLines are the box as drawn under the question: "answer:" before
// its first line, the rest under the first.
func (m model) answerLines() []string {
	// Each label as it reads ("answer: ", "request: "); the box is as
	// wide as the longer one leaves, so both wrap alike.
	head := m.styles.accent.Render(m.prompt) + " "
	pad := strings.Repeat(" ", len(m.prompt)+1)
	var out []string
	for i, l := range strings.Split(m.answerBox.View(), "\n") {
		if i == 0 {
			out = append(out, answerIndent+head+l)
		} else {
			out = append(out, answerIndent+pad+l)
		}
	}
	return out
}

// updateAnswer handles a key in the box: Enter sends what it holds, as
// the answer or the request, every other key edits it.
func (m model) updateAnswer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" && !msg.Paste {
		text := strings.TrimSpace(m.answerBox.Value())
		prompt := m.prompt
		m.prompt = ""
		m.answerBox.Blur()
		m.syncDetail()
		switch {
		case text == "" && prompt == requestPrompt:
			m.requestIDs = nil // the marks stay (ec6z6j check 5)
			m.message = "no request sent: it is empty (r again, then write what you want changed)"
		case text == "":
			m.message = "not answered: the answer is empty (a again, then type it)"
		case prompt == requestPrompt && len(m.requestIDs) > 0:
			m.sendMarkedRequest(text) // one request for the marked (ec6z6j check 5)
		case prompt == requestPrompt:
			m.request(text)
		default:
			m.answer(text)
		}
		return m, nil
	}
	var cmd tea.Cmd
	if msg.Paste {
		// A terminal pastes CRLF or CR line ends: one line break each,
		// not two (the textarea makes a line of every \r and \n).
		text := strings.ReplaceAll(string(msg.Runes), "\r\n", "\n")
		msg.Runes = []rune(strings.ReplaceAll(text, "\r", "\n"))
	}
	// Room first, so the textarea does not scroll its own view for the
	// line it is about to add; then the height its value needs.
	m.answerBox.SetHeight(max(1, m.height/2))
	m.answerBox, cmd = m.answerBox.Update(msg)
	m.fitAnswerBox()
	m.syncDetail() // the box sits in the page: its height moves what follows
	if m.prompt == requestPrompt && m.detail {
		m.revealRequestBox()
	}
	if m.prompt == answerPrompt {
		m.revealAnswerBox() // a growing box stays in view
	}
	return m, cmd
}

// requestKey opens the request box for the selected task, in its detail
// or on its row (hbgybn check 4, crab 01:30: r = request).
func (m *model) requestKey() tea.Cmd {
	t, ok := m.selected()
	if !ok {
		m.message = "no task selected: j/k to one, then r writes a request on it"
		return nil
	}
	m.confirmPath = t.Path
	if !m.detail {
		// On a row the box needs the task's page to sit in: open it.
		m.detailPath = t.Path
		m.detail = true
		m.startFocus()
	}
	cmd := m.openBox(requestPrompt)
	m.syncDetail()
	m.revealRequestBox()
	return cmd
}

// request stores text as the reader's request on the task r opened it on.
func (m *model) request(text string) {
	t, ok := m.taskAt(m.confirmPath)
	m.confirmPath = ""
	if !ok {
		m.message = "task disappeared; refresh and try again"
		return
	}
	who := m.me()
	if who == "" {
		who = "human"
	}
	defer m.hold()()
	r, err := m.store.Request(t.ID, text, who)
	m.finish(r, err, "request")
	m.deliver(r, err, "request", who)
}

// requestCount is the "req N" row mark while N requests are open, with a
// trailing space, or "".
func (m model) requestCount(t task.Task) string {
	if n := len(task.OpenRequests(t)); n > 0 {
		return fmt.Sprintf("req %d ", n)
	}
	return ""
}

// requestLines are the REQUESTS section: each request with who and when,
// its text verbatim, then its account once resolved, or "open"; with the
// box open, the box last (hbgybn check 4). Each block is wrapped here and
// hangs under its indent (cd63d1): the text and the targets under 3, the
// account under 5. The id markAt names is marked before the wrap, so its
// column is counted: request i's first in its head (a link's home, ec6z6j
// check 4), text and targets, then its first in the account.
func (m model) requestLines(t task.Task, markAt func(string, int) string) []string {
	var out []string
	for i, r := range t.Requests {
		r = m.shownRequest(r) // a link draws its home's text (ec6z6j check 4)
		w := markAt(stopRequest, i)
		marked := false
		mark := func(s string) string {
			if p := markPlain(s, w, true); !marked && p != s {
				marked = true
				return p
			}
			return s
		}
		head := fmt.Sprintf("%d. %s", r.N, oneLine(r.By))
		if s := strings.TrimPrefix(since(r.At), " · "); s != "" {
			head += " · " + s + " ago"
		}
		// One label: the request's receipt (gmzd87), "pending, delivered
		// to crab", "acknowledged by crab", ...
		state := m.styles.accent.Render(oneLine(r.Receipt.Label()))
		if r.Resolved != nil {
			state = m.styles.status["done"].Render("resolved")
		}
		line := m.styles.dim.Render(head+" · ") + state
		if home := requestHome(r); home != "" {
			line += m.styleMarked(" · "+mark(home), w, m.styles.dim) // where it lives (ec6z6j check 4)
		}
		out = append(out, line)
		text := mark(safe(strings.TrimRight(r.Text, "\n")))
		out = append(out, m.hangStyled("   ", text, 3, w, m.styles.normal)...)
		if tg := requestTargets(r); tg != "" {
			out = append(out, m.hangStyled("   ", mark(safe(tg)), 3, w, m.styles.dim)...) // each target, pending or resolved
		}
		if a := r.Resolved; a != nil {
			w := markAt(stopResolved, i)
			account := markPlain(safe(strings.TrimRight(a.Text, "\n")), w, true)
			out = append(out, m.hangStyled("   └ "+oneLine(a.By)+": ", account, 5, w, m.styles.status["done"])...)
		}
	}
	if m.prompt == requestPrompt {
		out = append(out, m.answerLines()...)
	}
	return out
}

// hangStyled is hang at the page's width, each line in style and the
// ▸w markPlain put there in the selection tint.
func (m model) hangStyled(prefix, body string, indent int, w string, style lipgloss.Style) []string {
	lines := hang(prefix, body, indent, m.width)
	for k, l := range lines {
		lines[k] = m.styleMarked(l, w, style)
	}
	return lines
}

// revealAnswerBox scrolls the page so the question the open answer box
// answers shows with the box under it, the box's last line (where the
// cursor is) always (qcq3tm check 4).
func (m *model) revealAnswerBox() {
	m.showMarked()
	_, e, ok := m.markedBlock()
	if !ok {
		return
	}
	if e >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(max(0, e-m.viewport.Height+1))
	}
}

// revealRequestBox scrolls the page so the open request box shows.
func (m *model) revealRequestBox() {
	t, ok := m.selected()
	if !ok {
		return
	}
	for i, line := range m.pageLines(t) {
		if strings.HasPrefix(strings.TrimSpace(line), requestPrompt) {
			if i < m.viewport.YOffset || i >= m.viewport.YOffset+m.viewport.Height {
				m.viewport.SetYOffset(max(0, i-m.viewport.Height/2))
			}
			return
		}
	}
}
