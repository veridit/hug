//go:build !windows

package gitbranch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/timing"
)

// lingeringGit puts a git first on PATH that runs the real git and, when its
// arguments contain sub, leaves a process holding its stdout and stderr open
// for 5 s after it exits.
func lingeringGit(t *testing.T, sub string) {
	t.Helper()
	real, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\n'" + real + "' \"$@\"\ncode=$?\ncase \" $* \" in *' " + sub + " '*) sleep 5 & ;; esac\nexit $code\n"
	if e := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// ad62aa: a git that exited 0 is its answer, however long anything else
// holds its output open. Under load 300-700 the test binary's goroutines
// that copy git's output pipes did not run within WaitDelay (2 s) after git
// exited, and os/exec returned "exec: WaitDelay expired before I/O complete"
// with the output dropped: a commit that landed was reported failed
// (34r1ft-full2.txt, TestTakeRecordsBranchAndWorktree), and a lost rev-parse
// made Locate fail, "hug task needs a Git checkout as its working directory"
// (TestDeduceTwoClonesOneLine). A process left holding the pipe opens the
// same 2 s window without needing load.
func TestGitAnswerOnceGitExits(t *testing.T) {
	timing.Budget(t)
	r := bornRepo(t, "main")
	lingeringGit(t, "rev-parse")
	start := time.Now()
	l, e := Locate(r.Dir)
	took := time.Since(start)
	if e != nil {
		t.Fatalf("Locate after a git that exited 0 (%s): %v", took, e)
	}
	want, _ := filepath.EvalSymlinks(r.Dir)
	if got, _ := filepath.EvalSymlinks(l.Root); got != want {
		t.Fatalf("Locate root %q, want %q", l.Root, r.Dir)
	}
	out, e := r.git("rev-parse", "--abbrev-ref", "HEAD")
	if e != nil || out != "main" {
		t.Fatalf("rev-parse: %q, %v", out, e)
	}
	// Errors still carry git's own stderr.
	_, e = r.git("rev-parse", "--verify", "no-such-ref")
	if e == nil || !strings.Contains(e.Error(), "Needed a single revision") {
		t.Fatalf("a failing git's error lacks its stderr: %v", e)
	}
	if took > 1500*time.Millisecond {
		t.Fatalf("Locate waited %s on a process holding git's output after git exited", took)
	}
}
