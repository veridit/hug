//go:build !windows

package tui

import "syscall"

// reexec replaces this process with path (j2vwk0): same pid, same terminal,
// never a second TUI.
func reexec(path string, args, env []string) error {
	return syscall.Exec(path, args, env)
}

// canReexec says whether this platform can replace the running process.
const canReexec = true
