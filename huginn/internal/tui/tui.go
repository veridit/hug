// Package tui provides Huginn's overview, ranked list and task board.
package tui

import (
	"context"
	"fmt"
	"github.com/veridit/hug/internal/sharefile"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/help"
	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

type Options struct {
	// User is who the TUI user is, as a name tasks write: an agent's
	// session name, else the person found by git email (Email) in
	// hug.toml [people], else Email before the @ (89y0b7, Jørgen 15:31).
	User string
	// Email is git config user.email, when the user is a person. Used
	// only when User is empty, and read again when hug.toml changes.
	Email  string
	Commit func(msg string) error
	// Turn, when set, takes the board's write turn and returns its release
	// (95ahtk). Every write holds it from changing a task file to the
	// commit, as a CLI write does: a hug task note from another agent at
	// the same moment would otherwise stage this write's temp file or
	// collide on git's index.lock (Seid, 2026-11-17 16:49:23). Commit does
	// not take it again. Take is not wrapped: it orders its locks itself.
	Turn func() func()
	// Input, when set, drives the TUI instead of the terminal, with Output
	// and no renderer (tests): q at its end quits.
	Input  io.Reader
	Output io.Writer
	// Take owns the complete distributed take transaction, including commit.
	Take func(ref string) (task.Result, error)
	// NoWatch turns off the reload on changes on disk (nfv18p); tests that
	// drive the model by hand leave it on through Run only.
	NoWatch bool
	// Version is what hug version prints ("v0.7.29"); the key bar ends with
	// "hug v0.7.29" when there is room (Jørgen, 2026-11-16).
	Version string
	// NoReload turns off the reload into a new hug at the started path
	// (j2vwk0); Binary is that path, the one the shell found ("" finds it).
	NoReload bool
	Binary   string
	// Getenv reads the environment (nil: os.Getenv): which terminal draws
	// images, tmux (5z4zt4). Tests choose a terminal with it.
	Getenv func(string) string
	// MainBranch is the repository's main line ("main"): IN PROGRESS rows
	// name a task's branch only when it is another one (x1t1t1).
	MainBranch string
	// Lint, when set, returns the findings of the board the store's last
	// Tasks returned, from the task cache (cache.LintListed). Then a reload
	// parses no task file: a task's file is read when its detail opens
	// (44w1ev: parsing 10,000 files on every reload was most of a 3-5 s
	// first screen).
	Lint func() ([]task.Finding, error)
	// Search, when set, ranks / search: the task paths matching the query,
	// best first, from the task cache's index (49n9kw, as hug task search).
	// Without it, / search filters by substring in rank order.
	Search func(query string) ([]string, error)
	// Initial, when set, is drawn before anything is read: the board the
	// task cache last saw (vnyz8z). Then Refresh, when set, brings the
	// board up to date (the cache's sync) before the first reload, off the
	// UI thread, behind the spinner. Without Initial the TUI reads the
	// board before its first frame, as before.
	Initial *Initial
	Refresh func(ctx context.Context) error
	// Progress, when set, says how many task files the refresh has read
	// of how many: a first start counts them on screen (vnyz8z).
	Progress func() (read, total int)
}

func Run(ctx context.Context, store *task.Store, opts Options) error {
	if store == nil {
		return fmt.Errorf("task store is required")
	}
	m := startModel(ctx, store, opts)
	// Quitting never waits for a refresh or the watches (vnyz8z).
	defer func() { m.release() }()
	// A hug that reloaded into this one left its place (j2vwk0).
	if h := takeHandoff(); h != nil {
		m.resume(*h)
		m.message = "reloaded hug " + opts.Version
	}
	if !opts.NoReload && canReexec {
		path := opts.Binary
		if path == "" {
			path = startedPath()
		}
		m.bin = startBinWatch(path)
	}
	popts := []tea.ProgramOption{tea.WithAltScreen()}
	if opts.Input != nil {
		out := opts.Output
		if out == nil {
			out = io.Discard
		}
		popts = []tea.ProgramOption{tea.WithInput(opts.Input), tea.WithOutput(out), tea.WithoutRenderer()}
	}
	for {
		done, err := runOnce(ctx, m, popts...)
		m = done
		if err != nil {
			return err
		}
		if done.reloadTo == "" {
			return nil
		}
		// The terminal is restored: hand over and become the new hug (the
		// watches' goroutines and close-on-exec files go with this process).
		// Exec returns only on failure; then this hug carries on where it was.
		err = reloadInto(done.bin.path, done.handoff())
		done.reloadTo = ""
		done.message = fmt.Sprintf("reload into the hug at %s failed (%v); still on hug %s\nfix: quit with q, then run hug tui", done.bin.path, err, opts.Version)
		m = done
	}
}

type editorMsg struct {
	path string // the task file the editor opened
	err  error
	// commit is the commit message; "" means "hug task edit" (zgr01t:
	// task files say which file).
	commit string
}
type model struct {
	store                      *task.Store
	opts                       Options
	all, visible               []task.Task
	findings                   []task.Finding
	docs                       map[string]*task.Document
	blocked, linted            map[string]bool
	broken                     map[string]bool   // paths with a broken finding (pbh9xq)
	stages                     map[string]string // task.Stage by path, per load (46ek1f)
	progressCols               int               // progressWidth, per load (46ek1f)
	epics                      map[string]bool   // lowercase ids of tasks with children
	cursor, acceptance, offset int
	dependency, detailPane     int
	link                       int // the task:ID marked > in Links (xxvm6t)
	file                       int // the file marked > in Files (zgr01t)
	// glance: a file shown in the detail pane (scrxak); glanceHead is its
	// "path · size · i of N" line, glanceBody its escaped text.
	glance                 bool
	glanceHead, glanceBody string
	// captured is the frame S took (5z4zt4), waiting for Enter to attach
	// it; capturedNote says what went to the clipboard.
	captured, capturedNote string
	// pictures keeps each question picture as last read, by file, width
	// and mtime, so a redraw does not read it again (5z4zt4).
	pictures               map[string]pictureMemo
	files                  map[string][]string         // task path -> its file names (zgr01t)
	people                 []task.Person               // hug.toml [people]: short names, long names, emails (89y0b7)
	lastExec               *exec.Cmd                   // the editor command last started, for tests (zgr01t)
	lastEdit               editorMsg                   // and the message it reports when it closes
	memo                   [2][maxSections]sectionMemo // overview and board: each section's scroll and row (h9zghj)
	sort, sortDraft        []sortKey                   // the order rows are in, and the one being built (mkzcqr)
	detailStack            []string
	width, height          int
	view                   string
	pane                   int
	detail, help, pendingG bool
	// column is the column the cursor marks (bejd89), an index into
	// gridColumns; -1 until Tab marks one. pendingV: v was pressed, a
	// view letter follows (v o, v i, v b, v !).
	column                             int
	pendingV                           bool
	helpTyping                         bool   // typing a help search after "/"
	helpQuery                          string // filters help rows by key or description
	prompt, query, message, detailPath string
	confirm, confirmPath, pendingValue string
	yanked                             string // y: the task id P and p paste (3xcesq)
	hidden                             string // with no row selected (cursor -1): the task kept, which a filter hides (mcvfbz)
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
	// closedFindings are advice findings on done or dropped tasks, hidden
	// from the header count, the lint view and the marks (6neax1, pbh9xq).
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
	// Reload into a new hug (j2vwk0): bin watches the started path; newer
	// is the newer hug installed there, shown in the corner with R;
	// reloadTo is its version once R quits the TUI to exec it.
	bin             *binWatch
	newer, reloadTo string
	// closes reads close times from git for RECENTLY DONE (zaec08);
	// closedAt is the last snapshot's copy.
	closes   *closeLog
	closedAt map[string]int64
	// Opening at once (vnyz8z): ctx ends with the TUI and cancels a running
	// refresh; refreshSince is when the running load began (the spinner
	// shows after spinAfter); spinning says a spinner tick is due.
	ctx          context.Context
	cancel       context.CancelFunc
	refreshSince time.Time
	spinning     bool
	pickPane     bool
	refreshed    bool // the start's refresh has landed
	// jumpPending: Enter on a # jump found nothing on the start's board;
	// the jump is tried again when the refresh lands.
	jumpPending bool
}

// firstLoad says the board shown is still the start's snapshot: the first
// refresh replaces it even under an open prompt (a # jump typed before it
// landed must find the tasks; vnyz8z), but not under the editor or a grab.
func (m model) firstLoad() bool {
	return m.opts.Initial != nil && !m.refreshed && !m.editing && !m.grab.active
}

// firstPane prefers an occupied overview pane, keeping the pane order.
func (m *model) firstPane() {
	for p := 0; p < len(overviewSections); p++ {
		m.pane = p
		if len(m.indices()) > 0 {
			break
		}
	}
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
	m.stale, m.loading, m.refreshSince = false, true, time.Now()
	s, loads, gen, closes, lint := m.store, m.loads, m.gen, m.closes, m.opts.Lint
	return func() tea.Msg { return loadedMsg{loadSnapshot(s, loads, closes, lint), gen} }
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
	none := !m.detail && m.cursor == -1 // no row selected stays so (mcvfbz)
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
	if none {
		m.cursor = -1
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
	// A load running: the spinner ticks until it ends (vnyz8z).
	if spin := m.spin(); spin != nil {
		cmd = tea.Batch(cmd, spin)
	}
	return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case changedMsg:
		cmd := m.reload()
		return m, tea.Batch(cmd, m.waitChange(), m.spin())
	case binaryMsg:
		return m, tea.Batch(m.probe(false), m.waitBinary())
	case probedMsg:
		cmd := m.probed(msg)
		return m, cmd
	case spinMsg:
		m.spinning = false
		if strings.HasPrefix(m.message, readingBoard) {
			m.message = m.readingMessage()
		}
		return m, m.spin()
	case watchStartedMsg:
		if msg.w != nil {
			m.watch = msg.w
			// A change made before the watch was up has no event: read
			// the board once more (np2kk2). A load still running is
			// followed by another, as for any change during a load.
			if m.opts.Initial != nil {
				cmd := m.reload()
				return m, tea.Batch(cmd, m.waitChange(), m.spin())
			}
			return m, m.waitChange()
		}
		return m, nil
	case loadedMsg:
		m.loading, m.refreshSince = false, time.Time{}
		if strings.HasPrefix(m.message, readingBoard) {
			m.message = ""
		}
		switch {
		case msg.gen != m.gen:
			// An action reloaded meanwhile, after this load began: the
			// screen already shows newer data.
		case m.busy() && !m.pickPane && !m.firstLoad():
			m.stale = true // a prompt opened meanwhile: wait for it
		default:
			m.applyKeeping(msg.sn)
			m.refreshed = true
			if m.pickPane && msg.sn.err == nil {
				m.pickPane = false
				m.firstPane()
				m.clamp()
			}
			if m.jumpPending && m.prompt == "#" {
				if m.message == "no matching id or alias" {
					m.message = ""
				}
				next, cmd := m.updateJump(tea.KeyMsg{Type: tea.KeyEnter})
				return next.(model).afterUpdate(cmd)
			}
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
	return newModelCtx(context.Background(), s, o)
}

func newModelCtx(ctx context.Context, s *task.Store, o Options) model {
	input := textinput.New()
	input.CharLimit = 512
	input.Cursor.Blink = false
	m := model{store: s, opts: o, width: 100, height: 30, view: "overview", column: -1, input: input, closes: &closeLog{}, pictures: map[string]pictureMemo{},
		viewport: viewport.New(100, 24), helpViewport: viewport.New(100, 24), helpModel: help.New(), styles: newTheme()}
	m.ctx, m.cancel = context.WithCancel(ctx)
	if o.Initial != nil {
		// The board the cache last saw, drawn now; the refresh brings the
		// rest (vnyz8z).
		m.apply(initialSnapshot(o.Initial), "")
		m.loading, m.refreshSince = true, time.Now()
		if len(o.Initial.Tasks) == 0 {
			m.message = readingBoard
		}
	} else {
		m.refresh("")
	}
	m.firstPane()
	// A first start drew an empty board: the first real one picks the
	// pane again, as a start with the board in hand would have.
	m.pickPane = o.Initial != nil && len(o.Initial.Tasks) == 0
	m.clamp()
	m.resize()
	return m
}
func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.waitChange(), m.waitBinary()}
	if m.opts.Initial != nil {
		// After the first frame: the refresh, the spinner, then the watch.
		s, loads, gen, closes, lint, refresh, ctx := m.store, m.loads, m.gen, m.closes, m.opts.Lint, m.opts.Refresh, m.ctx
		cmds = append(cmds, func() tea.Msg {
			if refresh != nil {
				if e := refresh(ctx); e != nil {
					if ctx.Err() != nil {
						return nil // the TUI ended
					}
					return loadedMsg{snapshot{err: e}, gen}
				}
			}
			return loadedMsg{loadSnapshot(s, loads, closes, lint), gen}
		}, tea.Tick(spinEvery, func(time.Time) tea.Msg { return spinMsg{} }))
		// (Init cannot mark the model spinning: spinMsg's first tick
		// re-arms through spin, which then sees loading.)
		if m.watch == nil {
			cmds = append(cmds, m.startWatching())
		}
	}
	return tea.Batch(cmds...)
}
func (m *model) refresh(path string) {
	m.apply(loadSnapshot(m.store, m.loads, m.closes, m.opts.Lint), path)
}

// snapshot is everything refresh reads from disk: the tasks, their parsed
// files and the lint findings. loadSnapshot builds it without touching the
// model, so a reload after a change on disk runs off the UI thread (nfv18p).
type snapshot struct {
	tasks            []task.Task
	docs             map[string]*task.Document
	findings, closed []task.Finding
	closedAt         map[string]int64 // close times from git (closeLog)
	files            map[string][]string
	people           []task.Person // hug.toml [people] (89y0b7)
	err, readErr     error
}

func loadSnapshot(s *task.Store, loads *loadCounter, closes *closeLog, lint func() ([]task.Finding, error)) snapshot {
	loads.add()
	ts, err := s.Tasks()
	if err != nil {
		return snapshot{err: err}
	}
	task.Sort(ts)
	sn := snapshot{tasks: ts, docs: map[string]*task.Document{}, closedAt: closes.times(s.Root), files: map[string][]string{}}
	sn.people, _ = task.People(s.Root) // a broken hug.toml is lint's to report
	// Task folders, from one ReadDir per group instead of a WalkDir per
	// task (44w1ev: 10,000 walks of folders that mostly do not exist were
	// half the first screen). Keys are lowercase: taskFiles decides.
	folders := map[string]bool{}
	listed := map[string]bool{}
	for _, t := range ts {
		g := filepath.Dir(filepath.FromSlash(t.Path))
		if listed[g] {
			continue
		}
		listed[g] = true
		entries, _ := os.ReadDir(filepath.Join(s.Root, g))
		for _, e := range entries {
			if e.IsDir() {
				folders[strings.ToLower(filepath.ToSlash(filepath.Join(g, e.Name())))] = true
			}
		}
	}
	for _, t := range ts {
		if !folders[strings.ToLower(task.FolderOf(t))] {
			continue
		}
		if fs := taskFiles(s.Root, t); len(fs) > 0 {
			sn.files[t.Path] = fs
		}
	}
	var findings []task.Finding
	if lint != nil {
		if findings, err = lint(); err != nil {
			return snapshot{err: err}
		}
	} else {
		var docs []*task.Document
		for _, t := range ts {
			b, e := sharefile.ReadFile(filepath.Join(s.Root, t.Path))
			if e != nil {
				sn.readErr = e
				continue
			}
			d := task.Parse(t.Path, b)
			docs = append(docs, d)
			sn.docs[t.Path] = d
		}
		findings = task.Lint(docs)
	}
	// As in hug task lint: broken findings show on every task; advice on
	// closed tasks (done, dropped) is hidden; closedFindings keeps it for
	// counts and for the detail of a closed task opened on purpose (6neax1,
	// pbh9xq).
	sn.findings, sn.closed = task.ShownFindings(findings, ts)
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
	m.files = sn.files
	m.people = sn.people
	m.blocked = map[string]bool{}
	m.linted = map[string]bool{}
	m.broken = map[string]bool{}
	m.epics = map[string]bool{}
	for id := range task.ChildIndex(ts) {
		m.epics[id] = true
	}
	for _, t := range ts {
		m.blocked[t.Path] = t.State != "done" && t.State != "dropped" && task.Waiting(t, ts)
	}
	m.computeStages()
	m.progressCols = progressWidthOf(ts)
	for _, f := range m.findings {
		m.linted[f.Path] = true
		if f.Severity() == task.Broken {
			m.broken[f.Path] = true
		}
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
	// The index's ranking of the query (49n9kw): its hits are kept even when
	// the substring rule misses them (a word in the notes), and come first.
	var rank map[string]int
	if m.query != "" && m.opts.Search != nil {
		if paths, e := m.opts.Search(m.query); e == nil {
			rank = make(map[string]int, len(paths))
			for i, p := range paths {
				rank[p] = i + 1
			}
		}
	}
	for _, t := range m.all {
		status := m.filter.State
		// One place per task (zaec08): "depends" is todo with an open
		// dependency, "todo" excludes it, "doing" keeps doing tasks that depend.
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
		if m.query != "" && rank[t.Path] == 0 && !searchMatches(t, m.query) {
			continue
		}
		m.visible = append(m.visible, t)
	}
	if rank != nil {
		// Best match first; substring-only matches keep board order after.
		sort.SliceStable(m.visible, func(i, j int) bool {
			ri, rj := rank[m.visible[i].Path], rank[m.visible[j].Path]
			if (ri == 0) != (rj == 0) {
				return rj == 0
			}
			return ri < rj
		})
	}
	m.sortVisible() // the order built with : sort (mkzcqr) wins; rank or relevance order without one
	m.offset = 0
	m.cursor = 0
	m.clamp()
}

var boardStates = []string{"todo", "doing", "depends", "done", "dropped"}

// Overview sections, top to bottom. Every task is in one place (zaec08).
// IN PROGRESS is what someone works on now (doing/active); WAITING is what
// is stuck on someone or something, open tasks waiting, in design or in
// review, todo or doing alike, and doing tasks with an open dependency
// (tje4sx, Jørgen 19:54, crab 20:10: reverses "waiting is a stage, not a
// section"). A todo task with an open dependency stays in NEXT UP with
// the stage depends: it is not ready yet, and nobody is asked anything.
var overviewSections = []string{"IN PROGRESS · assigned to", "WAITING · on you first", "NEXT UP · rank order", "RECENTLY DONE · last closed", "LINT · tasks needing attention"}

const (
	sectionDoing = iota
	sectionWaiting
	sectionNext
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
	memo.start, _ = m.window(indices, m.cursor, memo.start, m.sectionRows(m.pane), m.cursor)
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
// are; a todo task with an open dependency is depends (6ephqk, was
// blocked); anything else (a malformed status) is todo, still visible and
// repairable, marked with !.
func (m model) place(t task.Task) string {
	switch t.State {
	case "done", "dropped", "doing":
		return t.State
	}
	if m.blocked[t.Path] {
		return "depends"
	}
	return "todo"
}

// boardState is the status a detail shows: its place, except that a doing
// task waiting on a dependency is drawn depends where it stands.
func (m model) boardState(t task.Task) string {
	if t.State == "doing" && m.blocked[t.Path] {
		return "depends"
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
				match = t.State == "doing" && !m.waits(t)
			case sectionWaiting:
				match = m.waits(t)
			case sectionNext:
				// Unassigned todo, never an epic (it finishes through its
				// children); one that depends shows its stage (Jørgen 11:43).
				match = t.State == "todo" && !m.waits(t) && !m.epics[strings.ToLower(t.ID)] && len(t.Assignee) == 0
			case sectionLint:
				match = m.linted[t.Path]
			}
		case "board":
			match = m.place(t) == boardStates[p]
		case "lint":
			match = m.linted[t.Path]
		}
		if match {
			out = append(out, i)
		}
	}
	if m.view == "overview" && p == sectionWaiting {
		// A task waiting on you sorts to the top of its section (6ephqk
		// check 8, Jørgen 11:42), the rest keep their order.
		sort.SliceStable(out, func(a, b int) bool { return m.forYou(m.visible[out[a]]) && !m.forYou(m.visible[out[b]]) })
	}
	return out
}
func (m model) indices() []int { return m.paneIndices(m.pane) }

// waits says whether t goes to WAITING (tje4sx): an open task whose stage
// waits on a person or a thing (waiting, design, review), or a doing task
// held by an open dependency. A todo task that depends is not ready yet
// and stays in NEXT UP.
func (m model) waits(t task.Task) bool {
	if t.State != "todo" && t.State != "doing" {
		return false
	}
	switch m.stageOf(t) {
	case "waiting", "design", "review":
		return true
	case "depends":
		return t.State == "doing"
	}
	return false
}
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
	m.glance = false
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
	pos := -1
	for p, i := range indices {
		if i == m.cursor {
			pos = p
			break
		}
	}
	if pos < 0 && !absolute {
		// Nothing selected (the task kept after a search is hidden,
		// mcvfbz): a step selects the first row.
		pos, n = 0, 0
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
		m.pane = (m.pane + delta + len(boardStates)) % len(boardStates)
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
	if m.glance {
		// The file stays as it was read: a refresh does not read it again.
		if t, ok := m.selected(); ok {
			m.viewport.Height = max(1, m.bodyHeight()-len(m.detailHeader(t))-2)
		}
		return
	}
	if t, ok := m.selected(); ok {
		m.viewport.Height = max(1, m.bodyHeight()-len(m.detailHeader(t))-1)
		m.viewport.SetContent(m.detailText(t))
	}
}
func (m *model) openInput(prompt, value string) tea.Cmd {
	m.prompt = prompt
	m.pendingG = false
	m.jumpPending = false
	m.completion = 0
	m.input.Prompt = prompt + " "
	m.input.SetValue(value)
	m.input.CursorEnd()
	m.input.PromptStyle = m.styles.accent
	m.input.TextStyle = m.styles.normal
	return m.input.Focus()
}

// clearFilters removes the search and every filter; the selected task
// stays selected (mcvfbz).
func (m *model) clearFilters() { m.clearKeeping(false, m.dropFilters) }
func (m *model) dropFilters() {
	m.query = ""
	m.filter = task.Filter{}
	m.chips = nil
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
		if m.confirm == captureKey {
			// Esc keeps only the copy (5z4zt4): nothing is attached.
			m.message = m.capturedNote + " · not attached"
			m.captured, m.capturedNote = "", ""
		}
		m.confirm = ""
		m.pendingValue = ""
	case m.editNotice != "" && m.prompt == "":
		m.editNotice = ""
	case m.prompt == "filter" && m.pickerField != "":
		m.pickerField = ""
		m.input.SetValue("")
		m.completion = 0
	case m.prompt == "/":
		m.leaveSearch(false) // Esc clears the search, keeping the task (mcvfbz)
	case m.prompt != "":
		m.prompt = ""
		m.input.Blur()
	case m.detail && m.glance:
		// Up from a file is Files, with that file marked (scrxak).
		m.closeGlance()
	case m.detail && len(m.detailStack) > 0:
		m.detailPath = m.detailStack[len(m.detailStack)-1]
		m.detailStack = m.detailStack[:len(m.detailStack)-1]
		m.startFocus()
		m.syncDetail()
		m.openTop()
	case m.detail:
		t, ok := m.taskAt(m.detailPath)
		m.detail = false
		m.detailPath = ""
		if ok && !m.follow(t.ID) {
			// Opened with # while a filter or the view hides it: no
			// other task is selected in its place (mcvfbz).
			m.hide(t)
		}
	case m.query != "" || m.filter != (task.Filter{}):
		m.clearKeeping(true, m.dropFilters)
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
	case imageShown:
		// Back from an image (5z4zt4): Files, the image still marked.
		if msg.err != nil {
			m.message = msg.err.Error()
		}
		return m, tea.ClearScreen
	case editorMsg:
		m.refresh(msg.path)
		m.editNotice = ""
		if msg.err != nil {
			m.message = msg.err.Error()
		} else {
			m.message = "editor closed; re-linted"
			if m.opts.Commit != nil {
				subject := msg.commit
				if subject == "" {
					subject = "hug task edit"
				}
				// The turn covers the commit, not the editor, as hug task edit.
				release := m.hold()
				err := m.opts.Commit(subject)
				release()
				if err != nil {
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
		// S takes the screen as you see it (5z4zt4): copied to the
		// clipboard at once, then the bar asks to attach it to the task.
		if k == captureKey {
			m.capture()
			return m, nil
		}
		// R reloads into the newer hug in the corner (j2vwk0); without one
		// R is not bound. Prompts, help and confirms took their keys above.
		if k == reloadKey && m.newer != "" {
			m.message = "reloading into hug " + m.newer + " …"
			return m, m.probe(true)
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
		// v is the view branch of the header tree (0sw0dd, bejd89): v o
		// overview, v i (or v v, v l) list, v b board, v ! lint. o alone
		// orders by the marked column; i, b and ! stay one key.
		if m.pendingV {
			m.pendingV = false
			if view, ok := map[string]string{"o": "overview", "i": "list", "v": "list", "l": "list", "b": "board", "!": "lint"}[k]; ok {
				m.detail = false
				m.setView(view)
				m.message = ""
				return m, nil
			}
			m.message = ""
		} else if k == "v" {
			m.pendingV = true
			m.message = "view: o overview · i list · b board · ! lint"
			return m, nil
		}
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
			m.setFilter("state", []string{"todo", "doing", "done", "depends"}[int(k[0]-'1')])
			m.setView("list")
			return m, nil
		}
		if m.detail && m.glance {
			// In a file: n/p step files, j/k and pages scroll, Enter edits
			// it, E opens the folder, Esc and u go back (scrxak).
			switch k {
			case "n", "ctrl+n":
				m.stepGlance(1)
			case "p", "ctrl+p":
				m.stepGlance(-1)
			case "u":
				m.back()
			case "enter":
				return m, m.openFile()
			case "E":
				return m, m.openFolder()
			case " ":
				m.message = "a file is read only here: Enter opens it in the editor"
			default:
				m.scroll(&m.viewport, k)
			}
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
				case focusFiles:
					if t, ok := m.selected(); ok {
						m.file = max(0, min(m.file+step, len(m.filesOf(t))-1))
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
				case focusFiles:
					return m, m.openFile()
				}
			case "E":
				// The task's whole files folder, in the editor (zgr01t).
				return m, m.openFolder()
			case " ":
				// Space ticks the check marked >; with focus elsewhere no
				// check is marked, and ticking an unmarked one is a guess.
				// In Files it shows the marked file (scrxak).
				if m.detailPane == focusFiles {
					if t, ok := m.selected(); ok {
						if name, _, _ := m.glanceFile(t); task.IsImageName(name) {
							// An image is drawn outside the frame where the
							// terminal can (5z4zt4), else one line.
							return m, m.showImage(t, name)
						}
					}
					m.openGlance()
				} else if m.detailPane == focusAcceptance {
					m.requestAction(k, "")
				} else if t, ok := m.selected(); ok && len(t.Acceptance) > 0 {
					m.message = "Space ticks the check marked >: Tab focuses the checks"
				} else {
					m.message = "no acceptance checks to tick"
				}
			case "-":
				// - strikes the check marked > out as not relevant, asking for
				// the reason; on a struck check it asks to restore it (sfa8w4).
				return m, m.strikeKey()
			case "a":
				// a answers the question the task asks you, typed right under
				// it (3jtfw9).
				return m, m.answerKey()
			case "J", "K":
				// In a task J and K scroll half a page (x1ra0t, Jørgen
				// 14:49): Shift reads as more than j/k. Moving by rank is a
				// row action, where the rows are seen to move.
				if k == "J" {
					m.viewport.HalfViewDown()
				} else {
					m.viewport.HalfViewUp()
				}
			case "t", "d", "x", "e":
				return m, m.startAction(k)
			case "/", ":", "#":
				return m, m.openInput(k, "")
			case "f":
				return m, m.openFilter()
			case "N":
				return m, m.openInput("new title", "")
			case "s":
				return m, m.openInput(":", "state ")
			case "y":
				return m, m.openInput(":", "type ")
			case "r":
				if t, ok := m.selected(); ok {
					m.refresh(t.Path)
				}
			case "u":
				// u goes up one level, as Esc does in a task: back from a
				// followed link, else to the list it was opened from
				// (9xfnwr, Jørgen 2026-11-16). In : sort u is the
				// updated field; that prompt takes its keys before this.
				m.back()
			case "n", "p", "ctrl+n", "ctrl+p":
				// Next or previous task of the list the detail was opened
				// from, without going back first. n and p, not { and }, which
				// are hard to reach on a Norwegian keyboard (2026-11-16, user).
				step := 1
				if k == "p" || k == "ctrl+p" {
					step = -1
				}
				m.stepTask(step)
			case "i", "b", "!":
				// The tab bar stays visible in detail, so its keys work here
				// too: leave the task and switch view (2026-11-15, user).
				// The overview is v o (bejd89).
				m.detail = false
				m.setView(map[string]string{"i": "list", "b": "board", "!": "lint"}[k])
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
				} else if ok && m.detailPane == focusFiles {
					if next := m.file + step; next >= 0 && next < len(m.filesOf(t)) {
						m.file, moved = next, true
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
		case k == "tab":
			m.moveColumn(1) // the column cursor (bejd89); h/l change section
		case k == "shift+tab":
			m.moveColumn(-1)
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
			return m, m.openInput(":", "state ")
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
			m.orderByColumn() // bejd89: the overview is v o
		case k == "i":
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
			return m, m.startAction(k)
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
	if m.prompt == "/" && m.updateSearch(k) {
		return m, nil
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
		if m.prompt == strikePrompt {
			reason := strings.TrimSpace(m.input.Value())
			m.prompt = ""
			m.input.Blur()
			if reason == "" {
				m.message = "not struck out: a strike needs a reason (- again, then type why it no longer applies)"
				return m, nil
			}
			m.strike(reason)
			return m, nil
		}
		if m.prompt == answerPrompt {
			text := strings.TrimSpace(m.input.Value())
			m.prompt = ""
			m.input.Blur()
			if text == "" {
				m.message = "not answered: the answer is empty (a again, then type it)"
				return m, nil
			}
			m.answer(text)
			return m, nil
		}
		if m.prompt == dropPrompt {
			if why := strings.TrimSpace(m.input.Value()); why != "" {
				m.requestAction("x", why)
			} else {
				m.message = "not dropped: a drop needs a reason"
			}
		}
		m.prompt = ""
		m.input.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.completion = 0
	if m.prompt == "/" {
		m.searchTyped()
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

// searchMatches is / search: the query, case-insensitively, in the slug,
// title, id, assignee or an alias (vzghj8: imported ids such as
// STATBUS-402, which hug task show accepts too).
func searchMatches(t task.Task, query string) bool {
	q := strings.ToLower(query)
	return textMatches(t, q) || matchedAlias(t, q) != ""
}

func textMatches(t task.Task, q string) bool {
	return strings.Contains(strings.ToLower(t.Slug+" "+t.Title+" "+t.ID+" "+strings.Join(t.Assignee, " ")), q)
}

// matchedAlias is the first alias that contains query (lowercase), or "".
func matchedAlias(t task.Task, query string) string {
	if query == "" {
		return ""
	}
	for _, a := range t.Aliases {
		if strings.Contains(strings.ToLower(a), query) {
			return a
		}
	}
	return ""
}
