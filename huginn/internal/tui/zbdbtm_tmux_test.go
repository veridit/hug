package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/tmuxtest"
)

// zbdbtm in a real terminal: the hug binary in tmux at 100x30, run in a
// checkout that holds docs/DESIGN.md (testdata/zbdbtm-repo), on copies of
// two tasks from hug's board (testdata/zbdbtm-board):
//   - zbdbtm as Jørgen wrote it (hug/tasks 406a2cb2^), TRY IT HERE in its
//     Details, sample.txt and sample-notes.md attached;
//   - p87ngv as qcq3tm has it, with MOCK.md, before.md, after.md,
//     tui-100x30.txt and frame-1-100x30.ans from hug/tasks 069928d0.
//
// With ZBDBTM_FRAMES set, each screen waited for is kept there, with its
// colours, as an .ans: the frames attached to zbdbtm.

func newLinkTmux(t *testing.T, name string) *askedTmux {
	t.Helper()
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-zbdbtm-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	binary := filepath.Join(root, "hug")
	build := exec.Command("nice", "-n", "19", "env", "GOMAXPROCS=2", "GOFLAGS=-p=2", "go", "build", "-o", binary, "./cmd/huginn")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	home, repo := filepath.Join(root, "home"), filepath.Join(root, "repo")
	for _, p := range []string{home, repo} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=jhf", "HUG_MODEL=", "HUG_SYNC=off", "NO_COLOR=1", "TERM=xterm-256color", "GOMAXPROCS=2"}
	frames := os.Getenv("ZBDBTM_FRAMES")
	if frames != "" {
		// Frames for Jørgen: in the real theme's colours.
		env = append(env[:len(env)-3], "TERM=xterm-256color", "COLORTERM=truecolor", "GOMAXPROCS=2")
	}
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	copyBoard(t, filepath.Join("testdata", "zbdbtm-repo"), repo)
	run("git", "init", "-q", "-b", "main")
	run("git", "add", "-A")
	run("git", "commit", "-q", "-m", "fixture: docs/DESIGN.md")
	run(binary, "init")
	tasks := filepath.Join(repo, ".hug", "tasks")
	copyBoard(t, filepath.Join("testdata", "zbdbtm-board"), tasks)
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "zbdbtm and p87ngv from hug's board")

	socket := tmuxtest.Label("zbdbtm" + name)
	stop := startTmuxServer(t, socket)
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
	tmux("new-session", "-d", "-s", "q", "-x", "100", "-y", "30", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	tmux("set-option", "-s", "exit-empty", "off")
	t.Cleanup(func() {
		exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "q").Run()
		stop()
	})
	a := &askedTmux{t: t, tmux: tmux, frames: frames}
	a.wait("start", []string{"HUG · repo"})
	return a
}

// tabTo presses Tab until the screen shows want (a link marked ▸).
func (a *askedTmux) tabTo(want string) string {
	a.t.Helper()
	last := a.screen()
	for i := 0; i < 60; i++ {
		a.send("Tab")
		// Each Tab moves the mark: wait until it is drawn, so the next
		// Tab starts from the stop the screen shows.
		before := last
		for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(30 * time.Millisecond) {
			if last = a.screen(); last != before {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		if last = a.screen(); strings.Contains(last, want) {
			return a.wait("Tab to "+want, []string{want})
		}
	}
	a.t.Fatalf("Tab never reached %q:\n%s", want, last)
	return ""
}

// zbdbtm check 6 (and 1, 2): on zbdbtm itself, each of the four names in
// TRY IT HERE opens: Space shows it in line under its line, Enter in the
// file frame at the section, Esc comes back to the link.
func TestTmuxTryItHereLinksOpen(t *testing.T) {
	a := newLinkTmux(t, "try")
	a.open("zbdbtm", "A file named in a task opens from the task in the embedded viewer")
	// Details is a long section, folded: Tab to it, Enter opens it.
	a.tabTo("▸ DETAILS")
	a.send("Enter")
	a.wait("zbdbtm Details open", []string{"TRY IT HERE"})

	for _, c := range []struct {
		name, link, inline, frame, top string
	}{
		{"sample.txt", "▸sample.txt", "│ line 1: sample text file for zbdbtm", "sample.txt · 50 B · 2 of 2", "line 1: sample text file for zbdbtm"},
		{"file:sample-notes.md", "file:▸sample-notes.md", "│ # Sample notes for zbdbtm", "sample-notes.md · 244 B · 1 of 2", "Sample notes for zbdbtm"},
		{"sample-notes.md#Section two", "▸sample-notes.md#Section two", "│ ## Section two", "sample-notes.md · 244 B · 1 of 2", "Section two"},
		{"docs/DESIGN.md#Personal vs shared memories", "▸docs/DESIGN.md#Personal vs shared memories", "│ ## Personal vs shared memories (gav9nw", "docs/DESIGN.md · 9.8 KiB · repo", "Personal vs shared memories (gav9nw"},
	} {
		a.tabTo(c.link)
		a.send("Space")
		a.wait(c.name+" Space shows it in line", []string{c.link, c.inline, "Esc or Space closes"})
		a.send("Escape")
		a.wait(c.name+" Esc folds it away", []string{c.link}, c.inline)
		a.send("Enter")
		// At the section: once drawn (Markdown is formatted first), its
		// heading is the frame's first line.
		firstLine := func(screen string) string {
			lines := strings.Split(screen, "\n")
			for i, l := range lines {
				if strings.HasPrefix(l, "────") && i+1 < len(lines) {
					return lines[i+1]
				}
			}
			return ""
		}
		screen := ""
		for deadline := time.Now().Add(12 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			if screen = a.screen(); strings.Contains(screen, c.frame) && strings.Contains(firstLine(screen), c.top) {
				break
			}
		}
		if !strings.Contains(firstLine(screen), c.top) {
			t.Fatalf("%s: the frame's first line is %q, want %q:\n%s", c.name, firstLine(screen), c.top, screen)
		}
		a.wait(c.name+" Enter opens it in the file frame at the section", []string{c.frame, "Esc link", c.top})
		a.send("Escape")
		a.wait(c.name+" Esc comes back to the link", []string{c.link, "TRY IT HERE"}, c.frame)
	}
}

// zbdbtm check 5 (Jørgen on p87ngv, request 4): a file named in plain
// text in a question (MOCK.md, tui-100x30.txt) is a link: Space shows it
// in line under the line that names it, Enter in the file frame, Esc
// comes back.
func TestTmuxPlainFileNameInQuestionOpens(t *testing.T) {
	a := newLinkTmux(t, "plain")
	a.open("p87ngv", "Inline marks on a task's text; resolved dialogue moves to a history file")
	a.tabTo("▸ QUESTIONS")
	a.send("Enter")
	a.wait("p87ngv Questions open", []string{"▸ QUESTIONS (10) · Enter folds", "1. crab → you"})
	a.tabTo("approve the mock? ▸MOCK.md lists")
	a.send("Space")
	a.wait("MOCK.md Space shows it in line", []string{"▸MOCK.md lists", "┌ MOCK.md · 6.0 KiB", "more lines · Enter opens it"})
	a.send("Space")
	a.wait("MOCK.md Space again closes it", []string{"▸MOCK.md lists"}, "┌ MOCK.md")
	a.send("Enter")
	a.wait("MOCK.md Enter opens it in the file frame", []string{"MOCK.md · 6.0 KiB · 1 of 6", "Esc link"})
	a.send("Escape")
	a.wait("MOCK.md Esc comes back to the link", []string{"▸MOCK.md lists"}, "1 of 6")
	a.tabTo("▸tui-100x30.txt")
	a.send("Space")
	a.wait("tui-100x30.txt Space shows it in line", []string{"▸tui-100x30.txt", "┌ tui-100x30.txt · 19.4 KiB", "│ FRAME 1 · detail of w7tq3k"})
	a.send("Escape")
	a.send("Enter")
	a.wait("tui-100x30.txt Enter opens it in the file frame", []string{"tui-100x30.txt · 19.4 KiB · 6 of 6", "FRAME 1 · detail of w7tq3k"})
	a.send("Escape")
	a.wait("tui-100x30.txt Esc comes back to the link", []string{"▸tui-100x30.txt"}, "6 of 6")
}
