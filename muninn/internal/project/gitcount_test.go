package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// gitCalls puts a logging git first on PATH for the rest of the test and
// returns a function that runs f and gives the git calls it made, one line
// per process (as huginn's TestGitProcessesPerCommand).
func gitCalls(t *testing.T) func(f func()) []string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the git wrapper is a POSIX sh script")
	}
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	bin, logFile := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + logFile + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func(f func()) []string {
		_ = os.Remove(logFile)
		f()
		b, _ := os.ReadFile(logFile)
		return strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(b)), " ", "_"))
	}
}

// ce3dmq: every placement lookup runs Find, so its git processes (~45 ms
// each on the Windows runner) multiply across a session and timed the
// muninn app tests out at 10 minutes (run 44142356523). In a main worktree
// the top level is the main root: two processes, not four.
func TestFindInMainWorktreeStartsTwoGit(t *testing.T) {
	base := t.TempDir()
	main := filepath.Join(base, "main")
	os.MkdirAll(filepath.Join(main, "sub"), 0o700)
	gitInit(t, main)
	main, _ = filepath.EvalSymlinks(main)
	linked := filepath.Join(base, "linked")
	branchGit(t, main, "worktree", "add", "-q", "--detach", linked, "HEAD")
	count := gitCalls(t)
	var p Project
	var err error
	calls := count(func() { p, err = Find(filepath.Join(main, "sub"), false) })
	if err != nil || p.Notes != filepath.Join(main, NotesDir) {
		t.Fatalf("%+v %v", p, err)
	}
	if len(calls) != 2 {
		t.Errorf("Find in a main worktree starts %d git processes, want 2:\n  %s", len(calls), strings.Join(calls, "\n  "))
	}
	calls = count(func() { p, err = Find(linked, false) })
	if err != nil || p.Notes != filepath.Join(main, NotesDir) {
		t.Fatalf("linked worktree: Notes %q, want the main checkout's %q (%v)", p.Notes, filepath.Join(main, NotesDir), err)
	}
	// 3: the worktree list is read from the common git dir (0sbvnq).
	if len(calls) != 3 {
		t.Errorf("Find in a linked worktree starts %d git processes, want 3:\n  %s", len(calls), strings.Join(calls, "\n  "))
	}
	// A plain folder (no .git above it; t.TempDir is outside any checkout)
	// needs no git to learn that it is not one.
	plain := filepath.Join(base, "plain")
	os.MkdirAll(plain, 0o700)
	calls = count(func() { _, err = Find(plain, false) })
	if err != ErrNoProject {
		t.Fatalf("plain folder: %v, want ErrNoProject", err)
	}
	if len(calls) != 0 {
		t.Errorf("Find in a plain folder starts %d git processes, want 0:\n  %s", len(calls), strings.Join(calls, "\n  "))
	}
}

// 4vd1mz: one command asks Find for the same folder several times (a
// forget: its binding, the view, the refresh); the git processes run once.
// A failure is not kept: a folder that becomes a repository is found.
func TestFindAgainStartsNoGit(t *testing.T) {
	forgetFinds()
	t.Cleanup(forgetFinds)
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	os.MkdirAll(repo, 0o700)
	count := gitCalls(t)
	if calls := count(func() { Find(repo, false) }); len(calls) != 0 {
		t.Fatalf("a plain folder started git: %v", calls)
	}
	gitInit(t, repo)
	var p Project
	var err error
	if calls := count(func() { p, err = Find(repo, false) }); err != nil || len(calls) != 2 {
		t.Fatalf("first Find after git init: %+v %v, %d git processes:\n  %s", p, err, len(calls), strings.Join(calls, "\n  "))
	}
	for i := 0; i < 3; i++ {
		var again Project
		if calls := count(func() { again, err = Find(repo, false) }); err != nil || len(calls) != 0 || again != p {
			t.Fatalf("Find %d again: %+v %v, %d git processes:\n  %s", i+2, again, err, len(calls), strings.Join(calls, "\n  "))
		}
	}
}
