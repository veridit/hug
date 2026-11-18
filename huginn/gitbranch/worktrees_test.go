package gitbranch

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWorktreesFromFilesAgreeWithGit: Locate reads the worktree list from
// the common git dir instead of git worktree list (0sbvnq). Main, linked
// (absolute and relative gitdir), detached, a deleted folder (prunable) and
// a deleted but locked one give git's list; a bare repository and a
// per-worktree config fall back to git (ok=false).
func TestWorktreesFromFilesAgreeWithGit(t *testing.T) {
	root, _ := fixture(t, false)
	root, _ = filepath.EvalSymlinks(root)
	common := filepath.Join(root, ".git")
	base := filepath.Dir(root)
	command(t, root, "worktree", "add", "-q", "-b", "b1", filepath.Join(base, "w1"))
	command(t, root, "worktree", "add", "-q", "--detach", filepath.Join(base, "w2"))
	command(t, root, "-c", "worktree.useRelativePaths=true", "worktree", "add", "-q", "-b", "b3", filepath.Join(base, "w3"))
	command(t, root, "worktree", "add", "-q", "-b", "b4", filepath.Join(base, "w4"))
	command(t, root, "worktree", "add", "-q", "-b", "b5", filepath.Join(base, "w5"))
	command(t, root, "worktree", "lock", filepath.Join(base, "w5"))
	must(t, os.RemoveAll(filepath.Join(base, "w4")))
	must(t, os.RemoveAll(filepath.Join(base, "w5")))
	r := &Repository{Dir: root}
	want, e := r.worktrees(root)
	must(t, e)
	got, ok := fastWorktrees(common)
	if !ok {
		t.Fatal("fast worktree list refused a plain repository")
	}
	if len(got) != len(want) {
		t.Fatalf("fast %d worktrees, git %d:\nfast %+v\ngit  %+v", len(got), len(want), got, want)
	}
	byPath := map[string]worktree{}
	for _, w := range want {
		p, e := filepath.EvalSymlinks(w.Path)
		if e != nil {
			p = w.Path
		}
		byPath[filepath.Clean(p)] = w
	}
	if got[0].Path != filepath.Clean(want[0].Path) && got[0].Path != root {
		t.Fatalf("main first: fast %q, git %q", got[0].Path, want[0].Path)
	}
	for _, g := range got {
		p, e := filepath.EvalSymlinks(g.Path)
		if e != nil {
			p = g.Path
		}
		w, ok := byPath[filepath.Clean(p)]
		if !ok {
			t.Fatalf("fast lists %q, git does not: %+v", g.Path, want)
		}
		if g.Branch != w.Branch || g.Prunable != w.Prunable || g.Bare != w.Bare {
			t.Fatalf("%s: fast %+v, git %+v", g.Path, g, w)
		}
	}
	// What it cannot read with certainty goes to git.
	command(t, root, "config", "extensions.worktreeConfig", "true")
	if _, ok := fastWorktrees(common); ok {
		t.Fatal("per-worktree config read from the files")
	}
	bare := filepath.Join(t.TempDir(), "b.git")
	command(t, base, "init", "-q", "--bare", bare)
	if _, ok := fastWorktrees(bare); ok {
		t.Fatal("bare repository read from the files")
	}
}

// TestCommitTakesEarlierStagedChange: a change staged before a commit of
// the whole worktree (a write stopped between add and commit) is
// committed, though add has nothing new to stage (0sbvnq).
func TestCommitTakesEarlierStagedChange(t *testing.T) {
	root, _ := fixture(t, false)
	r := openRepo(t, root, "hug/tasks", ".hug/tasks")
	writeFile(t, r, "a.md", "a\n")
	must(t, r.Commit(nil, "a"))
	writeFile(t, r, "a.md", "staged\n")
	command(t, r.Dir, "add", "a.md")
	before, e := r.Head()
	must(t, e)
	must(t, r.Commit(nil, "takes the staged change"))
	after, e := r.Head()
	must(t, e)
	if before == after {
		t.Fatal("staged change left uncommitted")
	}
	if s := command(t, r.Dir, "status", "--porcelain"); s != "" {
		t.Fatalf("dirty after commit: %s", s)
	}
	// Nothing changed, nothing staged: no commit.
	must(t, r.Commit(nil, "empty"))
	if again, _ := r.Head(); again != after {
		t.Fatal("empty commit made")
	}
}
