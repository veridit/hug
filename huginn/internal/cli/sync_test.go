package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	// The settle a test gave its passes (clones: testSettle); the burst
	// test keeps the real one, which it is about.
	if d, e := time.ParseDuration(os.Getenv(childSettle)); e == nil {
		Settle = d
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
	quietRemote(t, bare)
	// Count pushes: every push appends its refs to bare/pushes.
	hook := "#!/bin/sh\ncat >> \"$GIT_DIR/pushes\"\n"
	if e := os.WriteFile(filepath.Join(bare, "hooks", "post-receive"), []byte(hook), 0o755); e != nil {
		t.Fatal(e)
	}
	// The repository's own config, which this process's git and the
	// spawned passes both read: the passes run without the machine's git
	// config (Sync.Env below), so a setting from it must not decide what
	// either sees. Windows runners set core.autocrlf=true system-wide: this
	// process checked task files out with CRLF, the pass (without it) saw
	// them modified and skipped as "uncommitted task edits" (CI run
	// 44114739181, 7 of 8 runs of TestBackgroundSyncConflictKeepsCommits).
	identity := func(dir string) {
		git(t, dir, "config", "user.name", "Test")
		git(t, dir, "config", "user.email", "test@example.invalid")
		git(t, dir, "config", "core.autocrlf", "false")
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
		// git's own trace of every pass, read when a wait times out (6ep7y2).
		trace := traceDir(dir)
		if e := os.MkdirAll(trace, 0o700); e != nil {
			t.Fatal(e)
		}
		return &App{Cwd: dir, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{},
			Env: map[string]string{"HUG_USER": "tester", "HUG_MODEL": "test-model", "JCODE_SESSION_ID": "", "JCODE_HOME": filepath.Join(base, ".no-jcode")},
			Sync: &SyncSettings{Spawn: []string{exe, "task", "sync", SyncChildArg}, FetchAfter: 500 * time.Millisecond,
				Env: []string{"GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_NOSYSTEM=1", childSettle + "=" + testSettle, "GIT_TRACE2_EVENT=" + trace}}}
	}
	a, b = mk(one), mk(two)
	execute(t, a, "init")
	execute(t, a, "sync")
	execute(t, b, "init")
	// Let any passes the setup started finish before a test counts.
	t.Cleanup(func() { settled(t, a); settled(t, b) })
	return a, b, bare
}

// offline runs f with a's background sync off, then turns it back on. A
// pass already started (by init or an earlier write) is waited for first:
// turning spawning off does not stop it, and one that ran inside f pushed
// the other clone's edit early (TestBackgroundSyncConflictKeepsCommits with
// the child held back 1 s: "[sync]: sync conflicts: ... (rebase aborted)").
func offline(t *testing.T, a *App, f func()) {
	t.Helper()
	settled(t, a)
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
	if why := settleWait(bg, settleStall, settleCap); why != "" {
		syncTimeoutFailure(t, bg, why)
	}
}

// settleStall is how long a pass may go without progress, settleCap how
// long settled waits in all (6zhyq2). A pass that stops is a stall after
// 30 s, as before. A pass that keeps going is waited for: on a slow
// Windows runner (CI 44522707343 attempt 1, sync tests 2 to 8 times their
// usual time) the pass after an upgrade started its last git 0.35 s
// before the old 30 s deadline, and failed as "still running".
const (
	settleStall = 30 * time.Second
	settleCap   = 3 * time.Minute
)

// settleWait waits for bg to be idle and says why it gave up, or "" when
// it is idle: stall without progress, or cap in all. Progress is the pass
// log and the child's git trace (one trace2 file per git process, events
// appended as it runs) growing.
func settleWait(bg gitbranch.Background, stall, cap time.Duration) string {
	start := time.Now()
	last, since := settleProgress(bg), start
	for {
		if !bg.Busy() && !bg.Pending() {
			return ""
		}
		if time.Since(since) >= stall {
			return fmt.Sprintf("no progress for %v (%v in all)", stall, time.Since(start).Round(time.Second))
		}
		if time.Since(start) >= cap {
			return fmt.Sprintf("still going after %v", cap)
		}
		time.Sleep(50 * time.Millisecond)
		if p := settleProgress(bg); p != last {
			last, since = p, time.Now()
		}
	}
}

// settleProgress is the size of bg's pass log and of every file in its
// trace2 folder, and how many there are.
func settleProgress(bg gitbranch.Background) string {
	size := func(p string) int64 {
		if st, e := os.Stat(p); e == nil {
			return st.Size()
		}
		return -1
	}
	p := fmt.Sprint(size(bg.Log()))
	dir := traceDirOf(bg.Log())
	ents, _ := os.ReadDir(dir)
	var n int64
	for _, e := range ents {
		n += size(filepath.Join(dir, e.Name()))
	}
	return fmt.Sprintf("%s %d %d", p, len(ents), n)
}

// syncTimeoutFailure reads the phase file before Fatalf triggers TempDir
// cleanup. Keep this path shared with the subprocess regression test: Go
// prints failed test output without -v, so ordinary CI retains the phases.
func syncTimeoutFailure(t *testing.T, bg gitbranch.Background, why string) {
	t.Helper()
	log, err := os.ReadFile(bg.Log())
	if err != nil {
		log = []byte(fmt.Sprintf("reading phase log: %v", err))
	}
	log = append(log, "\n"+unfinishedGit(traceDirOf(bg.Log()))+liveProcesses()...)
	t.Fatalf("background pass still running, "+why+": queued %v, busy %v, pending %v, turn held %v, run held %v (log %s):\n%s", bg.Queued(), bg.Busy(), bg.Pending(), bg.TurnHeld(), bg.RunHeld(), bg.Log(), log)
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

// childSettle is the quiet a spawned pass waits for (tests only), and
// testSettle the one clones gives: the real 1.5 s is waited for in every
// pass of every sync test (0sbvnq). Only TestBackgroundSyncBurstIsBatched
// needs the real settle.
const (
	childSettle = "HUG_TEST_SETTLE"
	testSettle  = "100ms"
)

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
		syncEarlyReturnFailure(t, a, got)
	}
}

// Keep the actual remote-head assertion's failure path shared with its
// deterministic subprocess regression, so ordinary CI retains diagnostics.
func syncEarlyReturnFailure(t *testing.T, a *App, got string) {
	t.Helper()
	t.Fatalf("settled returned before the pass pushed; remote head: %q; state: %s; log:\n%s", got, syncStateOf(t, a), syncLogOf(t, a))
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
	// The real settle: batching a burst is what it is for.
	a.Sync.Env = withoutEnv(a.Sync.Env, childSettle)
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
	// The remote hangs until the test releases it (at most 10 s): a note
	// that waited on the network would take that long. (A missing remote
	// fails at once, so it could not show a wait.) The release, not a
	// clock, ends each hang, so the passes after the timing check finish
	// at once: a fixed 5 s hang per network call cost this test 16 s of
	// settled on its own (0n4rnx).
	release := filepath.ToSlash(filepath.Join(t.TempDir(), "release"))
	git(t, a.Cwd, "remote", "set-url", "origin", "ssh://hang.invalid/remote.git")
	git(t, a.Cwd, "config", "core.sshCommand", "i=0; while [ ! -e '"+release+"' ] && [ $i -lt 100 ]; do sleep 0.1; i=$((i+1)); done; false")
	var baseline time.Duration
	offline(t, a, func() {
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
	// The remote now fails at once, still unreachable: the pass ends
	// with the 2 notes unpushed.
	if e := os.WriteFile(filepath.FromSlash(release), nil, 0o644); e != nil {
		t.Fatal(e)
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

// Competing titles keep the board's value and preserve the local value in a
// visible note. The detached pass reports it and retains later local work.
func TestBackgroundSyncConflictKeepsCommits(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	// Both clones edit offline from each other first.
	var id string
	offline(t, a, func() {
		offline(t, b, func() {
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
	execute(t, a, "sync")
	for _, c := range []*App{a, b} {
		text := taskFile(t, c, id)
		if !strings.Contains(text, "Title from A") || strings.Count(text, "Title from B") != 1 || strings.Count(text, "after the conflict") != 1 || !strings.Contains(text, strings.TrimSpace(mine)[:12]) {
			t.Fatalf("competing title lost content or source commit:\n%s\nsync log:\n%s", text, syncLogOf(t, b))
		}
	}
	if git(t, board, "rev-parse", "HEAD") != git(t, filepath.Join(a.Cwd, TaskDir), "rev-parse", "HEAD") {
		t.Fatal("background title resolution did not converge")
	}
	code, out := runText(b, "ls")
	if code != 0 || strings.Contains(out, "sync conflict in ") {
		t.Fatalf("ls after resolving competing values: %d\n%s\nB's state: %s\nB's sync log:\n%s", code, out, syncStateOf(t, b), syncLogOf(t, b))
	}
	if log := syncLogOf(t, b); strings.Count(log, "sync: "+id+" title:") != 1 {
		t.Fatalf("background pass did not report the retained title once:\n%s", log)
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

// syncStateOf is a's last recorded pass, as ls reads it.
func syncStateOf(t *testing.T, a *App) string {
	r, e := gitbranch.Open(a.Cwd, TaskBranch, TaskDir)
	if e != nil {
		return e.Error()
	}
	bg, e := r.Background()
	if e != nil {
		return e.Error()
	}
	raw, _ := json.Marshal(bg.State())
	return fmt.Sprintf("%s (busy %v, pending %v)", raw, bg.Busy(), bg.Pending())
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

// withoutEnv is env without the variable name.
func withoutEnv(env []string, name string) []string {
	out := []string{}
	for _, kv := range env {
		if !strings.HasPrefix(kv, name+"=") {
			out = append(out, kv)
		}
	}
	return out
}
