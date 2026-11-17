//go:build !windows

package sharefile

import "os"

// Unix opens never keep a file from being renamed over.
func open(path string) (*os.File, error) { return os.Open(path) }

func rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }
