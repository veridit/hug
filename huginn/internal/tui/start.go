package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/veridit/hug/huginn/internal/task"
)

// Opening at once (vnyz8z, Jørgen 2026-11-17: "during startup there is no
// spinner, it just hangs, and also during close down"). hug tui draws the
// board the task cache last saw (Options.Initial) before it reads anything,
// then refreshes in the background the way a change on disk does: the
// fresh board replaces the old one with the same task selected. The
// watcher is set up after the first frame. A load that runs past
// spinAfter shows a spinner in the header. Quitting never waits for a
// refresh or the watcher: Run cancels the refresh and closes the watches
// without waiting for them.

// Initial is the board as the cache last saw it: drawn before any read.
// Empty Tasks is a first start: the screen says it is reading the board.
type Initial struct {
	Tasks    []task.Task
	Findings []task.Finding
}

// spinAfter is how long a load may run before the header says so: shorter
// than that, a spinner would only flicker.
const spinAfter = 200 * time.Millisecond

// spinEvery is the spinner's frame time.
const spinEvery = 120 * time.Millisecond

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// readingBoard is the message of a first start, until the board arrives;
// readingMessage adds the count of files read.
const readingBoard = "reading the board: the first start builds the task cache"

func (m model) readingMessage() string {
	if m.opts.Progress == nil {
		return readingBoard
	}
	read, total := m.opts.Progress()
	if total == 0 {
		return readingBoard
	}
	return fmt.Sprintf("%s: %d of %d files", readingBoard, read, total)
}

type spinMsg struct{}

type watchStartedMsg struct{ w *watch }

// watchStarter sets up the watch on the task folders (a test replaces it).
var watchStarter = func(root string) *watch { return startWatch(root) }

// initialSnapshot is what the first frame shows: the cache's board and
// findings, without the files of tasks or close times (the refresh brings
// them, a moment later).
func initialSnapshot(in *Initial) snapshot {
	ts := append([]task.Task(nil), in.Tasks...)
	task.Sort(ts)
	sn := snapshot{tasks: ts, docs: map[string]*task.Document{}, files: map[string][]string{}}
	// The same judgement as every load, so the counts do not change when
	// the refresh lands.
	sn.findings = task.Judged(in.Findings, ts)
	return sn
}

// spin ticks the spinner while a load runs; one ticker at a time.
func (m *model) spin() tea.Cmd {
	if m.spinning || (!m.loading && !m.markdownLoading()) {
		return nil
	}
	m.spinning = true
	return tea.Tick(spinEvery, func(time.Time) tea.Msg { return spinMsg{} })
}

// spinner is the header's refresh mark: a frame and "refreshing" while a
// load has run longer than spinAfter, else nothing.
func (m model) spinner() string {
	if m.markdownLoading() {
		n := int(time.Since(m.glanceMD.since)/spinEvery) % len(spinFrames)
		return spinFrames[n] + " formatting Markdown"
	}
	if !m.loading || m.refreshSince.IsZero() || time.Since(m.refreshSince) < spinAfter {
		return ""
	}
	n := int(time.Since(m.refreshSince)/spinEvery) % len(spinFrames)
	return spinFrames[n] + " refreshing"
}

// startWatching sets up the watch off the UI thread, after the first
// frame. A watch that comes up after the TUI ended closes itself.
func (m model) startWatching() tea.Cmd {
	if m.opts.NoWatch {
		return nil
	}
	// The starter is read here, on the UI thread: the command may still
	// run after the TUI (and a test that replaced it) has ended.
	ctx, root, start := m.ctx, m.store.Root, watchStarter
	return func() tea.Msg {
		w := start(root)
		if w != nil && ctx != nil && ctx.Err() != nil {
			go w.close()
			return nil
		}
		return watchStartedMsg{w}
	}
}

// runProgram runs the TUI once and returns when it quits, without waiting
// for a refresh or the watches: the refresh's context is cancelled and the
// watches are closed in the background.
func runProgram(ctx context.Context, store *task.Store, opts Options, popts ...tea.ProgramOption) error {
	_, err := runOnce(ctx, startModel(ctx, store, opts), popts...)
	return err
}

func runOnce(ctx context.Context, m model, popts ...tea.ProgramOption) (model, error) {
	final, err := tea.NewProgram(m, append([]tea.ProgramOption{tea.WithContext(ctx)}, popts...)...).Run()
	done, ok := final.(model)
	if !ok {
		done = m
	}
	return done, err
}

// release cancels a running refresh and closes the watches without
// waiting for them: quitting returns the shell at once.
func (m model) release() {
	if m.cancel != nil {
		m.cancel()
	}
	if w := m.watch; w != nil {
		go w.close()
	}
	if b := m.bin; b != nil {
		go b.close()
	}
}

// terminalSize is stdout's size when it is a terminal (a test replaces it).
var terminalSize = func() (width, height int, ok bool) {
	if !term.IsTerminal(os.Stdout.Fd()) {
		return 0, 0, false
	}
	w, h, err := term.GetSize(os.Stdout.Fd())
	return w, h, err == nil && w > 0 && h > 0
}

// startModel is the model a TUI starts with. Its first frame is drawn
// before Bubble Tea's WindowSizeMsg arrives, so it takes the terminal's
// size here, read as Bubble Tea reads it (stdout): never the default
// 100x30 redrawn at the real size a moment later (vnyz8z; crab 17:16: a
// 140-column terminal showed a 100-column first frame, CI 44132567751).
func startModel(ctx context.Context, store *task.Store, opts Options) model {
	m := newModelCtx(ctx, store, opts)
	m.pulse.live = true // the focus sign pulses in the program (q3sh69)
	if w, h, ok := terminalSize(); ok {
		m.width, m.height = w, h
		m.resize()
	}
	return m
}
