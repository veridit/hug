package ingest

import (
	"errors"
	"os"
	"reflect"
	"sort"
	"sync"

	"github.com/veridit/hug/internal/flock"
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
	_, err := UpdateNativeChanged(path, change)
	return err
}

// Written is what one write did to the store file, for an index that
// updates only that (t0qap9): Before and After are the file's stat just
// before the load and just after the save, both under the lock, so an index
// that recorded Before differs from the file at After by exactly Changed
// (new or edited memories, as saved) and Removed (ids gone from the file).
// A write that saved nothing has After == Before and nothing changed.
type Written struct {
	Before, After FileStat
	Changed       []NativeMemory
	Removed       []string
}

// UpdateNativeChanged is UpdateNative, reporting what the write changed.
func UpdateNativeChanged(path string, change func(mems []NativeMemory) ([]NativeMemory, bool, error)) (Written, error) {
	var w Written
	unlock, err := LockNative(path)
	if err != nil {
		return w, err
	}
	defer unlock()
	if w.Before, err = StatFile(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return w, err
		}
		w.Before = FileStat{-1, -1} // no file yet: no index recorded it
	}
	mems, err := LoadNative(path)
	if err != nil {
		return w, err
	}
	if st, err := StatFile(path); err == nil && st != w.Before {
		// LoadNative repaired duplicates and saved: the file is no longer
		// the one an index may have recorded. Make Before say so.
		w.Before = FileStat{-1, -1}
	}
	before := make(map[string]NativeMemory, len(mems))
	for _, m := range CloneNative(mems) {
		before[m.ID] = m
	}
	out, save, err := change(mems)
	if err != nil || !save {
		w.After = w.Before
		return w, err
	}
	if err := SaveNative(path, out); err != nil {
		return w, err
	}
	kept := make(map[string]bool, len(out))
	for _, m := range out {
		kept[m.ID] = true
		if old, ok := before[m.ID]; !ok || !reflect.DeepEqual(old, m) {
			w.Changed = append(w.Changed, m)
		}
	}
	for id := range before {
		if !kept[id] {
			w.Removed = append(w.Removed, id)
		}
	}
	sort.Strings(w.Removed)
	w.Changed = CloneNative(w.Changed)
	w.After, err = StatFile(path)
	return w, err
}
