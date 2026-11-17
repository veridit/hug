package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Progress counts the files a sync reads, of the files it found: a first
// sync reads all of them, a sync of an unchanged board reads none (vnyz8z:
// hug tui counts them on its first start).
func TestProgressCountsReads(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-time.Hour) // past the racy window: markers are trusted
	for i := 1; i <= 7; i++ {
		p := fmt.Sprintf("todo/%s.t%d.md", testID(i), i)
		put(t, root, p, fixture(i))
		if e := os.Chtimes(filepath.Join(root, p), old, old); e != nil {
			t.Fatal(e)
		}
	}
	c := openTest(t, root, filepath.Join(t.TempDir(), "c.db"), &fakeGit{head: "h"})
	defer c.Close()
	if r, n := c.Progress(); r != 0 || n != 0 {
		t.Fatalf("before a sync: %d of %d", r, n)
	}
	if _, e := c.Sync(context.Background()); e != nil {
		t.Fatal(e)
	}
	if r, n := c.Progress(); r != 7 || n != 7 {
		t.Fatalf("first sync: %d of %d, want 7 of 7", r, n)
	}
	if _, e := c.Sync(context.Background()); e != nil {
		t.Fatal(e)
	}
	if r, _ := c.Progress(); r != 0 {
		t.Fatalf("unchanged board: read %d files, want 0", r)
	}
}
