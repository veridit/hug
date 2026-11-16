package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 065wwn: from a linked worktree (git worktree add --detach), hug task ls,
// new and take read and write the main worktree's board, and no second
// checkout of hug/tasks appears in git worktree list. Covers both a board
// that exists before the linked worktree and one created from it.
func TestLinkedWorktreeUsesMainBoard(t *testing.T) {
	for _, initFrom := range []string{"main", "linked"} {
		t.Run("init from "+initFrom, func(t *testing.T) {
			main := app(t)
			mainDir, _ := filepath.EvalSymlinks(main.Cwd)
			linkedDir := filepath.Join(t.TempDir(), "linked")
			git(t, mainDir, "worktree", "add", "--detach", linkedDir, "HEAD")
			linked := &App{Cwd: linkedDir, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
			if initFrom == "main" {
				execute(t, main, "init")
			} else {
				execute(t, linked, "init")
			}
			created := execute(t, linked, "new", "From the linked worktree", "--goal", "shared board", "--accept", "seen from main")
			id := created["task"].(Reply)["id"].(string)
			execute(t, main, "new", "From the main worktree", "--goal", "shared board", "--accept", "seen from linked")
			ls := execute(t, linked, "ls")
			if !strings.Contains(jsonString(t, ls), "From the main worktree") {
				t.Fatalf("linked ls misses the main task: %v", ls)
			}
			execute(t, linked, "take", id)
			shown := execute(t, main, "show", id)
			if s := jsonString(t, shown); !strings.Contains(s, `"assignee":["tester"]`) || !strings.Contains(s, `"status":"doing"`) || !strings.Contains(s, "From the linked worktree") {
				t.Fatalf("main does not see the linked take: %s", s)
			}
			if _, e := os.Stat(filepath.Join(linkedDir, ".hug")); !os.IsNotExist(e) {
				t.Fatalf("linked worktree got its own .hug: %v", e)
			}
			if _, e := os.Stat(filepath.Join(mainDir, TaskDir, ".git")); e != nil {
				t.Fatalf("board not in the main worktree: %v", e)
			}
			list := git(t, mainDir, "worktree", "list", "--porcelain")
			if n := strings.Count(list, "worktree "); n != 3 {
				t.Fatalf("want main, linked and one task checkout, got %d:\n%s", n, list)
			}
			// The CLI text path, as a user types it in the linked worktree.
			out := &bytes.Buffer{}
			linked.Out, linked.Err = out, out
			if code := linked.Run(context.Background(), []string{"ls"}); code != 0 || !strings.Contains(out.String(), "From the main worktree") {
				t.Fatalf("hug task ls in linked worktree: exit %d\n%s", code, out)
			}
		})
	}
}

func jsonString(t *testing.T, r Reply) string {
	t.Helper()
	b, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
