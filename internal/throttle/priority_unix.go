//go:build !windows

package throttle

import (
	"strconv"

	"golang.org/x/sys/unix"
)

// lowerPriority raises niceness to Nice, never lowering an already higher one.
func lowerPriority() {
	cur, err := unix.Getpriority(unix.PRIO_PROCESS, 0)
	if err != nil {
		return
	}
	// Linux's raw getpriority returns 20-nice; x/sys reports the raw value
	// there and the nice value on BSD/Darwin.
	nice := cur
	if runtimeLinux {
		nice = 20 - cur
	}
	if nice < Nice {
		_ = unix.Setpriority(unix.PRIO_PROCESS, 0, Nice)
	}
	// Linux: the call above lowered only this thread, and this thread's
	// value says nothing of the others (1rca5g). Off Linux a no-op.
	lowerThreads()
}

// Priority is this process's niceness as a word for a log line: "nice 10".
// Off Linux niceness is per process; on Linux, the calling thread's.
func Priority() string {
	p, err := unix.Getpriority(unix.PRIO_PROCESS, 0)
	if err != nil {
		return "nice ?"
	}
	if runtimeLinux {
		p = 20 - p
	}
	return "nice " + strconv.Itoa(p)
}

// Lowered is what Priority says in a process after Background: Nice, or
// the niceness it already had when that was higher.
func Lowered() string {
	p, err := unix.Getpriority(unix.PRIO_PROCESS, 0)
	if err != nil {
		return "nice ?"
	}
	if runtimeLinux {
		p = 20 - p
	}
	return "nice " + strconv.Itoa(max(Nice, p))
}
