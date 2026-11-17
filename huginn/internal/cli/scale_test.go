package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Notes come from the cache, not a re-read of every file (44w1ev). A hand
// edit of Notes, made without any hug command, must still show in ls --json
// and show: also a same-size edit inside the racy window, and also after
// the cache file comes from an older schema (crab's conditions, 2026-11-16).
func TestHandEditedNotesSeenByLsAndShow(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	r := execute(t, a, "new", "Hand edit", "--goal", "Notes are read", "--purpose", "P", "--principle", "Q", "--accept", "seen")
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
	// edit writes the file with modification time at (zero: the write's own).
	edit := func(from, to string, at time.Time) {
		t.Helper()
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, []byte(strings.Replace(string(b), from, to, 1)), 0o644); e != nil {
			t.Fatal(e)
		}
		if !at.IsZero() {
			if e = os.Chtimes(path, at, at); e != nil {
				t.Fatal(e)
			}
		}
	}
	// In place, a different size, as an editor or an agent's edit tool does.
	// Its modification time lies ahead of every sync in this test, so the
	// file stays inside the racy window (2 s before the last sync) however
	// slow the machine is.
	inWindow := time.Now().Add(time.Hour)
	edit("first note AAAA", "hand edited note", inWindow)
	notes("hand edited note")
	// A slow runner: Windows CI took 34 s for this test (44112302694). With
	// the old test (edit 2 kept edit 1's own modification time) this pause
	// alone failed it: edit 1 fell out of the racy window, and a same-size
	// edit outside the window is not promised to be seen.
	time.Sleep(2500 * time.Millisecond)
	// Same size and the same modification time, inside the racy window:
	// only the racy-clean rule can see it, and it must, every time.
	edit("hand edited note", "HAND EDITED NOTE", inWindow)
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

// lint takes file references from the cached parse (44w1ev); a reference
// added by hand is still reported, and forbid_cite still reads the text.
func TestLintSeesHandAddedFileRef(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	r := execute(t, a, "new", "Refs", "--goal", "Lint reads refs", "--purpose", "P", "--principle", "Q", "--accept", "seen")
	path := filepath.Join(a.Cwd, ".hug", "tasks", r["task"].(Reply)["path"].(string))
	execute(t, a, "lint")
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, append(b, []byte("\nsee file:missing.tsv and .jcode/scratch/x\n")...), 0o644); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(a.Cwd, ".hug", "tasks", "hug.toml"), []byte("forbid_cite = [\".jcode/scratch/\"]\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	out := execute(t, a, "lint", "--json")
	j := fmt.Sprint(out["findings"])
	if !strings.Contains(j, "file:missing.tsv: file not found") || !strings.Contains(j, "cites .jcode/scratch/") {
		t.Fatalf("lint after a hand edit: %s", j)
	}
}

// open asks git about the repository once (44w1ev), and the old-layout check
// reads both old refs in one for-each-ref. A ref below the old name
// (refs/heads/huginn/x) is not the old layout; the old branch itself, local
// or remote-tracking, still is. Outside git the error stays the same.
func TestOpenOldLayoutAndOutsideGit(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	git(t, a.Cwd, "branch", "huginn/x")
	execute(t, a, "ls")
	for _, ref := range []string{"refs/heads/huginn", "refs/remotes/origin/huginn"} {
		b := app(t)
		execute(t, b, "init")
		git(t, b.Cwd, "update-ref", ref, "HEAD")
		_, e := b.Execute(context.Background(), []string{"ls"})
		if _, p, fix := ErrorDetails(e); e == nil || !strings.Contains(p, "old layout") || fix != "hug upgrade" {
			t.Fatalf("%s: got %v", ref, e)
		}
	}
	outside := &App{Cwd: t.TempDir(), In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: a.Env}
	_, e := outside.Execute(context.Background(), []string{"ls"})
	if _, p, _ := ErrorDetails(e); e == nil || !strings.Contains(p, "needs a Git checkout") {
		t.Fatalf("outside git: %v", e)
	}
}
