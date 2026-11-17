package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 44w1ev step 8: a write rereads only its own files (ListAfterWrite), and
// that never hides a change anyone else made: the next full sync still sees
// it, because ListAfterWrite neither advances the sync time nor the head.
func TestListAfterWriteRereadsOnlyItsFiles(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db := filepath.Join(t.TempDir(), "cache.db")
	old := time.Now().Add(-time.Hour)
	for i := 1; i <= 4; i++ {
		p := filepath.Join(root, fmt.Sprintf("A.%s.task.md", testID(i)))
		put(t, root, filepath.Base(p), fixture(i))
		if e := os.Chtimes(p, old, old); e != nil {
			t.Fatal(e)
		}
	}
	g := &fakeGit{head: "h1"}
	c := openTest(t, root, db, g)
	defer c.Close()
	if _, e := c.List(ctx); e != nil {
		t.Fatal(e)
	}
	mine := fmt.Sprintf("A.%s.task.md", testID(1))
	other := fmt.Sprintf("A.%s.task.md", testID(2))
	gone := fmt.Sprintf("A.%s.task.md", testID(3))
	// Someone else edits other, keeping size and time: only a hash sees it.
	st, _ := os.Stat(filepath.Join(root, other))
	put(t, root, other, []byte(strings.Replace(string(fixture(2)), "Task 2", "Task Y", 1)))
	os.Chtimes(filepath.Join(root, other), st.ModTime(), st.ModTime())
	// This process writes mine, renames gone away and adds a new file.
	put(t, root, mine, []byte(strings.Replace(string(fixture(1)), "Task 1", "Task X", 1)))
	moved := fmt.Sprintf("B.%s.task.md", testID(3))
	if e := os.Rename(filepath.Join(root, gone), filepath.Join(root, moved)); e != nil {
		t.Fatal(e)
	}
	ts, e := c.ListAfterWrite(ctx, []string{mine, gone, moved})
	if e != nil {
		t.Fatal(e)
	}
	titles := map[string]string{}
	for _, x := range ts {
		titles[x.Path] = x.Title
	}
	if titles[mine] != "Task X" || titles[moved] != "Task 3" || titles[gone] != "" || len(ts) != 4 {
		t.Fatalf("after its own writes: %v", titles)
	}
	if titles[other] != "Task 2" {
		t.Fatalf("ListAfterWrite reread a file it did not write: %q", titles[other])
	}
	// The next full sync sees the other edit, though its marker is unchanged.
	g.head = "h2"
	g.changes = []Change{{Status: "M", Path: other}}
	ts, e = c.List(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, x := range ts {
		if x.Path == other && x.Title != "Task Y" {
			t.Fatalf("full sync after ListAfterWrite missed another writer's edit: %q", x.Title)
		}
	}
	// A reopened cache agrees with the files.
	c.Close()
	c2 := openTest(t, root, db, g)
	defer c2.Close()
	ts, e = c2.List(ctx)
	if e != nil {
		t.Fatal(e)
	}
	got := map[string]string{}
	for _, x := range ts {
		got[x.Path] = x.Title
	}
	if got[mine] != "Task X" || got[moved] != "Task 3" || got[other] != "Task Y" || len(got) != 4 {
		t.Fatalf("reopened: %v", got)
	}
}

// The racy window still applies after ListAfterWrite: a same-size edit in
// the same tick as the last full sync, to a file the write did not touch, is
// hashed by the next sync. ListAfterWrite must not move the sync time
// forward, or that edit would look settled.
func TestListAfterWriteKeepsRacyWindow(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db := filepath.Join(t.TempDir(), "cache.db")
	when := time.Now().Add(-5 * time.Second)
	for i := 1; i <= 2; i++ {
		p := fmt.Sprintf("A.%s.task.md", testID(i))
		put(t, root, p, fixture(i))
		if e := os.Chtimes(filepath.Join(root, p), when, when); e != nil {
			t.Fatal(e)
		}
	}
	g := &fakeGit{head: "h1"}
	c := openTest(t, root, db, g)
	defer c.Close()
	if _, e := c.List(ctx); e != nil {
		t.Fatal(e)
	}
	// As if that sync ran 1 s after the files were written: they are racy.
	c.synced = when.Add(time.Second).UnixNano()
	mine := fmt.Sprintf("A.%s.task.md", testID(1))
	other := fmt.Sprintf("A.%s.task.md", testID(2))
	put(t, root, other, []byte(strings.Replace(string(fixture(2)), "Task 2", "Task Z", 1)))
	os.Chtimes(filepath.Join(root, other), when, when)
	put(t, root, mine, []byte(strings.Replace(string(fixture(1)), "Task 1", "Task W", 1)))
	if _, e := c.ListAfterWrite(ctx, []string{mine}); e != nil {
		t.Fatal(e)
	}
	ts, e := c.List(ctx)
	if e != nil {
		t.Fatal(e)
	}
	seen := false
	for _, x := range ts {
		if x.Path == other {
			seen = x.Title == "Task Z"
		}
	}
	if !seen {
		t.Fatal("a same-size edit in the last sync's tick was missed after ListAfterWrite")
	}
}

// ListAfterWrite costs the written paths only: no walk, no hash of another
// file, even when the others are within the racy window.
func TestListAfterWriteStatsOnlyItsPaths(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	for i := 1; i <= 20; i++ {
		put(t, root, fmt.Sprintf("A.%s.task.md", testID(i)), fixture(i))
	}
	c := openTest(t, root, filepath.Join(t.TempDir(), "cache.db"), &fakeGit{head: "h"})
	defer c.Close()
	if _, e := c.List(ctx); e != nil {
		t.Fatal(e)
	}
	mine := fmt.Sprintf("A.%s.task.md", testID(1))
	put(t, root, mine, []byte(strings.Replace(string(fixture(1)), "Task 1", "Task V", 1)))
	c.mu.Lock()
	st, e := c.sync(ctx, map[string]bool{mine: true})
	c.mu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	if st.Statted != 1 || st.Hashed != 1 || st.Parsed != 1 {
		t.Fatalf("a write of 1 of 20 racy files: %+v, want 1 statted, hashed and parsed", st)
	}
}
