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
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
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
	// HistoryFile keeps the / and : prompt histories (xfv9pw), per board,
	// outside the task branch; "" keeps them for this session only.
	HistoryFile string
	// Getenv reads the environment (nil: os.Getenv): which terminal draws
	// images, tmux (5z4zt4). Tests choose a terminal with it.
	Getenv func(string) string
	// MainBranch is the repository's main line ("main"): IN PROGRESS rows
	// name a task's branch only when it is another one (x1t1t1).
	MainBranch string
	// Checkout is the top folder of the checkout hug tui runs in: its
	// last element names the project in the header, 'HUG · hug-testing'
	// (0sw0dd, Jørgen 2026-11-16), unless hug.toml [board] project says.
	Checkout string
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
	// Deliver, when set, hands a person's request or answer just written
	// on id (kind "request" or "answer") to the board's coordinator and
	// returns its receipt (gmzd87): the same function hug task request and
	// hug task answer call. It runs inside the write's turn, after its
	// commit.
	Deliver func(id, kind, by string) (string, error)
	// Run, when set, runs a hug task command (args after "hug task") as
	// the CLI and MCP do, its board turn and commit included, and returns
	// what it says in one line (05r3pg: the palette's task actions that
	// have no key of their own).
	Run func(args []string) (string, error)
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
	// ids are the marked tasks edited together (rxr4z8): unmarked once
	// the editor closes.
	ids []string
}
type model struct {
	store           *task.Store
	opts            Options
	all, visible    []task.Task
	findings        []task.Finding
	docs            map[string]*task.Document
	blocked, linted map[string]bool
	broken          map[string]bool   // paths with a broken finding (pbh9xq)
	stages          map[string]string // task.Stage by path, per load (46ek1f)
	visibleGen      int               // bumped on every write of visible (46ek1f)
	panes           *paneMemo         // paneIndices by visibleGen and view (46ek1f)
	progressCols    int               // progressWidth, per load (46ek1f)
	epics           map[string]bool   // lowercase ids of tasks with children
	epicOf          map[string]string // task path -> its epic's id, per load (kx2ndm)
	epicTitle       map[string]string // lowercase epic id -> its title (kx2ndm)
	// grouping is what rows are grouped by: "", "group" or "epic"
	// (kx2ndm); folded the headings folded, by foldKey; head the heading
	// under the cursor ("" for a task row; cursor is -1 then).
	grouping, head             string
	folded                     map[string]bool
	cursor, acceptance, offset int
	dependency, detailPane     int
	link                       int // the task:ID marked > in Links (xxvm6t)
	idAt                       int // with focusIDs: the task id marked, by place in idStops (4bwcgp)
	file                       int // the file marked > in Files (zgr01t)
	// question is the question block marked > (focusQuestion), by place
	// among the questions that name no check (q2adr8); answerN is the
	// number of the question the open answer box answers, 0 for none.
	question, answerN int
	// fold is the long section whose heading has the focus (focusFold),
	// by place among folds; unfolded the long sections opened on this
	// page, by key (8a0ybb). A new page opens with them all folded.
	fold     int
	unfolded map[string]bool
	// glance: a file shown in the detail pane (scrxak); glanceHead is its
	// "path · size · i of N" line, glanceBody its escaped text.
	glance                 bool
	glanceHead, glanceBody string
	glanceMD               glanceMarkdown
	glanceImage            *glanceImage // the image shown, drawn over the frame (2qndsd)
	imageWasDrawn          *glanceImage // what the last frame drew: eraseImage
	viewWasDrawn           picView      // and the crop of it (33v71x)
	glanceGen              uint64
	// glanceReturn is where Esc goes from a file Enter showed from a
	// question's link line (qcq3tm): back to that question, not Files.
	glanceReturn *glanceReturn
	// picZoom is the zoom step of the picture shown (33v71x), picX, picY
	// the corner of its crop, in pixels; 0, 0, 0 is fit.
	picZoom, picX, picY int
	// captured is the frame S took (5z4zt4), waiting for Enter to attach
	// it; capturedNote says what went to the clipboard.
	captured, capturedNote string
	// picking: s in the capture prompt chose a row for it (hjbfzd);
	// pickFrom is where S was pressed, to come back to. captureStamp is
	// the HHMM the capture was taken at; captureTyped says the reader has
	// typed in its prompt, so s and c are text.
	picking         bool
	pickFrom        pickReturn
	captureStamp    string
	captureTyped    bool
	captureKind     string                      // "" for S's screen, clipKind for V's paste (q7n2zd)
	captureExt      string                      // the capture file's extension: .ans for a screen
	hist            *promptHistory              // / and : history (xfv9pw, history.go)
	files           map[string][]string         // task path -> its file names (zgr01t)
	people          []task.Person               // hug.toml [people]: short names, long names, emails (89y0b7)
	project         string                      // hug.toml [board] project, "" when unset (0sw0dd)
	tagOrder        map[string]int              // hug.toml tags: tag -> its place on the list, for grouping by tag (txjkrm)
	lastExec        *exec.Cmd                   // the editor command last started, for tests (zgr01t)
	lastEdit        editorMsg                   // and the message it reports when it closes
	memo            [2][maxSections]sectionMemo // overview and board: each section's scroll and row (h9zghj)
	sort, sortDraft []sortKey                   // the order rows are in, and the one being built (mkzcqr)
	sortMoved       bool                        // j/k moved the sort picker's highlight since the last pick: Enter picks it (17w9r6)
	detailStack     []string
	width, height   int
	view            string
	pane            int
	detail, help    bool
	// column is the column the cursor marks (bejd89), an index into
	// gridColumns; -1 until Tab marks one. pendingV: v was pressed, a
	// view letter follows (v o, v i, v b, v !).
	column                             int
	pendingV                           bool
	pulse                              pulse  // the focus sign's pulse (q3sh69, pulse.go)
	helpTyping                         bool   // typing a help search after "/"
	helpQuery                          string // filters help rows by key or description
	prompt, query, message, detailPath string
	// messageAt is the screen the message was set on (messagePlace):
	// moving to another task, file or view clears it (z82qsy, tz9z14).
	messageAt                          string
	confirm, confirmPath, pendingValue string
	// pendingArgs is the hug task command a run confirm asks about (05r3pg).
	pendingArgs []string
	yanked      string // y: the task id P and p paste (3xcesq)
	// marks are the task ids Space marked in a list, in the order
	// marked; the footer acts on them (rxr4z8).
	marks []string
	// bulk is the marked ids a confirm acts on (rxr4z8), fixed when it
	// was asked.
	bulk []string
	// requestIDs are the marked ids the open request box writes one
	// request for (ec6z6j check 5), fixed when r opened it.
	requestIDs []string
	hidden     string // with no row selected (cursor -1): the task kept, which a filter hides (mcvfbz)
	// results: Enter left the search's matches on screen, not a kept
	// filter (5cjy88, search.go); searchFrom is where / was pressed, for
	// Esc while typing; searchPicked says ↑↓ picked a match, so ↑ picks.
	results                bool
	searchFrom             *searchReturn
	searchPicked           bool
	slot                   *slotGap // where the open task stood in its pane before it left it (24403z)
	movedTo                string   // "<ID> moved to <SECTION>", the last write's note, which a receipt follows (24403z)
	grab                   grabState
	find                   findState // / in a task or a file: matches in its text (4s3g5q)
	filter                 task.Filter
	input                  textinput.Model
	columnHinted           bool           // H/L or o last: the footer says what o does, on rows (qtj177)
	answerBox              textarea.Model // the answer under a question: lines of its own (41hhgn)
	vCursor                cursor.Model   // the blink of o l b !'s underlines while v waits (a82j3n)
	viewport, helpViewport viewport.Model
	helpModel              help.Model
	completion             int
	chips                  []filterChip
	pickerField            string
	pickerNot              bool // ! in the f picker: the value picked is negated (7zg9mf)
	// defaultOn: the list still opens on its default chip, 'state !done'
	// (7zg9mf), until F or ⌫ clears it or the reader sets a state
	// filter; defaultShown: that chip is up now, in the list.
	defaultOn, defaultShown bool
	styles                  theme
	// editNotice is the task file whose edit introduced lint findings: the
	// notice shows the first one with its fix, and e reopens the editor,
	// until Esc, another action, or the file is clean (rx74bp).
	editNotice string
	// editBefore holds the problems the task had when the editor opened.
	editBefore map[string]bool
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

// textInput says whether keys type text in the prompt open here (under
// help too): every prompt but the sort builder and the filter's field
// chooser, whose keys are letters to pick (t64v7d). In text every
// printable key is text; help is F1.
func (m model) textInput() bool {
	switch {
	case m.confirm != "" || m.prompt == "" || m.prompt == "sort":
		return false
	case m.prompt == "filter":
		return m.pickerField != ""
	}
	return true
}

// helpKey is the key that opens help here: F1 in text, else ?.
func (m model) helpKey() string {
	if m.textInput() {
		return "F1"
	}
	return "?"
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
	head := m.head                                    // a heading under the cursor stays so (kx2ndm)
	none := !m.detail && m.cursor == -1 && head == "" // no row selected stays so (mcvfbz)
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
	if head != "" && !m.detail {
		m.head, m.cursor = head, -1
		m.clamp() // the heading, if the new rows still have it
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
	if format := m.formatGlance(); format != nil {
		cmd = tea.Batch(cmd, format)
	}
	// A load running: the spinner ticks until it ends (vnyz8z).
	if spin := m.spin(); spin != nil {
		cmd = tea.Batch(cmd, spin)
	}
	if erase := m.eraseImage(); erase != nil {
		cmd = tea.Batch(cmd, erase)
	}
	return m, cmd
}

// Update is one step, after which a message left from another screen
// goes (clearStaleMessage, tz9z14).
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// A message set outside a step (Run's "reloaded hug") belongs to the
	// screen it first shows on.
	if m.message != "" && m.messageAt == "" {
		m.messageAt = m.messagePlace()
	}
	before := m.message
	height := m.viewHeight()
	next, cmd := m.step(msg)
	if n, ok := next.(model); ok {
		n.clearStaleMessage(before)
		// The focus sign pulses where no text cursor blinks (q3sh69).
		if pulse := n.afterPulse(msg); pulse != nil {
			cmd = tea.Batch(cmd, pulse)
		}
		if n.detail && n.glance {
			n.fitGlance() // the file has the rows no message takes (yp3c09)
		}
		if n.viewHeight() != height && n.detail && !n.glance {
			// The page is as tall as the palette leaves it (05r3pg): it
			// shrinks when the palette opens or grows, and grows back
			// when it closes, so its rows are where they were.
			n.syncDetail()
			if n.prompt == ":" {
				n.revealAcceptance()
			}
		}
		if n.detail && !n.glance {
			n.fitPage() // a prompt or confirm takes a row from the page (4ddcwa)
		}
		return n, cmd
	}
	return next, cmd
}

func (m model) step(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pulseMsg:
		return m.onPulse(msg)
	case glanceFormattedMsg:
		m.glanceFormatted(msg)
		return m.afterUpdate(nil)
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
			closes := m.readCloses()
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
				return next.(model).afterUpdate(tea.Batch(cmd, closes))
			}
			return m.afterUpdate(closes)
		}
		return m.afterUpdate(nil)
	case closesMsg:
		m.applyCloses()
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
	// Wherever you can type, the cursor blinks (q3sh69, Jørgen's rule;
	// qpp30y had only the box's move): the focus sign is still then.
	input.Cursor.SetMode(cursor.CursorBlink)
	m := model{store: s, opts: o, width: 100, height: 30, view: "overview", column: -1, input: input, answerBox: newAnswerBox(), closes: &closeLog{}, panes: &paneMemo{forYou: -1}, defaultOn: true,
		viewport: viewport.New(100, 24), helpViewport: viewport.New(100, 24), helpModel: help.New(), styles: newTheme()}
	styleAnswerCursor(&m.answerBox, m.styles.cursor)
	styleInputCursor(&m.input, m.styles.cursor)
	m.vCursor = cursor.New() // blinks at bubbles' 530 ms, as the box's
	m.ctx, m.cancel = context.WithCancel(ctx)
	if o.Initial != nil {
		// The board the cache last saw, drawn now; the refresh brings the
		// rest (vnyz8z).
		m.apply(initialSnapshot(o.Initial), "")
		// One small file, so the first frame names the project as every
		// later one does (0sw0dd).
		m.project, _ = task.ProjectName(s.Root)
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
	// After the TUI's own write, which committed with git already: the
	// close times are read here, not after (a reload's are, adqdfe).
	m.closes.times(m.store.Root)
	m.apply(loadSnapshot(m.store, m.loads, m.closes, m.opts.Lint), path)
}

// snapshot is everything refresh reads from disk: the tasks, their parsed
// files and the lint findings. loadSnapshot builds it without touching the
// model, so a reload after a change on disk runs off the UI thread (nfv18p).
type snapshot struct {
	tasks        []task.Task
	docs         map[string]*task.Document
	findings     []task.Finding
	closedAt     map[string]int64 // close times from git (closeLog)
	files        map[string][]string
	people       []task.Person // hug.toml [people] (89y0b7)
	project      string        // hug.toml [board] project (0sw0dd)
	tags         []string      // hug.toml tags, in their order (txjkrm)
	err, readErr error
}

func loadSnapshot(s *task.Store, loads *loadCounter, closes *closeLog, lint func() ([]task.Finding, error)) snapshot {
	loads.add()
	ts, err := s.Tasks()
	if err != nil {
		return snapshot{err: err}
	}
	task.Sort(ts)
	// The close times known so far: git is asked after the board is drawn
	// (readCloses), so a load never waits on starting a process (adqdfe).
	sn := snapshot{tasks: ts, docs: map[string]*task.Document{}, closedAt: closes.latest(), files: map[string][]string{}}
	sn.people, _ = task.People(s.Root) // a broken hug.toml is lint's to report
	sn.project, _ = task.ProjectName(s.Root)
	if tl, e := task.Tags(s.Root); e == nil {
		sn.tags = tl.List
	}
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
	// As in hug task lint: open tasks only, broken first (t01kqm).
	sn.findings = task.Judged(findings, ts)
	return sn
}

// apply shows a snapshot, keeping the task at path selected.
func (m *model) apply(sn snapshot, path string) {
	m.gen++
	slotID, slotRows, slotAt := m.slotBefore(sn) // read before the rows are replaced (24403z)
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
	m.findings = sn.findings
	m.closedAt = sn.closedAt
	m.files = sn.files
	m.people = sn.people
	m.project = sn.project
	m.tagOrder = map[string]int{}
	for i, tg := range sn.tags {
		if _, ok := m.tagOrder[tg]; !ok {
			m.tagOrder[tg] = i
		}
	}
	m.blocked = map[string]bool{}
	m.linted = map[string]bool{}
	m.broken = map[string]bool{}
	m.epics = map[string]bool{}
	m.epicOf = map[string]string{}
	m.epicTitle = map[string]string{}
	byID := make(map[string]task.Task, len(ts))
	for _, t := range ts {
		byID[strings.ToLower(t.ID)] = t
	}
	// One pass resolves every parent (kx2ndm): rows read epicOf and never
	// resolve their parent themselves (ls on 2,498 tasks: 19 s to 0.8 s).
	for id, children := range task.ChildIndex(ts) {
		m.epics[id] = true
		m.epicTitle[id] = oneLine(byID[id].Title)
		for _, c := range children {
			m.epicOf[c.Path] = byID[id].ID
		}
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
	m.keepSlot(slotID, slotRows, slotAt)
	m.syncDetail()
}
func (m *model) applyFilter() {
	m.visibleGen = nextVisibleGen()
	m.visible = nil
	// The index's ranking of the query (49n9kw): its hits are kept even when
	// the substring rule misses them (a word in the notes), and come first.
	var rank map[string]int
	terms, words := parseSearch(m.query)
	if words != "" && m.opts.Search != nil {
		if paths, e := m.opts.Search(words); e == nil {
			rank = make(map[string]int, len(paths))
			for i, p := range paths {
				rank[p] = i + 1
			}
		}
	}
	for _, t := range m.all {
		// One place per task (zaec08): "depends" is todo with an open
		// dependency, "todo" excludes it, "doing" keeps doing tasks that
		// depend. A value with ! first keeps all but those (7zg9mf).
		if !keeps(m.filter.State, func(v string) bool { return m.place(t) == v }) || !keeps(m.filter.Type, func(v string) bool { return optionalMatches(t.Type, v) }) || !keeps(m.filter.Group, func(v string) bool { return groupMatches(t.Group, v) }) {
			continue
		}
		if !keeps(m.filter.Label, func(v string) bool { return contains(t.Labels, v) }) { // the tag filter (txjkrm)
			continue
		}
		if !keeps(m.filter.Epic, func(v string) bool { return task.InEpic(t, m.all, v) }) {
			continue
		}
		if !keeps(m.filter.Priority, func(v string) bool { return optionalMatches(t.Priority, v) }) || !keeps(m.filter.Mine, func(v string) bool { return ownerMatches(t, v) }) {
			continue
		}
		if words != "" && rank[t.Path] == 0 && !searchMatches(t, words) {
			continue
		}
		// state:done type:bug in / (0sw0dd check 3, searchfields.go).
		if len(terms) > 0 && !m.fieldsMatch(t, terms) {
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
	m.visibleGen = nextVisibleGen() // sorted after the scan above
	m.sortVisible()                 // the order built with : sort (mkzcqr) wins; rank or relevance order without one
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
//
// ASKED is at the top (41hhgn check 10, Jørgen 00:12: "a separate section
// that I really need to address"): every task whose stage asks the viewer
// (forYou: asked, a design question or a review naming them), oldest
// first. WAITING keeps the waits with no person and what is asked of
// others ("a place to live for things that aren't active right now").
var overviewSections = []string{"ASKED · for you", "IN PROGRESS · assigned to", "WAITING", "NEXT UP · rank order", "RECENTLY DONE · last closed", "LINT · tasks needing attention"}

const (
	sectionAsked = iota
	sectionDoing
	sectionWaiting
	sectionNext
	sectionRecent
	sectionLint
)

// ownerSection says whether overview section p draws the owner cell:
// ASKED, IN PROGRESS and WAITING hold tasks someone has (102bc0, tje4sx).
func ownerSection(p int) bool { return p == sectionAsked || p == sectionDoing || p == sectionWaiting }

// recentDone is how many closed tasks RECENTLY DONE shows while the cursor
// is in another section; focused, it scrolls through all of them (h9zghj).
// Every other unfocused section shows as many rows.
const recentDone = 3

// maxSections is the most sections a view has: 6 overview sections, 5
// board columns.
const maxSections = 6

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
	memo.start, _ = m.window(m.paneRows(m.pane), m.rowCursor(m.pane), memo.start, m.sectionRows(m.pane), m.cursor)
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

// paneIndices are the rows of section p of the current view, as indices
// into visible. They depend only on the rows shown (visible, which a load,
// a filter, a sort or a grab replaces) and the view, so they are found
// once and kept (46ek1f: a key and its frame asked for them dozens of
// times, each a scan of every task: 52 ms a key on Seid). The memo is
// keyed by visibleGen, which every write of visible bumps.
func (m model) paneIndices(p int) []int {
	if m.grouping != "" {
		return m.grouped(p).tasks // folded tasks have no row (kx2ndm)
	}
	return m.rawPane(p)
}

// rawPane is section p's tasks in their order, before grouping.
func (m model) rawPane(p int) []int {
	if m.panes == nil {
		return m.scanPane(p)
	}
	return m.panes.get(m.visibleGen, m.view, p, m.scanPane)
}

// scanPane finds section p's rows by looking at every shown task.
func (m model) scanPane(p int) []int {
	if m.view == "overview" && p == sectionRecent {
		return m.recentIndices()
	}
	var out []int
	taskScans.Add(int64(len(m.visible)))
	for i, t := range m.visible {
		match := true
		switch m.view {
		case "overview":
			match = m.inSection(t, p) // one rule for the rows and placeName (24403z)
		case "board":
			match = m.place(t) == boardStates[p]
		case "lint":
			match = m.linted[t.Path]
		}
		if match {
			out = append(out, i)
		}
	}
	if m.view == "overview" && p == sectionAsked {
		// Oldest question first: it has waited longest (41hhgn check 10).
		sort.SliceStable(out, func(a, b int) bool {
			return askedSince(m.visible[out[a]]) < askedSince(m.visible[out[b]])
		})
	}
	if m.view == "overview" && p == sectionLint || m.view == "lint" {
		// What is wrong comes first (z6cb03): broken tasks before advice,
		// rank (or the chosen order) within each, as hug task lint lists
		// them. A section cut to its rows keeps the broken ones in view.
		sort.SliceStable(out, func(a, b int) bool {
			return m.broken[m.visible[out[a]].Path] && !m.broken[m.visible[out[b]].Path]
		})
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
	case "waiting", "asked", "design", "review":
		return true
	case "depends":
		return t.State == "doing"
	}
	return false
}
func (m *model) clamp() {
	if m.head != "" {
		// A task set as the cursor wins over the heading it replaced.
		if m.cursor < 0 && m.headUnder(m.pane) != -1 {
			m.cursor = -1 // on a heading (kx2ndm)
			return
		}
		m.head = ""
	}
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
	if m.resultsShown() {
		// A view switch ends the results; the picked task stays selected
		// where the new view shows it (5cjy88). A Tab chip stays.
		t, ok := m.rowTask()
		m.dropSearch()
		m.applyFilter()
		defer func() {
			if ok {
				m.follow(t.ID)
			}
		}()
	}
	m.view = view
	m.pane = 0
	m.offset = 0
	m.detail = false
	m.glance = false
	m.find = findState{}
	m.detailPath = ""
	m.detailStack = nil
	m.detailPane = 0
	m.head = ""
	if view == "overview" {
		m.firstPane() // ASKED is first and often empty (41hhgn)
	}
	m.showListDefault() // the list's 'state !done', up in the list only (7zg9mf)
	m.clamp()
}

// stepTask moves the open detail to the next or previous task in the pane it
// was opened from. It stops at either end.
func (m *model) stepTask(step int) {
	if m.stepFromGap(step) {
		return // the open task left the pane: from its slot (24403z)
	}
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
	if m.query != "" && memoView(m.view) >= 0 {
		m.walk(n, absolute) // every match, across sections (5cjy88)
		return
	}
	if m.grouping != "" && !m.detail {
		m.moveRows(n, absolute) // headings are rows too (kx2ndm)
		return
	}
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
		m.head = ""
		m.pane = (m.pane + delta + len(overviewSections)) % len(overviewSections)
		m.restore()
	case "board":
		m.remember()
		m.head = ""
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
	m.fitAnswerBox()
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
		m.resizeGlanceMarkdown()
		// The file stays as it was read: a refresh does not read it again.
		m.viewport.Height = m.glanceHeight()
		return
	}
	if t, ok := m.selected(); ok {
		// viewHeight: the palette takes the bottom rows (05r3pg), the page
		// above is shorter, so the check it acts on stays in sight.
		m.viewport.Height = m.pageHeight(t) // between the page's rules (4ddcwa)
		m.viewport.SetContent(m.detailText(t))
		if m.find.text != "" {
			m.refind() // the page drawn again keeps its matches (4s3g5q)
		}
	}
}
func (m *model) openInput(prompt, value string) tea.Cmd {
	m.bulk = nil // a prompt for the marked sets it after (rxr4z8)
	m.prompt = prompt
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
	m.results = false
	m.filter = task.Filter{}
	m.chips = nil
	if m.defaultShown || m.view == "list" {
		m.retireDefault() // F in the list clears its default for the session (7zg9mf)
	}
}
func (m *model) back() {
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
		m.pendingArgs = nil
		m.bulk = nil // the marks stay (rxr4z8)
	case m.prompt == captureTitlePrompt:
		// Esc on the new task's title goes back to where (hjbfzd).
		m.input.Blur()
		m.openCapture("")
	case m.prompt == capturePrompt:
		// Esc keeps only the copy (5z4zt4): nothing is attached.
		m.prompt = ""
		m.input.Blur()
		m.dropCapture()
	case m.picking && m.prompt == "":
		// Esc while picking: nothing attached, back where S was pressed. A
		// find or jump opened while picking closes first.
		m.endPick()
		m.dropCapture()
	case m.editNotice != "" && m.prompt == "":
		m.editNotice = ""
	case m.prompt == "filter" && m.pickerField != "":
		m.pickerField = ""
		m.input.SetValue("")
		m.completion = 0
	case m.prompt == ":" && strings.TrimSpace(m.input.Value()) != "":
		// Esc backs out one level of the tree, then closes (05r3pg).
		m.input.SetValue(paletteUp(m.input.Value()))
		m.input.CursorEnd()
		m.completion = 0
	case m.prompt == "/":
		m.cancelSearch() // Esc while typing: back where / was pressed (5cjy88)
	case m.prompt == findPrompt || (m.detail && m.prompt == "" && m.find.text != ""):
		// Esc clears the find and its tints; the page stays where it is
		// (4s3g5q check 4). The next Esc goes back.
		m.prompt = ""
		m.input.Blur()
		m.clearFind()
		m.message = ""
	case m.prompt != "":
		m.prompt = ""
		m.requestIDs = nil // the marks stay (ec6z6j check 5)
		m.input.Blur()
		m.answerBox.Blur()
		m.syncDetail() // a closed answer box gives its lines back
	case m.detail && m.glance:
		// Up from a file is Files, with that file marked (scrxak), or
		// the question whose link showed it (qcq3tm).
		m.closeGlance()
	case m.detail && len(m.detailStack) > 0:
		m.detailPath = m.detailStack[len(m.detailStack)-1]
		m.detailStack = m.detailStack[:len(m.detailStack)-1]
		m.startFocus()
		m.syncDetail()
		m.openTop()
	case m.detail && m.gap() != nil:
		// The open task left the pane: back on the row after its slot,
		// in the pane the reader came from (24403z), not in its new one.
		m.slot = nil
		m.detail = false
		m.find = findState{}
		m.detailPath = ""
		m.clamp()
	case m.detail:
		t, ok := m.taskAt(m.detailPath)
		m.detail = false
		m.find = findState{}
		m.detailPath = ""
		if ok && !m.follow(t.ID) {
			// Opened with # while a filter or the view hides it: no
			// other task is selected in its place (mcvfbz).
			m.hide(t)
		}
	case m.resultsShown():
		m.leaveResults(true) // all tasks, the picked one kept (5cjy88, mcvfbz)
	case m.view != "overview":
		// A chip is kept until F or ⌫ (5cjy88, crab 09:16Z): Esc goes up
		// a level with the chip line still drawn.
		m.setView("overview")
	}
}
func (m model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Bubble Tea coalesces fast typing, including gg, into rune batches.
	// A paste into a text field is text, whole: its line ends stay one
	// each (41hhgn check 8), and no letter in it runs as a key.
	if event, ok := msg.(tea.KeyMsg); ok && event.Type == tea.KeyRunes && len(event.Runes) > 1 && !(event.Paste && m.textInput() && !m.help) {
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
	case cursor.BlinkMsg:
		// The answer or request box's cursor blinks (qpp30y), the
		// one-line prompt's (q3sh69), and the underlines of o l b !
		// while v waits (a82j3n); the focus sign pulses only while none
		// of them does (pulse.go). They never show at once: v in a box
		// is text.
		if m.pendingV {
			var cmd tea.Cmd
			m.vCursor, cmd = m.vCursor.Update(msg)
			return m, cmd
		}
		m.vCursor.Blur() // v was answered: its last tick ends the blink
		if m.textInput() && !m.boxPrompt() {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, m.blinkBox(msg)
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
			if len(msg.ids) > 0 {
				m.unmark(msg.ids)
			}
			if m.opts.Commit != nil {
				subject := msg.commit
				if subject == "" {
					subject = "hug task edit"
				}
				if len(msg.ids) > 1 {
					subject = "hug task edit " + strings.Join(msg.ids, " ")
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
		// The column hint answers H/L and o; any other key ends it (qtj177).
		if k != "H" && k != "L" && k != "o" {
			m.columnHinted = false
		}
		// In text every printable key is text, ? too (t64v7d, Jørgen
		// 2026-11-18 01:52): there help is F1. F1 is help everywhere, and
		// leaves the input intact underneath.
		if k == "f1" || (k == "?" && ((m.help && !m.helpTyping) || (!m.help && !m.textInput()))) {
			m.help = !m.help
			m.pendingV = false // help answers the key, not the prefix
			m.vCursor.Blur()
			m.helpTyping = false
			m.helpQuery = ""
			m.helpViewport.SetContent(m.helpText())
			m.helpViewport.GotoTop()
			return m, nil
		}
		if k == "esc" {
			if m.pendingV && !m.help {
				// Esc cancels the prefix, and only that (qh9ep1 check 3).
				m.pendingV = false
				m.vCursor.Blur()
				m.message = ""
				return m, nil
			}
			if m.grab.active && !m.help {
				m.cancelGrab()
				return m, nil
			}
			if m.yanked != "" && !m.help && m.prompt == "" && m.confirm == "" {
				m.message = "forgot " + m.yanked + "; nothing changed"
				m.yanked = ""
				return m, nil
			}
			if len(m.marks) > 0 && !m.detail && !m.help && m.prompt == "" && m.confirm == "" {
				// Esc unmarks every mark first (rxr4z8 check 2).
				m.message = fmt.Sprintf("unmarked %d; nothing changed", len(m.marks))
				m.marks = nil
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
				return m, nil
			}
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
			if k == "enter" && m.confirm == runKey {
				args, path := m.pendingArgs, m.confirmPath
				m.confirm, m.confirmPath, m.pendingArgs = "", "", nil
				m.runArgs(args, path)
				return m, nil
			}
			if k == "enter" {
				if len(m.bulk) > 0 {
					return m, m.runBulk() // the marked tasks (rxr4z8)
				}
				cmd := m.executeAction()
				return m, cmd
			}
			return m, nil
		}
		if m.prompt != "" {
			return m.updateInput(msg)
		}
		// v is the view prefix (qh9ep1, 0sw0dd's key trie): vo, vl, vb
		// and v! pick a view, from a list or a task. It waits for the next
		// key, no timeout; that key picks, or cancels and does nothing
		// else (Esc, an unknown key). No single key switches views.
		if m.pendingV {
			m.pendingV = false
			m.vCursor.Blur() // the blink goes with the prefix (a82j3n)
			if view, ok := pickView(k); ok {
				m.detail = false
				m.setView(view)
				m.message = ""
				return m, nil
			}
			m.message = "v " + oneLine(k) + " is no view; nothing changed"
			if k == "esc" {
				m.message = ""
			}
			return m, nil
		}
		if k == "v" {
			m.pendingV = true
			m.message = viewPrefix()
			return m, m.vCursor.Focus() // the underlines show, then blink
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
		// clipboard at once, then the input line asks where it goes
		// (hjbfzd). While a task is picked for a capture, S is no new one.
		if k == captureKey && !m.picking {
			return m, m.capture()
		}
		// V pastes the clipboard (q7n2zd): in a task it attaches there, in
		// a list it asks where as S does. While picking, it is a writer.
		if k == clipKey && !m.picking {
			return m, m.pasteClipboard()
		}
		// Picking a task for the capture (hjbfzd): Enter attaches, the
		// writers say so, every other key is the rows' own.
		if m.picking && m.pickKey(k) {
			return m, nil
		}
		// R reloads into the newer hug in the corner (j2vwk0); without one
		// R is not bound. Prompts, help and confirms took their keys above.
		if k == reloadKey && m.newer != "" {
			m.message = "reloading into hug " + m.newer + " …"
			return m, m.probe(true)
		}
		// g groups the rows: flat, by group, by epic (kx2ndm, Jørgen
		// 2026-11-17 23:37: gg and G go, "never used it, vi specific";
		// top and bottom are < > Home End).
		if k == "g" && !m.detail {
			m.setGrouping("next")
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
		if m.detail && m.glance {
			// A picture first (33v71x): + - 0 zoom, h/j/k/l scroll it, o
			// and Enter open it in the system viewer.
			if m.glancePicture() && m.find.text == "" {
				if k == "enter" {
					k = "o" // a picture is looked at, never edited (check 5)
				}
				if m.pictureKey(k) {
					return m, nil
				}
			}
			// In a file: n/p step files, j/k and pages scroll, Enter edits
			// it, E opens the folder, Esc and u go back (scrxak).
			switch k {
			case "/":
				// / finds in the file (4s3g5q); n / N step its matches
				// while a find is on, else n / p step files.
				if m.markdownLoading() {
					m.message = "formatting Markdown · M shows raw source"
					break
				}
				m.openFind()
				m.message = m.findSays()
			case "M":
				m.toggleGlanceMarkdown()
			case "n", "N":
				if m.find.text != "" {
					m.stepHit(map[string]int{"n": 1, "N": -1}[k])
				} else if k == "n" {
					m.stepGlance(1)
				}
			case "ctrl+n":
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
					m.message = "[ and ] step through a list: l focuses the checks or dependencies"
				}
				m.clamp()
				m.syncDetail()
				m.revealAcceptance()
			case "h", "l", "left", "right", "tab", "shift+tab":
				// One focus ring, in reading order: the header's ids, the
				// page, the checks, each dependency, child and link, the
				// files, the ids in the Notes; Tab is l and Shift-Tab is h,
				// as in the rows (e3sevr, Jørgen 23:28).
				step := 1
				if k == "h" || k == "left" || k == "shift+tab" {
					step = -1
				}
				m.cycleFocus(step)
				m.syncDetail()
				m.revealAcceptance()
			case "enter":
				// On a question that names a file, Enter shows the file in
				// the file frame; Esc comes back to the question (qcq3tm).
				if m.showQuestionFile() {
					break
				}
				switch m.detailPane {
				case focusDeps:
					m.followDependency()
				case focusLinks:
					m.followLink()
				case focusIDs:
					m.followID()
				case focusFold:
					// Enter on a long section's heading opens it or folds it (8a0ybb).
					m.toggleSection()
				case focusFiles:
					if m.markedPicture() {
						// A picture opens in the system viewer, never the
						// editor (33v71x check 5, Jørgen on 911j70).
						m.openPicture()
						break
					}
					return m, m.openFile()
				}
			case "o":
				// o opens a picture marked in Files in the viewer, as in
				// its frame (33v71x); elsewhere in a task it does nothing.
				if m.detailPane == focusFiles && m.markedPicture() {
					m.openPicture()
				}
			case "E":
				// The task's whole files folder, in the editor (zgr01t).
				return m, m.openFolder()
			case " ":
				// Space ticks the check marked >; with focus elsewhere no
				// check is marked, and ticking an unmarked one is a guess.
				// In Files it shows the marked file (scrxak), an image too,
				// in the same frame (2qndsd).
				if m.detailPane == focusFiles {
					m.openGlance()
				} else if m.detailPane == focusAcceptance {
					m.requestAction(k, "")
				} else if m.detailPane == focusQuestion {
					// A question names no check to tick: a answers it.
					m.message = "a question has nothing to tick: a answers it"
				} else if t, ok := m.selected(); ok && len(t.Acceptance) > 0 {
					m.message = "Space ticks the check marked >: l focuses the checks"
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
			case "/":
				// / finds in this task's text (4s3g5q), never tasks: a
				// task's / is the page's, as in less.
				m.openFind()
				m.message = m.findSays()
			case ":", "#":
				return m, m.openInput(k, "")
			case "f":
				return m, m.openFilter()
			case "N":
				if m.find.text != "" {
					m.stepHit(-1) // N steps back through the matches while a find is on
					return m, nil
				}
				return m, m.openInput("new title", "")
			case "s":
				m.openSort() // sorts the list it came from, as f filters it (ahejza)
				return m, nil
			case "y":
				// y yanks the task, as on its row (e3sevr: the task's
				// actions are its row's); the type palette is T.
				m.yank()
			case "T":
				return m, m.openInput(":", "type ")
			case "w":
				return m, m.waitKey()
			case "r":
				// r writes a request to the agents (hbgybn, crab 01:30);
				// the board updates live, so refresh needs no key.
				return m, m.requestKey()
			case "u":
				// u goes up one level, as Esc does in a task: back from a
				// followed link, else to the view it was opened from
				// (9xfnwr, Jørgen 2026-11-16). In : sort u is the
				// updated field; that prompt takes its keys before this.
				// u goes up at once with a find on: only Esc clears it
				// first (4s3g5q check 7).
				m.find = findState{}
				m.back()
			case "n", "p", "ctrl+n", "ctrl+p":
				if k == "n" && m.find.text != "" {
					m.stepHit(1) // n is the next match while a find is on (4s3g5q)
					return m, nil
				}
				// Next or previous task of the list the detail was opened
				// from, without going back first. n and p, not { and }, which
				// are hard to reach on a Norwegian keyboard (2026-11-16, user).
				step := 1
				if k == "p" || k == "ctrl+p" {
					step = -1
				}
				m.stepTask(step)
			case "j", "down", "k", "up":
				// With a list focused, j/k step through it and scroll the
				// page past either end; with the page focused they scroll.
				step := 1
				if k == "k" || k == "up" {
					step = -1
				}
				t, ok := m.selected()
				moved := false
				// j and k mirror each other (ykta5g, Jørgen 19:55): j past the
				// last item scrolls the page with it still marked, so while
				// the marked item starts above the view k scrolls back a line
				// and keeps it; only once it is in view does k step. j, the
				// same way, reads a marked item taller than the view to its
				// last line before it steps (check 4).
				if s, e, ok := m.markedBlock(); ok && (step < 0 && s < m.viewport.YOffset || step > 0 && e >= m.viewport.YOffset+m.viewport.Height) {
					m.scroll(&m.viewport, k)
					return m, nil
				}
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
					// Scroll only as far as the new item needs, whole, its
					// question and all (ykta5g): never a re-centre.
					m.showMarked()
				} else {
					m.scroll(&m.viewport, k)
				}
			default:
				m.scroll(&m.viewport, k)
			}
			return m, nil
		}
		switch {
		case m.head != "" && (k == " " || k == "enter"):
			// Space or Enter on a heading folds or unfolds it (kx2ndm).
			m.toggleFold()
		case m.head != "" && contains([]string{"t", "d", "x", "e", "m", "y", "P", "p"}, k):
			// A heading is not a task: nothing to act on, and no guess.
			m.message = "a heading is not a task: Space folds it, j/k select a task"
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
		case k == "pgdown" || k == "J":
			// J/K page (e3sevr, Jørgen 23:20): Shift moves further. Moving
			// a task in rank is a grab (m), never a cursor key.
			m.move(m.bodyHeight(), false)
		case k == "pgup" || k == "K":
			m.move(-m.bodyHeight(), false)
		case k == "L":
			m.moveColumn(1) // the column cursor (bejd89, e3sevr)
		case k == "H":
			m.moveColumn(-1)
		case bkey.Matches(msg, keys.prev) || k == "shift+tab":
			m.changePane(-1) // Tab is l, Shift-Tab is h, in every view (e3sevr)
		case bkey.Matches(msg, keys.next) || k == "tab":
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
			return m, m.openSearch(m.query) // refines a search shown (5cjy88)
		case k == ":":
			return m, m.openInput(":", "")
		case k == "s":
			// s opens the sort picker (ahejza): the filter line's s, as
			// f is its filter; the state palette is ": state".
			m.openSort()
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
		case k == "m" && len(m.marks) > 0:
			return m, m.markedKey(k) // the marked as one block (rxr4z8 check 4)
		case k == "m":
			if m.resultsShown() {
				// Moved among its real neighbours, not the other
				// matches (5cjy88, crab 09:30Z).
				m.leaveResults(false)
			}
			m.startGrab()
		case k == "#":
			return m, m.openInput("#", "")
		case k == "f":
			return m, m.openFilter()
		case k == "c":
			m.clearFilters()
		case k == "o":
			m.orderByColumn() // the overview is vo (qh9ep1)
		case k == "N":
			// N, not n: n and p step like j and k, and to the next task
			// inside one (2026-11-16, user).
			return m, m.openInput("new title", "")
		case k == "r" && len(m.marks) > 0:
			return m, m.markedKey(k) // one request for the marked (ec6z6j check 5)
		case k == "r":
			return m, m.requestKey() // a request on the selected task (hbgybn)
		case k == " ":
			// Space marks the task and steps on (rxr4z8, Jørgen 23:42);
			// inside a task it ticks the focused check, and on a heading
			// it folds (above).
			m.toggleMark()
		case len(m.marks) > 0 && contains([]string{"t", "d", "x", "w", "e"}, k):
			return m, m.markedKey(k)
		case contains([]string{"t", "d", "x", "e"}, k):
			return m, m.startAction(k)
		}
	}
	return m, nil
}
func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.boxPrompt() {
		return m.updateAnswer(msg)
	}
	if m.prompt == capturePrompt || m.prompt == captureTitlePrompt {
		return m.updateCapture(msg)
	}
	if m.prompt == "sort" {
		return m.updateSort(msg)
	}
	if m.prompt == "filter" {
		return m.updateFilter(msg)
	}
	if m.prompt == "#" {
		return m.updateJump(msg)
	}
	if !(m.prompt == "/" && m.searchPicked) && m.historyKey(k) { // Up/Down recall, Enter records (xfv9pw)
		return m, nil
	}
	if m.prompt == "/" && m.updateSearch(k) {
		return m, nil
	}
	if m.prompt == findPrompt {
		if next, done := m.updateFind(k); done {
			return next, nil
		}
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
			// The context groups first (05r3pg): "edit" is this task's
			// edit, run as its key runs it, or asked in the confirm line.
			if value := m.input.Value(); m.contextReady(value) {
				m.prompt = ""
				m.input.Blur()
				m.input.SetValue("")
				cmd := m.runContext(value)
				m.syncDetail()
				return m, cmd
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
		if m.prompt == waitPrompt {
			on := strings.TrimSpace(m.input.Value())
			m.prompt = ""
			m.input.Blur()
			if on == "" {
				m.bulk = nil
				m.message = "not parked: a wait needs what it waits on (w again, then type it)"
				return m, nil
			}
			if len(m.bulk) > 0 {
				m.confirm, m.pendingValue = "w", on // one confirm for the marked (rxr4z8)
				return m, nil
			}
			m.wait(on)
			return m, nil
		}
		if m.prompt == dropPrompt {
			if why := strings.TrimSpace(m.input.Value()); why != "" && len(m.bulk) > 0 {
				m.confirm, m.pendingValue = "x", why
			} else if why != "" {
				m.requestAction("x", why)
			} else {
				m.bulk = nil
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
	if m.prompt == findPrompt {
		m.findTyped()
	}
	return m, cmd
}
func (m *model) scroll(v *viewport.Model, k string) {
	switch k {
	case "j", "down":
		v.ScrollDown(1)
	case "k", "up":
		v.ScrollUp(1)
	case ">", "end":
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

// visibleGens numbers every write of visible in the process: unique,
// so a model copy that refilters (valueCount) never shares a number with
// the model it came from.
var visibleGens atomic.Int64

func nextVisibleGen() int { return int(visibleGens.Add(1)) }

// paneMemo keeps paneIndices for one visibleGen (46ek1f). Model copies
// share it; Bubble Tea calls Update and View on one goroutine, and the
// mutex keeps a stray reader safe all the same.
type paneMemo struct {
	mu   sync.Mutex
	gen  int
	view string
	rows map[int][]int
	// forYou is forYouCount for forYouGen (any view), -1 for none yet.
	forYouGen, forYou int
	// groups is grouped by section, for groupGen and groupView (kx2ndm).
	groupGen  int
	groupView string
	groups    map[int]groupedRows
}

// count is forYouCount for gen, counted once.
func (c *paneMemo) count(gen int, scan func() int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.forYouGen != gen || c.forYou < 0 {
		c.forYouGen, c.forYou = gen, scan()
	}
	return c.forYou
}

// get is section p's rows for gen and view, scanned once.
func (c *paneMemo) get(gen int, view string, p int, scan func(int) []int) []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rows == nil || c.gen != gen || c.view != view {
		c.gen, c.view, c.rows = gen, view, map[int][]int{}
	}
	rows, ok := c.rows[p]
	if !ok {
		rows = scan(p)
		c.rows[p] = rows
	}
	// Clipped: a caller that appends gets its own copy, never the memo's.
	return slices.Clip(rows)
}

// groupsOf is section p's grouped rows for gen and view, built once
// (kx2ndm). A fold or a new grouping bumps visibleGen.
func (c *paneMemo) groupsOf(gen int, view string, p int, build func(int) groupedRows) groupedRows {
	c.mu.Lock()
	if c.groups == nil || c.groupGen != gen || c.groupView != view {
		c.groupGen, c.groupView, c.groups = gen, view, map[int]groupedRows{}
	}
	g, ok := c.groups[p]
	c.mu.Unlock()
	if ok {
		return g
	}
	g = build(p) // outside the lock: build reads rows through get
	c.mu.Lock()
	if c.groupGen == gen && c.groupView == view {
		c.groups[p] = g
	}
	c.mu.Unlock()
	return g
}
