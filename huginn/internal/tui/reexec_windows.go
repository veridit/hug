//go:build windows

package tui

import "errors"

// Windows cannot replace a running process, and a second TUI is never
// started instead (j2vwk0): the TUI does not watch for a new hug there.
func reexec(path string, args, env []string) error {
	return errors.New("reload needs exec, which Windows does not have")
}

const canReexec = false
