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

// 7zg9mf check 6 in a real terminal: the hug binary in tmux at 100x30 on
// five tasks, aaaaaa taken, bbbbbb dropped, cccccc done. 'v l' opens the
// list on 'filter: [state:!done ×]' without cccccc; Esc from a task keeps
// the chip; Esc up to the overview shows every task with 'filter: none';
// F in the list clears the chip to 'filter: none' and cccccc is back, and
// the list stays cleared. In the f picker, ! flips the highlighted value:
// '!done' and its count. With FRAMES_DIR set, the frames (tmux
// capture-pane -e) of the list as it opens and of the picker go there.
func TestTmuxListOpensOnNotDone(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-7zg9mf-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	binary := filepath.Join(root, "hug")
	build := exec.Command("nice", "-n", "19", "env", "GOMAXPROCS=2", "GOFLAGS=-p=2", "go", "build", "-o", binary, "./cmd/huginn")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	repo := viewRepo(t, root, binary)
	env := slotEnv(filepath.Join(root, "home"))
	for _, args := range [][]string{{"check", "cccccc", "1"}, {"check", "cccccc", "2"}, {"take", "cccccc", "--as", "crab", "--model", "human"}, {"done", "cccccc", "--as", "crab"}} {
		cmd := exec.Command(binary, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	if os.Getenv("FRAMES_DIR") != "" {
		for i, e := range env {
			if e == "NO_COLOR=1" {
				env[i] = "NO_COLOR=" // the frames are for people: colour on
			}
		}
	}
	socket := tmuxtest.Label("7zg9mf")
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
	tmux("set-option", "-s", "exit-empty", "off")
	quoted := make([]string, len(env))
	for i, e := range env {
		quoted[i] = shellQuote(e)
	}
	target := "n:0.0"
	tmux("new-session", "-d", "-s", "n", "-x", "100", "-y", "30", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "n").Run()
	tmux("set-option", "-t", "n", "status", "off")
	capture := func() string { return normalizeCapture(tmux("capture-pane", "-p", "-t", target)) }
	wait := func(want string) string {
		t.Helper()
		last, stable := "", 0
		for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			current := capture()
			if strings.Contains(current, want) && current == last {
				if stable++; stable >= 2 {
					return current
				}
			} else {
				stable = 0
			}
			last = current
		}
		t.Fatalf("never showed %q:\n%s", want, last)
		return ""
	}
	send := func(keys ...string) {
		for _, k := range keys {
			tmux("send-keys", "-t", target, k)
		}
	}
	frame := func(name string) {
		if dir := os.Getenv("FRAMES_DIR"); dir != "" {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(tmux("capture-pane", "-e", "-p", "-t", target)), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	rowOf := func(s, id string) bool {
		for _, line := range strings.Split(s, "\n") {
			if strings.Contains(line, id+" ") && !strings.HasPrefix(strings.TrimSpace(line), "HUG") {
				return true
			}
		}
		return false
	}
	s := wait("HUG · repo · overview")
	if !strings.Contains(s, "filter: none") {
		t.Fatalf("the overview opened filtered:\n%s", s)
	}
	send("v", "l")
	s = wait("filter: [state:!done ×]")
	frame("1-list-opens.ansi")
	if !strings.HasPrefix(s, "HUG · repo · list") || rowOf(s, "cccccc") || !rowOf(s, "bbbbbb") || !strings.Contains(s, "4/5 tasks") {
		t.Fatalf("the list did not open on state !done (cccccc out, the dropped bbbbbb in, 4/5):\n%s", s)
	}
	// Esc from a task keeps the chip.
	send("Enter")
	wait("› ")
	send("Escape")
	s = wait("HUG · repo · list")
	if !strings.Contains(s, "[state:!done ×]") {
		t.Fatalf("Esc from a task cleared the default:\n%s", s)
	}
	// Esc up to the overview: the overview's own sections, unfiltered.
	send("Escape")
	s = wait("HUG · repo · overview")
	if !strings.Contains(s, "filter: none") || !strings.Contains(s, "5/5 tasks") {
		t.Fatalf("the overview shows the list's default:\n%s", s)
	}
	// The picker: ! flips the highlighted value.
	send("v", "l")
	wait("[state:!done ×]")
	send("f", "s", "d", "o", "n", "e", "!")
	s = wait("> !done")
	frame("2-picker-not-done.ansi")
	if !strings.Contains(s, "FILTER · state · not") {
		t.Fatalf("the picker does not say not:\n%s", s)
	}
	send("Escape", "Escape")
	wait("HUG · repo · list")
	// F clears it to 'filter: none', and the list stays cleared.
	send("F")
	s = wait("filter: none")
	if !rowOf(s, "cccccc") || !strings.Contains(s, "5/5 tasks") {
		t.Fatalf("F did not bring the done task back:\n%s", s)
	}
	send("v", "o")
	wait("HUG · repo · overview")
	send("v", "l")
	s = wait("HUG · repo · list")
	if !strings.Contains(s, "filter: none") {
		t.Fatalf("the default came back after F:\n%s", s)
	}
}
