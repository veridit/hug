//go:build windows

package gitbranch

import (
	"os/exec"
	"syscall"
)

// createNoWindow gives the syncer a console that is never shown. Not
// DETACHED_PROCESS: a process without a console makes Windows open a new
// console for every console program it starts, here every git call, which
// is slow and flashes a window on the user's desktop each time.
const createNoWindow = 0x08000000

// detach starts a background syncer with a hidden console, in its own
// process group: it outlives the command that nudged it and gets no Ctrl-C.
func detach(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP, HideWindow: true}
}
