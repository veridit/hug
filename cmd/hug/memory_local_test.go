package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 34r1ft (crab review of e2582a0f): agents run fix: and next: lines as
// given. On a repository that keeps memories local, nothing hug prints may
// steer toward sharing, or an agent in a public repository runs it and
// publishes. hug mem sync there is a status, exit 0; hug task memory names
// the opt-in only in a plain sentence for a person.
func TestLocalRepoOutputNeverSteersToSharing(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	root := filepath.Join(base, "repo")
	bare := filepath.Join(base, "remote.git")
	env := append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "HUG_HOME="+filepath.Join(base, "store"), "JCODE_HOME="+filepath.Join(home, "jcode"), "HUG_SYNC=off", "HUG_NO_EMBED=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid",
		"GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=tester", "HUG_MODEL=test-model", "GOMAXPROCS=2")
	bin := filepath.Join(base, "hug")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = append(os.Environ(), "GOMAXPROCS=2", "GOFLAGS=-p=2")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	try := func(dir string, command string, args ...string) (string, error) {
		cmd := exec.Command(command, args...)
		cmd.Dir, cmd.Env = dir, env
		b, err := cmd.CombinedOutput()
		return string(b), err
	}
	run := func(dir, command string, args ...string) string {
		t.Helper()
		out, err := try(dir, command, args...)
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", command, args, err, out)
		}
		return out
	}
	run(base, "git", "init", "-q", "--bare", "-b", "main", bare)
	run(root, "git", "init", "-q", "-b", "main")
	run(root, "git", "commit", "-q", "--allow-empty", "-m", "code")
	run(root, "git", "remote", "add", "origin", bare)
	run(root, "git", "push", "-q", "origin", "main")
	run(root, bin, "task", "init")
	run(root, bin, "mem", "remember", "--project", "Synthetic fact that stays on this machine.")

	steers := func(what, out string) {
		t.Helper()
		for _, line := range strings.Split(out, "\n") {
			l := strings.TrimSpace(line)
			if (strings.HasPrefix(l, "fix:") || strings.HasPrefix(l, "next:")) && strings.Contains(l, "memory share") {
				t.Errorf("%s on a local repository steers an agent to share: %q\n%s", what, l, out)
			}
		}
	}
	// hug mem sync: a status, exit 0, nothing pushed.
	out, err := try(root, bin, "mem", "sync")
	if err != nil {
		t.Fatalf("hug mem sync on a local repository failed (local is not an error): %v\n%s", err, out)
	}
	if !strings.Contains(out, "project memories stay on this machine (hug.toml [memory] share is not set)") {
		t.Fatalf("hug mem sync on a local repository does not say so:\n%s", out)
	}
	steers("hug mem sync", out)
	if refs := run(root, "git", "ls-remote", bare); strings.Contains(refs, "hug/memory") || strings.Contains(refs, "muninn-notes") {
		t.Fatalf("a local repository's sync reached the remote:\n%s", refs)
	}
	// --json too: the reply an agent reads.
	out, err = try(root, bin, "mem", "sync", "--dry-run")
	if err != nil {
		t.Fatalf("hug mem sync --dry-run: %v\n%s", err, out)
	}
	steers("hug mem sync --dry-run", out)
	// hug task memory: the state, and the opt-in as a sentence for a person.
	for _, args := range [][]string{{"task", "memory"}, {"task", "memory", "--json"}, {"task", "help", "memory"}, {"task", "memory", "local"}} {
		out = run(root, bin, args...)
		steers("hug "+strings.Join(args, " "), out)
	}
	out = run(root, bin, "task", "memory")
	if !strings.Contains(out, "a private repository can share them with: hug task memory share") {
		t.Fatalf("hug task memory does not name the opt-in for a person:\n%s", out)
	}
	// A bad value in hug.toml is fixed by removing it, not by sharing.
	out, _ = try(root, bin, "task", "memory", "yes")
	steers("hug task memory yes", out)
}
