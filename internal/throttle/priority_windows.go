//go:build windows

package throttle

import (
	"strconv"

	"golang.org/x/sys/windows"
)

func lowerPriority() {
	if h, err := windows.GetCurrentProcess(); err == nil {
		_ = windows.SetPriorityClass(h, windows.BELOW_NORMAL_PRIORITY_CLASS)
	}
}

// Priority is this process's priority class as a word for a log line.
func Priority() string {
	h, err := windows.GetCurrentProcess()
	if err != nil {
		return "class ?"
	}
	c, err := windows.GetPriorityClass(h)
	if err != nil {
		return "class ?"
	}
	switch c {
	case windows.BELOW_NORMAL_PRIORITY_CLASS:
		return "class below-normal"
	case windows.NORMAL_PRIORITY_CLASS:
		return "class normal"
	case windows.IDLE_PRIORITY_CLASS:
		return "class idle"
	}
	return "class " + strconv.FormatUint(uint64(c), 16)
}

// Lowered is what Priority says in a process after Background.
func Lowered() string { return "class below-normal" }
