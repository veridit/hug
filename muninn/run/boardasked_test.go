package run

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/board"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/store"
)

// 72n8m4 check 2 (crab 2026-11-20: the cue said "every check of ddpxs8
// holds. Close it: hug task done ddpxs8" while ddpxs8 asked jhf a
// question; 4cn9vv and gvkpzt the same): the board cue never offers done
// for a task with an open question, whether in ## Questions or the asked
// stage of a task asked before q2adr8. A task with every check ticked and
// no open question still gets the line.
func TestBoardCueNoDoneWhileAsked(t *testing.T) {
	repo := t.TempDir()
	git := func(dir string, args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git(repo, "init", "-q", "-b", "main")
	git(repo, "commit", "-q", "--allow-empty", "-m", "init")
	git(repo, "worktree", "add", "-q", "--orphan", "-b", "hug/tasks", ".hug/tasks")
	head := func(id, extra string) string {
		return "---\nid: \"" + id + "\"\ntitle: \"Release " + id + "\"\nstate: \"doing\"\nassignee: [\"crab\"]\nbranch: \"main\"\n" + extra + "---\n\n## Goal\nG\n\n## Acceptance\n- [x] one\n- [x] two\n\n"
	}
	asked := "stage: \"asked\"\nasked_what: \"Does it read right?\"\nasked_for: \"jhf\"\nasked_since: \"2026-11-20T08:00\"\nasked_by: \"crab\"\n"
	files := map[string]string{
		// Asked through ## Questions (q2adr8).
		"d.ddpxs8.release.md": head("ddpxs8", asked) + "## Questions\n- 2026-11-20T08:00 question 1 by crab to jhf:\n  Does it read right?\n\n## History\n",
		// Asked before q2adr8: the asked stage, no section.
		"g.gvkpzt.release.md": head("gvkpzt", asked) + "## History\n",
		// Its question answered: done is right.
		"c.cccccc.release.md": head("cccccc", "") + "## Questions\n- 2026-11-20T08:00 question 1 by crab to jhf:\n  Does it read right?\n  answered 2026-11-20T08:10 by jhf:\n    yes\n\n## History\n",
	}
	for name, md := range files {
		if err := os.WriteFile(filepath.Join(repo, ".hug/tasks", name), []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(filepath.Join(repo, ".hug/tasks"), "add", "-A")
	git(filepath.Join(repo, ".hug/tasks"), "commit", "-q", "-m", "t")
	s, err := board.Open(context.Background(), repo)
	if err != nil || s == nil {
		t.Fatalf("open: %v %v", s, err)
	}
	defer s.Close()
	for _, only := range []string{"ddpxs8", "gvkpzt", "cccccc"} {
		var ts []board.Task
		for _, x := range s.Tasks {
			if x.ID == only {
				ts = append(ts, x)
			}
		}
		db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		e := &recall.Engine{DB: db}
		in := recall.BoardInput{Tasks: boardTasks(ts), Self: recall.BoardSelf{Name: "crab"}}
		line, kind := e.BoardCue("s", "go on", "", time.Now(), func(time.Time, bool) *recall.BoardInput { return &in })
		offers := strings.Contains(line, "hug task done "+only)
		if only == "cccccc" && (kind != "done" || !offers) {
			t.Fatalf("%s (question answered): kind %q line %q", only, kind, line)
		}
		if only != "cccccc" && (kind == "done" || offers) {
			t.Fatalf("%s has a question open to jhf, but the cue offers done: %q", only, line)
		}
	}
}
