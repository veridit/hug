//go:build !linux

package throttle

const runtimeLinux = false

// lowerThreads is a no-op off Linux: there niceness belongs to the
// process, and lowerPriority already lowered it.
func lowerThreads() {}
