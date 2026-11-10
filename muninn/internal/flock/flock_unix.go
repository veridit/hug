//go:build !windows

package flock

import (
	"os"

	"golang.org/x/sys/unix"
)

func lock(f *os.File) error {
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX)
		if err != unix.EINTR {
			return err
		}
	}
}
