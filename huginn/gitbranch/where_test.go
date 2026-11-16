package gitbranch

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWhereMainAndLinkedWorktree(t *testing.T) {
	root := t.TempDir()
	main := filepath.Join(root, "main")
	run := func(dir string, args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	os.MkdirAll(main, 0o755)
	run(main, "init", "-q", "-b", "main")
	run(main, "commit", "-q", "--allow-empty", "-m", "one")
	linked := filepath.Join(root, "wt-x")
	run(main, "worktree", "add", "-q", "-b", "feat/x", linked)
	sub := filepath.Join(linked, "sub")
	os.MkdirAll(sub, 0o755)

	p, err := Where(main)
	if err != nil || p.Branch != "main" || p.Linked || !samePath(p.Worktree, main) || !samePath(p.Main, main) {
		t.Fatalf("main: %+v %v", p, err)
	}
	p, err = Where(sub)
	if err != nil || p.Branch != "feat/x" || !p.Linked || !samePath(p.Worktree, linked) || !samePath(p.Main, main) {
		t.Fatalf("linked: %+v %v", p, err)
	}
	run(linked, "checkout", "-q", "--detach")
	if p, _ := Where(linked); p.Branch != "" || !p.Linked {
		t.Fatalf("detached: %+v", p)
	}
	if _, err := Where(t.TempDir()); err == nil {
		t.Fatal("outside git must fail")
	}
	if d := DefaultBranch(main); d != "main" {
		t.Fatalf("default branch %q, want main", d)
	}
}
