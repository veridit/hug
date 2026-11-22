package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/internal/fsnotify"
)

// gitBoard is a task store in a git repository on its own branch, like the
// hug/tasks worktree, with a watch started as Run starts it. commit commits
// every change as hug task commands do.
func gitBoard(t *testing.T, titles ...string) (model, func(msg string)) {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.invalid")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "hug/tasks")
	s := &task.Store{Root: root}
	for _, title := range titles {
		goal := "G"
		if strings.HasPrefix(title, "long:") {
			goal = strings.Repeat("A paragraph long enough to scroll the detail.\n\n", 40)
		}
		if _, err := s.New(task.NewOptions{Title: title, Goal: goal, Acceptance: []string{"C"}}); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(msg string) {
		t.Helper()
		git("add", "-A")
		git("commit", "-q", "--allow-empty", "-m", msg)
	}
	commit("seed")
	m := newModel(s, Options{User: "tester"})
	m.loads = &loadCounter{}
	m.width, m.height = 140, 40
	m.resize()
	m.watch = startWatch(root)
	t.Cleanup(m.watch.close)
	return m, commit
}

// otherProcess changes a task the way another terminal does: through the
// files and a commit, never through this model.
func otherProcess(t *testing.T, m model, id, note string, commit func(string)) {
	t.Helper()
	s := &task.Store{Root: m.store.Root}
	d, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.Root, d.Task.Path)
	b, _ := os.ReadFile(path)
	if err := task.AtomicWrite(path, []byte(strings.Replace(string(b), "## Notes\n", "## Notes\n"+note+"\n", 1))); err != nil {
		t.Fatal(err)
	}
	commit("hug task note " + id)
}

// settle feeds the model what Bubble Tea would: the watch signal, then the
// load it starts. It fails if no change arrives within the deadline.
func settle(t *testing.T, m model, within time.Duration) model {
	t.Helper()
	select {
	case <-m.watch.C:
	case <-time.After(within):
		t.Fatalf("no change signal within %v", within)
	}
	return feed(t, m, changedMsg{})
}

// feed runs msg and every reload command it returns, as Bubble Tea would,
// and returns the model once nothing is loading.
func feed(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()
	next, cmd := m.Update(msg)
	m = next.(model)
	for _, out := range drain(cmd, m.loading) {
		next, _ = m.Update(out)
		m = next.(model)
	}
	return m
}

// drain runs the commands Update returned, as Bubble Tea would, each in
// its own goroutine, and returns the load when loading says one was
// started. waitChange, which blocks on the watch, and the input cursor's
// blink, which sleeps, are left running: not part of this step. The load
// is awaited, not given a fixed slice of time: it runs git, which is slow
// to start on Windows under -race (4emme8).
func drain(cmd tea.Cmd, loading bool) []tea.Msg {
	if cmd == nil || !loading {
		return nil
	}
	out := make(chan tea.Msg, 64)
	var start func(tea.Cmd)
	start = func(c tea.Cmd) {
		go func() {
			msg := c()
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, b := range batch {
					if b != nil {
						start(b)
					}
				}
				return
			}
			out <- msg
		}()
	}
	start(cmd)
	deadline := time.After(10 * time.Second)
	for {
		select {
		case msg := <-out:
			if _, ok := msg.(loadedMsg); ok {
				return []tea.Msg{msg}
			}
		case <-deadline:
			return nil // the test then fails on what is not on screen
		}
	}
}

func screen(m model) string { return ansi.Strip(m.View()) }

func (m model) mustSelected(t *testing.T) task.Task {
	t.Helper()
	sel, ok := m.selected()
	if !ok {
		t.Fatal("nothing selected")
	}
	return sel
}

// Check 1 and 4: a task changed by another process appears in an open TUI
// without a key press, in the overview, the list and an open detail, with
// the selection and the detail's scroll kept.
func TestChangeOnDiskAppears(t *testing.T) {
	for _, view := range []string{"overview", "list", "detail"} {
		t.Run(view, func(t *testing.T) {
			m, commit := gitBoard(t, "Alpha", "long: Beta", "Gamma")
			ids := map[string]string{}
			for _, v := range m.all {
				ids[v.Title] = v.ID
			}
			switch view {
			case "overview":
				m.setView("overview")
			case "list":
				m.setView("list")
				m = press(m, "j") // select Beta
			case "detail":
				m.setView("list")
				m = press(m, "j", "enter", "ctrl+d")
				if m.viewport.YOffset == 0 {
					t.Fatal("the detail did not scroll; the scroll check would prove nothing")
				}
			}
			selected, _ := m.selected()
			viewport := m.viewport.YOffset
			// Another process renames Gamma and, in the detail case, adds a
			// note to the open task.
			s := &task.Store{Root: m.store.Root}
			if _, err := s.Set(ids["Gamma"], "title", "Gamma renamed elsewhere"); err != nil {
				t.Fatal(err)
			}
			if view == "detail" {
				otherProcess(t, m, selected.ID, "note written by another terminal", commit)
			} else {
				commit("hug task set")
			}
			start := time.Now()
			m = settle(t, m, 3*time.Second)
			took := time.Since(start)
			out := screen(m)
			want := "Gamma renamed elsewhere"
			if view == "detail" {
				// The note is below the scrolled window, in the Notes, a
				// long section the page folds (8a0ybb): opened, the content
				// holds it.
				out = m.withUnfolded("notes").detailText(m.mustSelected(t))
				want = "note written by another terminal"
			}
			if !strings.Contains(out, want) {
				t.Fatalf("%s: %q not on screen after %v:\n%s", view, want, took, out)
			}
			if now, _ := m.selected(); now.ID != selected.ID {
				t.Fatalf("%s: selection moved from %s to %s", view, selected.ID, now.ID)
			}
			if m.viewport.YOffset != viewport {
				t.Fatalf("%s: detail scroll moved %d -> %d", view, viewport, m.viewport.YOffset)
			}
			t.Logf("%s: change on screen %v after the commit", view, took)
		})
	}
}

// Check 1: the scroll of a long list stays where it was.
func TestChangeOnDiskKeepsListScroll(t *testing.T) {
	var titles []string
	for i := 0; i < 60; i++ {
		titles = append(titles, "Task "+string(rune('A'+i%26))+string(rune('a'+i/26)))
	}
	m, commit := gitBoard(t, titles...)
	m.setView("list")
	m = press(m, ">", "k", "k")
	selected, _ := m.selected()
	before := strings.Split(screen(m), "\n")
	s := &task.Store{Root: m.store.Root}
	if _, err := s.Set(m.all[0].ID, "title", "First renamed"); err != nil {
		t.Fatal(err)
	}
	commit("hug task set")
	m = settle(t, m, 3*time.Second)
	after := strings.Split(screen(m), "\n")
	if now, _ := m.selected(); now.ID != selected.ID {
		t.Fatalf("selection moved from %s to %s", selected.ID, now.ID)
	}
	// The rows on screen are the same rows: the first row of the window did
	// not jump back to the top.
	if before[5] != after[5] {
		t.Fatalf("list scrolled:\n%s\n%s", before[5], after[5])
	}
}

// Check 2: an open prompt, palette or confirm is never interrupted; the
// reload happens once it closes. The editor is covered by the same flag.
func TestChangeOnDiskWaitsForPrompt(t *testing.T) {
	for _, open := range []struct {
		name string
		keys []string
	}{
		{"search prompt", []string{"/", "Al"}},
		{"palette", []string{":"}}, // one Esc closes it; from "state " Esc goes up a level (05r3pg)
		{"confirm", []string{"t"}},
	} {
		t.Run(open.name, func(t *testing.T) {
			m, commit := gitBoard(t, "Alpha", "Beta")
			m.setView("list")
			m = press(m, open.keys...)
			typed, prompt, confirm := m.input.Value(), m.prompt, m.confirm
			s := &task.Store{Root: m.store.Root}
			if _, err := s.Set(m.all[1].ID, "title", "Beta renamed elsewhere"); err != nil {
				t.Fatal(err)
			}
			commit("hug task set")
			loads := m.loads.n.Load()
			m = settle(t, m, 3*time.Second)
			if m.prompt != prompt || m.confirm != confirm || m.input.Value() != typed {
				t.Fatalf("interrupted: prompt %q->%q confirm %q->%q input %q->%q", prompt, m.prompt, confirm, m.confirm, typed, m.input.Value())
			}
			if strings.Contains(screen(m), "Beta renamed elsewhere") || m.loads.n.Load() != loads || !m.stale {
				t.Fatalf("reloaded under an open %s (loads %d -> %d, stale %v)", open.name, loads, m.loads.n.Load(), m.stale)
			}
			m = feed(t, m, tea.KeyMsg{Type: tea.KeyEsc})
			if m.busy() {
				t.Fatalf("still open after Esc: prompt %q confirm %q", m.prompt, m.confirm)
			}
			if !strings.Contains(screen(m), "Beta renamed elsewhere") || m.stale {
				t.Fatalf("not reloaded after the %s closed:\n%s", open.name, screen(m))
			}
		})
	}
	// The editor: editing is set while it runs; a change waits; editorMsg
	// refreshes once it closes.
	m, commit := gitBoard(t, "Alpha", "Beta")
	m.editing = true
	s := &task.Store{Root: m.store.Root}
	if _, err := s.Set(m.all[1].ID, "title", "Beta renamed elsewhere"); err != nil {
		t.Fatal(err)
	}
	commit("hug task set")
	loads := m.loads.n.Load()
	m = settle(t, m, 3*time.Second)
	if m.loads.n.Load() != loads || !m.stale {
		t.Fatal("reloaded while the editor was open")
	}
	m = feed(t, m, editorMsg{path: m.all[0].Path})
	if m.editing || !strings.Contains(screen(m), "Beta renamed elsewhere") {
		t.Fatal("not reloaded after the editor closed", screen(m))
	}
}

// Check 3: no reload while nothing changes; a burst of commits gives one
// reload, not one per commit.
func TestNoReloadWhileIdleOneForABurst(t *testing.T) {
	m, commit := gitBoard(t, "Alpha", "Beta")
	m.setView("list")
	// Reading the board, moving around and the TUI's own reads change
	// nothing on disk that matters.
	m = press(m, "j", "k", "enter", "esc")
	start := m.loads.n.Load()
	select {
	case <-m.watch.C:
		t.Fatal("a change signal with nothing changed")
	case <-time.After(1500 * time.Millisecond):
	}
	if m.loads.n.Load() != start {
		t.Fatalf("loads while idle: %d", m.loads.n.Load()-start)
	}
	// Five commits in quick succession: one signal, one load.
	s := &task.Store{Root: m.store.Root}
	for i := 0; i < 5; i++ {
		if _, err := s.Set(m.all[0].ID, "priority", "p"+string(rune('0'+i))); err != nil {
			t.Fatal(err)
		}
		commit("hug task set")
	}
	m = settle(t, m, 3*time.Second)
	select {
	case <-m.watch.C:
		m = feed(t, m, changedMsg{})
	case <-time.After(600 * time.Millisecond):
	}
	if n := m.loads.n.Load() - start; n < 1 || n > 2 {
		t.Fatalf("a burst of 5 commits gave %d loads, want 1 (2 at most when the burst straddles the debounce)", n)
	}
	if !strings.Contains(screen(m), "p4") {
		t.Fatal("last commit of the burst not shown", screen(m))
	}
}

// 46ek1f check 3: a write is one load. hug task note changes the task file
// and then, a git commit later (about 300 ms on Seid), the ref: two bursts
// further apart than the debounce, which were two loads of the whole board.
// The write holds the board turn from the file to the commit; the watch
// signals once the turn is free.
func TestOneSignalPerWrite(t *testing.T) {
	m, commit := gitBoard(t, "Alpha", "Beta")
	common := filepath.Join(m.store.Root, ".git")
	release, err := gitbranch.BackgroundAt(common, "hug/tasks").Turn()
	if err != nil {
		t.Fatal(err)
	}
	s := &task.Store{Root: m.store.Root}
	if _, err := s.Set(m.all[0].ID, "priority", "p1"); err != nil {
		t.Fatal(err)
	}
	// Longer than the debounce and the poll of the turn, as git's commit
	// on a big board: the file's burst has settled before the ref moves.
	time.Sleep(3 * debounce)
	commit("hug task set")
	release()
	signals := 0
	quiet := time.After(4 * debounce)
	deadline := time.After(10 * time.Second)
	for done := false; !done; {
		select {
		case <-m.watch.C:
			signals++
			quiet = time.After(4 * debounce)
		case <-quiet:
			done = signals > 0
			quiet = time.After(4 * debounce)
		case <-deadline:
			done = true
		}
	}
	if signals != 1 {
		t.Fatalf("one write (task file, then the commit) gave %d change signals, want 1", signals)
	}
}

// The fallback: with no OS watch, the ref's modification time is checked.
func TestChangeOnDiskPollsWithoutWatcher(t *testing.T) {
	old, oldPoll := newWatcher, pollEvery
	newWatcher = func() (*fsnotify.Watcher, error) { return nil, os.ErrPermission }
	pollEvery = 100 * time.Millisecond
	t.Cleanup(func() { newWatcher, pollEvery = old, oldPoll })
	m, commit := gitBoard(t, "Alpha", "Beta")
	if m.watch.mode != "poll" {
		t.Fatal(m.watch.mode)
	}
	select {
	case <-m.watch.C:
		t.Fatal("polling signalled with nothing changed")
	case <-time.After(400 * time.Millisecond):
	}
	s := &task.Store{Root: m.store.Root}
	if _, err := s.Set(m.all[1].ID, "title", "Beta polled"); err != nil {
		t.Fatal(err)
	}
	commit("hug task set")
	m = settle(t, m, 3*time.Second)
	if !strings.Contains(screen(m), "Beta polled") {
		t.Fatal(screen(m))
	}
}

// An action's own reload wins over a background load that began before it.
func TestOlderLoadNeverOverwritesNewer(t *testing.T) {
	m, _ := gitBoard(t, "Alpha", "Beta")
	m.setView("list")
	cmd := m.reload()
	old := cmd().(loadedMsg)
	s := &task.Store{Root: m.store.Root}
	if _, err := s.Set(m.all[0].ID, "title", "Alpha newer"); err != nil {
		t.Fatal(err)
	}
	m.refresh("") // an action's own refresh after the load began
	next, _ := m.Update(old)
	m = next.(model)
	if !strings.Contains(screen(m), "Alpha newer") {
		t.Fatal("an older load overwrote a newer screen", screen(m))
	}
}
