// Package tui is an owner-operated terminal browser for muninn.
package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/muninn/internal/app"
	"github.com/veridit/muninn/internal/recall"
	"github.com/veridit/muninn/internal/search"
	"github.com/veridit/muninn/internal/store"
)

type screen int

const (
	overview screen = iota
	listing
	detail
	review
	preview
)

type inputMode int

const (
	noInput inputMode = iota
	findInput
	paletteInput
	correctionInput
	moveInput
	confirmInput
	jumpInput
	filterInput
)

type location struct {
	screen       screen
	title        string
	results      []search.Result
	cursor, pane int
	filter       string
	unit         store.Unit
	offset       int
}
type model struct {
	a                *app.App
	binding          app.Binding
	input            textinput.Model
	editor           textarea.Model
	viewport         viewport.Model
	screen           screen
	focus            inputMode
	title            string
	results          []search.Result
	cursor, pane     int
	selections       [4]int
	snapshot         snapshot
	history          []location
	unit             store.Unit
	metadata         detailMetadata
	related          []search.Result
	filter           string
	chips            []chip
	filterField      string
	jumpIDs          []string
	searchSeq        int
	completions      []string
	completion       int
	pending          action
	busy             bool
	quitAfter        bool
	help, explain, g bool
	status, block    string
	retainStatus     bool
	width, height    int
	reviewKind       string
	reviewIndex      int
	reviewOffset     int
	reviewDone       map[string]bool
}
type resultsMsg struct {
	res     []search.Result
	block   string
	elapsed time.Duration
	err     error
	seq     int
	query   string
}
type actionMsg struct {
	note string
	err  error
	next bool
	unit store.Unit
}

// Run starts the terminal browser. No writes occur until an owner confirms an action.
func Run(a *app.App, bindings ...app.Binding) error {
	m := newModel(a, bindings...)
	if m.binding.Err != nil {
		return m.binding.Err
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
func newModel(a *app.App, bindings ...app.Binding) model {
	cwd, _ := os.Getwd()
	b := a.Bound("", cwd)
	if len(bindings) > 0 {
		b = bindings[0]
	}
	in := textinput.New()
	in.Prompt = "/ "
	in.Placeholder = "find memories"
	in.CharLimit = 2000
	ed := textarea.New()
	ed.Placeholder = "Write the corrected memory. The original text and date are preserved."
	ed.CharLimit = 16000
	ed.ShowLineNumbers = false
	return model{a: a, binding: b, input: in, editor: ed, viewport: viewport.New(76, 14), width: 80, height: 24, title: "Overview", status: "Loading visible memories…", reviewDone: map[string]bool{}}
}
func (m model) Init() tea.Cmd { return m.load() }
func (m model) load() tea.Cmd {
	a, b := m.a, m.binding
	return func() tea.Msg { return loadSnapshot(a, b) }
}
func (m model) run() tea.Cmd {
	q, seq, a, b := m.input.Value(), m.searchSeq, m.a, m.binding
	return func() tea.Msg {
		b = b.Refresh(a)
		if b.Err != nil {
			return resultsMsg{err: b.Err, seq: seq, query: q}
		}
		if strings.TrimSpace(q) == "" {
			return resultsMsg{seq: seq, query: q}
		}
		t := time.Now()
		o := a.RecallOptions()
		o.MinScore = 0
		o.Limit = 100
		o.Scopes = b.Scopes
		o.ScopeFor = b.ScopeFor
		res, err := a.Searcher.Search(context.Background(), q, o)
		return resultsMsg{res: res, err: err, seq: seq, query: q, elapsed: time.Since(t)}
	}
}
func (m *model) push() {
	m.history = append(m.history, location{m.screen, m.title, m.results, m.cursor, m.pane, m.filter, m.unit, m.viewport.YOffset})
}
func (m *model) back() {
	if m.filter != "" {
		m.filter = ""
		m.input.SetValue("")
		m.searchSeq++
		m.results = m.visibleResults()
		m.title = "Recent"
		m.cursor = 0
		return
	}
	if m.screen == listing && len(m.chips) > 0 {
		m.chips = m.chips[:len(m.chips)-1]
		m.cursor = 0
		return
	}
	if n := len(m.history); n > 0 {
		p := m.history[n-1]
		m.history = m.history[:n-1]
		m.screen, m.title, m.results, m.cursor, m.pane, m.filter, m.unit = p.screen, p.title, p.results, p.cursor, p.pane, p.filter, p.unit
		if m.screen == detail {
			m.setDetail()
			m.viewport.SetYOffset(p.offset)
		}
		return
	}
	m.screen = overview
	m.title = "Overview"
	m.pane = 0
}
func (m *model) showList(title string, rows []search.Result) {
	m.push()
	m.screen = listing
	m.title = title
	m.results = rows
	m.cursor = 0
	m.pane = 0
	m.filter = ""
}
func (m *model) openUnit(ref string) {
	b := m.binding.Refresh(m.a)
	u, err := b.Resolve(m.a, ref)
	if err != nil {
		m.status = err.Error()
		return
	}
	m.push()
	m.binding = b
	m.unit = u
	m.screen = detail
	m.title = "Memory " + u.SID
	m.pane = 0
	m.cursor = 0
	m.filter = ""
	m.setDetail()
}
func (m *model) setDetail() {
	m.sizeViewport()
	m.metadata = m.loadDetailMetadata()
	m.related = nil
	res, err := m.a.Searcher.RelatedIn(context.Background(), m.unit.ID, 30, m.binding.Scopes, m.binding.ScopeFor)
	if err == nil {
		for _, r := range res {
			if u, ok := m.binding.Present(r.Unit); ok {
				r.Unit = u
				m.related = append(m.related, r)
			}
		}
	}
	m.viewport.SetContent(m.detailText())
	m.viewport.GotoTop()
}
func (m *model) sizeViewport() {
	h := max(3, m.height-4)
	related := max(4, min(8, h/3))
	m.viewport.Width = max(10, m.width-4)
	m.viewport.Height = max(1, h-related-5)
	if m.screen == preview {
		m.viewport.Height = max(1, h-2)
	}
}
func (m model) visibleResults() []search.Result {
	var rows []search.Result
	for _, u := range m.snapshot.units {
		rows = append(rows, search.Result{Unit: u})
	}
	return m.filtered(rows)
}
func (m model) selected() (store.Unit, bool) {
	if m.screen == detail {
		return m.unit, m.unit.SID != ""
	}
	rows := m.filtered(m.results)
	c := m.cursor
	if m.screen == overview {
		c = m.selections[m.pane]
		if m.pane == 1 {
			rows = m.visibleResults()
		} else if m.pane == 2 {
			if c >= 0 && c < len(m.snapshot.told) {
				return m.snapshot.told[c].unit, true
			}
			return store.Unit{}, false
		} else {
			return store.Unit{}, false
		}
	}
	if c >= 0 && c < len(rows) {
		return rows[c].Unit, true
	}
	return store.Unit{}, false
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(20, msg.Width)
		m.height = max(8, msg.Height)
		m.input.Width = max(1, m.width-6)
		m.editor.SetWidth(max(10, m.width-6))
		m.editor.SetHeight(max(3, m.height-10))
		m.sizeViewport()
		if m.screen == detail {
			offset := m.viewport.YOffset
			m.viewport.SetContent(m.detailText())
			m.viewport.SetYOffset(offset)
		} else if m.screen == preview {
			offset := m.viewport.YOffset
			m.viewport.SetContent(wrapText(m.block, m.width-4))
			m.viewport.SetYOffset(offset)
		}
		return m, nil
	case snapshot:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			m.snapshot = snapshot{}
			m.results = nil
			m.related = nil
			m.unit = store.Unit{}
			return m, nil
		}
		m.snapshot = msg
		m.binding = msg.binding
		if !m.busy && !m.retainStatus {
			m.status = fmt.Sprintf("%d visible memories", len(msg.units))
		}
		m.retainStatus = false
		return m, nil
	case resultsMsg:
		if msg.seq != m.searchSeq || msg.query != m.filter {
			return m, nil
		}
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			m.results = nil
			return m, nil
		}
		m.results = msg.res
		m.cursor = 0
		m.status = fmt.Sprintf("%d results in %s", len(msg.res), msg.elapsed.Round(time.Millisecond))
		return m, nil
	case actionMsg:
		m.busy = false
		m.focus = noInput
		if m.quitAfter {
			return m, tea.Quit
		}
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			return m, nil
		}
		m.status = msg.note
		m.retainStatus = true
		if msg.next {
			m.reviewDone[m.reviewKey()] = true
			m.reviewIndex++
			m.reviewOffset = 0
			m.clampReview()
		} else if msg.unit.SID != "" {
			m.unit = msg.unit
			m.screen = detail
			m.setDetail()
		} else {
			m.screen = overview
			m.history = nil
			m.filter = ""
		}
		return m, m.load()
	case tea.KeyMsg:
		// Bubble Tea groups adjacent printable characters from one PTY read.
		// Interpret command prefixes before passing the remaining text as one
		// input update, so /query, :command, fc and gg also work when typed fast.
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 && !msg.Alt && (m.focus == noInput || m.focus == confirmInput || m.focus == moveInput || (m.focus == filterInput && m.filterField == "") || strings.ContainsRune(string(msg.Runes), '?')) {
			first := msg
			first.Runes = msg.Runes[:1]
			next, cmd := m.Update(first)
			rest := msg
			rest.Runes = msg.Runes[1:]
			out, last := next.Update(rest)
			return out, tea.Batch(cmd, last)
		}
		k := msg.String()
		if k == "ctrl+c" {
			if m.busy {
				m.quitAfter = true
				m.status = "Finishing the confirmed write, then quitting…"
				return m, nil
			}
			return m, tea.Quit
		}
		if m.help {
			if k == "esc" || k == "?" {
				m.help = false
			}
			return m, nil
		}
		if k == "?" {
			m.help = true
			return m, nil
		}
		if m.busy {
			return m, nil
		}
		if k == "esc" {
			if m.focus == filterInput && m.filterField != "" {
				m.filterField = ""
				m.input.SetValue("")
				m.input.Blur()
				return m, nil
			}
			if m.focus != noInput {
				if m.focus == findInput {
					m.filter = ""
					m.searchSeq++
					m.backFind()
				}
				m.focus = noInput
				m.input.Blur()
				m.editor.Blur()
				return m, nil
			}
			m.back()
			return m, nil
		}
		if m.focus != noInput {
			return m.updateInput(msg)
		}
		switch k {
		case "q":
			return m, tea.Quit
		case "/":
			if m.screen != listing || m.title != "Find" {
				m.push()
			}
			m.screen = listing
			m.title = "Find"
			m.focus = findInput
			m.filter = ""
			m.input.SetValue("")
			m.input.Prompt = "/ "
			m.input.Placeholder = "find memories"
			m.results = m.visibleResults()
			m.cursor = 0
			return m, m.input.Focus()
		case "#":
			m.focus = jumpInput
			m.input.Prompt = "# "
			m.input.Placeholder = "id or prefix"
			m.input.SetValue("")
			m.jumpChoices()
			return m, m.input.Focus()
		case "f":
			m.focus = filterInput
			m.filterField = ""
			m.input.Prompt = "filter › "
			m.input.Placeholder = "type to narrow values"
			m.input.SetValue("")
			m.completion = 0
			return m, nil
		case "F":
			m.clearFilters()
			return m, nil
		case "1", "2", "3", "4":
			m.shortcutPlace(k)
			return m, nil
		case "backspace", "ctrl+h":
			if len(m.chips) > 0 {
				m.chips = m.chips[:len(m.chips)-1]
				m.cursor = 0
			}
			return m, nil
		case ":":
			m.focus = paletteInput
			m.input.Prompt = ": "
			m.input.Placeholder = "command"
			m.input.SetValue("")
			m.complete()
			return m, m.input.Focus()
		case "ctrl+e":
			m.explain = !m.explain
			if m.screen == detail {
				m.viewport.SetContent(m.detailText())
			}
			return m, nil
		case "ctrl+r":
			if u, ok := m.selected(); ok {
				u, err := m.binding.Resolve(m.a, u.SID)
				if err != nil {
					m.status = err.Error()
					return m, nil
				}
				rows, err := m.a.Searcher.RelatedIn(context.Background(), u.ID, 30, m.binding.Scopes, m.binding.ScopeFor)
				if err != nil {
					m.status = err.Error()
				} else {
					m.showList("Related to "+u.SID, rows)
				}
			}
			return m, nil
		case "e", "m", "x":
			return m.beginAction(k)
		case "enter":
			m.openSelected()
			return m, nil
		case "h", "left", "shift+tab":
			m.changePane(-1)
			return m, nil
		case "l", "right", "tab":
			m.changePane(1)
			return m, nil
		case "y", "n":
			if m.screen == review {
				if k == "y" {
					return m, m.applyReview()
				}
				m.reviewDone[m.reviewKey()] = true
				m.reviewIndex++
				m.reviewOffset = 0
				m.clampReview()
			}
			return m, nil
		case "g":
			if m.g {
				m.moveTo(0)
				m.g = false
			} else {
				m.g = true
			}
			return m, nil
		case "G", ">", "end":
			m.moveTo(1 << 30)
			m.g = false
			return m, nil
		case "<", "home":
			m.moveTo(0)
			m.g = false
			return m, nil
		case "j", "down":
			m.move(1)
		case "k", "up":
			m.move(-1)
		case "ctrl+d":
			m.move(max(1, (m.height-8)/2))
		case "ctrl+u":
			m.move(-max(1, (m.height-8)/2))
		case "pgdown":
			m.move(max(1, m.height-8))
		case "pgup":
			m.move(-max(1, m.height-8))
		}
		if k != "g" {
			m.g = false
		}
		return m, nil
	}
	if m.focus == findInput || m.focus == paletteInput || m.focus == jumpInput || m.focus == filterInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	if m.focus == correctionInput {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}
	return m, nil
}
func (m *model) backFind() {
	m.filter = ""
	if n := len(m.history); n > 0 {
		m.back()
	} else {
		m.screen = overview
		m.title = "Overview"
	}
}
func (m *model) changePane(delta int) {
	n := 1
	if m.screen == overview {
		n = 4
	}
	if m.screen == detail {
		n = 2
	}
	m.pane = (m.pane + delta + n) % n
	m.g = false
}
func (m *model) moveTo(n int) {
	if m.screen == detail && m.pane == 0 {
		if n == 0 {
			m.viewport.GotoTop()
		} else {
			m.viewport.GotoBottom()
		}
		return
	}
	count := len(m.filtered(m.results))
	c := &m.cursor
	if m.screen == overview {
		c = &m.selections[m.pane]
		count = []int{len(m.snapshot.places), len(m.visibleResults()), len(m.snapshot.told), 2}[m.pane]
	}
	if m.screen == detail {
		count = len(m.related)
	}
	if m.screen == review {
		m.reviewOffset = max(0, min(n, max(0, len(m.reviewLines())-(m.height-6))))
		return
	}
	*c = max(0, min(n, count-1))
}
func (m *model) move(n int) {
	if (m.screen == detail && m.pane == 0) || m.screen == preview {
		if n > 0 {
			m.viewport.ScrollDown(n)
		} else {
			m.viewport.ScrollUp(-n)
		}
		return
	}
	c := m.cursor
	if m.screen == overview {
		c = m.selections[m.pane]
	}
	if m.screen == review {
		c = m.reviewOffset
	}
	m.moveTo(c + n)
}
func (m *model) openSelected() {
	if m.screen == review {
		if m.reviewIndex < m.reviewCount() {
			ref := ""
			if m.reviewKind == "dupes" {
				ref = m.snapshot.dupes.Duplicates[m.reviewIndex].Keeper.SID
			} else {
				ref = m.snapshot.moves.Entries[m.reviewIndex].SID
			}
			m.openUnit(ref)
		}
		return
	}
	if m.screen == overview {
		c := m.selections[m.pane]
		switch m.pane {
		case 0:
			if c < len(m.snapshot.places) {
				p := m.snapshot.places[c]
				var rows []search.Result
				for _, u := range m.snapshot.units {
					if u.Scope == p.Scope {
						rows = append(rows, search.Result{Unit: u})
					}
				}
				m.showList("Place · "+p.Name, rows)
			}
		case 3:
			if c == 0 {
				m.startReview("dupes")
			} else {
				m.startReview("moves")
			}
		default:
			if u, ok := m.selected(); ok {
				m.openUnit(u.SID)
			}
		}
		return
	}
	if m.screen == detail {
		if m.pane == 1 && m.cursor < len(m.related) {
			m.openUnit(m.related[m.cursor].Unit.SID)
		}
		return
	}
	if u, ok := m.selected(); ok {
		m.openUnit(u.SID)
	}
}
func (m model) recallPreview(q string) tea.Cmd {
	a, b := m.a, m.binding
	return func() tea.Msg {
		b = b.Refresh(a)
		if b.Err != nil {
			return resultsMsg{err: b.Err}
		}
		r, err := a.Recall.Recall(context.Background(), recall.Request{Message: q, Harness: "tui", Scopes: b.Scopes, ScopeFor: b.ScopeFor})
		return resultsMsg{block: r.Block, err: err}
	}
}
