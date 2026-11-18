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
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/store"
)

// A person's answer on the real board reaches the asker's recall cue
// (3jtfw9 check 3): board.Open reads it from History, BoardTask carries it,
// BoardCue says it to guppy and to no one else.
func TestBoardAnswerReachesAsker(t *testing.T) {
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
	md := "---\nid: \"abc123\"\ntitle: \"Board nudges\"\nstate: \"doing\"\nassignee: [\"guppy\"]\nbranch: \"main\"\n---\n\n## Goal\nG\n\n## Acceptance\n- [x] one\n- [ ] two\n\n## History\n" +
		"- 2026-11-17T13:30 answered waiting: \"which colour?\" -> \"green\", by jhf, for guppy (doing/waiting -> doing/active)\n"
	if err := os.WriteFile(filepath.Join(repo, ".hug/tasks/a.abc123.board-nudges.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	git(filepath.Join(repo, ".hug/tasks"), "add", "-A")
	git(filepath.Join(repo, ".hug/tasks"), "commit", "-q", "-m", "t")
	s, err := board.Open(context.Background(), repo)
	if err != nil || s == nil {
		t.Fatalf("open: %v %v", s, err)
	}
	defer s.Close()
	in := recall.BoardInput{Tasks: boardTasks(s.Tasks)}
	for _, who := range []string{"guppy", "crab"} {
		db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		e := &recall.Engine{DB: db}
		in.Self = recall.BoardSelf{Name: who}
		c := in
		line, _ := e.BoardCue("s", "go on", "", time.Now(), func(time.Time, bool) *recall.BoardInput { return &c })
		want := "jhf answered your question on abc123: green (waiting: which colour?). hug task show abc123"
		if who == "guppy" && line != want {
			t.Fatalf("guppy: got %q, want %q", line, want)
		}
		if who == "crab" && strings.Contains(line, "answered") {
			t.Fatalf("crab got guppy's answer: %q", line)
		}
	}
}

// nudge_board = false in config.json turns board nudges off: the hook gets
// no board source, so BoardCue returns no line (qawmnb).
func TestNudgeBoardOff(t *testing.T) {
	for _, tc := range []struct {
		json string
		on   bool
	}{
		{`{}`, true},
		{`{"nudge_board": true}`, true},
		{`{"nudge_board": false}`, false},
	} {
		home := t.TempDir()
		t.Setenv("HUG_HOME", home)
		if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(config.ConfigDir(), "config.json"), []byte(tc.json), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.BoardNudges() != tc.on {
			t.Fatalf("%s: BoardNudges() = %v", tc.json, cfg.BoardNudges())
		}
		bs := boardSource(context.Background(), cfg.BoardNudges(), t.TempDir(), "guppy", "", func(time.Time) harness.Activity { return harness.Activity{} })
		if (bs != nil) != tc.on {
			t.Fatalf("%s: board source present = %v", tc.json, bs != nil)
		}
	}
}
