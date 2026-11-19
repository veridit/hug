package cli

import (
	"strings"
	"testing"
	"time"
)

// j0zvee: TestReleaseCloseTwoClones failed when the two clones' closes fell
// in different minutes (CI 44412479085, ubuntu: "release_test.go:148: ...
// 2 release lines", History lines at 22:34 and 22:35). Here the board clock
// is pinned and ticks one minute between the clones, every time, and the
// background passes are off, so only the foreground syncs merge. Not
// parallel: task.Now is shared.
func TestReleaseCloseTwoClonesInTwoMinutes(t *testing.T) {
	a, b, _ := clones(t)
	settled(t, a)
	settled(t, b)
	a.Sync.Spawn, b.Sync.Spawn = nil, nil
	minute := time.Date(2026, 1, 31, 12, 0, 0, 0, time.Local)
	pinClock(t, minute)
	x := newTicket(t, a, "Shared")
	execute(t, a, "take", x, "--as", "worker", "--model", "m")
	execute(t, a, "check", x, "1")
	execute(t, a, "wait", x, "landing", "--as", "worker")
	git(t, a.Cwd, "tag", "v0.1.0")
	git(t, a.Cwd, "push", "-q", "origin", "main", "v0.1.0")
	execute(t, a, "sync")
	git(t, b.Cwd, "fetch", "-q", "--tags", "origin")
	execute(t, b, "sync")
	execute(t, b, "sync") // b's first pass records where main and the tags are
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Ship it ("+x+")")
	git(t, a.Cwd, "tag", "v0.1.1")
	git(t, a.Cwd, "push", "-q", "origin", "main", "v0.1.1")
	git(t, b.Cwd, "fetch", "-q", "--tags", "origin")
	// A deduces at 12:01, B at 12:02: the same close, two minutes.
	pinClock(t, minute.Add(time.Minute))
	if got := deduced(t, a); len(got) != 2 {
		t.Fatalf("a deduced %q", got)
	}
	pinClock(t, minute.Add(2*time.Minute))
	if got := deduced(t, b); len(got) != 2 {
		t.Fatalf("b deduced %q", got)
	}
	execute(t, a, "sync")
	execute(t, b, "sync")
	execute(t, a, "sync")
	for _, c := range []*App{a, b} {
		f := taskFile(t, c, x)
		if s := stateOf(t, c, x); s.State != "done" {
			t.Fatalf("%s: %s\n%s", c.Cwd, s.State, f)
		}
		if n := strings.Count(f, "by release v0.1.1"); n != 1 {
			t.Fatalf("%s: %d release lines\n%s", c.Cwd, n, f)
		}
		if n := strings.Count(f, "shipped in v0.1.1 ("); n != 1 {
			t.Fatalf("%s: %d shipped lines\n%s", c.Cwd, n, f)
		}
	}
}
