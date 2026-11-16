package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// x1t1t1: hug task take in a linked worktree on branch feat/x records the
// branch and the worktree folder, without flags. show names the branch;
// ls rows name it because it is not the main branch; the main checkout's
// own branch is recorded but not repeated on ls rows. --branch/--worktree
// override detection.
func TestTakeRecordsBranchAndWorktree(t *testing.T) {
	main := app(t)
	mainDir, _ := filepath.EvalSymlinks(main.Cwd)
	execute(t, main, "init")
	linkedDir := filepath.Join(t.TempDir(), "feat-x")
	git(t, mainDir, "worktree", "add", "-b", "feat/x", linkedDir)
	linkedDir, _ = filepath.EvalSymlinks(linkedDir)
	linked := &App{Cwd: filepath.Join(linkedDir), In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}

	id := execute(t, linked, "new", "Worktree work", "--goal", "G", "--accept", "C")["task"].(Reply)["id"].(string)
	tk := execute(t, linked, "take", id)["task"].(Reply)
	if tk["branch"] != "feat/x" || tk["worktree"] != linkedDir {
		t.Fatalf("branch %v worktree %v, want feat/x %s", tk["branch"], tk["worktree"], linkedDir)
	}
	f := taskFile(t, main, id)
	if !strings.Contains(f, `branch: "feat/x"`) || !strings.Contains(f, `worktree: "`+linkedDir+`"`) {
		t.Fatalf("task file:\n%s", f)
	}
	if code, out := runText(main, "show", id); code != 0 || !strings.Contains(out, "taken by: tester · test-model · feat/x\n") {
		t.Fatalf("show: %d\n%s", code, out)
	}
	if code, out := runText(main, "ls"); code != 0 || !strings.Contains(out, "Worktree work  [tester · test-model · feat/x]\n") {
		t.Fatalf("ls: %d\n%s", code, out)
	}
	if s := jsonString(t, execute(t, main, "ls")); !strings.Contains(s, `"branch":"feat/x"`) {
		t.Fatalf("ls --json misses the branch: %s", s)
	}

	// On the main checkout: recorded, but ls rows do not repeat "main".
	other := execute(t, main, "new", "Main work", "--goal", "G", "--accept", "C")["task"].(Reply)["id"].(string)
	if tk := execute(t, main, "take", other)["task"].(Reply); tk["branch"] != "main" {
		t.Fatalf("main take: %v", tk)
	}
	if _, out := runText(main, "ls"); !strings.Contains(out, "Main work  [tester · test-model]\n") {
		t.Fatalf("ls on main branch:\n%s", out)
	}

	// Flags win over detection.
	third := execute(t, main, "new", "Flagged", "--goal", "G", "--accept", "C")["task"].(Reply)["id"].(string)
	if tk := execute(t, main, "take", third, "--branch", "feat/y", "--worktree", "/elsewhere")["task"].(Reply); tk["branch"] != "feat/y" || tk["worktree"] != "/elsewhere" {
		t.Fatalf("flags: %v", tk)
	}
}
