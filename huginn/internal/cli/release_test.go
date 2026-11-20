package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// stateOf is id's state and stage, read fresh.
func stateOf(t *testing.T, a *App, id string) task.Task {
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
	return d.Task
}

// yf39ya check 2: a release closes the tasks it contains whose checks all
// hold, with the version in History and done_release; a task with an open
// check, or one done refuses (an open dependency), is left as it was and
// keeps its finding; a todo task is never closed.
func TestReleaseClosesShippedTasks(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	held := newTicket(t, a, "Held")       // checks hold, waiting on landing
	open := newTicket(t, a, "Open check") // a check still open
	dep := newTicket(t, a, "Depends")     // checks hold, waits on open
	todo := newTicket(t, a, "Todo")       // never taken
	for _, id := range []string{held, open, dep} {
		execute(t, a, "take", id, "--as", "w-"+id, "--model", "m")
	}
	execute(t, a, "check", held, "1")
	execute(t, a, "wait", held, "landing by crab", "--as", "w-"+held)
	execute(t, a, "check", dep, "1")
	execute(t, a, "dep", dep, "--add", open)
	execute(t, a, "check", todo, "1")
	git(t, a.Cwd, "tag", "v0.1.0")
	deduced(t, a) // the first pass reads no old release
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Ship them ("+held+", "+open+", "+dep+", "+todo+")")
	git(t, a.Cwd, "tag", "v0.1.1")
	got0 := deduced(t, a)
	closedReport := false
	for _, l := range got0 {
		if strings.Contains(l, "naming "+held) && strings.Contains(l, "marked: shipped in v0.1.1, done (every check holds); if wrong: hug task set "+held+" --state todo") {
			closedReport = true
		}
	}
	if !closedReport {
		t.Fatalf("no close in the report: %q", got0)
	}
	h := stateOf(t, a, held)
	if h.State != "done" || h.Fields["done_release"] != "v0.1.1" {
		t.Fatalf("held: %s done_release %q\n%s", h.State, h.Fields["done_release"], taskFile(t, a, held))
	}
	if f := taskFile(t, a, held); !strings.Contains(f, "(done, by release v0.1.1: every check holds)") {
		t.Fatalf("held History lacks the release:\n%s", f)
	}
	for _, id := range []string{open, dep} {
		if s := stateOf(t, a, id); s.State != "doing" {
			t.Fatalf("%s closed: %s", id, s.State)
		}
	}
	if s := stateOf(t, a, todo); s.State != "todo" {
		t.Fatalf("todo task moved: %s", s.State)
	}
	// Nothing silenced: the open task keeps its question (landed-unchecked
	// ranks first while none of its checks is ticked), the todo keeps
	// shipped-not-done, the closed one has none. The refused one is in
	// depends: its open dependency says why, and the release step runs
	// when it is done (yf39ya check 9).
	r := execute(t, a, "lint")
	got := map[string]string{}
	for _, f := range r["findings"].([]task.Finding) {
		id, _ := task.IDOf(f.Path)
		if c := task.FindingCode(f.Problem); task.QuestionCode(c) {
			got[id] = c
		}
	}
	if got[open] == "" || got[todo] != "shipped-not-done" || got[dep] != "" || got[held] != "" {
		t.Fatalf("question findings %v (open %s todo %s dep %s held %s)", got, open, todo, dep, held)
	}
	// Again: nothing new, nothing written twice.
	deduced(t, a)
	if n := strings.Count(taskFile(t, a, held), "by release v0.1.1"); n != 1 {
		t.Fatalf("held: %d release lines", n)
	}
}

// yf39ya check 2 across clones (crab 19:59 constraint 2): two machines
// deducing the same tag before either syncs give one done, one shipped
// note and one release line, and both syncs succeed.
func TestReleaseCloseTwoClones(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
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
	settled(t, a)
	settled(t, b)
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Ship it ("+x+")")
	git(t, a.Cwd, "tag", "v0.1.1")
	git(t, a.Cwd, "push", "-q", "origin", "main", "v0.1.1")
	git(t, b.Cwd, "fetch", "-q", "--tags", "origin")
	// Both deduce the tag offline, then sync.
	offline(t, a, func() {
		offline(t, b, func() {
			// landed and shipped, each closing x on its own clone.
			for _, c := range []*App{a, b} {
				if got := deduced(t, c); len(got) != 2 {
					t.Fatalf("%s deduced %q", c.Cwd, got)
				}
				if s := stateOf(t, c, x); s.State != "done" {
					t.Fatalf("%s before sync: %s", c.Cwd, s.State)
				}
			}
		})
	})
	for i := 0; i < 2; i++ {
		execute(t, a, "sync")
		execute(t, b, "sync")
	}
	settled(t, a)
	settled(t, b)
	execute(t, a, "sync")
	for _, c := range []*App{a, b} {
		f := taskFile(t, c, x)
		if s := stateOf(t, c, x); s.State != "done" {
			t.Fatalf("%s: %s\n%s", c.Cwd, s.State, f)
		}
		if n := strings.Count(f, "shipped in v0.1.1 ("); n != 1 {
			t.Fatalf("%s: %d shipped lines\n%s", c.Cwd, n, f)
		}
		if n := strings.Count(f, "by release v0.1.1"); n != 1 {
			t.Fatalf("%s: %d release lines\n%s", c.Cwd, n, f)
		}
	}
}
