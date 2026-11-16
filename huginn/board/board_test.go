package board_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/huginn/board"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

const doing = `---
id: "abc123"
title: "Board nudges"
status: "doing"
assignee: ["guppy"]
branch: "main"
---

## Goal
G

## Acceptance
- [x] one
- [ ] two
`

func TestOpenReadsBoardWithoutWriting(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "init")
	if _, err := board.Open(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	if s, _ := board.Open(context.Background(), repo); s != nil {
		t.Fatal("no board: want nil snapshot")
	}
	if _, err := os.Stat(filepath.Join(repo, ".hug")); !os.IsNotExist(err) {
		t.Fatalf("Open created the board folder: %v", err)
	}
	git(t, repo, "worktree", "add", "-q", "--orphan", "-b", "hug/tasks", ".hug/tasks")
	os.WriteFile(filepath.Join(repo, ".hug/tasks/a.abc123.board-nudges.md"), []byte(doing), 0o644)
	git(t, filepath.Join(repo, ".hug/tasks"), "add", "-A")
	git(t, filepath.Join(repo, ".hug/tasks"), "commit", "-q", "-m", "t")
	sub := filepath.Join(repo, "src")
	os.MkdirAll(sub, 0o755)
	s, err := board.Open(context.Background(), sub)
	if err != nil || s == nil {
		t.Fatalf("open: %v %v", s, err)
	}
	defer s.Close()
	if len(s.Tasks) != 1 || s.Tasks[0].ID != "abc123" || s.Tasks[0].Status != "doing" || s.Tasks[0].Checks != 2 || s.Tasks[0].Open != 1 || s.Tasks[0].Branch != "main" {
		t.Fatalf("%+v", s.Tasks)
	}
	if s.Place.Branch != "main" || s.Place.Linked {
		t.Fatalf("%+v", s.Place)
	}
	if fs, err := s.Findings(); err != nil {
		t.Fatal(err)
	} else {
		for _, f := range fs {
			if f.Fix == "" || f.TaskID != "abc123" {
				t.Fatalf("finding without fix or task: %+v", f)
			}
		}
	}
	out, _ := exec.Command("git", "-C", filepath.Join(repo, ".hug/tasks"), "status", "--porcelain").Output()
	if len(out) != 0 {
		t.Fatalf("board changed: %s", out)
	}
}

func TestFilesSnapshot(t *testing.T) {
	f := board.NewFiles()
	f.Set("a.abc123.board-nudges.md", []byte(doing))
	f.Set("notes.txt", []byte("x"))
	if s := f.Snapshot(); len(s.Tasks) != 1 || s.Tasks[0].Assignee[0] != "guppy" {
		t.Fatalf("%+v", s.Tasks)
	}
	f.Delete("a.abc123.board-nudges.md")
	if s := f.Snapshot(); len(s.Tasks) != 0 {
		t.Fatalf("%+v", s.Tasks)
	}
}
