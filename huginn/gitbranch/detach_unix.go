//go:build !windows

package gitbranch

import (
	"os/exec"
	"syscall"
)

// detach gives a background syncer its own session: it outlives the
// command that nudged it and gets no terminal signals.
func detach(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
