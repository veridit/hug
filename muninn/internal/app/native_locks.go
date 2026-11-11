package app

import "github.com/veridit/muninn/internal/ingest"

// nativeLocks holds the write lock of each native store a multi-file flow
// (correction, duplicate tidy) loads, from load until save. Release them all
// before Reindex: indexing writes SIDs back under the same lock.
type nativeLocks map[string]*ingest.NativeLock

func (n nativeLocks) hold(path string) error {
	l := n[path]
	if l == nil {
		l = &ingest.NativeLock{Path: path}
		n[path] = l
	}
	return l.Hold()
}

func (n nativeLocks) holdAll() error {
	for _, l := range n {
		if err := l.Hold(); err != nil {
			return err
		}
	}
	return nil
}

func (n nativeLocks) release() {
	for _, l := range n {
		l.Release()
	}
}

// restoreLocked rolls files back with their locks held, then releases them.
func (n nativeLocks) restore(files []tidyFile) error {
	if err := n.holdAll(); err != nil {
		return err
	}
	defer n.release()
	return restoreTidyFiles(files)
}
