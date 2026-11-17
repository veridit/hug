package cli

import (
	"bytes"
	"encoding/json"
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
	t.Parallel()
	main := app(t)
	mainDir, _ := filepath.EvalSymlinks(main.Cwd)
	execute(t, main, "init")
	linkedDir := filepath.Join(t.TempDir(), "feat-x")
	git(t, mainDir, "worktree", "add", "-b", "feat/x", linkedDir)
	linkedDir, _ = filepath.EvalSymlinks(linkedDir)
	linked := &App{Cwd: filepath.Join(linkedDir), In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: main.Env}

	id := execute(t, linked, "new", "Worktree work", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	tk := execute(t, linked, "take", id)["task"].(Reply)
	if tk["branch"] != "feat/x" || tk["worktree"] != linkedDir {
		t.Fatalf("branch %v worktree %v, want feat/x %s", tk["branch"], tk["worktree"], linkedDir)
	}
	f := taskFile(t, main, id)
	// Frontmatter values are JSON strings: a Windows path's backslashes are escaped.
	quoted, _ := json.Marshal(linkedDir)
	if !strings.Contains(f, `branch: "feat/x"`) || !strings.Contains(f, `worktree: `+string(quoted)) {
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
	other := execute(t, main, "new", "Main work", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	if tk := execute(t, main, "take", other)["task"].(Reply); tk["branch"] != "main" {
		t.Fatalf("main take: %v", tk)
	}
	if _, out := runText(main, "ls"); !strings.Contains(out, "Main work  [tester · test-model]\n") {
		t.Fatalf("ls on main branch:\n%s", out)
	}

	// Flags win over detection.
	third := execute(t, main, "new", "Flagged", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	if tk := execute(t, main, "take", third, "--branch", "feat/y", "--worktree", "/elsewhere")["task"].(Reply); tk["branch"] != "feat/y" || tk["worktree"] != "/elsewhere" {
		t.Fatalf("flags: %v", tk)
	}
}

// x1t1t1 check 6: lint flags a doing task whose branch is merged into main
// or deleted, with fix: hug task done ID; an unmerged branch is not flagged.
func TestLintFlagsDoingTaskOnMergedOrDeletedBranch(t *testing.T) {
	t.Parallel()
	main := app(t)
	mainDir, _ := filepath.EvalSymlinks(main.Cwd)
	execute(t, main, "init")
	take := func(title, branch string) string {
		t.Helper()
		id := execute(t, main, "new", title, "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
		execute(t, main, "take", id, "--branch", branch)
		return id
	}
	git(t, mainDir, "branch", "feat/merged")
	git(t, mainDir, "branch", "feat/open")
	git(t, mainDir, "checkout", "-q", "feat/open")
	git(t, mainDir, "commit", "-q", "--allow-empty", "-m", "unmerged work")
	git(t, mainDir, "checkout", "-q", "main")
	merged, open, gone := take("Merged work", "feat/merged"), take("Open work", "feat/open"), take("Gone work", "feat/gone")

	code, out := runText(main, "lint", "--code", "branch-gone")
	for _, want := range []string{
		"branch feat/merged is merged into main, but " + merged + " is still doing",
		"fix: hug task done " + merged,
		"branch feat/gone is deleted, but " + gone + " is still doing",
		"fix: hug task done " + gone,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("lint (exit %d) misses %q:\n%s", code, want, out)
		}
	}
	if strings.Contains(out, "feat/open") || strings.Count(out, "[branch-gone]") != 2 {
		t.Fatalf("unmerged branch %s flagged:\n%s", open, out)
	}
	// Closing the task clears its finding.
	execute(t, main, "check", merged, "1")
	execute(t, main, "done", merged)
	if _, out := runText(main, "lint"); strings.Contains(out, "feat/merged") {
		t.Fatalf("done task still flagged:\n%s", out)
	}
}
