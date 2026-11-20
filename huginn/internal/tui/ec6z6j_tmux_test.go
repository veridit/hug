package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/tui/testfixture"
	"github.com/veridit/hug/internal/tmuxtest"
)

// ec6z6j check 5 in a real terminal: the hug binary in tmux at 80x24, the
// LINT list (v!), three rows marked with Space. The footer offers r
// request; r opens the box at the bottom with the three marked rows still
// on screen; Esc writes nothing and keeps the marks; r, the text, Enter
// writes one request for exactly the three, in one commit
// "hug task request a,b,c". With EC6Z6J_FRAMES set, the screens are saved
// there in colour (capture-pane -e), for the task.
func TestTmuxMarkedRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-ec6z6j-")
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
	frames := os.Getenv("EC6Z6J_FRAMES")
	env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=jhf", "HUG_MODEL=", "HUG_SYNC=off", "NO_COLOR=1", "TERM=xterm-256color", "GOMAXPROCS=2"}
	if frames != "" {
		env = append(env[:len(env)-3], "TERM=xterm-256color", "COLORTERM=truecolor", "GOMAXPROCS=2")
	}
	run := func(name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
		return string(out)
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "commit", "-q", "--allow-empty", "-m", "fixture")
	run(binary, "init")
	tasks := filepath.Join(repo, ".hug", "tasks")
	if err := testfixture.Seed(tasks); err != nil {
		t.Fatal(err)
	}
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "fixture tasks")

	socket := tmuxtest.Label("ec6z6j")
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
	tmux("new-session", "-d", "-s", "r", "-x", "80", "-y", "24", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	tmux("set-option", "-s", "exit-empty", "off")
	t.Cleanup(func() {
		exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "r").Run()
		stop()
	})
	screen := func() string { return tmux("capture-pane", "-p", "-t", "r:0.0") }
	wait := func(what string, want []string, gone ...string) string {
		t.Helper()
		last := ""
		for deadline := time.Now().Add(12 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			last = screen()
			ok := true
			for _, w := range want {
				ok = ok && strings.Contains(last, w)
			}
			for _, g := range gone {
				ok = ok && !strings.Contains(last, g)
			}
			if ok {
				if frames != "" {
					ans := tmux("capture-pane", "-e", "-p", "-t", "r:0.0")
					if err := os.WriteFile(filepath.Join(frames, "after-"+what+".ans"), []byte(ans), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				return last
			}
		}
		t.Fatalf("%s: never showed %q without %q:\n%s", what, want, gone, last)
		return ""
	}
	send := func(keys ...string) { tmux(append([]string{"send-keys", "-t", "r:0.0"}, keys...)...) }

	wait("start", []string{"HUG · repo"})
	send("v!")
	wait("lint", []string{"HUG · repo · lint"})
	send("Space", "Space", "Space")
	s := wait("marked", []string{"3 marked · "})
	markedRow := regexp.MustCompile(`(?m)^◆ .(\w{6}) `)
	var ids []string
	for _, m := range markedRow.FindAllStringSubmatch(s, -1) {
		ids = append(ids, m[1])
	}
	if len(ids) != 3 {
		t.Fatalf("3 marked rows, found %v:\n%s", ids, s)
	}
	if !strings.Contains(s, "3 marked · ") || !strings.Contains(s, " · request · Esc unmark") {
		t.Fatalf("the marked footer does not offer request:\n%s", s)
	}

	// r: the box at the bottom, the three marked rows above it.
	send("r")
	send("-l", "each lacks a purpose")
	s = wait("box", []string{"▸ REQUEST FOR 3 MARKED · " + strings.Join(ids, " "), "request: each lacks a purpose", "Enter send"})
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	head := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "▸ REQUEST FOR 3 MARKED") {
			head = i
		}
	}
	if head < len(lines)-5 {
		t.Fatalf("the box is not at the bottom (heading on line %d of %d):\n%s", head+1, len(lines), s)
	}
	for _, id := range ids {
		if !regexp.MustCompile(`(?m)^◆ .` + id + ` `).MatchString(strings.Join(lines[:head], "\n")) {
			t.Fatalf("marked %s is not in view above the box:\n%s", id, s)
		}
	}

	// Esc: nothing written, the marks stay.
	send("Escape")
	wait("esc", []string{"3 marked · "}, "REQUEST FOR")
	if log := run("git", "-C", tasks, "log", "--format=%s"); strings.Contains(log, "request") {
		t.Fatalf("Esc wrote a request:\n%s", log)
	}

	// Enter: one request for exactly the three, one commit.
	send("r")
	send("-l", "write a purpose from the goal")
	wait("box-again", []string{"request: write a purpose from the goal"})
	send("Enter")
	wait("sent", []string{"request 1 for 3 tasks, on " + ids[0]}, "◆ ")
	// One request commit, then its one delivery record (gmzd87), on the
	// home: nothing per other target.
	log := strings.Split(strings.TrimSpace(run("git", "-C", tasks, "log", "-3", "--format=%s")), "\n")
	if want := "hug task request " + strings.Join(ids, ","); len(log) < 3 || log[1] != want || !strings.HasPrefix(log[0], "hug task request 1 on "+ids[0]+" ") || log[2] != "fixture tasks" {
		t.Fatalf("commits %q, want the delivery on %s over %q over the fixture", log, ids[0], want)
	}
	// The CLI reads the same request: the text once, every target pending.
	out := run(binary, "show", ids[0]) // hug task show: this binary is huginn
	if !strings.Contains(out, "write a purpose from the goal") || !strings.Contains(out, "targets: "+ids[0]+" pending, "+ids[1]+" pending, "+ids[2]+" pending") {
		t.Fatalf("hug task show %s lacks the request for the three:\n%s", ids[0], out)
	}
	for _, id := range ids[1:] {
		if out := run(binary, "show", id); !strings.Contains(out, "see "+ids[0]+" request 1") {
			t.Fatalf("hug task show %s does not link to %s request 1:\n%s", id, ids[0], out)
		}
	}
}
