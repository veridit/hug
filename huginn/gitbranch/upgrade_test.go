package gitbranch

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpgradeFailuresKeepOriginalLayout(t *testing.T) {
	for _, failure := range []string{"destination", "unreachable", "rejected-push", "submodule-move", "lost-ack", "unknown-before", "unknown-after", "unknown-advanced"} {
		t.Run(failure, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("JCODE_HOME", filepath.Join(home, "jcode"))
			t.Setenv("HUG_HOME", filepath.Join(home, "hug"))
			t.Setenv("HUG_SYNC", "off")
			base := t.TempDir()
			remote := filepath.Join(base, "remote.git")
			run(t, base, "init", "-q", "--bare", remote)
			root := filepath.Join(base, "repo")
			run(t, base, "clone", "-q", remote, root)
			run(t, root, "commit", "-q", "--allow-empty", "-m", "code")
			run(t, root, "push", "-q", "origin", "HEAD")
			r, err := Open(root, "muninn-notes", ".muninn-notes")
			if err != nil {
				t.Fatal(err)
			}
			raw := []byte("exact project memory bytes\n")
			if err := os.WriteFile(filepath.Join(r.Dir, "memories.jsonl"), raw, 0o644); err != nil {
				t.Fatal(err)
			}
			run(t, r.Dir, "add", "-A")
			run(t, r.Dir, "commit", "-qm", "memory")
			run(t, r.Dir, "push", "-qu", "origin", "muninn-notes")
			switch failure {
			case "destination":
				if err := os.MkdirAll(filepath.Join(root, ".hug", "memory"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "unreachable":
				run(t, root, "remote", "set-url", "origin", filepath.Join(base, "missing.git"))
			case "rejected-push":
				if err := os.WriteFile(filepath.Join(remote, "hooks", "pre-receive"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "submodule-move":
				sub := filepath.Join(base, "sub")
				run(t, base, "init", "-q", sub)
				run(t, sub, "commit", "-qm", "submodule", "--allow-empty")
				run(t, r.Dir, "-c", "protocol.file.allow=always", "submodule", "add", "-q", sub, "nested")
				run(t, r.Dir, "commit", "-qam", "submodule blocks worktree move")
			case "lost-ack":
				if err := os.WriteFile(filepath.Join(remote, "hooks", "post-receive"), []byte("#!/bin/sh\nprintf committed > ack-marker\nsleep 3\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "unknown-before", "unknown-after", "unknown-advanced":
				hook := "post-receive"
				script := "#!/bin/sh\ngit --git-dir=" + quote(filepath.ToSlash(filepath.Join(root, ".git"))) + " config remote.origin.url " + quote(filepath.ToSlash(filepath.Join(base, "missing.git"))) + "\n"
				if failure == "unknown-before" {
					hook = "pre-receive"
					script += "exit 1\n"
				} else {
					script += "sleep 3\n"
				}
				if err := os.WriteFile(filepath.Join(remote, "hooks", hook), []byte(script), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			beforeRefs := run(t, root, "show-ref")
			beforeRemote := run(t, remote, "show-ref")
			beforeTrees := run(t, root, "worktree", "list", "--porcelain")
			beforeStatus := run(t, r.Dir, "status", "--porcelain")
			beforeHead := run(t, root, "rev-parse", "HEAD")
			config := filepath.Join(root, ".git", "config")
			beforeConfig, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			if failure == "lost-ack" || strings.HasPrefix(failure, "unknown-") {
				before := NetTimeout
				NetTimeout = time.Second
				defer func() { NetTimeout = before }()
			}
			m, err := Upgrade(root, "muninn-notes", ".muninn-notes", "hug/memory", ".hug/memory")
			if failure == "lost-ack" {
				if _, e := os.Stat(filepath.Join(remote, "ack-marker")); e != nil {
					t.Fatalf("lost-ack fixture did not reach committed remote: %v", e)
				}
				if err != nil || !m.RemoteBranch || !m.Worktree || !m.LocalBranch {
					t.Fatalf("committed remote with lost ack must be verified as success, not rolled back locally: moved=%+v err=%v", m, err)
				}
				if b, e := os.ReadFile(filepath.Join(root, ".hug", "memory", "memories.jsonl")); e != nil || !bytes.Equal(b, raw) {
					t.Fatalf("verified remote success lost local memory: %v %q", e, b)
				}
				if heads := run(t, remote, "show-ref"); strings.Contains(heads, "refs/heads/muninn-notes") || !strings.Contains(heads, "refs/heads/hug/memory") {
					t.Fatalf("remote not atomically migrated: %s", heads)
				}
				return
			}
			if err == nil {
				t.Fatalf("%s should fail, moved=%+v", failure, m)
			}
			unknown := strings.HasPrefix(failure, "unknown-")
			if unknown {
				for _, want := range []string{"local original layout restoration VERIFIED", "remote outcome UNKNOWN", "refs/heads/muninn-notes", "refs/heads/hug/memory", "fix: git -C", "ls-remote --heads", "&& hug upgrade"} {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("unknown outcome lacks %q: %v", want, err)
					}
				}
				for _, hook := range []string{"pre-receive", "post-receive"} {
					if e := os.WriteFile(filepath.Join(remote, "hooks", hook), []byte("#!/bin/sh\nexit 0\n"), 0o755); e != nil {
						t.Fatal(e)
					}
				}
			}
			if m.LocalBranch || m.Worktree || m.RemoteBranch {
				t.Errorf("failure falsely reports an installed migration: %+v", m)
			}
			if after := run(t, root, "show-ref"); after != beforeRefs {
				t.Errorf("failure changed local refs:\n%s\nwant:\n%s", after, beforeRefs)
			}
			if after := run(t, remote, "show-ref"); !unknown && after != beforeRemote {
				t.Errorf("failure changed remote refs:\n%s\nwant:\n%s", after, beforeRemote)
			}
			if after := run(t, root, "worktree", "list", "--porcelain"); after != beforeTrees {
				t.Errorf("failure changed worktrees:\n%s\nwant:\n%s", after, beforeTrees)
			}
			if after := run(t, root, "rev-parse", "HEAD"); after != beforeHead {
				t.Errorf("failure changed code HEAD: %s", after)
			}
			if after, err := os.ReadFile(config); err != nil || !bytes.Equal(after, beforeConfig) {
				t.Errorf("failure changed config: %v\n%s\nwant:\n%s", err, after, beforeConfig)
			}
			if after, err := os.ReadFile(filepath.Join(r.Dir, "memories.jsonl")); err != nil || !bytes.Equal(after, raw) {
				t.Fatalf("failure lost original path/bytes: %v %q", err, after)
			}
			if after := run(t, r.Dir, "status", "--porcelain"); after != beforeStatus {
				t.Errorf("failure changed original status: %q != %q", after, beforeStatus)
			}
			if unknown {
				if failure == "unknown-advanced" {
					parent := run(t, root, "rev-parse", "muninn-notes")
					advance := run(t, root, "commit-tree", parent+"^{tree}", "-p", parent, "-m", "another writer advanced the new branch")
					run(t, root, "push", "-q", "origin", advance+":refs/heads/hug/memory")
					localBefore := run(t, root, "show-ref")
					remoteBefore := run(t, remote, "show-ref")
					if _, e := Upgrade(root, "muninn-notes", ".muninn-notes", "hug/memory", ".hug/memory"); e == nil || !strings.Contains(e.Error(), "nothing changed or rewound") {
						t.Fatalf("advanced remote retry was not safely refused: %v", e)
					}
					if run(t, root, "show-ref") != localBefore || run(t, remote, "show-ref") != remoteBefore {
						t.Fatal("advanced-writer retry changed refs")
					}
					return
				}
				remoteBefore := run(t, remote, "show-ref")
				m, e := Upgrade(root, "muninn-notes", ".muninn-notes", "hug/memory", ".hug/memory")
				if e != nil || !m.LocalBranch || !m.Worktree || m.RemoteBranch != (failure == "unknown-before") {
					t.Fatalf("recovered retry: moved=%+v err=%v", m, e)
				}
				if failure == "unknown-after" && run(t, remote, "show-ref") != remoteBefore {
					t.Fatal("retry destructively changed an already committed remote")
				}
				refs := run(t, root, "show-ref")
				remoteRefs := run(t, remote, "show-ref")
				m, e = Upgrade(root, "muninn-notes", ".muninn-notes", "hug/memory", ".hug/memory")
				if e != nil || m.LocalBranch || m.Worktree || m.RemoteBranch || run(t, root, "show-ref") != refs || run(t, remote, "show-ref") != remoteRefs {
					t.Fatalf("recovered second retry not idempotent: %+v %v", m, e)
				}
			}
		})
	}
}

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
