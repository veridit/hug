package cli

import (
	"bytes"
	"context"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
)

// Background sync tests (0hkaa5) run the real detached child: this test
// binary, re-run with the child argument (see TestMain). Spawning is set
// only here, never by default, so other tests cannot fork.

// runSyncChild is TestMain's child mode: run the background pass and exit.
func runSyncChild() {
	// A slow start, as a loaded Windows runner gives (TestSettledWaitsForSlowChild).
	if d, e := time.ParseDuration(os.Getenv(childDelay)); e == nil {
		time.Sleep(d)
	}
	cwd, _ := os.Getwd()
	a := &App{Cwd: cwd, In: strings.NewReader(""), Out: os.Stdout, Err: os.Stderr}
	os.Exit(a.Run(context.Background(), []string{"sync", SyncChildArg}))
}

// clones makes a bare remote and two clones with boards, background sync on.
// Everything it sets is the clones' own (repository config, App.Env,
// App.Sync), never the process's, so these tests run in parallel (0sbvnq).
func clones(t *testing.T) (a, b *App, bare string) {
	t.Helper()
	base := t.TempDir()
	bare = filepath.Join(base, "remote.git")
	git(t, base, "init", "--bare", "-b", "main", bare)
	// Count pushes: every push appends its refs to bare/pushes.
	hook := "#!/bin/sh\ncat >> \"$GIT_DIR/pushes\"\n"
	if e := os.WriteFile(filepath.Join(bare, "hooks", "post-receive"), []byte(hook), 0o755); e != nil {
		t.Fatal(e)
	}
	identity := func(dir string) {
		git(t, dir, "config", "user.name", "Test")
		git(t, dir, "config", "user.email", "test@example.invalid")
	}
	one := filepath.Join(base, "one")
	git(t, base, "init", "-b", "main", one)
	identity(one)
	git(t, one, "commit", "--allow-empty", "-m", "init")
	git(t, one, "remote", "add", "origin", bare)
	git(t, one, "push", "-q", "origin", "main")
	two := filepath.Join(base, "two")
	git(t, base, "clone", "-q", bare, two)
	identity(two)
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	// The passes run in a fresh process, with the real settle time (1.5 s),
	// and without the machine's git config; FetchAfter is the parent's, for reads.
	mk := func(dir string) *App {
		return &App{Cwd: dir, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{},
			Env: map[string]string{"HUG_USER": "tester", "HUG_MODEL": "test-model", "JCODE_SESSION_ID": "", "JCODE_HOME": filepath.Join(base, ".no-jcode")},
			Sync: &SyncSettings{Spawn: []string{exe, "task", "sync", SyncChildArg}, FetchAfter: 500 * time.Millisecond,
				Env: []string{"GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_NOSYSTEM=1"}}}
	}
	a, b = mk(one), mk(two)
	execute(t, a, "init")
	execute(t, a, "sync")
	execute(t, b, "init")
	// Let any passes the setup started finish before a test counts.
	t.Cleanup(func() { settled(t, a); settled(t, b) })
	return a, b, bare
}

// offline runs f with a's background sync off, then turns it back on.
func offline(a *App, f func()) {
	saved := a.Sync.Spawn
	a.Sync.Spawn = nil
	defer func() { a.Sync.Spawn = saved }()
	f()
}

// settled waits until no background pass runs, is queued, or has been
// started and not yet queued for a's board. It waits on the child, never on
// a clock: a child slower to start than pendingFor (Windows CI under
// parallel tests, run 44107306302) is still waited for.
func settled(t *testing.T, a *App) {
	t.Helper()
	r, e := gitbranch.Open(a.Cwd, TaskBranch, TaskDir)
	if e != nil || r.Remote == "" {
		return
	}
	bg, e := r.Background()
	if e != nil {
		return
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if !bg.Busy() && !bg.Pending() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("background pass still running after 30 s (log %s)", bg.Log())
}

// arrival is how long a change may take to reach the other clone: 10 s,
// the acceptance bound, where a hug command costs what it does on a laptop.
// The Windows CI runner spends ~1.9 s on one note even without background
// sync (run 43943367365: "without background sync 1.875s"), so the same
// pipeline of three commands gets four times the time there.
func arrival() time.Duration {
	if runtime.GOOS == "windows" {
		return 40 * time.Second
	}
	return 10 * time.Second
}

// until polls f every 100 ms for up to d.
func until(d time.Duration, f func() bool) bool {
	for deadline := time.Now().Add(d); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if f() {
			return true
		}
	}
	return f()
}

// childDelay holds a spawned pass back before it starts (tests only).
const childDelay = "HUG_TEST_CHILD_DELAY"

// settled waits for the pass a write started, however slowly the child
// starts: one slower than pendingFor (5 s) made settled return before the
// pass ran, and TestBackgroundSyncConflictKeepsCommits failed on Windows CI
// ("ls after a conflict: 0", runs 44096836969, 44098427171, 44107306302).
func TestSettledWaitsForSlowChild(t *testing.T) {
	t.Parallel()
	a, _, bare := clones(t)
	id := execute(t, a, "new", "Slow start", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	settled(t, a)
	a.Sync.Env = append(a.Sync.Env, childDelay+"=6s")
	execute(t, a, "note", id, "after a slow start")
	settled(t, a)
	if got := git(t, bare, "log", "-1", "--format=%s", TaskBranch); !strings.Contains(got, "note "+id) {
		t.Fatalf("settled returned before the pass pushed; remote head: %q", got)
	}
}

// Check 1: a note in clone A shows up in clone B within 10 s, nobody syncs.
func TestBackgroundSyncNoteReachesOtherClone(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	id := execute(t, a, "new", "Shared work", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	start := time.Now()
	execute(t, a, "note", id, "from clone A")
	seen := until(arrival(), func() bool {
		r, e := b.Execute(context.Background(), []string{"show", id})
		if e != nil {
			return false
		}
		return strings.Contains(jsonString(t, r), "from clone A")
	})
	if !seen {
		t.Fatalf("clone B did not see the note within %v; A's log:\n%s", arrival(), syncLogOf(t, a))
	}
	t.Logf("note reached clone B in %v", time.Since(start).Round(10*time.Millisecond))
}

// Check 2: a burst of 20 notes in 2 s produces at most 2 pushes.
func TestBackgroundSyncBurstIsBatched(t *testing.T) {
	t.Parallel()
	a, _, bare := clones(t)
	settle := Settle // the child's: it never sees a test's change to it
	id := execute(t, a, "new", "Burst", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	settled(t, a)
	before := pushes(t, bare)
	start := time.Now()
	for i := 0; i < 20; i++ {
		execute(t, a, "note", id, "note "+strconv.Itoa(i))
	}
	d := time.Since(start)
	settled(t, a)
	n := pushes(t, bare) - before
	// A burst is writes closer together than the settle time: it is one or
	// two pushes. Where one note alone takes longer than half of that (the
	// Windows CI runner: ~1.9 s a note), 20 notes are no burst, and at most
	// one push per settle time is allowed.
	most := 2
	if per := d / 20; per > settle/2 {
		most = int(d/settle) + 1
		t.Logf("20 notes took %v (%v each), no burst on this machine: up to %d pushes", d, per, most)
	}
	if n < 1 || n > most {
		t.Fatalf("20 notes in %v made %d pushes, want 1 to %d; log:\n%s", d, n, most, syncLogOf(t, a))
	}
	if got := git(t, bare, "log", "-1", "--format=%s", TaskBranch); !strings.Contains(got, "note "+id) {
		t.Fatalf("remote head %q is not the last note", got)
	}
}

// Check 3: with the remote unreachable a note is fast, ls reports the
// unpushed commits with a fix line, and they go out once it is back.
func TestBackgroundSyncOfflineCatchesUp(t *testing.T) {
	t.Parallel()
	a, _, bare := clones(t)
	id := execute(t, a, "new", "Offline", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	settled(t, a)
	// The remote hangs: any network call by the note would cost 5 s. (A
	// missing remote fails at once, so it could not show a wait.)
	git(t, a.Cwd, "remote", "set-url", "origin", "ssh://hang.invalid/remote.git")
	git(t, a.Cwd, "config", "core.sshCommand", "sleep 5; false")
	var baseline time.Duration
	offline(a, func() {
		start := time.Now()
		execute(t, a, "note", id, "baseline, no background sync")
		baseline = time.Since(start)
	})
	start := time.Now()
	execute(t, a, "note", id, "while offline")
	offline := time.Since(start)
	t.Logf("note with the remote hanging %v, without background sync %v", offline.Round(time.Millisecond), baseline.Round(time.Millisecond))
	// Starting the syncer is a process start (slow under -race and on the
	// Windows runner, ~1.9 s a note there); waiting on the remote is 5 s.
	if offline > baseline+max(2*time.Second, baseline) {
		t.Fatalf("note took %v with the remote hanging, %v without background sync: it waited on the network; log:\n%s", offline, baseline, syncLogOf(t, a))
	}
	settled(t, a)
	code, out := runText(a, "ls")
	if code != 0 || !strings.Contains(out, "sync: 2 unpushed: ") || !strings.Contains(out, "\nfix: hug task sync\n") {
		t.Fatalf("ls offline: %d\n%s", code, out)
	}
	git(t, a.Cwd, "remote", "set-url", "origin", bare)
	git(t, a.Cwd, "config", "--unset", "core.sshCommand")
	execute(t, a, "note", id, "back online")
	settled(t, a)
	if got := git(t, bare, "log", "-3", "--format=%s", TaskBranch); strings.Count(got, "note "+id) != 3 {
		t.Fatalf("remote after reconnect:\n%s", got)
	}
	if code, out := runText(a, "ls"); code != 0 || strings.Contains(out, "unpushed") {
		t.Fatalf("ls still reports unpushed after catching up: %d\n%s", code, out)
	}
}

// Check 5: a rebase conflict keeps both clones' commits and says so.
func TestBackgroundSyncConflictKeepsCommits(t *testing.T) {
	t.Parallel()
	a, b, bare := clones(t)
	// Both clones edit offline from each other first.
	var id string
	offline(a, func() {
		offline(b, func() {
			id = execute(t, a, "new", "Contested", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
			execute(t, a, "sync")
			execute(t, b, "sync")
			execute(t, a, "set", id, "--title", "Title from A")
			execute(t, b, "set", id, "--title", "Title from B")
			execute(t, a, "sync")
		})
	})
	mine := git(t, filepath.Join(b.Cwd, TaskDir), "rev-parse", "HEAD")
	execute(t, b, "note", id, "after the conflict")
	settled(t, b)
	board := filepath.Join(b.Cwd, TaskDir)
	if !isAncestor(board, mine) {
		t.Fatalf("clone B lost its own commit %s", mine)
	}
	code, out := runText(b, "ls")
	if code != 0 || !strings.Contains(out, "sync conflict in ") || !strings.Contains(out, "\nfix: hug task sync\n") {
		t.Fatalf("ls after a conflict: %d\n%s", code, out)
	}
	if got := git(t, bare, "log", "-1", "--format=%s", TaskBranch); !strings.Contains(got, "set "+id) {
		t.Fatalf("remote lost clone A's commit: %s", got)
	}
}

// The board turn: 20 parallel notes on one clone all land, none lost to a
// background rebase moving files underneath (m8shdj check 1 shape).
func TestBackgroundSyncParallelNotesAllLand(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	id := execute(t, a, "new", "Parallel", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := &App{Cwd: a.Cwd, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: a.Env, Sync: a.Sync}
			if _, e := c.Execute(context.Background(), []string{"note", id, "parallel " + strconv.Itoa(i)}); e != nil {
				errs <- e
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("parallel note: %v", e)
	}
	settled(t, a)
	seen := until(arrival(), func() bool {
		r, e := b.Execute(context.Background(), []string{"show", id})
		return e == nil && strings.Count(jsonString(t, r), "parallel ") == 20
	})
	if !seen {
		r, _ := b.Execute(context.Background(), []string{"show", id})
		t.Fatalf("clone B does not have all 20 notes: %d\n%s", strings.Count(jsonString(t, r), "parallel "), syncLogOf(t, a))
	}
}

func isAncestor(dir, commit string) bool {
	return osexec.Command("git", "-C", dir, "merge-base", "--is-ancestor", commit, "HEAD").Run() == nil
}

// pushes counts pushes to the task branch, from the hook clones installs.
func pushes(t *testing.T, bare string) int {
	t.Helper()
	b, _ := os.ReadFile(filepath.Join(bare, "pushes"))
	return strings.Count(string(b), "refs/heads/"+TaskBranch)
}

func syncLogOf(t *testing.T, a *App) string {
	r, e := gitbranch.Open(a.Cwd, TaskBranch, TaskDir)
	if e != nil {
		return e.Error()
	}
	bg, e := r.Background()
	if e != nil {
		return e.Error()
	}
	b, _ := os.ReadFile(bg.Log())
	return string(b)
}
