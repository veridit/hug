//go:build !windows

package flock

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// 3qa9bt: a released lock is free even while another descriptor of the
// same open file lives on. A fork in another goroutine (any exec.Command,
// e.g. git) copies every descriptor into the child until it execs; a dup
// shares the open file the same way, and holds it for the whole check
// instead of a few racy milliseconds. Releasing by Close alone fails here:
// the copy keeps the open file, and with it the lock (on a synctest clock
// settleWait counted each poll in that gap as 50 ms: 1m31.95s, 1m32.6s
// on CI where it wanted 1m30s).
func TestReleaseFreesLockACopiedDescriptorShares(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run")
	for _, take := range []struct {
		name    string
		acquire func(t *testing.T) func()
	}{
		{"Acquire", func(t *testing.T) func() {
			release, err := Acquire(path)
			if err != nil {
				t.Fatal(err)
			}
			return release
		}},
		{"TryAcquire", func(t *testing.T) func() {
			release, ok, err := TryAcquire(path)
			if err != nil || !ok {
				t.Fatalf("TryAcquire of a free lock: ok %v, error %v", ok, err)
			}
			return release
		}},
	} {
		t.Run(take.name, func(t *testing.T) {
			release := take.acquire(t)
			copied := dupOf(t, path)
			defer unix.Close(copied)
			release()
			again, ok, err := TryAcquire(path)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("lock still held after release while a copy of its descriptor is open (a forked child's, until it execs): release must unlock, not only close (3qa9bt)")
			}
			again()
		})
	}
}

// dupOf duplicates this process's open descriptor of path, found by its
// inode: the lock's own descriptor, which Acquire does not hand out.
func dupOf(t *testing.T, path string) int {
	t.Helper()
	var want unix.Stat_t
	if err := unix.Stat(path, &want); err != nil {
		t.Fatal(err)
	}
	for fd := 0; fd < 4096; fd++ {
		var st unix.Stat_t
		if unix.Fstat(fd, &st) != nil || st.Dev != want.Dev || st.Ino != want.Ino {
			continue
		}
		copied, err := unix.Dup(fd)
		if err != nil {
			t.Fatal(err)
		}
		return copied
	}
	t.Fatalf("no open descriptor of %s", path)
	return -1
}
