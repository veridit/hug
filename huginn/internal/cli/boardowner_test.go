package cli

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ygay9c: a copy of a repo keeps its board checkout's link to the original
// (wyvern 2026-11-18: hug in an APFS copy committed 37 notes to the
// original's board). hug in the copy refuses before it reads or writes,
// names both repositories, and its fix line, run as given, makes the copy's
// own board work; the original is untouched throughout.
func TestBoardOfAnotherRepoRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("copies the repo with cp -Rp")
	}
	t.Parallel()
	orig := app(t)
	execute(t, orig, "init")
	id := execute(t, orig, "new", "Copied", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	origBoard := filepath.Join(orig.Cwd, TaskDir)
	origHead := git(t, origBoard, "rev-parse", "HEAD")

	dir := filepath.Join(t.TempDir(), "copy")
	if out, e := exec.Command("cp", "-Rp", orig.Cwd, dir).CombinedOutput(); e != nil {
		t.Fatalf("cp: %v %s", e, out)
	}
	cp := *orig
	cp.Cwd = dir
	copyBoard := filepath.Join(dir, TaskDir)

	_, err := cp.Execute(context.Background(), []string{"note", id, "from the copy"})
	if err == nil {
		t.Fatalf("note in the copy went through")
	}
	msg := err.Error()
	_, after, _ := strings.Cut(msg, "\nfix: ")
	fix, _, _ := strings.Cut(after, "\n")
	t.Logf("refusal:\n%s", msg)
	// The message names the original as git resolves it (a symlinked
	// TMPDIR, /private/var on macOS): the same folder, not the same string.
	origShown, _ := filepath.EvalSymlinks(orig.Cwd)
	if !strings.Contains(msg, "belongs to another repository") || !(strings.Contains(msg, orig.Cwd) || strings.Contains(msg, origShown)) || !strings.Contains(fix, "worktree repair") {
		t.Fatalf("refusal:\n%s\nfix: %s", msg, fix)
	}
	if now := git(t, origBoard, "rev-parse", "HEAD"); now != origHead {
		t.Fatalf("the original's board moved: %s", git(t, origBoard, "log", "--oneline", "-3"))
	}
	if s := git(t, origBoard, "status", "--porcelain"); s != "" {
		t.Fatalf("the original's board is dirty: %s", s)
	}

	// The fix, as given.
	if out, e := exec.Command("sh", "-c", fix).CombinedOutput(); e != nil {
		t.Fatalf("fix %q: %v %s", fix, e, out)
	}
	execute(t, &cp, "note", id, "from the copy")
	if f := git(t, copyBoard, "log", "-1", "--format=%s"); !strings.Contains(f, id) {
		t.Fatalf("the copy's board did not get the note: %s", f)
	}
	if now := git(t, origBoard, "rev-parse", "HEAD"); now != origHead {
		t.Fatalf("after the fix the original's board moved")
	}
	// The original keeps working on its own board.
	execute(t, orig, "note", id, "from the original")
	if now := git(t, origBoard, "rev-parse", "HEAD"); now == origHead {
		t.Fatalf("the original's note did not land")
	}
}
