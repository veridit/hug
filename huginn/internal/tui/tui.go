// Package tui provides Huginn's overview, ranked list and task board.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

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
	// NoWatch turns off the reload on changes on disk (nfv18p); tests that
	// drive the model by hand leave it on through Run only.
	NoWatch bool
}

func Run(ctx context.Context, store *task.Store, opts Options) error {
	if store == nil {
		return fmt.Errorf("task store is required")
	}
	m := newModel(store, opts)
	if !opts.NoWatch {
		m.watch = startWatch(store.Root)
		defer m.watch.close()
	}
	_, err := tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen()).Run()
	return err
}

type editorMsg struct {
	path string // the task file the editor opened
	err  error
}
type model struct {
	store                              *task.Store
	opts                               Options
	all, visible                       []task.Task
	findings                           []task.Finding
	docs                               map[string]*task.Document
	blocked, linted                    map[string]bool
	epics                              map[string]bool // lowercase ids of tasks with children
	cursor, acceptance, offset         int
	dependency, detailPane             int
	link                               int                         // the task:ID marked > in Links (xxvm6t)
	memo                               [2][maxSections]sectionMemo // overview and board: each section's scroll and row (h9zghj)
	sort, sortDraft                    []sortKey                   // the order rows are in, and the one being built (mkzcqr)
	detailStack                        []string
	width, height                      int
	view                               string
	pane                               int
	detail, help, pendingG             bool
	helpTyping                         bool   // typing a help search after "/"
	helpQuery                          string // filters help rows by key or description
	prompt, query, message, detailPath string
	confirm, confirmPath, pendingValue string
	yanked                             string // y: the task id P and p paste (3xcesq)
	grab                               grabState
	filter                             task.Filter
	input                              textinput.Model
	viewport, helpViewport             viewport.Model
	helpModel                          help.Model
	completion                         int
	chips                              []filterChip
	pickerField                        string
	styles                             theme
	// editNotice is the task file whose edit introduced lint findings: the
	// notice shows the first one with its fix, and e reopens the editor,
	// until Esc, another action, or the file is clean (rx74bp).
	editNotice string
	// editBefore holds the problems the task had when the editor opened.
	editBefore map[string]bool
	// closedFindings are findings on done or dropped tasks, hidden from the
	// header count, the lint view and the ! marks (6neax1).
	closedFindings []task.Finding
	// Reload on changes on disk (nfv18p): watch signals a settled burst of
	// changes; loading runs off the UI thread; stale says a change came while
	// a prompt, confirm, palette or the editor was open, so the reload waits
	// for it to close. gen counts applied snapshots, so an older background
	// load never overwrites a newer one.
	watch          *watch
	loading, stale bool
	editing        bool
	gen            int
	loads          *loadCounter
	// closes reads close times from git for RECENTLY DONE (zaec08);
	// closedAt is the last snapshot's copy.
	closes   *closeLog
	closedAt map[string]int64
}

// loadCounter counts loads from disk, for the idle-load test.
type loadCounter struct{ n atomic.Int64 }

func (c *loadCounter) add() {
	if c != nil {
		c.n.Add(1)
	}
}

type changedMsg struct{}
type loadedMsg struct {
	sn  snapshot
	gen int
}

// waitChange waits for the next settled change on disk.
func (m model) waitChange() tea.Cmd {
	if m.watch == nil {
		return nil
	}
	c := m.watch.C
	return func() tea.Msg {
		if _, ok := <-c; !ok {
			return nil
		}
		return changedMsg{}
	}
}

// busy is true while a reload would interrupt the reader: a prompt or the
// palette, a confirm question, or the editor.
func (m model) busy() bool {
	return m.prompt != "" || m.confirm != "" || m.editing || m.grab.active
}

// reload starts a background load unless one is running or the reader is busy.
func (m *model) reload() tea.Cmd {
	if m.busy() || m.loading {
		m.stale = true
		return nil
	}
	m.stale, m.loading = false, true
	s, loads, gen, closes := m.store, m.loads, m.gen, m.closes
	return func() tea.Msg { return loadedMsg{loadSnapshot(s, loads, closes), gen} }
}

// selectionID is the task to keep selected across a reload: by id, since a
// move elsewhere renames the file.
func (m model) selectionID() string {
	if t, ok := m.selected(); ok {
		return t.ID
	}
	return ""
}

// applyKeeping shows a reloaded snapshot as if nothing moved: the same task
// selected, the list scrolled the same, the detail at the same line.
func (m *model) applyKeeping(sn snapshot) {
	id, offset, y := m.selectionID(), m.offset, m.viewport.YOffset
	message := m.message
	path, found := "", false
	for _, t := range sn.tasks {
		if id != "" && t.ID == id {
			path, found = t.Path, true
			break
		}
	}
	if m.detail && m.detailPath != "" && sn.err == nil {
		if found {
			m.detailPath = path
		} else {
			message = id + " is gone: it was removed elsewhere"
		}
	}
	m.apply(sn, path)
	if sn.err == nil {
		m.message = message
	}
	m.offset = offset
	m.viewport.SetYOffset(y)
}

// afterUpdate runs a reload that waited for a prompt, confirm or editor to
// close, once it has closed.
func (m model) afterUpdate(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.remember()
	if m.stale && !m.busy() && !m.loading {
		cmd = tea.Batch(cmd, m.reload())
	}
	return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case changedMsg:
		cmd := m.reload()
		return m, tea.Batch(cmd, m.waitChange())
	case loadedMsg:
		m.loading = false
		switch {
		case msg.gen != m.gen:
			// An action reloaded meanwhile, after this load began: the
			// screen already shows newer data.
		case m.busy():
			m.stale = true // a prompt opened meanwhile: wait for it
		default:
			m.applyKeeping(msg.sn)
		}
		return m.afterUpdate(nil)
	case editorMsg:
		// The editor's own refresh reads everything that changed meanwhile.
		m.editing, m.stale = false, false
	}
	next, cmd := m.update(msg)
	return next.(model).afterUpdate(cmd)
}

func newModel(s *task.Store, o Options) model {
	input := textinput.New()
	input.CharLimit = 512
	input.Cursor.Blink = false
	m := model{store: s, opts: o, width: 100, height: 30, view: "overview", input: input, closes: &closeLog{},
		viewport: viewport.New(100, 24), helpViewport: viewport.New(100, 24), helpModel: help.New(), styles: newTheme()}
	m.refresh("")
	// Prefer an occupied overview pane, while retaining the same pane order.
	for p := 0; p < len(overviewSections); p++ {
		m.pane = p
		if len(m.indices()) > 0 {
			break
		}
	}
	m.clamp()
	m.resize()
	return m
}
func (m model) Init() tea.Cmd { return m.waitChange() }
func (m *model) refresh(path string) {
	m.apply(loadSnapshot(m.store, m.loads, m.closes), path)
}

// snapshot is everything refresh reads from disk: the tasks, their parsed
// files and the lint findings. loadSnapshot builds it without touching the
// model, so a reload after a change on disk runs off the UI thread (nfv18p).
type snapshot struct {
	tasks            []task.Task
	docs             map[string]*task.Document
	findings, closed []task.Finding
	closedAt         map[string]int64 // close times from git (closeLog)
	err, readErr     error
}

func loadSnapshot(s *task.Store, loads *loadCounter, closes *closeLog) snapshot {
	loads.add()
	ts, err := s.Tasks()
	if err != nil {
		return snapshot{err: err}
	}
	task.Sort(ts)
	sn := snapshot{tasks: ts, docs: map[string]*task.Document{}, closedAt: closes.times(s.Root)}
	var docs []*task.Document
	for _, t := range ts {
		b, e := os.ReadFile(filepath.Join(s.Root, t.Path))
		if e != nil {
			sn.readErr = e
			continue
		}
		d := task.Parse(t.Path, b)
		docs = append(docs, d)
		sn.docs[t.Path] = d
	}
	// Findings on closed tasks (done, dropped) are hidden by default, as in
	// hug task lint; closedFindings keeps them for counts and for the detail
	// of a closed task opened on purpose (6neax1).
	sn.findings, sn.closed = task.OpenFindings(task.Lint(docs), ts)
	return sn
}

// apply shows a snapshot, keeping the task at path selected.
func (m *model) apply(sn snapshot, path string) {
	m.gen++
	if sn.err != nil {
		m.message = sn.err.Error()
		return
	}
	ts := sn.tasks
	m.all = ts
	m.docs = sn.docs
	if sn.readErr != nil {
		m.message = sn.readErr.Error()
	}
	m.findings, m.closedFindings = sn.findings, sn.closed
	m.closedAt = sn.closedAt
	m.blocked = map[string]bool{}
	m.linted = map[string]bool{}
	m.epics = map[string]bool{}
	for id := range task.ChildIndex(ts) {
		m.epics[id] = true
	}
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
		// One place per task (zaec08): "blocked" is todo with an unmet
		// dependency, "todo" excludes it, "doing" keeps blocked doing tasks.
		if status != "" && m.place(t) != status || m.filter.Type != "" && !optionalMatches(t.Type, m.filter.Type) || m.filter.Group != "" && !groupMatches(t.Group, m.filter.Group) {
			continue
		}
		if m.filter.Label != "" && !contains(t.Labels, m.filter.Label) {
			continue
		}
		if !task.InEpic(t, m.all, m.filter.Epic) {
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
	m.sortVisible() // the order built with : sort (mkzcqr); rank order without one
	m.offset = 0
	m.cursor = 0
	m.clamp()
}

var boardStatuses = []string{"todo", "doing", "blocked", "done", "dropped"}

// Overview sections, top to bottom. Every task is in one place (zaec08): a
// doing task stays in IN PROGRESS even when a dependency is unmet, drawn
// blocked; BLOCKED holds only todo tasks that cannot start.
var overviewSections = []string{"IN PROGRESS · assigned to", "NEXT UP · rank order", "BLOCKED · unmet dependencies", "RECENTLY DONE · last closed", "LINT · tasks needing attention"}

const (
	sectionDoing = iota
	sectionNext
	sectionBlocked
	sectionRecent
	sectionLint
)

// recentDone is how many closed tasks RECENTLY DONE shows while the cursor
// is in another section; focused, it scrolls through all of them (h9zghj).
// Every other unfocused section shows as many rows.
const recentDone = 3

// maxSections is the most sections a view has: 5 overview sections, 5
// board columns.
const maxSections = 5

// sectionMemo is where a section was when the cursor left it: its first
// row shown and the task selected in it (h9zghj).
type sectionMemo struct {
	start int
	path  string
}

// memoView is the memo slot of a view with sections, or -1.
func memoView(view string) int {
	switch view {
	case "overview":
		return 0
	case "board":
		return 1
	}
	return -1
}

// remember records the focused section's scroll and selected task, so it
// is where it was when the cursor comes back.
func (m *model) remember() {
	v := memoView(m.view)
	if v < 0 || m.pane < 0 || m.pane >= maxSections {
		return
	}
	indices := m.indices()
	memo := &m.memo[v][m.pane]
	memo.start, _ = window(indices, m.cursor, memo.start, m.sectionRows(m.pane))
	memo.path = ""
	for _, i := range indices {
		if i == m.cursor {
			memo.path = m.visible[i].Path
		}
	}
}

// restore selects the task the focused section had when the cursor left.
func (m *model) restore() {
	if c := m.memoCursor(m.pane); c >= 0 {
		m.cursor = c
	}
}

// place is the one status a task counts under, in the header, the board
// columns and the status filter (zaec08): done, dropped and doing as they
// are; a todo task with an unmet dependency is blocked; anything else
// (a malformed status) is todo, still visible and repairable, marked with !.
func (m model) place(t task.Task) string {
	switch t.Status {
	case "done", "dropped", "doing":
		return t.Status
	}
	if m.blocked[t.Path] {
		return "blocked"
	}
	return "todo"
}

// boardStatus is the status a row shows: its place, except that a doing
// task waiting on a dependency is drawn blocked where it stands.
func (m model) boardStatus(t task.Task) string {
	if t.Status == "doing" && m.blocked[t.Path] {
		return "blocked"
	}
	return m.place(t)
}
func (m model) paneIndices(p int) []int {
	if m.view == "overview" && p == sectionRecent {
		return m.recentIndices()
	}
	var out []int
	for i, t := range m.visible {
		match := true
		switch m.view {
		case "overview":
			switch p {
			case sectionDoing:
				match = t.Status == "doing"
			case sectionNext:
				// Same rule as hug task next: unblocked, unassigned todo,
				// never an epic (it finishes through its children).
				match = m.place(t) == "todo" && t.Status == "todo" && !m.epics[strings.ToLower(t.ID)] && len(t.Assignee) == 0
			case sectionBlocked:
				match = m.place(t) == "blocked"
			case sectionLint:
				match = m.linted[t.Path]
			}
		case "board":
			match = m.place(t) == boardStatuses[p]
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

// stepTask moves the open detail to the next or previous task in the pane it
// was opened from. It stops at either end.
func (m *model) stepTask(step int) {
	m.detailStack = nil
	m.detailPath = ""
	before := m.cursor
	m.move(step, false)
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		m.detailPath = m.visible[m.cursor].Path
	}
	if m.cursor != before {
		m.startFocus()
		m.syncDetail()
		m.openTop()
		return
	}
	m.syncDetail()
}

// detailPosition is the open task's place in its pane: "3 of 22".
func (m model) detailPosition() string {
	indices := m.indices()
	for p, i := range indices {
		if i == m.cursor && i < len(m.visible) && m.visible[i].Path == m.detailPath {
			return fmt.Sprintf("%d of %d", p+1, len(indices))
		}
	}
	return ""
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
		m.remember()
		m.pane = (m.pane + delta + len(overviewSections)) % len(overviewSections)
		m.restore()
	case "board":
		m.remember()
		m.pane = (m.pane + delta + len(boardStatuses)) % len(boardStatuses)
		m.restore()
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
	case m.editNotice != "" && m.prompt == "":
		m.editNotice = ""
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
		m.startFocus()
		m.syncDetail()
		m.openTop()
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
func (m model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.refresh(msg.path)
		m.editNotice = ""
		if msg.err != nil {
			m.message = msg.err.Error()
		} else {
			m.message = "editor closed; re-linted"
			if m.opts.Commit != nil {
				if err := m.opts.Commit("hug task edit"); err != nil {
					m.message = err.Error()
				}
			}
			m.editNotice = msg.path
			if _, _, ok := m.noticeTask(); ok {
				// Say what the edit broke and how to repair it, in the
				// footer and the bar under it, instead of a quiet message.
				// A failed commit keeps its message: that matters more.
				if m.message == "editor closed; re-linted" {
					m.message = ""
				}
			} else {
				m.editNotice = ""
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
			if m.grab.active && !m.help {
				m.cancelGrab()
				return m, nil
			}
			if m.yanked != "" && !m.help && m.prompt == "" && m.confirm == "" {
				m.message = "forgot " + m.yanked + "; nothing changed"
				m.yanked = ""
				return m, nil
			}
			m.back()
			return m, nil
		}
		// While a task is grabbed, keys move it; nothing else runs (3xcesq).
		if m.grab.active && !m.help {
			return m.updateGrab(k)
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
		// After an edit that broke the file, e goes straight back into the
		// editor on that task: the notice is the question (rx74bp).
		if k == "e" {
			if t, _, ok := m.noticeTask(); ok {
				cmd := m.runEditor(t)
				return m, cmd
			}
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
				// [ and ] step through the focused list; the page has none.
				step := 1
				if k == "[" {
					step = -1
				}
				switch m.detailPane {
				case focusAcceptance:
					m.acceptance += step
				case focusDeps:
					if t, ok := m.selected(); ok {
						m.dependency = max(0, min(m.dependency+step, len(t.Dependencies)-1))
					}
				case focusLinks:
					if t, ok := m.selected(); ok {
						m.link = max(0, min(m.link+step, len(m.links(t))-1))
					}
				default:
					m.message = "[ and ] step through a list: Tab focuses the checks or dependencies"
				}
				m.clamp()
				m.syncDetail()
				m.revealAcceptance()
			case "h", "l", "left", "right", "tab", "shift+tab":
				// Focus cycles page, acceptance, dependencies, skipping
				// empty sections (2026-11-16, user).
				step := 1
				if k == "h" || k == "left" || k == "shift+tab" {
					step = -1
				}
				m.cycleFocus(step)
				m.syncDetail()
				m.revealAcceptance()
			case "enter":
				switch m.detailPane {
				case focusDeps:
					m.followDependency()
				case focusLinks:
					m.followLink()
				}
			case " ":
				// Space ticks the check marked >; with focus elsewhere no
				// check is marked, and ticking an unmarked one is a guess.
				if m.detailPane == focusAcceptance {
					m.requestAction(k, "")
				} else if t, ok := m.selected(); ok && len(t.Acceptance) > 0 {
					m.message = "Space ticks the check marked >: Tab focuses the checks"
				} else {
					m.message = "no acceptance checks to tick"
				}
			case "t", "d", "x", "J", "K", "e":
				m.requestAction(k, "")
			case "/", ":", "#":
				return m, m.openInput(k, "")
			case "f":
				return m, m.openFilter()
			case "N":
				return m, m.openInput("new title", "")
			case "s":
				return m, m.openInput(":", "status ")
			case "y":
				return m, m.openInput(":", "type ")
			case "r":
				if t, ok := m.selected(); ok {
					m.refresh(t.Path)
				}
			case "n", "p", "ctrl+n", "ctrl+p":
				// Next or previous task of the list the detail was opened
				// from, without going back first. n and p, not { and }, which
				// are hard to reach on a Norwegian keyboard (2026-11-16, user).
				step := 1
				if k == "p" || k == "ctrl+p" {
					step = -1
				}
				m.stepTask(step)
			case "o", "i", "v", "b", "!":
				// The tab bar stays visible in detail, so its keys work here
				// too: leave the task and switch view (2026-11-15, user).
				m.detail = false
				m.setView(map[string]string{"o": "overview", "i": "list", "v": "list", "b": "board", "!": "lint"}[k])
			case "j", "down", "k", "up":
				// With a list focused, j/k step through it and scroll the
				// page past either end; with the page focused they scroll.
				step := 1
				if k == "k" || k == "up" {
					step = -1
				}
				t, ok := m.selected()
				moved := false
				if ok && m.detailPane == focusAcceptance && len(t.Acceptance) > 0 {
					next := m.acceptance + step
					if next >= 0 && next < len(t.Acceptance) {
						m.acceptance, moved = next, true
					}
				} else if ok && m.detailPane == focusDeps && len(t.Dependencies) > 0 {
					next := m.dependency + step
					if next >= 0 && next < len(t.Dependencies) {
						m.dependency, moved = next, true
					}
				} else if ok && m.detailPane == focusLinks {
					if next := m.link + step; next >= 0 && next < len(m.links(t)) {
						m.link, moved = next, true
					}
				}
				if moved {
					m.syncDetail()
					m.revealAcceptance()
				} else {
					m.scroll(&m.viewport, k)
				}
			default:
				m.scroll(&m.viewport, k)
			}
			return m, nil
		}
		switch {
		case bkey.Matches(msg, keys.down) || k == "n":
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
				m.startFocus()
				m.syncDetail()
				m.openTop()
			}
		case k == "/":
			return m, m.openInput("/", m.query)
		case k == ":":
			return m, m.openInput(":", "")
		case k == "s":
			return m, m.openInput(":", "status ")
		case k == "y":
			// y yanks the selected task; P and p paste it (3xcesq). The
			// type palette is :type, f t, or y inside a task.
			m.yank()
		case k == "P" || k == "p":
			// In lists P and p only paste; k moves up (crab, 2026-11-16).
			if why := m.moveRefusal(); why != "" && len(m.sort) > 0 {
				m.message = why
			} else if m.yanked == "" {
				m.message = "nothing yanked: y on a task first"
			} else {
				m.paste(k)
			}
		case k == "m":
			m.startGrab()
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
		case k == "N":
			// N, not n: n and p step like j and k, and to the next task
			// inside one (2026-11-16, user).
			return m, m.openInput("new title", "")
		case k == "r":
			path := ""
			if t, ok := m.selected(); ok {
				path = t.Path
			}
			m.refresh(path)
		case k == " ":
			// Ticking a check you cannot see is a guess: Space ticks only
			// inside the task, where the checks are shown (2026-11-16, user).
			if t, ok := m.selected(); ok {
				m.message = fmt.Sprintf("Space ticks a check inside the task: Enter opens %s", t.ID)
			}
		case contains([]string{"t", "d", "x", "J", "K", "e"}, k):
			m.requestAction(k, "")
		}
	}
	return m, nil
}
func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.prompt == "sort" {
		return m.updateSort(msg)
	}
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
			if strings.TrimSpace(unfilter(m.input.Value())) == "sort rank" {
				m.applySort(nil)
				m.prompt = ""
				m.input.Blur()
				return m, nil
			}
			if strings.TrimSpace(m.input.Value()) == "sort" {
				m.input.SetValue("")
				m.openSort()
				return m, nil
			}
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
