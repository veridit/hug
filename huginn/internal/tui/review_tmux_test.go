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

// 6ephqk check 6 (tmux, the real binary in a real git repo): t on a todo
// task waiting for review takes it into doing/review with review_by = the
// TUI user, committed as Model: human; t on a task you have gives it back
// to todo, keeping its stage.
func TestTmuxTakeForReviewAndGiveBack(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-review-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	binary := filepath.Join(root, "huginn")
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
	env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=Jørgen", "HUG_MODEL=", "HUG_SYNC=off", "NO_COLOR=1", "TERM=xterm-256color", "GOMAXPROCS=2"}
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
	write := func(id, title, front string) {
		body := "---\nid: \"" + id + "\"\ntitle: \"" + title + "\"\n" + front + "\ntype: \"feature\"\npriority: \"high\"\nlabels: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Acceptance\n- [x] a\n"
		if err := os.WriteFile(filepath.Join(tasks, "0"+id[:1]+"."+id+".t.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("aaaaaa", "Approve the status colours", "state: \"todo\"\nstage: \"review\"\nreview_since: \"2026-11-17T09:00\"\nassignee: []")
	write("bbbbbb", "My own work", "state: \"doing\"\nstage: \"active\"\ndoing_since: \"2026-11-17T08:00\"\nassignee: [\"Jørgen\"]\nmodel: \"human\"")
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "fixture tasks")

	socket := tmuxtest.Label("review")
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
	tmux("new-session", "-d", "-s", "review", "-x", "140", "-y", "40", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	tmux("set-option", "-s", "exit-empty", "off")
	defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "review").Run()
	wait := func(want string) string {
		t.Helper()
		last := ""
		for deadline := time.Now().Add(12 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			last = tmux("capture-pane", "-p", "-t", "review:0.0")
			if strings.Contains(last, want) {
				return last
			}
		}
		t.Fatalf("never showed %q:\n%s", want, last)
		return ""
	}
	send := func(keys ...string) { tmux(append([]string{"send-keys", "-t", "review:0.0"}, keys...)...) }

	wait("HUG · repo · overview")
	send("#aaaaaa")
	wait("JUMP TO ID")
	send("Enter")
	wait("Approve the status colours")
	send("t")
	wait("Take aaaaaa for review as Jørgen?")
	send("Enter")
	screen := wait("aaaaaa moved to ASKED") // taken for review by you: out of NEXT UP (24403z)
	file := func(id string) string {
		matches, _ := filepath.Glob(filepath.Join(tasks, "*"+id+"*.md"))
		// Layout 3 (w40cmk): C/ID.md.
		more, _ := filepath.Glob(filepath.Join(tasks, "?", id+".md"))
		matches = append(matches, more...)
		if len(matches) != 1 {
			t.Fatalf("%s: %v", id, matches)
		}
		b, _ := os.ReadFile(matches[0])
		return string(b)
	}
	a := file("aaaaaa")
	for _, want := range []string{"state: \"doing\"", "stage: \"review\"", "review_by: \"Jørgen\"", "doing_since: ", "todo/review -> doing/review (take for review)"} {
		if !strings.Contains(a, want) {
			t.Fatalf("after t on todo/review, the file lacks %q:\n%s\n%s", want, a, screen)
		}
	}
	commit := run("git", "-C", tasks, "log", "-1", "--format=%s%n%(trailers:key=Model,valueonly)")
	if !strings.Contains(commit, "aaaaaa") || !strings.Contains(commit, "human") {
		t.Fatalf("take commit: %q", commit)
	}

	// t on my own task: give it back, keeping its stage.
	send("Escape")
	wait("HUG · repo · overview   ") // # opened it over the overview (ht3cdx)
	send("#bbbbbb")
	wait("JUMP TO ID")
	send("Enter")
	wait("My own work")
	send("t")
	wait("Give back bbbbbb to todo (ready)?")
	send("Enter")
	wait("bbbbbb moved to NEXT UP") // given back: out of IN PROGRESS (24403z)
	b := file("bbbbbb")
	for _, want := range []string{"state: \"todo\"", "stage: \"ready\"", "doing/active -> todo/ready (give back; was assignee: Jørgen)"} {
		if !strings.Contains(b, want) {
			t.Fatalf("after t on my own task, the file lacks %q:\n%s", want, b)
		}
	}
	if front := b[:strings.Index(b[4:], "---")+4]; strings.Contains(front, "doing_since") || !strings.Contains(front, "assignee: []") {
		t.Fatalf("doing_since or the assignee stayed:\n%s", b)
	}
	// Taken for review and given back: todo/review again, review_by kept.
	send("Escape")
	wait("HUG · repo · overview   ") // # opened it over the overview (ht3cdx)
	send("#aaaaaa")
	wait("JUMP TO ID")
	send("Enter")
	send("t")
	wait("Give back aaaaaa to todo (review)?")
	send("Enter")
	wait("gave back aaaaaa")
	if a = file("aaaaaa"); !strings.Contains(a, "state: \"todo\"") || !strings.Contains(a, "stage: \"review\"") {
		t.Fatalf("review given back:\n%s", a)
	}
}
