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

// qcq3tm in a real terminal: the hug binary in tmux at 100x30 on copies of
// three tasks from hug's own board as Jørgen opened them
// (testdata/qcq3tm-board):
//   - tb0dra asked of jhf on check 5 with --show agentsection-proposal.md
//     (hug/tasks 7349821f^), with tzc9hb, its dependency;
//   - hjbfzd asked on check 5, then on check 6 with --show
//     capture-wording-100x30.ans (hug/tasks 4aea4bdb);
//   - p87ngv with old asked_* fields, question 11 on check 5 and question
//     12 in its own block with --show frame-1-100x30.ans (hug/tasks 069928d0).

// askedTmux is a hug tui in tmux on the qcq3tm board; the server and the
// session end with the test.
type askedTmux struct {
	t    *testing.T
	tmux func(args ...string) string
	// frames, when set, is where save keeps each screen (zbdbtm), else
	// $QCQ3TM_CAPTURES.
	frames string
}

func newAskedTmux(t *testing.T, name string) *askedTmux {
	t.Helper()
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-qcq3tm-")
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
	if os.Getenv("QCQ3TM_CAPTURES") != "" {
		// Captures for Jørgen (qcq3tm): in the real theme's colours.
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
	run("git", "init", "-q", "-b", "main")
	run("git", "commit", "-q", "--allow-empty", "-m", "fixture")
	run(binary, "init")
	tasks := filepath.Join(repo, ".hug", "tasks")
	copyBoard(t, filepath.Join("testdata", "qcq3tm-board"), tasks)
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "tb0dra, hjbfzd and p87ngv from hug's board")

	socket := tmuxtest.Label("qcq3tm" + name)
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
	a := &askedTmux{t: t, tmux: tmux}
	a.wait("start", []string{"HUG · repo"})
	return a
}

func (a *askedTmux) screen() string { return a.tmux("capture-pane", "-p", "-t", "q:0.0") }

// wait is the screen once it shows every want and none of the gone.
func (a *askedTmux) wait(what string, want []string, gone ...string) string {
	a.t.Helper()
	last := ""
	for deadline := time.Now().Add(12 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		last = a.screen()
		ok := true
		for _, w := range want {
			ok = ok && strings.Contains(last, w)
		}
		for _, g := range gone {
			ok = ok && !strings.Contains(last, g)
		}
		if ok {
			a.save(what)
			return last
		}
	}
	a.t.Fatalf("%s: never showed %q without %q:\n%s", what, want, gone, last)
	return ""
}

// save keeps the screen with its colours as an .ans in
// $QCQ3TM_CAPTURES, when set: the captures attached to qcq3tm.
func (a *askedTmux) save(what string) {
	dir := os.Getenv("QCQ3TM_CAPTURES")
	if a.frames != "" {
		dir = a.frames
	}
	if dir == "" {
		return
	}
	name := strings.NewReplacer(" ", "-", "(", "", ")", "", "'", "", ",", "", "/", "-", "#", "-", "▸", "").Replace(what)
	ans := a.tmux("capture-pane", "-e", "-p", "-t", "q:0.0")
	if err := os.WriteFile(filepath.Join(dir, name+".ans"), []byte(ans), 0o644); err != nil {
		a.t.Fatal(err)
	}
}

func (a *askedTmux) send(keys ...string) {
	a.tmux(append([]string{"send-keys", "-t", "q:0.0"}, keys...)...)
}

// open jumps to id and waits for its title.
func (a *askedTmux) open(id, title string) {
	a.t.Helper()
	a.send("#" + id)
	a.wait("jump to "+id, []string{"JUMP TO ID"})
	a.send("Enter")
	a.wait("open "+id, []string{title})
	time.Sleep(300 * time.Millisecond) // the first frame of the page settles
	a.save(id + " as it opens")
}

// qcq3tm checks 1 and 2: opening tb0dra and hjbfzd lands on the question
// asked of jhf, below the fold, focused; no file is drawn above the page;
// the question's file is one link line, Enter shows it in the file frame
// and Esc goes back to the question.
func TestTmuxAskedTaskOpensOnItsQuestion(t *testing.T) {
	a := newAskedTmux(t, "open")

	a.open("tb0dra", "Agents reflect on the instructions and propose fixes")
	tb0dra := []string{"> ? ☐ 5. after a week of use", "Approve these three additions to the AGENTS.md section", "→ agentsection-proposal.md · Enter shows", "└ piglet"}
	inline := []string{"▸ agentsection-proposal.md", "# Three additions to the"}
	a.wait("tb0dra opens on its question (check 1), its file not drawn above the page (check 2)", tb0dra, inline...)
	a.send("Enter")
	a.wait("tb0dra Enter shows the file in the file frame (check 2)", []string{"tb0dra › t/tb0dra/agentsection-proposal.md", "agentsection-proposal.md · 3.0 KiB · 1 of 1", "Esc question"})
	a.send("Escape")
	a.wait("tb0dra Esc goes back to the question (check 2)", tb0dra, inline...)
	a.send("Escape")
	// # opened it over the overview, so Esc goes back there (ht3cdx).
	a.wait("back to the rows", []string{"HUG · repo · overview   4/4 tasks"}, "→ agentsection-proposal.md")

	// hjbfzd: the oldest question is on check 5; check 6's has a file.
	a.open("hjbfzd", "Screen capture: attach to a chosen task, pick one, or create a new task")
	a.wait("hjbfzd opens on its question (check 1)", []string{"> ? ☐ 5. Jørgen confirms on his board after release", "└ question 1 · crab"}, "capture, as it was taken", "▸ capture-wording-100x30.ans")
	a.send("j")
	q6 := []string{"> ? ☐ 6.", "→ capture-wording-100x30.ans · Enter shows", "└ question 2 · crab"}
	a.wait("hjbfzd j reaches check 6 and its link", q6, "capture, as it was taken")
	a.send("Enter")
	a.wait("hjbfzd Enter shows the capture in the file frame (check 2)", []string{"hjbfzd › h/hjbfzd/capture-wording-100x30.ans", "capture-wording-100x30.ans · 2.7 KiB · 1 of 2"})
	a.send("Escape")
	a.wait("hjbfzd Esc goes back to check 6's question (check 2)", q6, "capture, as it was taken")
}

// qcq3tm check 4 (Jørgen 2026-11-20 10:13Z on p87ngv: "nothing happens, I
// can't enter any text"): with the page scrolled away from question 11,
// a opens the box under it, in view, and what is typed shows there.
func TestTmuxAnswerBoxOpensInView(t *testing.T) {
	a := newAskedTmux(t, "answer")
	a.open("p87ngv", "Inline marks on a task's text; resolved dialogue moves to a history file")
	a.send("Home") // the top of the page, as Jørgen had it
	a.wait("p87ngv at the top", []string{"Inline marks on a task's text", "└ question 12 · seedling"}, "└ question 11 · crab")
	a.send("a")
	a.wait("p87ngv a opens the answer box in view (check 4)", []string{"? ☐ 5. Jørgen approves a mock", "answer:"})
	a.send("-l", "typed by qcq3tm")
	a.wait("p87ngv typing shows in the box (check 4)", []string{"answer: typed by qcq3tm"})
	a.send("Escape")
	a.wait("p87ngv Esc closes the box, nothing sent", []string{"└ question 11 · crab"}, "answer: typed by qcq3tm")
}

// copyBoard writes every file under from into to, over what hug init wrote
// there (hug.toml).
func copyBoard(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(from, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(to, rel)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
