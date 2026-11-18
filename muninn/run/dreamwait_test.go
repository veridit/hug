package run

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/store"
)

// fakeDream drives the waiter by counting: a clock that only sleeps move,
// a load per step, the binary on disk, and what the waiter spawned or exec'd.
type fakeDream struct {
	now     time.Time
	sleeps  int
	loads   []float64 // load at each look; the last repeats
	looks   int
	spawned [][]string
	execed  [][]string
}

func newFakeDream(t *testing.T) *fakeDream {
	t.Helper()
	f := &fakeDream{now: time.Date(2026, 11, 18, 3, 0, 0, 0, time.UTC), loads: []float64{10}}
	clock, load, sleep, step := app.DreamClock, app.DreamLoad, dreamSleep, dreamStep
	spawn, exe, self, hugExe := dreamSpawn, dreamExec, dreamSelf, app.HugExecutable
	t.Cleanup(func() {
		app.DreamClock, app.DreamLoad, dreamSleep, dreamStep = clock, load, sleep, step
		dreamSpawn, dreamExec, dreamSelf, app.HugExecutable = spawn, exe, self, hugExe
	})
	app.DreamClock = func() time.Time { return f.now }
	app.DreamLoad = func() (float64, bool) {
		l := f.loads[min(f.looks, len(f.loads)-1)]
		f.looks++
		return l, true
	}
	dreamStep = 5 * time.Minute
	dreamSleep = func(d time.Duration) { f.sleeps++; f.now = f.now.Add(d) }
	dreamSpawn = func(argv []string) error { f.spawned = append(f.spawned, argv); return nil }
	dreamExec = func(argv []string) error { f.execed = append(f.execed, argv); return nil }
	// This hug and the installed one are the same file unless a test swaps it.
	bin := filepath.Join(t.TempDir(), "hug")
	os.WriteFile(bin, []byte("hug v1"), 0o755)
	dreamSelf = func() string { return bin }
	app.HugExecutable = func() string { return bin }
	return f
}

func dreamDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func lastTurn(t *testing.T, db *store.DB, at time.Time) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO session_turns(session,last_ns,turns) VALUES('s',?,1) ON CONFLICT(session) DO UPDATE SET last_ns=excluded.last_ns`, at.UnixNano()); err != nil {
		t.Fatal(err)
	}
}

// statsDream is the "dream" line of hug mem stats.
func statsDream(t *testing.T) string {
	t.Helper()
	out, errOut, code := runCLI(t, "", "stats")
	var s struct{ Dream string }
	if code != 0 || json.Unmarshal([]byte(out[:strings.LastIndex(out, "}")+1]), &s) != nil {
		t.Fatalf("stats: %d %q %q", code, out, errOut)
	}
	return s.Dream
}

func hookLog(t *testing.T) string {
	t.Helper()
	b, _ := os.ReadFile(filepath.Join(config.DataDir(), "hook.log"))
	return string(b)
}

// h00cs2 check 1: with no serve, the waiter runs one pass once the person
// has been away 30 min and the load is below 120, and records it.
func TestDreamWaiterRunsOnceQuietAndIdle(t *testing.T) {
	memHome(t)
	f := newFakeDream(t)
	db := dreamDB(t)
	lastTurn(t, db, f.now.Add(-5*time.Minute)) // away 5 min: wait 25 more
	f.loads = []float64{700, 700, 50}          // looked at only once the person is away
	if !app.DreamDue(db, f.now) {
		t.Fatal("never-run store is not due")
	}
	if err := waitAndDream(t.Context()); err != nil {
		t.Fatal(err)
	}
	// 5 sleeps reach 30 min away; the load is then 700 for 2 more looks.
	if f.sleeps != 7 || f.looks != 3 {
		t.Fatalf("waited %d steps and read the load %d times, want 7 (5 for quiet, 2 for load) and 3", f.sleeps, f.looks)
	}
	l := app.ReadDreamLast(db)
	if l.At.IsZero() || l.How != "while quiet" || !l.At.Equal(f.now) {
		t.Fatalf("pass not recorded: %+v", l)
	}
	if db.GetMeta("dream_last_ns") == "" {
		t.Fatal("dream_last_ns unset after a pass")
	}
	if log := hookLog(t); !strings.Contains(log, "while quiet: retired 0") || !strings.Contains(log, "waited 35m0s for quiet") {
		t.Fatalf("hook.log: %q", log)
	}
	// Not due again for a day: a second waiter does nothing.
	f.sleeps = 0
	if err := waitAndDream(t.Context()); err != nil || f.sleeps != 0 || !app.ReadDreamLast(db).At.Equal(l.At) {
		t.Fatalf("second waiter ran again: %v sleeps=%d", err, f.sleeps)
	}
}

// crab 2026-11-18: a busy machine is waited out, and after 12 h the waiter
// gives up and says why.
func TestDreamWaiterGivesUpOnABusyMachine(t *testing.T) {
	memHome(t)
	f := newFakeDream(t)
	db := dreamDB(t)
	f.loads = []float64{700}
	if err := waitAndDream(t.Context()); err != nil {
		t.Fatal(err)
	}
	if want := int(app.DreamWaitMax / dreamStep); f.sleeps != want {
		t.Fatalf("slept %d steps, want %d (12 h)", f.sleeps, want)
	}
	if !app.ReadDreamLast(db).At.IsZero() {
		t.Fatal("a pass ran at load 700")
	}
	if log := hookLog(t); !strings.Contains(log, "dream: gave up after 12h0m0s waiting: load is 700.0") {
		t.Fatalf("hook.log: %q", log)
	}
}

// crab 2026-11-18 (the mbdfzs rule): a waiter that slept while hug was
// upgraded runs the installed hug, never itself.
func TestDreamWaiterHandsThePassToTheInstalledHug(t *testing.T) {
	memHome(t)
	f := newFakeDream(t)
	db := dreamDB(t)
	lastTurn(t, db, f.now.Add(-20*time.Minute))
	upgraded := filepath.Join(t.TempDir(), "hug")
	os.WriteFile(upgraded, []byte("hug v2, installed while the waiter slept"), 0o755)
	f.loads = []float64{10}
	slept := dreamSleep
	dreamSleep = func(d time.Duration) {
		slept(d)
		app.HugExecutable = func() string { return upgraded }
	}
	if err := waitAndDream(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.execed) != 1 || f.execed[0][0] != upgraded || strings.Join(f.execed[0][1:], " ") != "mem dream --when-quiet" {
		t.Fatalf("exec'd %v, want the installed hug's waiter", f.execed)
	}
	if !app.ReadDreamLast(db).At.IsZero() {
		t.Fatal("the old binary ran the pass itself")
	}
	if log := hookLog(t); !strings.Contains(log, "dream: hug changed while waiting (10m0s); handing the pass to "+upgraded) {
		t.Fatalf("hook.log: %q", log)
	}
	// The installed hug, started fresh, finds the gate open and runs it.
	dreamSelf = func() string { return upgraded }
	f.execed = nil
	if err := waitAndDream(t.Context()); err != nil || len(f.execed) != 0 || app.ReadDreamLast(db).At.IsZero() {
		t.Fatalf("the installed hug did not run the pass: %v %v", err, f.execed)
	}
}

// The capture hook starts a waiter when a pass is due, once: never while
// one waits, never when the last pass is recent.
func TestCaptureHookStartsTheDreamWaiter(t *testing.T) {
	home := memHome(t)
	f := newFakeDream(t)
	t.Setenv("JCODE_HOME", filepath.Join(home, "jcode"))
	t.Setenv("JCODE_HOOK_SESSION_ID", "no-such-session")
	t.Setenv("JCODE_HOOK_EVENT", "turn_end")
	hook := func() {
		c := captureCmd()
		c.SetArgs([]string{"--hook"})
		if err := c.Execute(); err != nil {
			t.Fatal(err) // a hook never fails the harness
		}
	}
	hook()
	if len(f.spawned) != 1 || f.spawned[0][0] != app.HugExecutable() || strings.Join(f.spawned[0][1:], " ") != "mem dream --when-quiet" {
		t.Fatalf("spawned %v, want one waiter of the installed hug", f.spawned)
	}
	release, ok, err := app.TryDreamWait()
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	hook()
	release()
	if len(f.spawned) != 1 {
		t.Fatalf("a second waiter started while one waits: %v", f.spawned)
	}
	db := dreamDB(t)
	if _, err := app.RecordDream(db, f.now.Add(-time.Hour), "by hand", app.DreamResult{}); err != nil {
		t.Fatal(err)
	}
	hook()
	if len(f.spawned) != 1 {
		t.Fatalf("a waiter started an hour after a pass: %v", f.spawned)
	}
}

// h00cs2 check 2 and crab's rule 3: every surface names the last pass and
// its receipt, and says never when none ran.
func TestDreamStatusNamesTheLastPass(t *testing.T) {
	memHome(t)
	f := newFakeDream(t)
	out, errOut, code := runCLI(t, "", "dream", "--status")
	if code != 0 || !strings.HasPrefix(out, "Last pass: never (dream runs by itself once you have been away 30m0s and the load is below 120") {
		t.Fatalf("status before any pass: %d %q %q", code, out, errOut)
	}
	out, _, _ = runCLI(t, "", "dream", "--dry-run")
	if !strings.HasPrefix(out, "Last pass: never") {
		t.Fatalf("dry-run header: %q", out)
	}
	if got := statsDream(t); !strings.HasPrefix(got, "dream: never ran") {
		t.Fatalf("stats: %q", got)
	}
	db := dreamDB(t)
	receipt := filepath.Join(config.DataDir(), "tidy", "apply-20261118T030000Z-1")
	if _, err := app.RecordDream(db, f.now, "while quiet", app.DreamResult{Retired: 3, Queued: 40, Open: 44, Backup: receipt}); err != nil {
		t.Fatal(err)
	}
	want := "while quiet: retired 3, queued 40, open 44, undo: hug mem tidy undo " + receipt
	out, _, _ = runCLI(t, "", "dream", "--status")
	if !strings.Contains(out, want) || !strings.Contains(out, "Next pass: after ") {
		t.Fatalf("status after a pass: %q", out)
	}
	// Decoded, not matched as text: JSON escapes a Windows path's backslashes.
	if got := statsDream(t); !strings.Contains(got, want) {
		t.Fatalf("stats after a pass: %q", got)
	}
	// A failed pass is recorded too: the next try is a day away, and the
	// line says it failed.
	app.RecordDreamFailed(db, f.now, errors.New("disk full\nfix: free space"))
	if l := app.ReadDreamLast(db); l.How != "failed (disk full)" || app.DreamDue(db, f.now.Add(time.Hour)) {
		t.Fatalf("failed pass: %+v", l)
	}
}
