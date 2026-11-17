//go:build !windows

package gitbranch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// unbornRepo is a repository whose branch has no commit yet, as a freshly
// initialised board is: HEAD names refs/heads/BRANCH, which does not exist.
func unbornRepo(t *testing.T, branch string) *Repository {
	t.Helper()
	dir := t.TempDir()
	if out, e := exec.Command("git", "init", "-q", "-b", branch, dir).CombinedOutput(); e != nil {
		t.Fatalf("git init: %v %s", e, out)
	}
	return &Repository{Dir: dir, Branch: branch}
}

// crashingGit puts a git first on PATH that kills itself with SIGSEGV when
// its arguments contain sub, and runs the real git otherwise.
func crashingGit(t *testing.T, sub string) {
	t.Helper()
	real, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \" $* \" in *' " + sub + " '*) kill -SEGV $$ ;; esac\nexec '" + real + "' \"$@\"\n"
	if e := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// xv39b9 check 2: an unborn branch is an empty head, not an error.
func TestHeadUnbornIsEmpty(t *testing.T) {
	r := unbornRepo(t, "hug/tasks")
	h, e := r.Head()
	if h != "" || e != nil {
		t.Fatalf("unborn branch: head %q, error %v", h, e)
	}
}

// xv39b9 check 1: when the call that tells an unborn branch from a broken
// HEAD crashes, the error names that call and its signal, not rev-parse's
// "Needed a single revision" (kestrel stress run 2026-11-17, load ~150).
func TestHeadNamesTheCallThatCrashed(t *testing.T) {
	for _, sub := range []string{"show-ref", "symbolic-ref"} {
		t.Run(sub, func(t *testing.T) {
			r := unbornRepo(t, "hug/tasks")
			crashingGit(t, sub)
			h, e := r.Head()
			if e == nil {
				t.Fatalf("head %q and no error though %s crashed", h, sub)
			}
			msg := e.Error()
			if strings.Contains(msg, "Needed a single revision") || !strings.HasPrefix(msg, sub) {
				t.Fatalf("the error does not name %s first:\n%s", sub, msg)
			}
			if !strings.Contains(msg, "segmentation fault") {
				t.Fatalf("the error does not say how %s died:\n%s", sub, msg)
			}
			if !strings.Contains(msg, "\nfix: git -C ") || !strings.Contains(msg, sub) {
				t.Fatalf("no fix: line re-running %s:\n%s", sub, msg)
			}
		})
	}
}
