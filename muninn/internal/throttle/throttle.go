// Package throttle keeps muninn's background work polite: hooks, capture,
// indexing and the watcher run at lowered CPU priority and with bounded
// parallelism, so they take a little longer instead of spiking the machine.
package throttle

import (
	"github.com/veridit/hug/internal/env"
	"os"
	"runtime"
)

// Nice is the Unix niceness background work runs at.
const Nice = 10

// Procs is the parallelism background work uses: a quarter of the cores,
// at least one, at most four.
func Procs() int {
	return min(4, max(1, runtime.NumCPU()/4))
}

// Background lowers this process's CPU priority and caps GOMAXPROCS.
// HUG_FULL_SPEED=1 opts out. An explicit GOMAXPROCS is respected.
// Failures are ignored: being unable to lower priority is never fatal.
func Background() {
	if env.Get(env.FullSpeed) != "" {
		return
	}
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(Procs())
	}
	lowerPriority()
}
