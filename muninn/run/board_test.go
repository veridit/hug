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

// bbxdd7 check 2: a request hug already delivered to the coordinator's
// session is not named again by the board cue; one not delivered still is.
func TestBoardCueSkipsDeliveredRequest(t *testing.T) {
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
	md := "---\nid: \"abc123\"\ntitle: \"Tags\"\nstate: \"todo\"\n---\n\n## Goal\nG\n\n## Acceptance\n- [ ] one\n\n" +
		"## Requests\n- 2026-11-18T12:00 request 1 by jhf:\n  make it per board\n- 2026-11-18T12:05 request 2 by jhf:\n  shorter title\n\n" +
		"## History\n- 2026-11-18T12:00 request 1 delivered to crab (session_crab_1_aa)\n- 2026-11-18T12:05 request 2 not delivered: no jcode session named crab on this machine\n"
	if err := os.WriteFile(filepath.Join(repo, ".hug/tasks/a.abc123.tags.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	git(filepath.Join(repo, ".hug/tasks"), "add", "-A")
	git(filepath.Join(repo, ".hug/tasks"), "commit", "-q", "-m", "t")
	s, err := board.Open(context.Background(), repo)
	if err != nil || s == nil {
		t.Fatalf("open: %v %v", s, err)
	}
	defer s.Close()
	if ts := boardTasksFor(s.Tasks, "nautilus"); len(ts) != 1 || len(ts[0].Requests) != 2 {
		t.Fatalf("another session lost requests: %+v", ts)
	}
	in := recall.BoardInput{Tasks: boardTasksFor(s.Tasks, "crab"), Coordinator: "crab", Self: recall.BoardSelf{Name: "crab"}}
	db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	e := &recall.Engine{DB: db}
	var lines []string
	for i := 0; i < 3; i++ {
		line, _ := e.BoardCue("s", "go on", "", time.Now(), func(time.Time, bool) *recall.BoardInput { c := in; return &c })
		lines = append(lines, line)
	}
	all := strings.Join(lines, "\n")
	if strings.Contains(all, "make it per board") || strings.Contains(all, "(request 1)") {
		t.Fatalf("the cue names request 1, already delivered to crab:\n%s", all)
	}
	if !strings.Contains(all, "shorter title") {
		t.Fatalf("the cue lost request 2, not delivered:\n%s", all)
	}
}

// 41hhgn check 11: an outcome hug already delivered to the coordinator's
// session (or that session acknowledged) is not said again by the board
// cue: it is in that session's transcript. One that could not be delivered
// is still said, once, as before.
func TestBoardCueSkipsDeliveredOutcome(t *testing.T) {
	for _, tc := range []struct {
		receipt string
		said    bool
	}{
		{"- 2026-11-18T01:00 outcome 1 delivered to crab (session_crab_1_aa)\n", false},
		{"- 2026-11-18T01:00 outcome 1 not delivered to crab: no jcode session named crab on this machine\n", true},
		{"", true},
	} {
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
		md := "---\nid: \"abc123\"\ntitle: \"Tags\"\nstate: \"doing\"\nassignee: [nautilus]\n---\n\n## Goal\nG\n\n## Acceptance\n- [ ] one\n- [ ] two\n\n" +
			"## History\n- 2026-11-18T00:50 answered asked (check 2), by jhf, for nautilus (doing/active):\n  > which rows?\n  the open ones\n" +
			"- 2026-11-18T00:59 nautilus read the answer: hug task note abc123 \"rows now list the open ones first\"\n" + tc.receipt
		if err := os.WriteFile(filepath.Join(repo, ".hug/tasks/a.abc123.tags.md"), []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
		git(filepath.Join(repo, ".hug/tasks"), "add", "-A")
		git(filepath.Join(repo, ".hug/tasks"), "commit", "-q", "-m", "t")
		s, err := board.Open(context.Background(), repo)
		if err != nil || s == nil {
			t.Fatalf("open: %v %v", s, err)
		}
		in := recall.BoardInput{Tasks: boardTasksFor(s.Tasks, "crab"), Coordinator: "crab", Self: recall.BoardSelf{Name: "crab"}}
		s.Close()
		db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
		if err != nil {
			t.Fatal(err)
		}
		e := &recall.Engine{DB: db}
		var lines []string
		for i := 0; i < 3; i++ {
			line, _ := e.BoardCue("s", "go on", "", time.Now(), func(time.Time, bool) *recall.BoardInput { c := in; return &c })
			lines = append(lines, line)
		}
		db.Close()
		all := strings.Join(lines, "\n")
		if said := strings.Contains(all, "nautilus acted on jhf's answer on abc123"); said != tc.said {
			t.Fatalf("receipt %q: the cue said the outcome = %v, want %v:\n%s", tc.receipt, said, tc.said, all)
		}
	}
}

// 5yh7hr: a review hug already delivered to the reviewer's session (or
// that session acknowledged) is not said again by the board cue; one that
// could not be delivered still is, once.
func TestBoardCueSkipsDeliveredReview(t *testing.T) {
	for _, tc := range []struct {
		receipt string
		said    bool
	}{
		{"- 2026-11-27T10:00 review 1 delivered to crab (session_crab_1_aa)\n", false},
		{"- 2026-11-27T10:00 review 1 not delivered to crab: no jcode session named crab on this machine\n- 2026-11-27T10:05 review 1 acknowledged by crab\n", false},
		{"- 2026-11-27T10:00 review 1 not delivered to crab: no jcode session named crab on this machine\n", true},
	} {
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
		md := "---\nid: \"abc123\"\ntitle: \"Tags\"\nstate: \"doing\"\nstage: \"review\"\nassignee: [\"wolf\"]\nreview_by: \"crab\"\n---\n\n## Goal\nG\n\n## Acceptance\n- [x] one\n\n" +
			"## History\n- 2026-11-27T10:00 doing/active -> doing/review (review: wolf ticked the last own check)\n" + tc.receipt
		if err := os.WriteFile(filepath.Join(repo, ".hug/tasks/a.abc123.tags.md"), []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
		git(filepath.Join(repo, ".hug/tasks"), "add", "-A")
		git(filepath.Join(repo, ".hug/tasks"), "commit", "-q", "-m", "t")
		s, err := board.Open(context.Background(), repo)
		if err != nil || s == nil {
			t.Fatalf("open: %v %v", s, err)
		}
		if ts := boardTasksFor(s.Tasks, "otter"); len(ts) != 1 || ts[0].Step.Kind != "review" {
			t.Fatalf("another session lost the step: %+v", ts)
		}
		in := recall.BoardInput{Tasks: boardTasksFor(s.Tasks, "crab"), Coordinator: "crab", Self: recall.BoardSelf{Name: "crab"}}
		db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
		if err != nil {
			t.Fatal(err)
		}
		e := &recall.Engine{DB: db}
		line, _ := e.BoardCue("s", "go on", "", time.Now(), func(time.Time, bool) *recall.BoardInput { c := in; return &c })
		db.Close()
		s.Close()
		if said := strings.Contains(line, "asks you to review abc123"); said != tc.said {
			t.Fatalf("receipt %q: review said %v, want %v: %q", tc.receipt, said, tc.said, line)
		}
	}
}
