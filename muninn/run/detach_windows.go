//go:build windows

package run

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// spawnDetached starts argv with a hidden console in its own process group.
func spawnDetached(argv []string) error {
	c := exec.Command(argv[0], argv[1:]...)
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP, HideWindow: true}
	if err := c.Start(); err != nil {
		return err
	}
	return c.Process.Release()
}

// execInstalled has no exec on Windows: start the installed hug detached,
// and this waiter ends.
func execInstalled(argv []string) error { return spawnDetached(argv) }
