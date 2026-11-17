package throttle

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const runtimeLinux = true

// lowerThreads raises every thread of this process to Nice (1rca5g).
// Linux keeps niceness per thread: setpriority(PRIO_PROCESS, 0) lowers only
// the calling thread, and the Go runtime runs goroutines on many. A new
// thread inherits the niceness of the thread that creates it, so once every
// existing thread is lowered, later ones are too. Passes repeat until one
// finds nothing left to lower (a thread may start during a pass).
func lowerThreads() {
	for range 5 {
		ents, err := os.ReadDir("/proc/self/task")
		if err != nil {
			return
		}
		changed := false
		for _, e := range ents {
			tid, err := strconv.Atoi(e.Name())
			if err != nil {
				continue
			}
			if threadNice(tid) >= Nice {
				continue
			}
			if unix.Setpriority(unix.PRIO_PROCESS, tid, Nice) == nil {
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

// threadNice is a thread's niceness from /proc/self/task/TID/stat (field
// 19); on a read error it is taken as already lowered.
func threadNice(tid int) int {
	b, err := os.ReadFile("/proc/self/task/" + strconv.Itoa(tid) + "/stat")
	if err != nil {
		return Nice
	}
	s := string(b)
	f := strings.Fields(s[strings.LastIndexByte(s, ')')+1:])
	if len(f) <= 19-3 {
		return Nice
	}
	n, err := strconv.Atoi(f[19-3])
	if err != nil {
		return Nice
	}
	return n
}
