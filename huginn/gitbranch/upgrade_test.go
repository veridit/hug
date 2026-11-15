package gitbranch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// Two clones share a remote with branch huginn. Upgrading the first renames
// the remote branch; the second, upgraded later, ends on hug/tasks with the
// same history and its own unpushed commit intact. Re-running is a no-op.
func TestUpgradeTwoCheckouts(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	run(t, base, "init", "-q", "--bare", remote)
	a := filepath.Join(base, "a")
	run(t, base, "clone", "-q", remote, a)
	run(t, a, "commit", "-q", "--allow-empty", "-m", "init")
	run(t, a, "push", "-q", "origin", "HEAD")
	if _, err := Open(a, "huginn", ".huginn"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(a, ".huginn", "t1.md"), []byte("one\n"), 0o644)
	run(t, filepath.Join(a, ".huginn"), "add", "-A")
	run(t, filepath.Join(a, ".huginn"), "commit", "-q", "-m", "t1")
	run(t, filepath.Join(a, ".huginn"), "push", "-q", "origin", "huginn")

	b := filepath.Join(base, "b")
	run(t, base, "clone", "-q", remote, b)
	if _, err := Open(b, "huginn", ".huginn"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(b, ".huginn", "t2.md"), []byte("two\n"), 0o644)
	run(t, filepath.Join(b, ".huginn"), "add", "-A")
	run(t, filepath.Join(b, ".huginn"), "commit", "-q", "-m", "t2 unpushed")

	m, err := Upgrade(a, "huginn", ".huginn", "hug/tasks", ".hug/tasks")
	if err != nil {
		t.Fatal(err)
	}
	if !m.LocalBranch || !m.Worktree || !m.RemoteBranch {
		t.Fatalf("first upgrade: %+v", m)
	}
	if heads := run(t, a, "ls-remote", "--heads", "origin"); strings.Contains(heads, "refs/heads/huginn") || !strings.Contains(heads, "refs/heads/hug/tasks") {
		t.Fatalf("remote heads after upgrade: %s", heads)
	}
	if m, err = Upgrade(a, "huginn", ".huginn", "hug/tasks", ".hug/tasks"); err != nil || m.LocalBranch || m.Worktree || m.RemoteBranch {
		t.Fatalf("second run should do nothing: %+v %v", m, err)
	}
	if _, err := Open(a, "hug/tasks", ".hug/tasks"); err != nil {
		t.Fatalf("open after upgrade: %v", err)
	}

	if _, err = Upgrade(b, "huginn", ".huginn", "hug/tasks", ".hug/tasks"); err != nil {
		t.Fatal(err)
	}
	rb, err := Open(b, "hug/tasks", ".hug/tasks")
	if err != nil {
		t.Fatal(err)
	}
	if err := rb.Sync(); err != nil {
		t.Fatalf("sync after upgrade: %v", err)
	}
	for _, f := range []string{"t1.md", "t2.md"} {
		if _, err := os.Stat(filepath.Join(b, ".hug", "tasks", f)); err != nil {
			t.Errorf("b lost %s after upgrade+sync", f)
		}
	}
	if heads := run(t, b, "ls-remote", "--heads", "origin"); strings.Contains(heads, "refs/heads/huginn") {
		t.Fatalf("b resurrected the old remote branch: %s", heads)
	}
}
