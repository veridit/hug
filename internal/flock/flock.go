// Package flock takes exclusive cross-process locks on lock files.
package flock

import (
	"os"
	"path/filepath"
)

// Acquire blocks until it holds an exclusive lock on path (created if
// missing) and returns the function that releases it. The lock is per open
// file: a second Acquire of the same path in one process also blocks, so
// callers that may nest serialize within the process themselves.
//
// Release unlocks before it closes (3qa9bt). Close alone frees the lock
// only when the last descriptor of the open file goes, and a fork in
// another goroutine (any exec.Command: git) copies every descriptor into a
// child that keeps it until it execs. Released, the lock then still looked
// held for some milliseconds; settleWait on a synctest clock counted each
// poll in that gap as 50 ms and returned 1-3 s late (CI 44720742380,
// 44907060648). An unlock frees the open file itself, whatever copies of
// its descriptor live on.
func Acquire(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lock(f); err != nil {
		f.Close()
		return nil, err
	}
	return releaser(f), nil
}

// TryAcquire is Acquire without waiting: ok is false, with no error, when
// another open file holds the lock.
func TryAcquire(path string) (release func(), ok bool, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, err
	}
	held, err := tryLock(f)
	if err != nil || !held {
		f.Close()
		return nil, false, err
	}
	return releaser(f), true, nil
}

// releaser unlocks f, then closes it (see Acquire).
func releaser(f *os.File) func() {
	return func() {
		_ = unlock(f)
		f.Close()
	}
}
