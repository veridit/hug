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
	return func() { f.Close() }, nil
}
