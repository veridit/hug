package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/gitbranch"
)

// fcf4fg check 1: a sync stopped mid-rebase (Seid 2026-11-16: rebase-merge
// with only an autostash, every agent's write failing) is ended by the
// next write, which succeeds; the note and the autostashed change both
// reach the branch. A concurrent edit without the board turn makes the
// rebase autostash (rebase.autoStash, user config), and a smudge filter
// that fails stops git while it resets that file, where a kill landed.
func TestWriteEndsInterruptedSyncRebase(t *testing.T) {
	a, b, _ := clones(t)
	id := newTicket(t, a, "Stuck rebase")
	execute(t, a, "sync")
	execute(t, b, "sync")
	settled(t, a)
	settled(t, b)
	offline(t, b, func() {
		offline(t, a, func() {
			execute(t, a, "note", id, "from a")
			execute(t, a, "sync")
		})
		board := filepath.Join(b.Cwd, TaskDir)
		file := taskPath(t, b, id)
		rel, err := filepath.Rel(board, file)
		if err != nil {
			t.Fatal(err)
		}
		git(t, b.Cwd, "config", "rebase.autoStash", "true")
		git(t, b.Cwd, "config", "filter.die.clean", "cat")
		git(t, b.Cwd, "config", "filter.die.smudge", "exit 1")
		git(t, b.Cwd, "config", "filter.die.required", "true")
		common := git(t, b.Cwd, "rev-parse", "--path-format=absolute", "--git-common-dir")
		attrs := filepath.Join(strings.TrimSpace(common), "info", "attributes")
		if err = os.MkdirAll(filepath.Dir(attrs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(attrs, []byte(filepath.ToSlash(rel)+" filter=die\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitbranch.BeforeRebase = func(dir string) {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0)
			if err == nil {
				_, _ = f.WriteString("\nedited while the sync ran\n")
				f.Close()
			}
		}
		_, err = b.Execute(context.Background(), []string{"sync"})
		gitbranch.BeforeRebase = nil
		if err == nil {
			t.Fatal("the sync with a failing filter succeeded")
		}
		if err = os.Remove(attrs); err != nil {
			t.Fatal(err)
		}
		gitdir := strings.TrimSpace(git(t, board, "rev-parse", "--absolute-git-dir"))
		entries, _ := os.ReadDir(filepath.Join(gitdir, "rebase-merge"))
		if len(entries) != 1 || entries[0].Name() != "autostash" {
			t.Fatalf("rebase-merge holds %v, want only the autostash (Seid's state)", entries)
		}
		code, out := runText(b, "note", id, "from b")
		if code != 0 {
			t.Fatalf("note on the stuck board: exit %d\n%s", code, out)
		}
		if strings.Count(out, "sync: ended an interrupted hug rebase") != 1 {
			t.Fatalf("no single repair line:\n%s", out)
		}
		if _, err = os.Stat(filepath.Join(gitdir, "rebase-merge")); !os.IsNotExist(err) {
			t.Fatalf("rebase-merge left: %v", err)
		}
		if st := git(t, board, "status", "--porcelain"); st != "" {
			t.Fatalf("board not clean after the note:\n%s", st)
		}
		execute(t, b, "sync")
	})
	execute(t, a, "sync")
	text := taskFile(t, a, id)
	for _, want := range []string{"from a", "from b", "edited while the sync ran"} {
		if strings.Count(text, want) != 1 {
			t.Fatalf("%q not once on the branch:\n%s", want, text)
		}
	}
}
