package cache

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// The board blob keeps only the references of each parse; notes and
// description texts live in table texts and come back on request (44w1ev:
// they were 14 of the blob's 21.6 MB at 10,000 tasks, decoded by every
// command for ls alone). FillTexts returns exactly what a fresh parse gives,
// also after a reopen and after an edit; List itself carries no bodies.
func TestTextsLeaveTheBlob(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db := filepath.Join(t.TempDir(), "cache.db")
	body := func(i int, note string) []byte {
		return []byte(fmt.Sprintf("---\nid: %s\ntitle: Task %d\nstate: todo\n---\n\n## Goal\nA visible result.\n\n## Purpose\nWhy %d.\n\n## Relevant details\nSee file:x.txt\n\n## Acceptance\n- [ ] passes\n\n## Notes\n%s\n", testID(i), i, i, note))
	}
	for i := 1; i <= 3; i++ {
		put(t, root, fmt.Sprintf("A.%06d.task.md", i), body(i, fmt.Sprintf("note %d", i)))
	}
	check := func(c *Cache, when string) {
		t.Helper()
		ts, e := c.List(ctx)
		if e != nil {
			t.Fatal(e)
		}
		for _, x := range ts {
			if x.Texts == nil || x.Texts.Bodies || x.Texts.Notes != "" || len(x.Texts.Description) != 0 {
				t.Fatalf("%s: List carries bodies for %s: %+v", when, x.ID, x.Texts)
			}
			if len(x.Texts.Refs) != 1 {
				t.Fatalf("%s: List lost the file reference of %s: %+v", when, x.ID, x.Texts.Refs)
			}
		}
		if e = c.FillTexts(ctx, ts); e != nil {
			t.Fatal(e)
		}
		for _, x := range ts {
			b, e := os.ReadFile(filepath.Join(root, x.Path))
			if e != nil {
				t.Fatal(e)
			}
			want := task.Parse(x.Path, b).Task.Texts
			if !x.Texts.Bodies || x.Texts.Notes != want.Notes || fmt.Sprint(x.Texts.Description) != fmt.Sprint(want.Description) || len(x.Texts.Refs) != len(want.Refs) {
				t.Fatalf("%s: FillTexts for %s = %+v, parse gives %+v", when, x.ID, x.Texts, want)
			}
		}
	}
	c := openTest(t, root, db, nil)
	check(c, "first open")
	var blob []byte
	c.db.QueryRow(`SELECT value FROM meta WHERE key='board'`).Scan(&blob)
	if bytes.Contains(blob, []byte("note 2")) || bytes.Contains(blob, []byte("Why 2.")) {
		t.Fatal("the board blob still holds notes or description texts")
	}
	// An edit lands in the delta and in table texts.
	put(t, root, "A.000002.task.md", body(2, "edited note, longer"))
	check(c, "after edit")
	c.Close()
	c = openTest(t, root, db, nil)
	check(c, "after reopen")
	// A file changed after the last sync: its stored texts are stale, and
	// FillTexts leaves the task as it was (the caller reads the file).
	ts, e := c.List(ctx)
	if e != nil {
		t.Fatal(e)
	}
	put(t, root, "A.000003.task.md", body(3, "changed behind the cache"))
	c.mu.Lock()
	r := c.files["A.000003.task.md"]
	r.SHA256 = strings.Repeat("0", 64)
	c.files["A.000003.task.md"] = r
	c.mu.Unlock()
	if e = c.FillTexts(ctx, ts); e != nil {
		t.Fatal(e)
	}
	for _, x := range ts {
		if x.Path == "A.000003.task.md" && x.Texts.Bodies {
			t.Fatalf("stale texts filled for %s: %+v", x.ID, x.Texts)
		}
	}
	c.Close()
}

func TestBodiesRoundTrip(t *testing.T) {
	for _, x := range []*task.Texts{
		{Bodies: true, Description: map[string]string{}},
		{Bodies: true, Notes: "a:b\n12:x", Description: map[string]string{"Purpose": "p", "Principle": "", "Relevant details": "ünï\x00:"}},
	} {
		got, e := decodeBodies(encodeBodies(x))
		if e != nil {
			t.Fatal(e)
		}
		for _, sec := range task.DescriptionSections {
			if got.Description[sec] != x.Description[sec] {
				t.Fatalf("%s: %q, want %q", sec, got.Description[sec], x.Description[sec])
			}
		}
		if got.Notes != x.Notes || !got.Bodies {
			t.Fatalf("notes %q, want %q", got.Notes, x.Notes)
		}
	}
	for _, bad := range []string{"", "x", "5:ab", "-1:", "2:ab"} {
		if _, e := decodeBodies([]byte(bad)); e == nil {
			t.Fatalf("decoded %q", bad)
		}
	}
}
