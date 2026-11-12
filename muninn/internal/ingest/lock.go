package ingest

import (
	"sync"

	"github.com/veridit/hug/muninn/internal/flock"
)

// Several processes write the native store: recall hooks, capture, MCP servers
// for different harnesses (jcode, Claude Code, the Claude app), the CLI and
// tidy. Each write is load, change, save of the whole file, so two at once could
// lose one change (measured 2026-11-10: 29 of 30 memories survived two
// concurrent writers). LockNative serializes those spans across processes with a
// lock file next to the store, and within a process with a mutex (flock is per
// open file on BSD/macOS, so a second open in one process would also block).

var nativeMu sync.Mutex

// LockNative holds the native store's write lock until the returned func runs.
// Do not call a function that locks again while holding it.
func LockNative(path string) (func(), error) {
	nativeMu.Lock()
	release, err := flock.Acquire(path + ".lock")
	if err != nil {
		nativeMu.Unlock()
		return nil, err
	}
	return func() {
		release()
		nativeMu.Unlock()
	}, nil
}

// NativeLock is a holdable LockNative for flows that take and drop the lock at
// several points (apply, then roll back on error) in one goroutine. Hold and
// Release are idempotent, so a rollback can Hold without knowing whether the
// lock is already held, and a deferred Release is always safe.
type NativeLock struct {
	Path   string
	unlock func()
}

func (l *NativeLock) Hold() error {
	if l.unlock != nil {
		return nil
	}
	u, err := LockNative(l.Path)
	if err != nil {
		return err
	}
	l.unlock = u
	return nil
}

func (l *NativeLock) Release() {
	if l.unlock != nil {
		l.unlock()
		l.unlock = nil
	}
}

// UpdateNative loads the store, lets change edit it, and saves the result, all
// under the write lock. change returns save=false to leave the file untouched.
func UpdateNative(path string, change func(mems []NativeMemory) ([]NativeMemory, bool, error)) error {
	unlock, err := LockNative(path)
	if err != nil {
		return err
	}
	defer unlock()
	mems, err := LoadNative(path)
	if err != nil {
		return err
	}
	out, save, err := change(mems)
	if err != nil || !save {
		return err
	}
	return SaveNative(path, out)
}
