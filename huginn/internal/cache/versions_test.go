package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// gjmqhn: two hug versions on one machine keep their own cache. Observed
// 2026-11-17: a v0.7.41 hug tui (schema 12) and v0.7.42 (schema 14) shared
// .git/huginn/cache.db and dropped each other's tables on every open.
func TestTwoSchemasAlternateWithoutRebuild(t *testing.T) {
	root := t.TempDir()
	put(t, root, "A.000001.task.md", fixture(1))
	common := t.TempDir()
	open := func(version string) Stats {
		t.Helper()
		defer func(v string) { schema = v }(schema)
		schema = version
		c := openTest(t, root, File(common), nil)
		defer c.Close()
		st, e := c.Sync(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		return st
	}
	for round := 0; round < 4; round++ {
		for _, v := range []string{"old", SchemaVersion} {
			st := open(v)
			if want := round == 0; st.Rebuilt != want {
				t.Fatalf("round %d schema %s: rebuilt %v, want %v", round, v, st.Rebuilt, want)
			}
		}
	}
	for _, v := range []string{"old", SchemaVersion} {
		if _, e := os.Stat(filepath.Join(common, "huginn", "cache-"+v+".db")); e != nil {
			t.Errorf("schema %s has no file of its own: %v", v, e)
		}
	}
}

// A cache file no hug has used for a week is removed by the next open;
// fresher ones, and the one opened, are kept.
func TestStaleCacheFilesRemoved(t *testing.T) {
	root := t.TempDir()
	put(t, root, "A.000001.task.md", fixture(1))
	common := t.TempDir()
	dir := filepath.Join(common, "huginn")
	if e := os.MkdirAll(dir, 0o755); e != nil {
		t.Fatal(e)
	}
	old := time.Now().Add(-8 * 24 * time.Hour)
	recent := time.Now().Add(-6 * 24 * time.Hour)
	files := map[string]time.Time{
		"cache.db": old, "cache.db-wal": old, "cache.db-shm": old, // legacy, a week unused
		"cache-12.db": old, "cache-12.db-wal": old,
		"cache-13.db": old, "cache-13.db-wal": recent, // its WAL was written 6 days ago: in use
		"cache-14.db": recent,
		"notes.txt":   old, // not a cache file
	}
	for name, at := range files {
		p := filepath.Join(dir, name)
		if e := os.WriteFile(p, []byte("x"), 0o644); e != nil {
			t.Fatal(e)
		}
		if e := os.Chtimes(p, at, at); e != nil {
			t.Fatal(e)
		}
	}
	c := openTest(t, root, File(common), nil)
	c.Close()
	for name := range files {
		_, e := os.Stat(filepath.Join(dir, name))
		gone := os.IsNotExist(e)
		want := name == "cache.db" || name == "cache.db-wal" || name == "cache.db-shm" || name == "cache-12.db" || name == "cache-12.db-wal"
		if gone != want {
			t.Errorf("%s: removed %v, want %v", name, gone, want)
		}
	}
	if _, e := os.Stat(File(common)); e != nil {
		t.Errorf("the cache opened is gone: %v", e)
	}
}

// Opening a cache marks it used, even when the sync writes nothing: a
// board read every day but changed for a week is not removed.
func TestOpenMarksCacheUsed(t *testing.T) {
	root := t.TempDir()
	put(t, root, "A.000001.task.md", fixture(1))
	common := t.TempDir()
	c := openTest(t, root, File(common), nil)
	if _, e := c.Sync(context.Background()); e != nil {
		t.Fatal(e)
	}
	c.Close()
	old := time.Now().Add(-8 * 24 * time.Hour)
	for _, s := range []string{"", "-wal", "-shm"} {
		os.Chtimes(File(common)+s, old, old)
	}
	c = openTest(t, root, File(common), nil)
	c.Close()
	st, e := os.Stat(File(common))
	if e != nil || time.Since(st.ModTime()) > time.Hour {
		t.Fatalf("open did not mark the cache used: %v %v", st, e)
	}
}
