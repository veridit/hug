package tui

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// vnyz8z: hug tui draws at once from the last snapshot, refreshes behind a
// spinner, and quits without waiting for a sync or the watcher.

// slowRefresh is a Refresh stub: it reports each call, then blocks until
// release is closed (or the context ends), then lists the store.
type slowRefresh struct {
	called  chan struct{}
	release chan struct{}
}

func newSlowRefresh() *slowRefresh {
	return &slowRefresh{called: make(chan struct{}, 8), release: make(chan struct{})}
}

func (r *slowRefresh) fn(s *task.Store) func(context.Context) error {
	return func(ctx context.Context) error {
		r.called <- struct{}{}
		select {
		case <-r.release:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}
}

// Check 1: the first frame is the last snapshot, with the spinner, before
// the refresh returns; the fresh board then replaces it and the selection
// stays on the same task.
func TestStartDrawsSnapshotThenRefreshes(t *testing.T) {
	m := fixture(t)
	old := m.all // the snapshot the cache had: Alpha and Beta
	beta := ""
	for _, tk := range old {
		if strings.HasPrefix(tk.Title, "Beta") {
			beta = tk.ID
		}
	}
	// Meanwhile a third task landed on disk; only the refresh sees it.
	if _, err := m.store.New(task.NewOptions{Title: "Gamma task", Goal: "G", Acceptance: []string{"c"}}); err != nil {
		t.Fatal(err)
	}
	r := newSlowRefresh()
	start := newModel(m.store, Options{User: "tester", NoWatch: true, Initial: &Initial{Tasks: old}, Refresh: r.fn(m.store)})
	start.setView("list")
	start.width, start.height = 120, 20
	start.resize()
	for i, tk := range start.visible {
		if tk.ID == beta {
			start.cursor = i
		}
	}
	if len(start.all) != 2 || !start.loading {
		t.Fatalf("first frame: %d tasks, refreshing %v; want the 2 of the snapshot, refreshing", len(start.all), start.loading)
	}
	// Past the 200 ms grace the header says so.
	start.refreshSince = time.Now().Add(-time.Second)
	if v := ansi.Strip(start.View()); !strings.Contains(v, "refreshing") || !strings.Contains(v, "Beta task") {
		t.Fatalf("first frame lacks the board or the spinner:\n%s", v)
	}
	// The refresh command blocks in the stub, off the UI thread.
	cmd := start.Init()
	msgs := make(chan tea.Msg, 4)
	go func() { msgs <- runBatch(cmd) }()
	select {
	case <-r.called:
	case <-time.After(5 * time.Second):
		t.Fatal("Init did not start the refresh")
	}
	close(r.release)
	var got tea.Msg
	select {
	case got = <-msgs:
	case <-time.After(5 * time.Second):
		t.Fatal("the refresh never delivered")
	}
	next, _ := start.Update(got)
	after := next.(model)
	if len(after.all) != 3 || after.loading {
		t.Fatalf("after the refresh: %d tasks, refreshing %v", len(after.all), after.loading)
	}
	if tk, ok := after.selected(); !ok || tk.ID != beta {
		t.Fatalf("selection moved: %v %v, want %s", tk.ID, ok, beta)
	}
	if v := ansi.Strip(after.View()); strings.Contains(v, "refreshing") || !strings.Contains(v, "Gamma task") {
		t.Fatalf("after the refresh:\n%s", v)
	}
}

// runBatch runs a command and, for a batch, its parts until one returns a
// message other than nil (the refresh; the others are watch waits that
// return nil without a watch).
func runBatch(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if b, ok := msg.(tea.BatchMsg); ok {
		out := make(chan tea.Msg, len(b))
		for _, c := range b {
			c := c
			go func() {
				if c == nil {
					return
				}
				if m := c(); m != nil {
					if _, tick := m.(spinMsg); !tick {
						out <- m
					}
				}
			}()
		}
		return <-out
	}
	return msg
}

// Check 2: q quits while the refresh is still blocked: Run returns, the
// stub's context is cancelled, nothing waits for it.
func TestQuitDoesNotWaitForRefresh(t *testing.T) {
	m := fixture(t)
	r := newSlowRefresh()
	in, w := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- runProgram(context.Background(), m.store, Options{User: "tester", NoWatch: true, NoReload: true, Initial: &Initial{Tasks: m.all}, Refresh: r.fn(m.store)},
			tea.WithInput(in), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	}()
	select {
	case <-r.called:
	case <-time.After(5 * time.Second):
		t.Fatal("the refresh never started")
	}
	start := time.Now()
	go w.Write([]byte("q")) // never blocks the test when nothing reads
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("q did not return while the refresh was blocked")
	}
	t.Logf("quit took %v with the refresh blocked", time.Since(start).Round(time.Millisecond))
	w.Close()
}

// Check 2, the watcher: q returns while the watch setup is still running.
func TestQuitDoesNotWaitForWatchSetup(t *testing.T) {
	m := fixture(t)
	block := make(chan struct{})
	defer close(block)
	old := watchStarter
	watchStarter = func(root string) *watch { <-block; return nil }
	defer func() { watchStarter = old }()
	in, w := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- runProgram(context.Background(), m.store, Options{User: "tester", NoReload: true, Initial: &Initial{Tasks: m.all}},
			tea.WithInput(in), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	}()
	time.Sleep(100 * time.Millisecond)
	go w.Write([]byte("q")) // never blocks the test when nothing reads
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("q did not return while the watch setup was blocked")
	}
	w.Close()
}

// Check 3: a first start without a cache shows the spinner and what it
// reads, never a blank screen.
func TestFirstStartWithoutCacheSaysSo(t *testing.T) {
	m := fixture(t)
	r := newSlowRefresh()
	start := newModel(m.store, Options{User: "tester", NoWatch: true, Initial: &Initial{}, Refresh: r.fn(m.store),
		Progress: func() (int, int) { return 1234, 3630 }})
	start.width, start.height = 100, 12
	start.resize()
	start.refreshSince = time.Now().Add(-time.Second)
	// A spinner tick counts the files the refresh has read.
	next, _ := start.Update(spinMsg{})
	start = next.(model)
	var b bytes.Buffer
	b.WriteString(ansi.Strip(start.View()))
	text := strings.Join(strings.Fields(strings.ReplaceAll(b.String(), "\n", " ⏎ ")), " ")
	if !strings.Contains(text, "refreshing") || !strings.Contains(text, "reading the board") || !strings.Contains(text, "1234 of 3630 files") {
		t.Fatalf("first start screen:\n%s", b.String())
	}
	golden(t, "testdata/rows/first-start-100.txt", "TestFirstStartWithoutCacheSaysSo", spinnerFree(b.String()))
}

// spinnerFree replaces the spinner frame, so a golden does not depend on
// which frame was drawn.
func spinnerFree(s string) string {
	for _, f := range spinFrames {
		s = strings.ReplaceAll(s, f, "*")
	}
	return s
}

// A # jump typed before the first board arrives opens its task when the
// board lands, instead of answering "no matching id" from the empty first
// frame (the tmux review test typed #aaaaaa Enter before the refresh).
func TestJumpTypedBeforeBoardOpensWhenItArrives(t *testing.T) {
	m := fixture(t)
	beta := ""
	for _, tk := range m.all {
		if strings.HasPrefix(tk.Title, "Beta") {
			beta = tk.ID
		}
	}
	r := newSlowRefresh()
	start := newModel(m.store, Options{User: "tester", NoWatch: true, Initial: &Initial{}, Refresh: r.fn(m.store)})
	start = press(start, "#", beta, "enter")
	if start.prompt != "#" || start.detail {
		t.Fatalf("before the board: prompt %q detail %v, want the jump still open", start.prompt, start.detail)
	}
	next, _ := start.Update(loadedMsg{loadSnapshot(start.store, start.loads, start.closes, start.opts.Lint), start.gen})
	after := next.(model)
	if tk, ok := after.taskAt(after.detailPath); !after.detail || !ok || tk.ID != beta || after.prompt != "" {
		t.Fatalf("after the board: detail %v on %q, prompt %q; want %s open", after.detail, after.detailPath, after.prompt, beta)
	}
	// Editing after Enter, even back to the same id, means a new question
	// that has not been sent: the board landing does not answer the old one.
	start2 := newModel(m.store, Options{User: "tester", NoWatch: true, Initial: &Initial{}, Refresh: r.fn(m.store)})
	start2 = press(start2, "#", beta, "enter", "backspace", beta[len(beta)-1:])
	next, _ = start2.Update(loadedMsg{loadSnapshot(start2.store, start2.loads, start2.closes, start2.opts.Lint), start2.gen})
	if after := next.(model); after.detail || after.prompt != "#" {
		t.Fatalf("edited jump opened: detail %v prompt %q", after.detail, after.prompt)
	}
}

// The first frame has the terminal's size, before any WindowSizeMsg: a
// 140x40 terminal never sees a 100-column frame first (vnyz8z).
func TestFirstFrameHasTerminalSize(t *testing.T) {
	m := fixture(t)
	old := terminalSize
	terminalSize = func() (int, int, bool) { return 140, 40, true }
	defer func() { terminalSize = old }()
	start := startModel(context.Background(), m.store, Options{User: "tester", NoWatch: true, Initial: &Initial{Tasks: m.all}})
	lines := strings.Split(ansi.Strip(start.View()), "\n")
	if len(lines) != 40 {
		t.Fatalf("first frame has %d lines, want 40", len(lines))
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w > 140 {
			t.Fatalf("line %d is %d wide", i+1, w)
		}
	}
	if w := ansi.StringWidth(lines[0]); w != 140 {
		t.Fatalf("first frame header is %d columns wide, want 140:\n%s", w, lines[0])
	}
}
