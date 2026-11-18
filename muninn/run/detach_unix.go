//go:build !windows

package run

import (
	"os"
	"os/exec"
	"syscall"
)

// spawnDetached starts argv in its own session: it outlives the hook that
// started it and gets no terminal signals.
func spawnDetached(argv []string) error {
	c := exec.Command(argv[0], argv[1:]...)
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		return err
	}
	return c.Process.Release()
}

// execInstalled replaces this process with argv[0], same pid.
func execInstalled(argv []string) error { return syscall.Exec(argv[0], argv, os.Environ()) }
