package cli

import (
	"strings"
	"testing"
)

// syncPair is a fresh board in two clones of one remote, background sync
// off, one task with three checks, synced into both.
func syncPair(t *testing.T) (a, b *App, id string) {
	t.Helper()
	a, b, _ = clones(t)
	settled(t, a)
	settled(t, b)
	a.Sync.Spawn, b.Sync.Spawn = nil, nil
	id = execute(t, a, "new", "Shared task", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "one", "--accept", "two", "--accept", "three")["task"].(Reply)["id"].(string)
	execute(t, a, "sync")
	execute(t, b, "sync")
	if !strings.Contains(taskFile(t, b, id), "Shared task") {
		t.Fatal("fixture: B lacks the task")
	}
	return a, b, id
}

// 8zwy6r check 1 (kestrel's baseline on a13d7c74: B's sync exits 1 with
// "sync conflicts: ... (rebase aborted)"): two clones each note the same
// task offline; both syncs succeed, both notes are in the file once, in
// time order, and both clones end on the same commit.
// Check 2: the same for a different check ticked on each side, an answer
// against a note, and a request on each side.
func TestSyncMergesBothSidesAppends(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		before func(t *testing.T, a, b *App, id string) // both sides, synced
		sideA  []string
		sideB  []string
		want   []string // in this order in the file
	}{
		{name: "note on each side",
			sideA: []string{"note", "{id}", "A noted first"},
			sideB: []string{"note", "{id}", "B noted second"},
			want:  []string{"A noted first", "B noted second"}},
		{name: "a different check ticked on each side",
			sideA: []string{"check", "{id}", "1"},
			sideB: []string{"check", "{id}", "3"},
			want:  []string{"- [x] one", "- [ ] two", "- [x] three"}},
		{name: "an answer against a note",
			before: func(t *testing.T, a, b *App, id string) {
				execute(t, a, "take", id, "--as", "nautilus")
				execute(t, a, "ask", id, "--to", "jhf", "which rows?", "--as", "nautilus")
				execute(t, a, "sync")
				execute(t, b, "sync")
			},
			sideA: []string{"note", "{id}", "A noted while asked"},
			sideB: []string{"answer", "{id}", "the IN PROGRESS rows", "--as", "jhf"},
			want:  []string{"A noted while asked", "the IN PROGRESS rows"}},
		{name: "a request on each side",
			sideA: []string{"request", "{id}", "A wants the rows wider", "--as", "jhf"},
			sideB: []string{"request", "{id}", "B wants a key for it", "--as", "alice"},
			want:  []string{"A wants the rows wider", "B wants a key for it"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a, b, id := syncPair(t)
			if c.before != nil {
				c.before(t, a, b, id)
			}
			args := func(s []string) []string {
				out := append([]string{}, s...)
				for i := range out {
					out[i] = strings.ReplaceAll(out[i], "{id}", id)
				}
				return out
			}
			execute(t, a, args(c.sideA)...)
			execute(t, b, args(c.sideB)...)
			for _, side := range []*App{a, b, a} {
				if code, out := runText(side, "sync"); code != 0 {
					t.Fatalf("sync in %s: exit %d\n%s", side.Cwd, code, out)
				}
			}
			for _, side := range []*App{a, b} {
				text := taskFile(t, side, id)
				at := -1
				for _, w := range c.want {
					if n := strings.Count(text, w); n != 1 {
						t.Fatalf("%s has %q %d times:\n%s", side.Cwd, w, n, text)
					}
					i := strings.Index(text, w)
					if i < at {
						t.Fatalf("%s: %q is out of order:\n%s", side.Cwd, w, text)
					}
					at = i
				}
			}
			if git(t, a.Cwd+"/"+TaskDir, "rev-parse", "HEAD") != git(t, b.Cwd+"/"+TaskDir, "rev-parse", "HEAD") {
				t.Fatal("the clones did not converge")
			}
		})
	}
}

// 8zwy6r check 3, with crab's 04:44 ruling for single values: state set
// differently on both sides does not stop the sync; the board (remote)
// keeps its value, the local one is kept as a note, and the sync prints
// one line naming the task and the field.
func TestSyncBothSidesStateRemoteWinsAndSays(t *testing.T) {
	t.Parallel()
	a, b, id := syncPair(t)
	execute(t, a, "set", id, "--state", "doing")
	execute(t, b, "drop", id, "not needed on this side")
	execute(t, a, "sync")
	code, out := runText(b, "sync")
	if code != 0 {
		t.Fatalf("state on both sides: exit %d\n%s", code, out)
	}
	if n := strings.Count(out, "sync: "+id+" state:"); n != 1 {
		t.Fatalf("want one line naming %s state, got %d:\n%s", id, n, out)
	}
	execute(t, a, "sync")
	for _, side := range []*App{a, b} {
		text := taskFile(t, side, id)
		if !strings.Contains(text, `state: "doing"`) || !strings.Contains(text, "dropped") {
			t.Fatalf("%s: the board's state or the local one is lost:\n%s\nB's sync said:\n%s", side.Cwd, text, out)
		}
		// The board's state is whole: no field of the losing state stays
		// behind as a broken finding (dropped_reason on a doing task).
		if code, lint := runText(side, "lint", id); strings.Contains(lint, "broken") {
			t.Fatalf("%s: the merged task lints broken (exit %d):\n%s\n%s", side.Cwd, code, lint, text)
		}
	}
}
