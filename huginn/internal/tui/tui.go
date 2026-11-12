// Package tui provides Huginn's overview, ranked list and task board.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

type Options struct {
	User   string
	Commit func(msg string) error
	// Take owns the complete distributed take transaction, including commit.
	Take func(ref string) (task.Result, error)
}

func Run(ctx context.Context, store *task.Store, opts Options) error {
	if store == nil {
		return fmt.Errorf("task store is required")
	}
	_, err := tea.NewProgram(newModel(store, opts), tea.WithContext(ctx), tea.WithAltScreen()).Run()
	return err
}

type editorMsg struct{ err error }
type model struct {
	store                              *task.Store
	opts                               Options
	all, visible                       []task.Task
	findings                           []task.Finding
	docs                               map[string]*task.Document
	blocked, linted                    map[string]bool
	cursor, acceptance, offset         int
	dependency, detailPane             int
	detailStack                        []string
	width, height                      int
	view                               string
	pane                               int
	detail, help, pendingG             bool
	helpTyping                         bool   // typing a help search after "/"
	helpQuery                          string // filters help rows by key or description
	prompt, query, message, detailPath string
	confirm, confirmPath, pendingValue string
	filter                             task.Filter
	input                              textinput.Model
	viewport, helpViewport             viewport.Model
	helpModel                          help.Model
	completion                         int
	chips                              []filterChip
	pickerField                        string
	styles                             theme
}

func newModel(s *task.Store, o Options) model {
	input := textinput.New()
	input.CharLimit = 512
	input.Cursor.Blink = false
	m := model{store: s, opts: o, width: 100, height: 30, view: "overview", input: input,
		viewport: viewport.New(100, 24), helpViewport: viewport.New(100, 24), helpModel: help.New(), styles: newTheme()}
	m.refresh("")
	// Prefer an occupied overview pane, while retaining the same pane order.
	for p := 0; p < 4; p++ {
		m.pane = p
		if len(m.indices()) > 0 {
			break
		}
	}
	m.clamp()
	m.resize()
	return m
}
func (m model) Init() tea.Cmd { return nil }
func (m *model) refresh(path string) {
	ts, err := m.store.Tasks()
	if err != nil {
		m.message = err.Error()
		return
	}
	task.Sort(ts)
	m.all = ts
	m.docs = map[string]*task.Document{}
	var docs []*task.Document
	for _, t := range ts {
		b, e := os.ReadFile(filepath.Join(m.store.Root, t.Path))
		if e != nil {
			m.message = e.Error()
			continue
		}
		d := task.Parse(t.Path, b)
		docs = append(docs, d)
		m.docs[t.Path] = d
	}
	m.findings = task.Lint(docs)
	m.blocked = map[string]bool{}
	m.linted = map[string]bool{}
	for _, t := range ts {
		m.blocked[t.Path] = t.Status != "done" && t.Status != "dropped" && len(task.Blocked(t, ts)) > 0
	}
	for _, f := range m.findings {
		m.linted[f.Path] = true
	}
	m.applyFilter()
	for i, t := range m.visible {
		if t.Path == path {
			m.cursor = i
			break
		}
	}
	m.clamp()
	m.syncDetail()
}
func (m *model) applyFilter() {
	m.visible = nil
	for _, t := range m.all {
		status := m.filter.Status
		if status == "blocked" && !m.blocked[t.Path] || status != "" && status != "blocked" && t.Status != status || m.filter.Type != "" && !optionalMatches(t.Type, m.filter.Type) || m.filter.Group != "" && !groupMatches(t.Group, m.filter.Group) {
			continue
		}
		if m.filter.Label != "" && !contains(t.Labels, m.filter.Label) {
			continue
		}
		if m.filter.Priority != "" && !optionalMatches(t.Priority, m.filter.Priority) || m.filter.Mine != "" && !ownerMatches(t, m.filter.Mine) {
			continue
		}
		if m.query != "" && !strings.Contains(strings.ToLower(t.Slug+" "+t.Title+" "+t.ID+" "+strings.Join(t.Assignee, " ")), strings.ToLower(m.query)) {
			continue
		}
		m.visible = append(m.visible, t)
	}
	m.offset = 0
	m.cursor = 0
	m.clamp()
}

var boardStatuses = []string{"todo", "doing", "blocked", "done", "dropped"}

func (m model) boardStatus(t task.Task) string {
	if m.blocked[t.Path] {
		return "blocked"
	}
	if contains(boardStatuses, t.Status) {
		return t.Status
	}
	return "todo" // Malformed tasks remain visible and repairable, marked with !.
}
func (m model) paneIndices(p int) []int {
	var out []int
	for i, t := range m.visible {
		match := true
		switch m.view {
		case "overview":
			switch p {
			case 0:
				match = t.Status == "doing"
			case 1:
				match = t.Status == "todo" && !m.blocked[t.Path]
			case 2:
				match = m.blocked[t.Path]
			case 3:
				match = m.linted[t.Path]
			}
		case "board":
			match = m.boardStatus(t) == boardStatuses[p]
		case "lint":
			match = m.linted[t.Path]
		}
		if match {
			out = append(out, i)
		}
	}
	return out
}
func (m model) indices() []int { return m.paneIndices(m.pane) }
func (m *model) clamp() {
	indices := m.indices()
	found := false
	for _, i := range indices {
		if m.cursor == i {
			found = true
			break
		}
	}
	if !found {
		m.cursor = 0
		if len(indices) > 0 {
			m.cursor = indices[0]
		}
	}
	if t, ok := m.selected(); ok {
		m.acceptance = min(m.acceptance, max(0, len(t.Acceptance)-1))
	}
	m.acceptance = max(0, m.acceptance)
}
func (m model) selected() (task.Task, bool) {
	if m.detail && m.detailPath != "" {
		for _, t := range m.all {
			if t.Path == m.detailPath {
				return t, true
			}
		}
	}
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return task.Task{}, false
	}
	for _, i := range m.indices() {
		if i == m.cursor {
			return m.visible[i], true
		}
	}
	return task.Task{}, false
}
func (m *model) setView(view string) {
	m.view = view
	m.pane = 0
	m.offset = 0
	m.detail = false
	m.detailPath = ""
	m.detailStack = nil
	m.detailPane = 0
	m.pendingG = false
	m.clamp()
}
func (m *model) move(n int, absolute bool) {
	indices := m.indices()
	if len(indices) == 0 {
		return
	}
	pos := 0
	for p, i := range indices {
		if i == m.cursor {
			pos = p
			break
		}
	}
	if absolute {
		pos = n
	} else {
		pos += n
	}
	pos = max(0, min(pos, len(indices)-1))
	m.cursor = indices[pos]
	m.clamp()
}
func (m *model) changePane(delta int) {
	switch m.view {
	case "overview":
		m.pane = (m.pane + delta + 4) % 4
	case "board":
		m.pane = (m.pane + delta + len(boardStatuses)) % len(boardStatuses)
	default:
		views := []string{"overview", "list", "board", "lint"}
		for i, v := range views {
			if m.view == v {
				m.setView(views[(i+delta+len(views))%len(views)])
				return
			}
		}
	}
	m.offset = 0
	m.clamp()
}
func (m *model) resize() {
	m.width = max(1, m.width)
	m.height = max(1, m.height)
	m.input.Width = max(1, m.width-6)
	m.viewport.Width = m.width
	m.viewport.Height = m.bodyHeight()
	m.helpViewport.Width = m.width
	m.helpViewport.Height = m.bodyHeight()
	m.helpModel.Width = m.width
	m.helpModel.Styles.ShortKey = m.styles.accent
	m.helpModel.Styles.ShortDesc = m.styles.dim
	m.helpModel.Styles.ShortSeparator = m.styles.dim
	m.helpModel.Styles.FullKey = m.styles.accent
	m.helpModel.Styles.FullDesc = m.styles.normal
	m.helpModel.Styles.FullSeparator = m.styles.dim
	m.helpViewport.SetContent(m.helpText())
	m.syncDetail()
}
func (m *model) syncDetail() {
	if t, ok := m.selected(); ok {
		m.viewport.Height = max(1, m.bodyHeight()-len(m.detailHeader(t))-1)
		m.viewport.SetContent(m.detailText(t))
	}
}
func (m *model) openInput(prompt, value string) tea.Cmd {
	m.prompt = prompt
	m.pendingG = false
	m.completion = 0
	m.input.Prompt = prompt + " "
	m.input.SetValue(value)
	m.input.CursorEnd()
	m.input.PromptStyle = m.styles.accent
	m.input.TextStyle = m.styles.normal
	return m.input.Focus()
}
func (m *model) clearFilters() {
	m.query = ""
	m.filter = task.Filter{}
	m.chips = nil
	m.applyFilter()
}
func (m *model) back() {
	m.pendingG = false
	switch {
	case m.help && (m.helpTyping || m.helpQuery != ""):
		// Esc clears a help search first; the next Esc closes help.
		m.helpTyping = false
		m.helpQuery = ""
		m.helpViewport.SetContent(m.helpText())
		m.helpViewport.GotoTop()
	case m.help:
		m.help = false
	case m.confirm != "":
		m.confirm = ""
		m.pendingValue = ""
	case m.prompt == "filter" && m.pickerField != "":
		m.pickerField = ""
		m.input.SetValue("")
		m.completion = 0
	case m.prompt != "":
		if m.prompt == "/" {
			m.query = ""
			m.applyFilter()
		}
		m.prompt = ""
		m.input.Blur()
	case m.detail && len(m.detailStack) > 0:
		m.detailPath = m.detailStack[len(m.detailStack)-1]
		m.detailStack = m.detailStack[:len(m.detailStack)-1]
		m.detailPane = 0
		m.syncDetail()
		m.viewport.GotoTop()
	case m.detail:
		m.detail = false
		m.detailPath = ""
		m.clamp()
	case m.query != "" || m.filter != (task.Filter{}):
		m.clearFilters()
	case m.view != "overview":
		m.setView("overview")
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Bubble Tea coalesces fast typing, including gg, into rune batches.
	if event, ok := msg.(tea.KeyMsg); ok && event.Type == tea.KeyRunes && len(event.Runes) > 1 {
		var cmds []tea.Cmd
		for _, r := range event.Runes {
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = next.(model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Some PTYs report zero before their first usable size is available.
		// Keep the default or last layout until both dimensions are known.
		if msg.Width > 0 && msg.Height > 0 {
			m.width = msg.Width
			m.height = msg.Height
			m.resize()
		}
	case editorMsg:
		m.refresh("")
		if msg.err != nil {
			m.message = msg.err.Error()
		} else {
			m.message = "editor closed; re-linted"
			if m.opts.Commit != nil {
				if err := m.opts.Commit("huginn: edit task"); err != nil {
					m.message = err.Error()
				}
			}
		}
	case tea.KeyMsg:
		k := msg.String()
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		// Help always wins over text input, and leaves that input intact underneath.
		if k == "?" && !(m.help && m.helpTyping) {
			m.help = !m.help
			m.helpTyping = false
			m.helpQuery = ""
			m.helpViewport.SetContent(m.helpText())
			m.helpViewport.GotoTop()
			return m, nil
		}
		if k == "esc" {
			m.back()
			return m, nil
		}
		if m.help {
			if m.helpTyping {
				switch {
				case k == "enter":
					m.helpTyping = false
				case k == "backspace":
					if r := []rune(m.helpQuery); len(r) > 0 {
						m.helpQuery = string(r[:len(r)-1])
					}
				case msg.Type == tea.KeyRunes || k == " ":
					m.helpQuery += string(msg.Runes)
				}
				m.helpViewport.SetContent(m.helpText())
				m.helpViewport.GotoTop()
				return m, nil
			}
			if k == "/" {
				m.helpTyping = true
				m.pendingG = false
				return m, nil
			}
			if k == "g" {
				if m.pendingG {
					m.helpViewport.GotoTop()
					m.pendingG = false
				} else {
					m.pendingG = true
				}
				return m, nil
			}
			m.pendingG = false
			if k == "q" && m.prompt == "" {
				return m, tea.Quit
			}
			m.scroll(&m.helpViewport, k)
			return m, nil
		}
		if m.confirm != "" {
			if k == "q" {
				return m, tea.Quit
			}
			if k == "enter" {
				cmd := m.executeAction()
				return m, cmd
			}
			return m, nil
		}
		if m.prompt != "" {
			return m.updateInput(msg)
		}
		if k == "q" {
			return m, tea.Quit
		}
		if k == "g" {
			if m.pendingG {
				m.pendingG = false
				if m.detail {
					m.viewport.GotoTop()
				} else {
					m.move(0, true)
				}
			} else {
				m.pendingG = true
			}
			return m, nil
		}
		m.pendingG = false
		// Filtering remains available from detail, without interfering with focused inputs.
		if k == "F" {
			m.clearFilters()
			m.syncDetail()
			return m, nil
		}
		if k == "backspace" {
			m.removeChip()
			m.syncDetail()
			return m, nil
		}
		if k == "1" || k == "2" || k == "3" || k == "4" {
			m.setFilter("status", []string{"todo", "doing", "done", "blocked"}[int(k[0]-'1')])
			m.setView("list")
			return m, nil
		}
		if m.detail {
			switch k {
			case "[", "]":
				step := 1
				if k == "[" {
					step = -1
				}
				if m.detailPane == 0 {
					m.acceptance += step
				} else {
					if t, ok := m.selected(); ok {
						m.dependency = max(0, min(m.dependency+step, len(t.Dependencies)-1))
					}
				}
				m.clamp()
				m.syncDetail()
				m.revealAcceptance()
			case "h", "l", "left", "right", "tab", "shift+tab":
				m.detailPane = 1 - m.detailPane
				m.syncDetail()
				m.revealAcceptance()
			case "enter":
				if m.detailPane == 1 {
					m.followDependency()
				}
			case "t", "d", "x", "J", "K", " ", "e":
				m.requestAction(k, "")
			case "/", ":", "#":
				return m, m.openInput(k, "")
			case "f":
				return m, m.openFilter()
			case "n":
				return m, m.openInput("new title", "")
			case "s":
				return m, m.openInput(":", "status ")
			case "y":
				return m, m.openInput(":", "type ")
			case "r":
				if t, ok := m.selected(); ok {
					m.refresh(t.Path)
				}
			default:
				m.scroll(&m.viewport, k)
			}
			return m, nil
		}
		switch {
		case bkey.Matches(msg, keys.down):
			m.move(1, false)
		case bkey.Matches(msg, keys.up):
			m.move(-1, false)
		case bkey.Matches(msg, keys.top):
			m.move(0, true)
		case bkey.Matches(msg, keys.bottom):
			m.move(len(m.visible), true)
		case k == "ctrl+d":
			m.move(max(1, m.bodyHeight()/2), false)
		case k == "ctrl+u":
			m.move(-max(1, m.bodyHeight()/2), false)
		case k == "pgdown":
			m.move(m.bodyHeight(), false)
		case k == "pgup":
			m.move(-m.bodyHeight(), false)
		case bkey.Matches(msg, keys.prev):
			m.changePane(-1)
		case bkey.Matches(msg, keys.next):
			m.changePane(1)
		case k == "enter":
			if t, ok := m.selected(); ok {
				m.detailPath = t.Path
				m.detail = true
				m.syncDetail()
				m.viewport.GotoTop()
			}
		case k == "/":
			return m, m.openInput("/", m.query)
		case k == ":":
			return m, m.openInput(":", "")
		case k == "s":
			return m, m.openInput(":", "status ")
		case k == "y":
			return m, m.openInput(":", "type ")
		case k == "#":
			return m, m.openInput("#", "")
		case k == "f":
			return m, m.openFilter()
		case k == "c":
			m.clearFilters()
		case k == "b":
			m.setView("board")
		case k == "o":
			m.setView("overview")
		case k == "v" || k == "i":
			m.setView("list")
		case k == "!":
			m.setView("lint")
		case k == "n":
			return m, m.openInput("new title", "")
		case k == "r":
			path := ""
			if t, ok := m.selected(); ok {
				path = t.Path
			}
			m.refresh(path)
		case contains([]string{"t", "d", "x", "J", "K", " ", "e"}, k):
			m.requestAction(k, "")
		}
	}
	return m, nil
}
func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.prompt == "filter" {
		return m.updateFilter(msg)
	}
	if m.prompt == "#" {
		return m.updateJump(msg)
	}
	if m.prompt == ":" {
		cs := m.completions()
		switch k {
		case "ctrl+n", "down":
			if len(cs) > 0 {
				m.completion = (m.completion + 1) % len(cs)
			}
			return m, nil
		case "ctrl+p", "up":
			if len(cs) > 0 {
				m.completion = (m.completion + len(cs) - 1) % len(cs)
			}
			return m, nil
		case "tab":
			m.complete()
			return m, nil
		case "enter":
			if m.commandReady() {
				if m.executeCommand() {
					m.prompt = ""
					m.input.Blur()
				}
			} else {
				m.complete()
			}
			return m, nil
		}
	} else if k == "enter" {
		if m.prompt == "new title" && strings.TrimSpace(m.input.Value()) != "" {
			m.requestAction("n", strings.TrimSpace(m.input.Value()))
		}
		m.prompt = ""
		m.input.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.completion = 0
	if m.prompt == "/" {
		m.query = m.input.Value()
		m.applyFilter()
	}
	return m, cmd
}
func (m *model) scroll(v *viewport.Model, k string) {
	switch k {
	case "j", "down":
		v.ScrollDown(1)
	case "k", "up":
		v.ScrollUp(1)
	case "G", ">", "end":
		v.GotoBottom()
	case "<", "home":
		v.GotoTop()
	case "ctrl+d":
		v.HalfViewDown()
	case "ctrl+u":
		v.HalfViewUp()
	case "pgdown":
		v.ViewDown()
	case "pgup":
		v.ViewUp()
	}
}
func contains(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
