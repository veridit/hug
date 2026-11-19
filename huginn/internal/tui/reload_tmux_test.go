package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/tui/testfixture"
	"github.com/veridit/hug/internal/tmuxtest"
)

// TestTmuxReload is j2vwk0 end to end with the real binary, Jørgen's test:
// hug v0.0.1 runs through a bin/hug symlink into a Cellar folder, as brew
// installs it. The reader filters, sorts and opens a task; then the symlink
// moves to v0.0.2 (brew upgrade). The TUI keeps running v0.0.1 and only the
// corner changes, to "hug v0.0.1 → v0.0.2 · R reloads". R inside the
// palette is just a letter; R outside it brings the same process (same
// pane, same pid) back as v0.0.2 on the same task with the same filters,
// saying "reloaded hug v0.0.2". A broken v0.0.3 leaves v0.0.2 running
// with a fix: line.
func TestTmuxReload(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if runtime.GOOS == "windows" {
		t.Skip("no exec on Windows: the reload is off there")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required for terminal acceptance")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "huginn-reload-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	cellar := func(v string) string { return filepath.Join(root, "Cellar", "hug", v, "bin", "hug") }
	for _, v := range []string{"0.0.1", "0.0.2"} {
		build := exec.Command("nice", "-n", "19", "env", "GOMAXPROCS=2", "GOFLAGS=-p=2", "go", "build", "-ldflags", "-X main.Version=v"+v+" -X github.com/veridit/hug/huginn/run.Version=v"+v, "-o", cellar(v), "./cmd/hug")
		build.Dir = "../../.."
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", v, err, out)
		}
	}
	link := filepath.Join(root, "bin", "hug")
	os.MkdirAll(filepath.Dir(link), 0o755)
	if err := os.Symlink(cellar("0.0.1"), link); err != nil {
		t.Fatal(err)
	}
	// brew upgrade: a new symlink renamed over the old one.
	swap := func(target string) {
		t.Helper()
		tmp := link + ".new"
		if err := os.Symlink(target, tmp); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(tmp, link); err != nil {
			t.Fatal(err)
		}
	}
	home, repo := filepath.Join(root, "home"), filepath.Join(root, "repo")
	for _, p := range []string{home, repo} {
		os.MkdirAll(p, 0o755)
	}
	env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=terminal-fixture", "TERM=xterm-256color", "NO_COLOR=1", "GOMAXPROCS=2", "PATH=" + filepath.Dir(link) + ":/usr/bin:/bin"}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir, cmd.Env = repo, env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "commit", "-q", "--allow-empty", "-m", "synthetic fixture")
	run(link, "task", "init")
	if err := testfixture.Seed(filepath.Join(repo, ".hug", "tasks")); err != nil {
		t.Fatal(err)
	}
	socket := tmuxtest.Label("reload")
	defer startTmuxServer(t, socket)()
	tmux := func(args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "tmux", append([]string{"-N", "-L", socket, "-f", "/dev/null"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	quoted := make([]string, len(env))
	for i, e := range env {
		quoted[i] = shellQuote(e)
	}
	// "hug tui" as typed: the shell finds hug on PATH, through the symlink.
	// exec: the pane's pid is hug's own, so a second TUI would show.
	tmux("new-session", "-d", "-s", "r", "-x", "140", "-y", "40", "-c", repo, "exec env "+strings.Join(quoted, " ")+" hug tui")
	tmux("set-option", "-s", "exit-empty", "off")
	tmux("set-option", "-g", "remain-on-exit", "on")
	defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "r").Run()
	tmux("set-option", "-g", "status", "off")
	send := func(keys ...string) { tmux(append([]string{"send-keys", "-t", "r:0.0"}, keys...)...) }
	screen := func() string { return normalizeCapture(tmux("capture-pane", "-p", "-t", "r:0.0")) }
	// waitFor returns the first capture ok accepts.
	waitFor := func(what string, ok func(string) bool, within time.Duration) string {
		t.Helper()
		deadline := time.Now().Add(within)
		last := ""
		for time.Now().Before(deadline) {
			if last = screen(); ok(last) {
				return last
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("screen never showed %s\n%s", what, last)
		return ""
	}
	wait := func(want string, within time.Duration) string {
		t.Helper()
		return waitFor(fmt.Sprintf("%q", want), func(s string) bool { return strings.Contains(s, want) }, within)
	}
	pid := func() string { return strings.TrimSpace(tmux("display-message", "-p", "-t", "r:0.0", "#{pane_pid}")) }
	wait("hug v0.0.1", 15*time.Second)
	started := pid()
	// A place worth keeping: list view, a filter, a sort, a search, then a
	// task open with the checks focused.
	send("-l", "vl") // the list (qh9ep1)
	wait("HUG · repo · list", 5*time.Second)
	send("-l", ":doing") // the digits are gone (ahejza)
	send("Enter")
	wait("state:doing", 5*time.Second)
	send("-l", ":sort")
	send("Enter")
	wait("HUG · repo · sort", 5*time.Second)
	send("-l", "p")
	send("Enter")
	wait("sort: priority", 5*time.Second)
	send("-l", "jj")
	send("Enter")
	wait("HUG · repo · list ›", 5*time.Second) // the breadcrumb band (e3sevr)
	send("Tab")
	time.Sleep(300 * time.Millisecond)
	before := waitFor("the task, no load running", settled, 5*time.Second)
	header := strings.SplitN(before, "\n", 6)
	// The new hug installed (brew upgrade): the old one keeps running and
	// only the corner changes (Jørgen's test).
	swap(cellar("0.0.2"))
	offered := waitFor("the R offer, no load running", func(s string) bool {
		return strings.Contains(s, "hug v0.0.1 → v0.0.2 · R reloads") && settled(s)
	}, 15*time.Second)
	if lines, was := strings.Split(strings.TrimSuffix(offered, "\n"), "\n"), strings.Split(strings.TrimSuffix(before, "\n"), "\n"); len(lines) != len(was) {
		t.Fatal("screen size changed")
	} else {
		for i := 0; i < len(lines)-1; i++ {
			if lines[i] != was[i] {
				t.Fatalf("line %d changed before R; only the corner may:\n%q\n%q", i+1, was[i], lines[i])
			}
		}
	}
	// R with the palette open is typed into the palette: no reload (check 2).
	send("-l", ":")
	wait("GLOBAL", 5*time.Second) // the palette's global group (05r3pg)
	send("-l", "R")
	time.Sleep(1500 * time.Millisecond)
	if s := screen(); strings.Contains(s, "reloaded hug") {
		t.Fatalf("reloaded with the palette open:\n%s", s)
	}
	send("Escape") // the typed R goes (05r3pg: Esc goes up a level)
	send("Escape") // the palette closes
	wait("hug v0.0.1 → v0.0.2 · R reloads", 5*time.Second)
	send("-l", "R")
	after := waitFor("the reloaded hug, its first load done", reloadShown, 15*time.Second)
	// The new hug's first frame has the terminal's width: the 140-column
	// divider, never the default 100 redrawn a moment later (vnyz8z;
	// CI 44132567751 caught a 100-column first frame).
	if !strings.Contains(after, strings.Repeat("─", 140)) {
		t.Fatalf("the reloaded hug's first frame is not 140 columns wide:\n%s", after)
	}
	if dir := os.Getenv("HUG_TUI_ARTIFACTS"); dir != "" {
		os.MkdirAll(filepath.Join(dir, "reload"), 0o755)
		os.WriteFile(filepath.Join(dir, "reload", "before.txt"), []byte(before), 0o644)
		os.WriteFile(filepath.Join(dir, "reload", "offered.txt"), []byte(offered), 0o644)
		os.WriteFile(filepath.Join(dir, "reload", "after.txt"), []byte(after), 0o644)
	}
	if got := pid(); got != started {
		t.Fatalf("a second TUI started: pane pid %s, was %s", got, started)
	}
	// Same task, same focus, same filters and sort (check 1); only the
	// footer and the version in the key bar differ (check 4).
	norm := func(s string) string {
		lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
		lines[len(lines)-3] = ""
		return strings.ReplaceAll(strings.Join(lines, "\n"), "hug v0.0.2", "hug v0.0.1")
	}
	if norm(after) != norm(before) {
		t.Fatalf("not the same place after the reload:\nBEFORE:\n%s\nAFTER:\n%s", before, after)
	}
	if !strings.Contains(after, strings.TrimSpace(header[0])) {
		t.Fatalf("header changed: %q\n%s", header[0], after)
	}
	// The line belongs to the screen it came back on: leaving that task
	// clears it (tz9z14, Jørgen 15:19: "noise because it's from the main
	// screen").
	send("Escape")
	// The detail's crumb also reads "HUG TASKS · list › ID": wait for the
	// list itself, the crumb without › (CI 44326448795 asserted too early).
	listed := waitFor("the list, drawn whole, after Esc left the task", listShown, 5*time.Second)
	if strings.Contains(listed, "reloaded hug") {
		t.Fatalf("the reload line stayed after leaving the task:\n%s", listed)
	}
	send("Enter")
	wait("› ", 5*time.Second)
	// A broken binary: v0.0.2 keeps running and says how to fix it (check 3).
	broken := cellar("0.0.3")
	os.MkdirAll(filepath.Dir(broken), 0o755)
	os.WriteFile(broken, []byte("#!/bin/sh\necho 'dyld: Library not loaded: libhug.dylib' >&2\nexit 1\n"), 0o755)
	swap(broken)
	failed := wait("fix: brew reinstall veridit/tap/hug", 15*time.Second)
	if strings.Contains(failed, "R reloads") {
		t.Fatalf("a broken binary must not offer R:\n%s", failed)
	}
	if !strings.Contains(failed, "still on hug v0.0.2: the new hug does not start") || !strings.Contains(failed, "hug v0.0.2") {
		t.Fatalf("broken binary:\n%s", failed)
	}
	send("Escape")
	wait("HUG · repo · list", 5*time.Second) // still working
	send("-l", "q")
	t.Log("tmux reload: PASS (v0.0.1 -> v0.0.2 by symlink swap: corner only until R, R reloads, same pane pid, same place; broken v0.0.3 refused)")
}

// frameLines are a capture's first line, the header, and its last, the
// key bar.
func frameLines(s string) (first, last string) {
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	return lines[0], lines[len(lines)-1]
}

// settled says no load is running: the header shows "refreshing" while a
// load runs past spinAfter, which under load can be any frame. The frames
// TestTmuxReload compares line by line must not differ by it.
func settled(s string) bool {
	first, _ := frameLines(s)
	return !strings.Contains(first, "refreshing")
}

// reloadShown says a capture shows the reloaded hug, its first load done.
func reloadShown(s string) bool {
	return strings.Contains(s, "reloaded hug v0.0.2") && settled(s)
}

// listShown says a capture shows the list after Esc left the task, drawn
// whole. A capture can catch a frame half drawn: under load the tmux server
// reads hug's output in pieces, and hug writes a frame top line first, the
// key bar last (hfkt1n). So the crumb without › is not enough; the key bar
// must be the list's own, "j/k move", not the detail's "j/k scroll".
func listShown(s string) bool {
	first, last := frameLines(s)
	return !strings.Contains(first, "›") && strings.HasPrefix(last, "j/k move")
}

// TestReloadFramesSettle holds TestTmuxReload's waits to frames it can
// compare (hfkt1n): captures from that test under machine load.
func TestReloadFramesSettle(t *testing.T) {
	frame := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("testdata", "reload", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	for _, c := range []struct {
		name string
		want bool
		ok   func(string) bool
	}{
		// The reloaded hug back on the task, its first load done.
		{"after.txt", true, reloadShown},
		// The same, its first load still running: "⠸ refreshing" in the
		// header, which "the same place" would then report.
		{"after-refreshing.txt", false, reloadShown},
		// Esc's frame half drawn: the tmux server had read the list's top
		// lines, not yet the rest. The detail's key bar and its "reloaded
		// hug" line are still on screen (hjbfzd's run; then a tmux server
		// stopped mid-frame, hfkt1n).
		{"torn-hjbfzd.txt", false, listShown},
		{"torn-after-esc.txt", false, listShown},
		// The detail, before Esc.
		{"after.txt", false, listShown},
		// The task before R, then the R offer, then the same with a load
		// running: line 1 differs from the task's before R.
		{"before.txt", true, settled},
		{"offered.txt", true, settled},
		{"offered-refreshing.txt", false, settled},
	} {
		if got := c.ok(frame(c.name)); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
