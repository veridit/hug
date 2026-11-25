//go:build !windows

package main

import (
	"os"
	"syscall"
)

// fileInode is path's inode now, 0 when it cannot be read.
func fileInode(path string) uint64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Ino)
	}
	return 0
}
