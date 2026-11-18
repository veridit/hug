//go:build freebsd || openbsd || netbsd || dragonfly || darwin

package fsnotify

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// fsnotify v1.10.1's kqueue Close closed every watch descriptor while
// readEvents closed the same ones on the Remove events of files being
// deleted (fsnotify issue 788). The second close hit whatever had reused the
// number: on macOS CI a t.TempDir cleanup failed "unlinkat ...: bad file
// descriptor" in TestChangeBeforeWatchIsUpIsSeen (mq5ger).
//
// TestCloseWhileFilesGoClosesNothingElse closes watchers while their files
// are deleted, and holds files open meanwhile: none of them may lose its
// descriptor, and the deletion may not fail.
func TestCloseWhileFilesGoClosesNothingElse(t *testing.T) {
	rounds := 300
	if testing.Short() {
		rounds = 30
	}
	base := t.TempDir()
	sentinel := filepath.Join(base, "sentinel")
	if err := os.WriteFile(sentinel, []byte("s"), 0o644); err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	stray, first := 0, ""
	fail := func(format string, a ...any) {
		stray++
		if first == "" {
			first = fmt.Sprintf(format, a...)
		}
	}
	for r := range rounds {
		root := filepath.Join(base, fmt.Sprint(r))
		w, err := NewWatcher()
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			for range w.Events {
			}
		}()
		go func() {
			for range w.Errors {
			}
		}()
		for g := range 4 {
			dir := filepath.Join(root, fmt.Sprint("g", g))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for f := range 20 {
				if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("t%02d.md", f)), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Add(dir); err != nil {
				t.Fatal(err)
			}
		}
		removed, closed := make(chan error, 1), make(chan struct{})
		go func() { removed <- os.RemoveAll(root) }()
		go func() { w.Close(); close(closed) }()
		var held []*os.File
		for done := false; !done && len(held) < 2000; {
			select {
			case <-closed:
				done = true
			default:
			}
			f, err := os.Open(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			held = append(held, f)
		}
		<-closed
		if err := <-removed; err != nil {
			fail("round %d: %v", r, err)
		}
		for _, f := range held {
			if fi, err := f.Stat(); err != nil || !os.SameFile(fi, want) {
				fail("round %d: a file held open lost its descriptor: %v", r, err)
			}
			f.Close()
		}
	}
	if stray > 0 {
		t.Fatalf("%d descriptors closed by someone else in %d rounds; first: %s", stray, rounds, first)
	}
}
