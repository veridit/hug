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

// zhvkmj check 5 in a real terminal: the hug binary in tmux, on a copy of
// hug's own board as it was when Jørgen could not reach Files (zhvkmj
// asked of jhf, testdata/zhvkmj-board). From the top of zhvkmj's detail,
// Tab reaches FILES and it is on screen, j marks screen-1821.ans, Space
// shows it in the file frame.
func TestTmuxZhvkmjFilesOnItself(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-zhvkmj-")
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
	home, repo := filepath.Join(root, "home"), filepath.Join(root, "repo")
	for _, p := range []string{home, repo} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=jhf", "HUG_MODEL=", "HUG_SYNC=off", "NO_COLOR=1", "TERM=xterm-256color", "GOMAXPROCS=2"}
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
	// The task, its two captures and the board's hug.toml (people: jhf).
	for _, rel := range []string{"z/zhvkmj.md", "z/zhvkmj/colours-overview.ans", "z/zhvkmj/screen-1821.ans", "hug.toml"} {
		b, e := os.ReadFile(filepath.Join("testdata", "zhvkmj-board", rel))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.MkdirAll(filepath.Dir(filepath.Join(tasks, rel)), 0o755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(tasks, rel), b, 0o644); e != nil {
			t.Fatal(e)
		}
	}
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "zhvkmj from hug's board")

	socket := tmuxtest.Label("zhvkmj")
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
	tmux("new-session", "-d", "-s", "zhv", "-x", "140", "-y", "40", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	tmux("set-option", "-s", "exit-empty", "off")
	defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "zhv").Run()
	wait := func(want string) string {
		t.Helper()
		last := ""
		for deadline := time.Now().Add(12 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			last = tmux("capture-pane", "-p", "-t", "zhv:0.0")
			if strings.Contains(last, want) {
				return last
			}
		}
		t.Fatalf("never showed %q:\n%s", want, last)
		return ""
	}
	send := func(keys ...string) { tmux(append([]string{"send-keys", "-t", "zhv:0.0"}, keys...)...) }

	wait("HUG · repo")
	send("#zhvkmj")
	wait("JUMP TO ID")
	send("Enter")
	wait("capture, as it was taken")
	send("g") // the top of the detail
	// Tab: the checks (the question is on check 4), then Files.
	for i := 0; i < 6; i++ {
		send("Tab")
		time.Sleep(150 * time.Millisecond)
		if strings.Contains(tmux("capture-pane", "-p", "-t", "zhv:0.0"), "▸ FILES (2)") {
			break
		}
	}
	wait("> ⎘ colours-overview.ans")
	send("j")
	wait("> ⎘ screen-1821.ans")
	send("Space")
	screen := wait("screen-1821.ans · 10.0 KiB · 2 of 2")
	if first := strings.SplitN(screen, "\n", 2)[0]; !strings.Contains(first, "zhvkmj › z/zhvkmj/screen-1821.ans") {
		t.Fatalf("Space did not show the capture in the file frame:\n%s", screen)
	}
}
