package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// The answer box (41hhgn check 8, Jørgen 23:50: "my answers can be
// multiple lines"): a textarea under the question that grows with what is
// typed, up to half the screen. Enter sends; Alt-Enter (Shift-Enter where
// the terminal reports it, ctrl-j everywhere) starts a new line; a paste
// keeps its lines. As every text field, ? is text in it and help is F1
// (t64v7d).

// newAnswerBox is the empty, unfocused answer box.
func newAnswerBox() textarea.Model {
	a := textarea.New()
	a.ShowLineNumbers = false
	a.Prompt = ""
	a.CharLimit = 0
	a.MaxHeight = 0 // the box grows; answerHeight caps what shows
	a.Cursor.SetMode(cursor.CursorStatic)
	// Enter sends: only these keys start a new line.
	a.KeyMap.InsertNewline = bkey.NewBinding(bkey.WithKeys("alt+enter", "shift+enter", "ctrl+j"))
	a.FocusedStyle.CursorLine = lipgloss.NewStyle()
	a.FocusedStyle.Base = lipgloss.NewStyle()
	a.BlurredStyle.Base = lipgloss.NewStyle()
	a.SetHeight(1)
	return a
}

// answerIndent is the margin of the box under the question: "  answer: ".
const answerIndent = "  "

// answerHeight is how many lines the box shows for its value: every line
// it holds, wrapped, up to half the screen.
func (m model) answerHeight() int {
	w := max(1, m.width-len(answerIndent)-len(answerPrompt)-1)
	lines := 0
	for _, l := range strings.Split(m.answerBox.Value(), "\n") {
		lines += max(1, (lipgloss.Width(l)+w)/w)
	}
	return max(1, min(lines, max(1, m.height/2)))
}

// fitAnswerBox sizes the box to the screen and to what it holds.
func (m *model) fitAnswerBox() {
	m.answerBox.SetWidth(max(1, m.width-len(answerIndent)-len(answerPrompt)-1))
	m.answerBox.SetHeight(m.answerHeight())
}

// openAnswer opens the empty box under the question.
func (m *model) openAnswer() tea.Cmd {
	m.prompt = answerPrompt
	m.answerBox.Reset()
	m.fitAnswerBox()
	return m.answerBox.Focus()
}

// answerLines are the box as drawn under the question: "answer:" before
// its first line, the rest under the first.
func (m model) answerLines() []string {
	head := m.styles.accent.Render(answerPrompt) + " "
	pad := strings.Repeat(" ", len(answerPrompt)+1)
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

// updateAnswer handles a key in the answer box: Enter sends what it holds,
// every other key edits it.
func (m model) updateAnswer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" && !msg.Paste {
		text := strings.TrimSpace(m.answerBox.Value())
		m.prompt = ""
		m.answerBox.Blur()
		m.syncDetail()
		if text == "" {
			m.message = "not answered: the answer is empty (a again, then type it)"
			return m, nil
		}
		m.answer(text)
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
	return m, cmd
}
