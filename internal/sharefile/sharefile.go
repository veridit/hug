// Package sharefile reads and replaces files that other processes read and
// replace at the same time (kbc99j). On Windows, os.ReadFile opens without
// FILE_SHARE_DELETE and os.Rename replaces with MoveFileEx, so a read and a
// rename over the same file fail each other: "The process cannot access the
// file because it is being used by another process" and "Access is
// denied" (CI run 44086864807: 6 of 11646 reads, 165 of 169 renames). Here a
// read lets the file be replaced under it, and a replace is POSIX: readers
// keep the old bytes, the name points at the new ones at once.
package sharefile

import (
	"io"
	"os"
)

// Open opens path for reading; on Windows the file may be renamed over or
// removed while it is open.
func Open(path string) (*os.File, error) { return open(path) }

// ReadFile is os.ReadFile through Open.
func ReadFile(path string) ([]byte, error) {
	f, err := open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Rename is os.Rename for a file, except on Windows it replaces newpath
// with POSIX semantics: it succeeds while others have newpath open.
func Rename(oldpath, newpath string) error { return rename(oldpath, newpath) }
