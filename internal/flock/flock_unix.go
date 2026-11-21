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

func tryLock(f *os.File) (bool, error) {
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		switch err {
		case nil:
			return true, nil
		case unix.EWOULDBLOCK:
			return false, nil
		case unix.EINTR:
			continue
		}
		return false, err
	}
}

func unlock(f *os.File) error {
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_UN)
		if err != unix.EINTR {
			return err
		}
	}
}
