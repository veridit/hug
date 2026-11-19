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

// 5mmqjk: Peek takes a task whose file still has the marker the cache
// recorded from the cache, and parses only files that changed since: a
// file rewritten with the same size and time (the racy case a sync guards
// by its window) still reads as cached, proof the cache answered; a file
// whose time moved reads as it is now. It writes nothing.
func TestPeekUsesFreshRecords(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	dbDir := t.TempDir()
	db := filepath.Join(dbDir, "cache.db")
	old := time.Now().Add(-time.Hour)
	name := func(n int) string { return fmt.Sprintf("A.%s.task.md", testID(n)) }
	for n := 1; n <= 3; n++ {
		put(t, root, name(n), fixture(n))
		os.Chtimes(filepath.Join(root, name(n)), old, old)
	}
	c := openTest(t, root, db, nil)
	if _, e := c.Sync(ctx); e != nil {
		t.Fatal(e)
	}
	c.Close()
	// Same size and time, other bytes: only the cache still says Task 1.
	put(t, root, name(1), []byte(strings.Replace(string(fixture(1)), "Task 1", "Task X", 1)))
	os.Chtimes(filepath.Join(root, name(1)), old, old)
	// A real edit: the time moves.
	put(t, root, name(2), []byte(strings.Replace(string(fixture(2)), "Task 2", "Task two", 1)))
	put(t, root, name(4), fixture(4))
	before := dirState(t, dbDir)
	ts, e := Peek(ctx, root, db)
	if e != nil {
		t.Fatal(e)
	}
	var got []string
	for _, x := range ts {
		got = append(got, x.Title)
	}
	if want := "Task 1,Task two,Task 3,Task 4"; strings.Join(got, ",") != want {
		t.Errorf("Peek titles %q, want %s", got, want)
	}
	if ts[0].ID != testID(1) || ts[0].State != "todo" {
		t.Errorf("Peek first brief %+v", ts[0])
	}
	if after := dirState(t, dbDir); after != before {
		t.Errorf("Peek changed the cache folder:\n%s\n%s", before, after)
	}
	// No cache: every file parsed, nothing created.
	none := filepath.Join(t.TempDir(), "none.db")
	if ts, e = Peek(ctx, root, none); e != nil || len(ts) != 4 || ts[0].Title != "Task X" {
		t.Errorf("Peek without a cache: %v %v", ts, e)
	}
	if _, e := os.Stat(none); !os.IsNotExist(e) {
		t.Errorf("Peek created %s", none)
	}
}

func dirState(t *testing.T, dir string) string {
	t.Helper()
	entries, e := os.ReadDir(dir)
	if e != nil {
		t.Fatal(e)
	}
	var out []string
	for _, d := range entries {
		info, _ := d.Info()
		out = append(out, fmt.Sprint(d.Name(), " ", info.Size(), " ", info.ModTime().UnixNano()))
	}
	return strings.Join(out, "\n")
}
