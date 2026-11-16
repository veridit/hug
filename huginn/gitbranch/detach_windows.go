//go:build windows

package gitbranch

import (
	"os/exec"
	"syscall"
)

const detachedProcess = 0x00000008

// detach starts a background syncer without a console, in its own process
// group: it outlives the command that nudged it and gets no Ctrl-C.
func detach(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP, HideWindow: true}
}
