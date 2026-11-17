package cli

import (
	"context"
	"strings"
	"testing"
)

// newTicket makes a ticket with one check and returns its id.
func newTicket(t *testing.T, a *App, title string) string {
	t.Helper()
	r := execute(t, a, "new", title, "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")
	return r["task"].(Reply)["id"].(string)
}

// notesOf is the Notes section of id, read fresh.
func notesOf(t *testing.T, a *App, id string) string {
	t.Helper()
	s, e := a.open(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer s.c.Close()
	d, e := s.s.Get(id)
	if e != nil {
		t.Fatal(e)
	}
	return d.SectionText("Notes")
}

func deduced(t *testing.T, a *App, args ...string) []string {
	t.Helper()
	r := execute(t, a, append([]string{"deduce"}, args...)...)
	lines, _ := r["deductions"].([]string)
	return lines
}

// Checks 1-3 on one clone: a commit naming two tickets reaching main adds
// one landed line to each, once, however often deduce runs (and on a
// replay); a release tag adds shipped (done_release on a done ticket); each
// is reported as observed / marked / if wrong.
func TestDeduceLandedAndShipped(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	x, y := newTicket(t, a, "First"), newTicket(t, a, "Second")
	git(t, a.Cwd, "tag", "v0.1.0")
	// The first pass reads nothing old: it records where main and the tags are.
	if got := deduced(t, a); len(got) != 0 {
		t.Fatalf("first pass wrote %v", got)
	}
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Fix both ("+x+", "+y+")")
	sha := git(t, a.Cwd, "rev-parse", "--short=7", "HEAD")
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Unrelated (CI run 123)")
	got := deduced(t, a)
	if len(got) != 2 {
		t.Fatalf("want 2 deductions, got %q", got)
	}
	want := "observed: commit " + sha + " naming " + x + " reached main; marked: landed " + sha + `; if wrong: hug task note ` + x + ` "not landed: ` + sha + ` (why)"`
	if got[0] != want {
		t.Fatalf("report:\n got %q\nwant %q", got[0], want)
	}
	for _, id := range []string{x, y} {
		if n := strings.Count(notesOf(t, a, id), "landed "+sha+" on main "); n != 1 {
			t.Fatalf("%s: %d landed lines:\n%s", id, n, notesOf(t, a, id))
		}
	}
	// Again, and replayed from the start: nothing new.
	if got := deduced(t, a); len(got) != 0 {
		t.Fatalf("second pass wrote %v", got)
	}
	if got := deduced(t, a, "--since", "v0.1.0"); len(got) != 0 {
		t.Fatalf("replay wrote %v", got)
	}
	for _, id := range []string{x, y} {
		if n := strings.Count(notesOf(t, a, id), "landed "+sha); n != 1 {
			t.Fatalf("%s after replay: %d landed lines", id, n)
		}
	}
	// y is done before the release: it gets done_release, x a note.
	execute(t, a, "take", y, "--model", "test-model")
	execute(t, a, "check", y, "1")
	execute(t, a, "done", y)
	git(t, a.Cwd, "tag", "v0.1.1")
	got = deduced(t, a)
	if len(got) != 2 || !strings.Contains(got[0], "release v0.1.1 contains commit "+sha) || !strings.Contains(got[0], "marked: shipped in v0.1.1;") {
		t.Fatalf("tag deductions: %q", got)
	}
	if n := strings.Count(notesOf(t, a, x), "shipped in v0.1.1 ("+sha+")"); n != 1 {
		t.Fatalf("%s: %d shipped lines", x, n)
	}
	s, e := a.open(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	dy, _ := s.s.Get(y)
	dx, _ := s.s.Get(x)
	s.c.Close()
	if dy.Task.Fields["done_release"] != "v0.1.1" || dx.Task.Fields["done_release"] != "" {
		t.Fatalf("done_release: y %q x %q", dy.Task.Fields["done_release"], dx.Task.Fields["done_release"])
	}
	// Never ticks, never done: x stays todo with its check open.
	if dx.Task.State != "todo" || dx.Task.Acceptance[0].Checked {
		t.Fatalf("x changed: %s %v", dx.Task.State, dx.Task.Acceptance)
	}
	if got := deduced(t, a, "--since", "v0.1.0"); len(got) != 0 {
		t.Fatalf("replay after tag wrote %v", got)
	}
}

// A replay as the first pass: --since v0.1.0 reads v0.1.1 only, and the
// passes after it (plain deduce, the background pass) treat every release
// up to since as read, so no old release is deduced behind the replay's back.
func TestDeduceReplayFirstLeavesOldReleases(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	x := newTicket(t, a, "Old")
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Old work ("+x+")")
	git(t, a.Cwd, "tag", "v0.1.0")
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "New work ("+x+")")
	sha := git(t, a.Cwd, "rev-parse", "--short=7", "HEAD")
	git(t, a.Cwd, "tag", "v0.1.1")
	got := deduced(t, a, "--since", "v0.1.0")
	if len(got) != 2 || !strings.Contains(got[0], "landed "+sha) || !strings.Contains(got[1], "shipped in v0.1.1") {
		t.Fatalf("replay: %q", got)
	}
	if got := deduced(t, a); len(got) != 0 {
		t.Fatalf("pass after the replay wrote %q", got)
	}
	if n := notesOf(t, a, x); strings.Contains(n, "v0.1.0") {
		t.Fatalf("old release deduced:\n%s", n)
	}
}

// Check 1 across clones: a commit reaching main is noted once on the
// board, though both clones run deduce through hug task sync.
func TestDeduceTwoClonesOneLine(t *testing.T) {
	t.Parallel() // clones() sets nothing process-wide (0sbvnq)
	a, b, _ := clones(t)
	x := newTicket(t, a, "Shared")
	execute(t, a, "sync")
	execute(t, b, "sync")
	execute(t, b, "sync") // b's first pass records where main is
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Land it ("+x+")")
	git(t, a.Cwd, "push", "-q", "origin", "main")
	sha := git(t, a.Cwd, "rev-parse", "--short=7", "HEAD")
	git(t, b.Cwd, "fetch", "-q", "origin")
	for i := 0; i < 2; i++ {
		execute(t, a, "sync")
		execute(t, b, "sync")
	}
	settled(t, a)
	settled(t, b)
	execute(t, a, "sync")
	for _, c := range []*App{a, b} {
		if n := strings.Count(notesOf(t, c, x), "landed "+sha); n != 1 {
			t.Fatalf("%s: %d landed lines:\n%s", c.Cwd, n, notesOf(t, c, x))
		}
	}
}
