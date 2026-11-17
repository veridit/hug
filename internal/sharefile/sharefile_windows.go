//go:build windows

package sharefile

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// open shares delete as well as read and write, so a writer can rename a
// new version over the file while it is read.
func open(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}

// rename replaces newpath with FILE_RENAME_POSIX_SEMANTICS, as os.Root does
// on Windows (Go's internal/syscall/windows Renameat): it succeeds while
// others hold newpath open with delete sharing, and they keep the old bytes.
func rename(oldpath, newpath string) error {
	dir := filepath.Dir(newpath)
	if filepath.Dir(oldpath) != dir {
		return os.Rename(oldpath, newpath)
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer r.Close()
	if err = r.Rename(filepath.Base(oldpath), filepath.Base(newpath)); err != nil {
		if le, ok := err.(*os.LinkError); ok {
			err = le.Err
		}
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: err}
	}
	return nil
}
