package tui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// The test binary doubles as a hug for the reload (j2vwk0): with
// HUG_TEST_FAKE_VERSION set, "version" prints "hug <that>", or exits 1 with
// a message when it is "broken".
const fakeVersion = "HUG_TEST_FAKE_VERSION"

func fakeHug() {
	v, ok := os.LookupEnv(fakeVersion)
	if !ok || len(os.Args) < 2 || os.Args[len(os.Args)-1] != "version" {
		return
	}
	if v == "broken" {
		os.Stderr.WriteString("dyld: Library not loaded: libhug.dylib\n")
		os.Exit(1)
	}
	os.Stdout.WriteString("hug " + v + "\n")
	os.Exit(0)
}

// reloadModel is the seeded board with a reload watch on a fake path, and
// hooks that record the exec instead of doing it.
func reloadModel(t *testing.T) (model, *[]string) {
	t.Helper()
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	m.opts.Version = "v0.7.33"
	m.bin = &binWatch{path: filepath.Join(t.TempDir(), "bin", "hug"), C: make(chan struct{}, 1)}
	var execed []string
	execBinary = func(path string, args, env []string) error {
		execed = append(execed, path)
		for _, e := range env {
			if strings.HasPrefix(e, handoffEnv+"=") {
				// What the new hug reads at start, before this returns.
				b, err := os.ReadFile(strings.TrimPrefix(e, handoffEnv+"="))
				if err != nil {
					t.Errorf("state file unreadable in the new hug: %v", err)
				}
				execed = append(execed, e, string(b))
			}
		}
		return nil
	}
	probeBinary = func(string) (string, error) { return "v0.7.34", nil }
	t.Cleanup(func() { execBinary, probeBinary = reexec, probeVersion })
	return m, &execed
}

// step feeds msg and runs every command it returns, until the model quits
// or nothing more comes back. It returns whether it quit.
func step(t *testing.T, m model, msg tea.Msg) (model, bool) {
	t.Helper()
	queue := []tea.Msg{msg}
	for n := 0; len(queue) > 0 && n < 50; n++ {
		next, cmd := m.Update(queue[0])
		m, queue = next.(model), queue[1:]
		for _, c := range flatten(cmd) {
			out := runCmd(c)
			switch out.(type) {
			case tea.QuitMsg:
				return m, true
			case nil, binaryMsg, changedMsg:
				// waits on a channel: nothing more until the watch fires
			default:
				queue = append(queue, out)
			}
		}
	}
	return m, false
}

// runCmd runs c with a timeout, so a command that waits on a watch channel
// counts as no message.
func runCmd(c tea.Cmd) tea.Msg {
	out := make(chan tea.Msg, 1)
	go func() { out <- c() }()
	select {
	case msg := <-out:
		return msg
	case <-time.After(200 * time.Millisecond):
		return nil
	}
}

func flatten(cmd tea.Cmd) []tea.Cmd {
	if cmd == nil {
		return nil
	}
	msg := runCmd(cmd)
	if b, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Cmd
		for _, c := range b {
			out = append(out, flatten(c)...)
		}
		return out
	}
	return []tea.Cmd{func() tea.Msg { return msg }}
}

// j2vwk0 check 1 (model side; TestTmuxReload drives the real binary): the
// handoff carries view, selection, filters, sort, open task, focus and
// scroll, and a new model resumes exactly there.
func TestReloadHandsOverPlace(t *testing.T) {
	m, execed := reloadModel(t)
	m.setView("list")
	m.setFilter("type", "feature")
	m.query = "task"
	m.applySort([]sortKey{{"priority", true}})
	m.move(2, true)
	m = press(m, "enter", "l")
	m.viewport.SetYOffset(1)
	before := ansi.Strip(m.View())
	want := m.handoff()
	if want.View != "list" || want.Detail == "" || len(want.Chips) != 1 || want.Query != "task" || len(want.Sort) != 1 {
		t.Fatalf("handoff misses state: %+v", want)
	}
	m, _ = step(t, m, binaryMsg{})
	m, quit := step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	if !quit || m.reloadTo != "v0.7.34" {
		t.Fatalf("no reload: quit %v reloadTo %q message %q", quit, m.reloadTo, m.message)
	}
	if err := reloadInto(m.bin.path, m.handoff()); err != nil {
		t.Fatal(err)
	}
	if len(*execed) != 3 || (*execed)[0] != m.bin.path {
		t.Fatalf("exec: %v", *execed)
	}
	// The new hug: a fresh model from disk and the state file. reloadInto
	// removed the file after the fake exec returned; put it back as the
	// new process found it.
	file := strings.TrimPrefix((*execed)[1], handoffEnv+"=")
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("a failed exec must remove the state file")
	}
	os.WriteFile(file, []byte((*execed)[2]), 0o600)
	t.Setenv(handoffEnv, file)
	h := takeHandoff()
	if h == nil {
		t.Fatal("no handoff read")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) || os.Getenv(handoffEnv) != "" {
		t.Fatal("the new hug must delete the state file and forget its name")
	}
	fresh := newModel(m.store, Options{User: "tester", Version: "v0.7.34"})
	fresh.width, fresh.height = 140, 40
	fresh.resize()
	fresh.resume(*h)
	if got := fresh.handoff(); got.View != want.View || got.Row != want.Row || got.Detail != want.Detail || got.DetailPane != want.DetailPane || got.Scroll != want.Scroll || got.Query != want.Query || len(got.Chips) != 1 || got.Chips[0] != want.Chips[0] || got.Sort[0] != want.Sort[0] {
		t.Fatalf("resumed elsewhere:\n got %+v\nwant %+v", got, want)
	}
	after := ansi.Strip(fresh.View())
	// Same screen but for the version in the key bar and the footer, which
	// Run sets to "reloaded hug v0.7.34" (TestTmuxReload checks it).
	footer := func(s string) string {
		lines := strings.Split(s, "\n")
		lines[len(lines)-3] = ""
		return strings.ReplaceAll(strings.Join(lines, "\n"), "v0.7.34", "v0.7.33")
	}
	if footer(after) != footer(before) {
		t.Fatalf("screen differs after reload:\n%s\n---\n%s", before, after)
	}
}

// Overview sections keep their own scroll and row across the reload.
func TestReloadKeepsSectionScroll(t *testing.T) {
	m, _ := reloadModel(t)
	m.setView("overview")
	m.height = 16
	m.resize()
	m.changePane(1)
	m.move(6, false)
	m.remember()
	before, h := ansi.Strip(m.View()), m.handoff()
	fresh := newModel(m.store, m.opts)
	fresh.width, fresh.height = m.width, m.height
	fresh.resize()
	fresh.resume(h)
	if after := ansi.Strip(fresh.View()); after != before {
		t.Fatalf("overview differs:\n%s\n---\n%s", before, after)
	}
}

// Jørgen's ruling: never by itself. A newer hug only changes the version
// corner, "hug v0.7.33 → v0.7.34 · R reloads"; R reloads; R is not bound
// while no newer hug exists; and like every key R does nothing while a
// prompt, the palette, help or a confirm has the keys (check 2: the
// reload waits until it closes, then R).
func TestReloadOnlyOnR(t *testing.T) {
	m, execed := reloadModel(t)
	m.setView("list")
	before := ansi.Strip(m.View())
	if m, quit := step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); quit || m.message != "" || ansi.Strip(m.View()) != before {
		t.Fatalf("R without a newer hug must do nothing: quit %v message %q", quit, m.message)
	}
	m, quit := step(t, m, binaryMsg{})
	if quit || m.reloadTo != "" || len(*execed) != 0 {
		t.Fatal("reloaded by itself")
	}
	after := ansi.Strip(m.View())
	bar := func(s string) string { l := strings.Split(s, "\n"); return l[len(l)-1] }
	if !strings.HasSuffix(bar(after), "hug v0.7.33 → v0.7.34 · R reloads") {
		t.Fatalf("corner: %q", bar(after))
	}
	lines, was := strings.Split(after, "\n"), strings.Split(before, "\n")
	for i := range lines[:len(lines)-1] {
		if lines[i] != was[i] {
			t.Fatalf("line %d changed, only the corner may:\n%q\n%q", i, was[i], lines[i])
		}
	}
	if help := ansi.Strip(press(m, "?").View()); !strings.Contains(help, "reload into hug v0.7.34") {
		t.Fatalf("help lacks R:\n%s", help)
	}
	for _, open := range []struct {
		name string
		keys []string
	}{{"find prompt", []string{"/"}}, {"palette", []string{":"}}, {"help", []string{"?"}}} {
		o := press(m, open.keys...)
		if o, quit = step(t, o, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); quit || o.reloadTo != "" {
			t.Fatalf("R reloaded with the %s open", open.name)
		}
	}
	if m, quit = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); !quit || m.reloadTo != "v0.7.34" {
		t.Fatalf("R did not reload: quit %v %q %q", quit, m.reloadTo, m.message)
	}
}

// A newer hug installed while the editor is open shows when it closes;
// R then reloads.
func TestReloadAfterEditor(t *testing.T) {
	m, _ := reloadModel(t)
	m.setView("list")
	m.editing = true
	m, _ = step(t, m, binaryMsg{})
	m, _ = step(t, m, editorMsg{path: m.visible[m.cursor].Path})
	if !strings.Contains(ansi.Strip(m.View()), "R reloads") {
		t.Fatal("corner lost while editing")
	}
	if m, quit := step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); !quit || m.reloadTo != "v0.7.34" {
		t.Fatal("R after the editor did not reload")
	}
}

// j2vwk0 check 3: a new binary that does not start leaves this TUI running
// with the reason and a fix: line; a later good binary still reloads.
func TestReloadBrokenBinaryKeepsRunning(t *testing.T) {
	m, execed := reloadModel(t)
	m.setView("list")
	probeBinary = func(string) (string, error) {
		return "", errors.New("exit status 1: dyld: Library not loaded: libhug.dylib")
	}
	m, quit := step(t, m, binaryMsg{})
	if quit || m.reloadTo != "" || len(*execed) != 0 {
		t.Fatal("reloaded into a broken binary")
	}
	screen := ansi.Strip(m.View())
	for _, want := range []string{"still on hug v0.7.33: the new hug does not start (exit status 1: dyld: Library not loaded", "fix: go install github.com/veridit/hug/cmd/hug@latest"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing %q:\n%s", want, screen)
		}
	}
	if got := reinstallFix("/opt/homebrew/Cellar/hug/0.7.34/bin/hug"); got != "brew reinstall veridit/tap/hug" {
		t.Fatal(got)
	}
	if strings.Contains(screen, "R reloads") {
		t.Fatal("a broken binary must not offer R")
	}
	m = press(m, "j") // the TUI still works
	probeBinary = func(string) (string, error) { return "v0.7.35", nil }
	m, _ = step(t, m, binaryMsg{})
	if m, quit = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); !quit || m.reloadTo != "v0.7.35" {
		t.Fatal("a good binary after a broken one must reload on R")
	}
	// Broken between the corner and R: R asks again and stays.
	m, _ = reloadModel(t)
	m, _ = step(t, m, binaryMsg{})
	probeBinary = func(string) (string, error) { return "", errors.New("exit status 1: killed") }
	if m, quit = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); quit || !strings.Contains(m.message, "does not start") || strings.Contains(ansi.Strip(m.View()), "R reloads") {
		t.Fatalf("R into a binary broken since: quit %v %q", quit, m.message)
	}
}

// Only the same or a newer hug gets the state (Jørgen: no backwards
// compatibility): an older one is refused with a fix: line.
func TestReloadRefusesOlder(t *testing.T) {
	m, _ := reloadModel(t)
	probeBinary = func(string) (string, error) { return "v0.7.9", nil }
	m, quit := step(t, m, binaryMsg{})
	if quit || !strings.Contains(m.message, "still on hug v0.7.33: hug v0.7.9, which is older") || !strings.Contains(m.message, "\nfix: ") {
		t.Fatalf("quit %v message %q", quit, m.message)
	}
	for _, c := range []struct {
		a, b string
		want bool
	}{{"v0.7.9", "v0.7.33", true}, {"v0.7.33", "v0.7.33", false}, {"v0.8.0", "v0.7.33", false}, {"dev", "v0.7.33", false}, {"v0.7.33", "dev", false}} {
		if older(c.a, c.b) != c.want {
			t.Errorf("older(%s, %s) != %v", c.a, c.b, c.want)
		}
	}
}

// The real probe runs "path version" (the test binary as a fake hug).
func TestProbeVersion(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(fakeVersion, "v9.9.9")
	if v, err := probeVersion(self); err != nil || v != "v9.9.9" {
		t.Fatalf("%q %v", v, err)
	}
	t.Setenv(fakeVersion, "broken")
	if _, err := probeVersion(self); err == nil || !strings.Contains(err.Error(), "Library not loaded") {
		t.Fatalf("broken: %v", err)
	}
	if _, err := probeVersion(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("a missing binary must fail")
	}
}

// The watch sees brew's symlink swap (new target, same path) and a file
// replaced in place, once it holds still.
func TestBinWatchSeesSwap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows, and the reload is off there")
	}
	binPoll = 20 * time.Millisecond
	t.Cleanup(func() { binPoll = 2 * time.Second })
	dir := t.TempDir()
	old, next := filepath.Join(dir, "Cellar", "0.7.33", "hug"), filepath.Join(dir, "Cellar", "0.7.34", "hug")
	for _, p := range []string{old, next} {
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(p), 0o755)
	}
	link := filepath.Join(dir, "bin", "hug")
	os.MkdirAll(filepath.Dir(link), 0o755)
	if err := os.Symlink(old, link); err != nil {
		t.Fatal(err)
	}
	w := startBinWatch(link)
	defer w.close()
	select {
	case <-w.C:
		t.Fatal("signal without a change")
	case <-time.After(150 * time.Millisecond):
	}
	// brew: a new symlink renamed over the old one.
	tmp := link + ".new"
	os.Symlink(next, tmp)
	os.Rename(tmp, link)
	select {
	case <-w.C:
	case <-time.After(2 * time.Second):
		t.Fatal("symlink swap not seen")
	}
	// go install: same path, new file.
	os.WriteFile(next+".tmp", []byte("rebuilt hug, longer"), 0o755)
	os.Rename(next+".tmp", next)
	select {
	case <-w.C:
	case <-time.After(2 * time.Second):
		t.Fatal("replaced file not seen")
	}
}
