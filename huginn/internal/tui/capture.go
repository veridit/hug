package tui

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// Where the screen S took goes (hjbfzd, Jørgen 2026-11-18: "attach it to
// ... ticket so-and-so or S for select or C for create"). S copies the
// screen at once, then the input line asks:
//
//	Attach this screen to: 000002 → Title · select · create
//
// The field starts with the task in view, so Enter alone attaches there
// as before; typing replaces it with an id (a prefix completes, Tab fills
// it in). While the field holds the offer or nothing, s picks the task
// from the rows and c creates one; once typing, they are text. Esc keeps
// only the copy: nothing is written.

const (
	capturePrompt      = "Attach this screen to:"
	captureTitlePrompt = "title:"
)

// pickReturn is the screen S was pressed on: s comes back to it.
type pickReturn struct {
	view, detailPath, id string
	detail               bool
	pane                 int
}

// openCapture asks where the capture goes, offering offer ("" for none).
// The prompt names what goes: this screen (S) or the clipboard (V,
// q7n2zd).
func (m *model) openCapture(offer string) tea.Cmd {
	m.captureTyped = false
	m.message = m.capturedNote // what went to the clipboard, above the question
	cmd := m.openInput(capturePrompt, offer)
	m.input.Prompt = "Attach " + m.captureWhat() + " to: "
	return cmd
}

// captureWhat is what is being attached: "this screen" (S) or "the
// clipboard" (V).
func (m model) captureWhat() string {
	if m.captureKind == clipKind {
		return "the clipboard"
	}
	return "this screen"
}

// captureNoun is the short form for the footers: "the screen" or "the
// clipboard".
func (m model) captureNoun() string {
	if m.captureKind == clipKind {
		return "the clipboard"
	}
	return "the screen"
}

// captureWhy is the attach note's reason, after hug task attach's name,
// size and sha256.
func (m model) captureWhy() string {
	if m.captureKind == clipKind {
		return "pasted from the clipboard in hug tui"
	}
	return "screen capture from hug tui"
}

// forgetCapture clears what S or V took.
func (m *model) forgetCapture() {
	m.captured, m.capturedNote, m.captureTyped = "", "", false
	m.captureKind, m.captureExt = "", ""
}

// captureUntouched says whether s and c are keys: the field holds the
// offered id, as opened, or nothing.
func (m model) captureUntouched() bool {
	return !m.captureTyped || m.input.Value() == ""
}

// captureName is the file name the capture takes in t: screen-HHMM.ans,
// or paste-HHMM with V's extension (q7n2zd), numbered from 2 when t has
// one already.
func (m model) captureName(t task.Task) string {
	prefix, ext := "screen-", ".ans"
	if m.captureKind == clipKind {
		prefix, ext = "paste-", m.captureExt
	}
	base := prefix + m.captureStamp
	name := base + ext
	for i := 2; m.hasFile(t, name); i++ {
		name = fmt.Sprintf("%s-%d%s", base, i, ext)
	}
	return name
}

// dropCapture forgets the capture: only the copy stays.
func (m *model) dropCapture() {
	m.message = m.capturedNote + " · not attached"
	m.forgetCapture()
}

// captureMatches are the tasks whose id or alias starts with ref, by id.
func (m model) captureMatches(ref string) []task.Task {
	var out []task.Task
	for _, t := range m.jumpChoices() {
		if ref != "" {
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b task.Task) int { return strings.Compare(a.ID, b.ID) })
	return out
}

// updateCapture is a key in the capture prompt or its title prompt.
func (m model) updateCapture(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.prompt == captureTitlePrompt {
		if k != "enter" {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		title := strings.TrimSpace(m.input.Value())
		if title == "" {
			m.message = "a new task needs a title: type it, or Esc goes back"
			return m, nil
		}
		m.prompt = ""
		m.input.Blur()
		m.createForCapture(title)
		return m, nil
	}
	value := strings.TrimSpace(m.input.Value())
	switch {
	case k == "s" && m.captureUntouched():
		m.startPick()
		return m, nil
	case k == "c" && m.captureUntouched():
		cmd := m.openInput(captureTitlePrompt, "")
		m.input.Prompt = "new task with " + m.captureWhat() + ", title: "
		return m, cmd
	case k == "tab":
		if cs := m.captureMatches(value); len(cs) > 0 {
			m.input.SetValue(cs[0].ID)
			m.input.CursorEnd()
			m.captureTyped = true
		}
		return m, nil
	case k == "enter":
		if value == "" {
			m.message = "type a task id, s selects one, c creates one"
			return m, nil
		}
		t, err := task.Resolve(m.all, value)
		if err != nil {
			m.message = m.captureMiss(value, err)
			return m, nil
		}
		m.prompt = ""
		m.input.Blur()
		m.attachCapture(t, m.captureName(t))
		return m, nil
	}
	if !m.captureTyped && (msg.Type == tea.KeyRunes || k == "backspace") {
		// The first key replaces the offered id; Backspace clears it.
		m.captureTyped = true
		m.input.SetValue("")
		if k == "backspace" {
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if msg.Type == tea.KeyRunes {
		m.captureTyped = true
	}
	return m, cmd
}

// captureMiss says why ref names no one task.
func (m model) captureMiss(ref string, err error) string {
	var e *task.Error
	if errors.As(err, &e) && e.Kind == "ambiguous" {
		ids := []string{}
		for i, x := range e.Matches {
			if i == 3 {
				ids = append(ids, "…")
				break
			}
			ids = append(ids, x.ID)
		}
		return fmt.Sprintf("%s matches %d tasks (%s): type more, or Tab takes the first", oneLine(ref), len(e.Matches), strings.Join(ids, ", "))
	}
	return "no task " + oneLine(ref) + ": type a task id, s selects one, c creates one"
}

// captureLine is the capture prompt's input line: the field, what it
// names, then the keys s and c while they are keys. Only the title is
// cut to fit; the keys stay.
func (m model) captureLine() string {
	// The field as wide as its text: bubbles pads it to Width, which would
	// push the title and the keys off the line. Its cursor is the theme's
	// underline, still (qpp30y: only the box cursor moves), never a
	// reversed block: no background fills.
	in := m.input
	in.Width = 0
	cur := m.styles.cursor
	in.Cursor.Style = lipgloss.NewStyle().Transform(func(cell string) string {
		return "\x1b[0m" + cur.Render(ansi.Strip(cell))
	})
	field := in.View()
	if m.prompt == captureTitlePrompt {
		return field
	}
	keys := ""
	if m.captureUntouched() {
		keys = " · " + mk("s") + "elect · " + mk("c") + "reate"
	}
	value := strings.TrimSpace(m.input.Value())
	named := ""
	if value != "" {
		if t, err := task.Resolve(m.all, value); err == nil {
			named = "→ " + oneLine(t.Title)
		} else if cs := m.captureMatches(value); len(cs) > 1 {
			named = fmt.Sprintf("→ %d tasks match", len(cs))
		} else {
			named = "→ no such task"
		}
	}
	room := m.width - ansi.StringWidth(ansi.Strip(field)) - ansi.StringWidth(plainMarks(keys)) - 1
	out := field
	if named != "" && room >= 8 {
		if !strings.HasSuffix(ansi.Strip(field), " ") {
			out += " " // the cursor cell at the end is the space before it
		}
		out += m.styles.dim.Render(ansi.Truncate(named, room, "…"))
	}
	if keys == "" {
		return out
	}
	// The key letters underlined in the accent, not bold; one cue per
	// meaning (qh9ep1 check 9). The view line's v is bold since r80svs;
	// this prompt keeps its look.
	dim, key := m.styles.dim, m.styles.viewPrefixKey.UnsetBold().Underline(true)
	return out + dim.Render(" · ") + key.Render("s") + dim.Render("elect · ") + key.Render("c") + dim.Render("reate")
}

// startPick leaves for the rows: the reader picks the task with the list's
// own keys, and Enter attaches the capture to it (hjbfzd check 2).
func (m *model) startPick() {
	from := pickReturn{view: m.view, detail: m.detail, detailPath: m.detailPath, pane: m.pane}
	if t, ok := m.selected(); ok {
		from.id = t.ID
	}
	m.prompt = ""
	m.input.Blur()
	m.message = ""
	m.pickFrom, m.picking = from, true
	m.setView("list")
	if from.id != "" {
		m.follow(from.id)
	}
}

// pickFooter says what Enter does while picking.
func (m model) pickFooter() string {
	return "pick the task for " + m.captureNoun() + " · Enter attaches it there · Esc not attached"
}

// pickWriters are the keys that write or start a write: none runs while
// a task is picked for the capture.
var pickWriters = []string{"t", "d", "x", "e", "m", "y", "P", "p", "r", "N", "S", clipKey, "a", "w", "-", " "}

// pickKey is a key while picking: Enter attaches, writers say what Enter
// does. Every other key moves, finds or filters as in the rows.
func (m *model) pickKey(k string) bool {
	switch {
	case k == "enter":
		t, ok := m.selected()
		if !ok || m.head != "" {
			m.message = "a heading is not a task: j/k select a task, Enter attaches " + m.captureNoun() + " to it"
			return true
		}
		name := m.captureName(t)
		m.endPick()
		m.attachCapture(t, name)
		return true
	case slices.Contains(pickWriters, k):
		m.message = "picking the task for " + m.captureNoun() + ": Enter attaches it there · Esc not attached"
		return true
	}
	return false
}

// endPick goes back to the screen S was pressed on.
func (m *model) endPick() {
	from := m.pickFrom
	m.picking, m.pickFrom = false, pickReturn{}
	m.setView(from.view)
	m.pane = from.pane
	m.clamp()
	if from.id != "" {
		m.follow(from.id)
	}
	if from.detail {
		if _, ok := m.taskAt(from.detailPath); ok {
			m.detailPath, m.detail = from.detailPath, true
			m.startFocus()
			m.syncDetail()
			m.openTop()
		}
	}
}

// createForCapture makes a task titled title, as N does, with the capture
// attached, in one commit, and opens it (hjbfzd check 3).
func (m *model) createForCapture(title string) {
	body, note, why := []byte(m.captured), m.capturedNote, m.captureWhy()
	name := m.captureName(task.Task{})
	m.forgetCapture()
	release := m.hold()
	r, err := m.store.New(task.NewOptions{Title: title, Goal: title, Acceptance: []string{"Verify " + title}})
	if err != nil {
		release()
		m.message = note + " · not attached: " + err.Error()
		return
	}
	said := "created " + r.Task.ID + " · " + note + " · attached as " + name
	if _, _, e := m.store.Attach(r.Task.ID, name, body, why, false); e != nil {
		said = "created " + r.Task.ID + " · " + note + " · not attached: " + e.Error()
	}
	if m.opts.Commit != nil {
		if e := m.opts.Commit("hug task new " + r.Task.ID); e != nil {
			said += " · " + e.Error()
		}
	}
	release()
	m.refresh(r.Task.Path)
	if t, ok := m.taskAt(r.Task.Path); ok {
		m.openTask(t)
	}
	m.message = said
}
