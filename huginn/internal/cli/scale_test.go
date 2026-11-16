package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Notes come from the cache, not a re-read of every file (44w1ev). A hand
// edit of Notes, made without any hug command, must still show in ls --json
// and show: also a same-size edit inside the racy window, and also after
// the cache file comes from an older schema (crab's conditions, 2026-11-16).
func TestHandEditedNotesSeenByLsAndShow(t *testing.T) {
	a := app(t)
	execute(t, a, "init")
	r := execute(t, a, "new", "Hand edit", "--goal", "Notes are read", "--accept", "seen")
	id := r["task"].(Reply)["id"].(string)
	execute(t, a, "note", id, "first note AAAA")
	path := filepath.Join(a.Cwd, ".hug", "tasks", r["task"].(Reply)["path"].(string))
	notes := func(want string) {
		t.Helper()
		ls := execute(t, a, "ls")["tasks"].([]Reply)
		if len(ls) != 1 || !strings.Contains(ls[0]["notes"].(string), want) {
			t.Fatalf("ls --json notes: %q, want %q", ls[0]["notes"], want)
		}
		show := execute(t, a, "show", id)["task"].(Reply)
		if !strings.Contains(show["notes"].(string), want) {
			t.Fatalf("show notes: %q, want %q", show["notes"], want)
		}
	}
	notes("first note AAAA")
	edit := func(from, to string, keepTime bool) {
		t.Helper()
		st, e := os.Stat(path)
		if e != nil {
			t.Fatal(e)
		}
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, []byte(strings.Replace(string(b), from, to, 1)), 0o644); e != nil {
			t.Fatal(e)
		}
		if keepTime {
			if e = os.Chtimes(path, st.ModTime(), st.ModTime()); e != nil {
				t.Fatal(e)
			}
		}
	}
	// In place, a different size, as an editor or an agent's edit tool does.
	edit("first note AAAA", "hand edited note", false)
	notes("hand edited note")
	// Same size and the same modification time, right after a sync: only
	// the racy-clean rule can see it.
	edit("hand edited note", "HAND EDITED NOTE", true)
	notes("HAND EDITED NOTE")
	// An older cache: another schema version rebuilds on first open, and
	// the unchanged file's notes come back from that rebuild.
	s, e := a.open(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	if e = s.c.SetSchemaForTest("6"); e != nil {
		t.Fatal(e)
	}
	s.c.Close()
	s, e = a.open(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	rebuilt := s.c.Rebuilt()
	s.c.Close()
	if !rebuilt {
		t.Fatal("a schema 6 cache was not rebuilt")
	}
	notes("HAND EDITED NOTE")
}
