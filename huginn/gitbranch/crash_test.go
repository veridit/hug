//go:build !windows

package gitbranch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// bornRepo is a repository on branch with one commit and a changed file.
func bornRepo(t *testing.T, branch string) *Repository {
	t.Helper()
	r := unbornRepo(t, branch)
	for _, args := range [][]string{
		{"config", "user.name", "t"}, {"config", "user.email", "t@example.invalid"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
	} {
		if out, e := exec.Command("git", append([]string{"-C", r.Dir}, args...)...).CombinedOutput(); e != nil {
			t.Fatalf("git %v: %v %s", args, e, out)
		}
	}
	if e := os.WriteFile(filepath.Join(r.Dir, "a.md"), []byte("a\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	return r
}

// 3z65k1 check 2: a git that dies of a signal (SIGSEGV on the owner's Mac,
// 2026-11-17) is reported as a crash, with a fix: line that retries it, never
// as a bare "signal: segmentation fault" that reads as an answer or a bug in
// the board.
func TestCommitReportsGitCrash(t *testing.T) {
	r := bornRepo(t, "hug/tasks")
	crashingGit(t, "add")
	e := r.Commit(nil, "hug task note x")
	if e == nil {
		t.Fatal("commit succeeded though git add crashed")
	}
	msg := e.Error()
	for _, want := range []string{"died before it finished", "segmentation fault", "no answer about the board", "run it again", "\nfix: git -C "} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in:\n%s", want, msg)
		}
	}
	if !Crashed(e) {
		t.Fatalf("Crashed(err) is false for a signal death: %v", e)
	}
	if strings.Contains(msg, "index.lock") {
		t.Fatalf("names an index.lock that is not there:\n%s", msg)
	}
}

// A git killed while writing the index leaves index.lock, and every later
// write then fails "File exists": the crash report names the lock and how
// to remove it once no git is running.
func TestCrashNamesLeftoverIndexLock(t *testing.T) {
	r := bornRepo(t, "hug/tasks")
	lock := filepath.Join(r.Dir, ".git", "index.lock")
	if e := os.WriteFile(lock, nil, 0o644); e != nil {
		t.Fatal(e)
	}
	crashingGit(t, "add")
	e := r.Commit(nil, "hug task note x")
	if e == nil {
		t.Fatal("commit succeeded though git add crashed")
	}
	msg := e.Error()
	i := strings.Index(msg, "\nfix: ")
	detail, fix := msg[:i], msg[i:]
	if !strings.Contains(fix, "rm "+quote(lock)+" && git -C ") || !strings.Contains(detail, lock) || !strings.Contains(detail, "no other git") {
		t.Fatalf("fix does not remove the leftover lock %s:\n%s", lock, msg)
	}
}

// Exiting non-zero is git's answer, not a crash: the message stays git's own.
func TestFailureIsNotACrash(t *testing.T) {
	r := bornRepo(t, "hug/tasks")
	_, e := r.git("rev-parse", "--verify", "nosuchref")
	if e == nil || strings.Contains(e.Error(), "died before it finished") || Crashed(e) {
		t.Fatalf("an exit 128 reads as a crash: %v", e)
	}
}
