package memshare

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid",
		"GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commitConfig commits hug.toml with body on branch hug/tasks of repo, as
// hug task does (its own orphan branch), and returns to main.
func commitConfig(t *testing.T, repo, body string) {
	t.Helper()
	wt := filepath.Join(t.TempDir(), "tasks")
	if strings.Contains(git(t, repo, "branch", "--list", "hug/tasks"), "hug/tasks") {
		git(t, repo, "worktree", "add", "-q", wt, "hug/tasks")
	} else {
		git(t, repo, "worktree", "add", "-q", "--orphan", "-b", "hug/tasks", wt)
	}
	if err := os.WriteFile(filepath.Join(wt, "hug.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, wt, "add", "hug.toml")
	git(t, wt, "commit", "-q", "-m", "hug.toml")
	git(t, repo, "worktree", "remove", "--force", wt)
}

func newRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	os.MkdirAll(repo, 0o755)
	git(t, repo, "init", "-q", "-b", "main")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "code")
	return repo
}

// 34r1ft check 1 (Jørgen 18:18, "Clearly A"): the setting is hug.toml
// [memory] share = true, committed on hug/tasks; absent means local.
func TestSharesReadsCommittedHugToml(t *testing.T) {
	repo := newRepo(t)
	if s, err := Shares(repo); s || err != nil {
		t.Fatalf("no hug/tasks branch: Shares = %v, %v; want local", s, err)
	}
	commitConfig(t, repo, "[board]\ncoordinator = \"crab\"\n")
	if s, err := Shares(repo); s || err != nil {
		t.Fatalf("hug.toml without [memory]: Shares = %v, %v; want local", s, err)
	}
	// share under another section is not the setting.
	commitConfig(t, repo, "share = true\n[board]\nshare = true\n")
	if s, err := Shares(repo); s || err != nil {
		t.Fatalf("share outside [memory]: Shares = %v, %v; want local", s, err)
	}
	commitConfig(t, repo, "[board]\ncoordinator = \"crab\"\n\n[memory]\nshare = true # private repo\n")
	if s, err := Shares(repo); !s || err != nil {
		t.Fatalf("[memory] share = true: Shares = %v, %v; want shares", s, err)
	}
	// From a subdirectory and a linked worktree of the code branch too.
	sub := filepath.Join(repo, "sub")
	os.MkdirAll(sub, 0o755)
	if s, err := Shares(sub); !s || err != nil {
		t.Fatalf("from a subdirectory: Shares = %v, %v", s, err)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	git(t, repo, "worktree", "add", "-q", "-b", "work", linked)
	if s, err := Shares(linked); !s || err != nil {
		t.Fatalf("from a linked worktree: Shares = %v, %v", s, err)
	}
	commitConfig(t, repo, "[memory]\nshare = false\n")
	if s, err := Shares(repo); s || err != nil {
		t.Fatalf("[memory] share = false: Shares = %v, %v; want local", s, err)
	}
	// Uncommitted is not the repo's setting: the committed one decides.
	if err := os.WriteFile(filepath.Join(repo, "hug.toml"), []byte("[memory]\nshare = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if s, err := Shares(repo); s || err != nil {
		t.Fatalf("a loose hug.toml on the code branch: Shares = %v, %v; want local", s, err)
	}
}

// A fresh clone has no local hug/tasks yet: the remote's committed setting
// decides, so every clone follows the repo.
func TestSharesFreshCloneFollowsRemote(t *testing.T) {
	repo := newRepo(t)
	commitConfig(t, repo, "[memory]\nshare = true\n")
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, repo, "init", "-q", "--bare", "-b", "main", bare)
	git(t, repo, "push", "-q", bare, "main", "hug/tasks")
	clone := filepath.Join(t.TempDir(), "clone")
	git(t, filepath.Dir(clone), "clone", "-q", bare, clone)
	if s, err := Shares(clone); !s || err != nil {
		t.Fatalf("fresh clone of a sharing repo: Shares = %v, %v; want shares", s, err)
	}
}

// Outside a repository nothing can be pushed: local, no error.
func TestSharesOutsideGit(t *testing.T) {
	if s, err := Shares(t.TempDir()); s || err != nil {
		t.Fatalf("outside git: Shares = %v, %v", s, err)
	}
}

// A value hug cannot read is an error with a fix, never a quiet share, and
// the fix goes back to local, never to sharing.
func TestSharesRejectsUnknownValue(t *testing.T) {
	repo := newRepo(t)
	commitConfig(t, repo, "[memory]\nshare = \"yes\"\n")
	s, err := Shares(repo)
	if s || err == nil || !strings.HasSuffix(err.Error(), "\nfix: hug task memory local") {
		t.Fatalf("share = \"yes\": Shares = %v, %v; want an error with a fix", s, err)
	}
}

// Local is a status, not a refusal with a fix (crab review of e2582a0f):
// nothing an agent would run names the opt-in.
func TestErrLocalSteersNowhere(t *testing.T) {
	msg := ErrLocal.Error()
	if msg != "project memories stay on this machine (hug.toml [memory] share is not set)" || strings.Contains(msg, "fix:") || strings.Contains(msg, "next:") {
		t.Fatalf("ErrLocal = %q", msg)
	}
	if strings.HasPrefix(PersonHint, "fix:") || strings.HasPrefix(PersonHint, "next:") {
		t.Fatalf("PersonHint = %q", PersonHint)
	}
	if !errors.Is(ErrLocal, ErrLocal) {
		t.Fatal("ErrLocal is not itself")
	}
}
