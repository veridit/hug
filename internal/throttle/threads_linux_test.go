package throttle

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/veridit/hug/internal/env"
)

// threadNices reads the nice value of every thread of this process from
// /proc/self/task/TID/stat (field 19). Linux keeps niceness per thread.
func threadNices(t *testing.T) map[string]int {
	t.Helper()
	dirs, err := filepath.Glob("/proc/self/task/*")
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no /proc/self/task: %v", err)
	}
	out := map[string]int{}
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "stat"))
		if err != nil {
			continue // the thread exited
		}
		s := string(b)
		// Fields after the ")" that ends comm: state is field 3, nice field 19.
		f := strings.Fields(s[strings.LastIndexByte(s, ')')+1:])
		n, err := strconv.Atoi(f[19-3])
		if err != nil {
			t.Fatalf("stat %s: %q", d, s)
		}
		out[filepath.Base(d)] = n
	}
	return out
}

// TestBackgroundNicesEveryThread (1rca5g): Linux niceness is per thread,
// so Background must lower every thread the process runs on, including
// threads the Go runtime starts later. Run in a child (niceness cannot be
// raised back). Busy goroutines force several OS threads before and after
// Background, then every /proc/self/task entry is read.
func TestBackgroundNicesEveryThread(t *testing.T) {
	if os.Getenv("THROTTLE_THREADS_CHILD") == "1" {
		// Each goroutine locks its own OS thread and holds it until the
		// threads are read, so the threads made before and after
		// Background are all alive at the read.
		release := make(chan struct{})
		var ready sync.WaitGroup
		spawn := func(n int) {
			for range n {
				ready.Add(1)
				go func() {
					runtime.LockOSThread()
					ready.Done()
					<-release
				}()
			}
			ready.Wait()
		}
		spawn(4) // threads that exist before Background
		Background()
		spawn(4) // threads made after Background
		for tid, n := range threadNices(t) {
			os.Stdout.WriteString("tid=" + tid + " nice=" + strconv.Itoa(n) + "\n")
		}
		close(release)
		return
	}
	if os.Getenv(env.FullSpeed) != "" {
		t.Skip("full speed set in the environment")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBackgroundNicesEveryThread$")
	cmd.Env = append(os.Environ(), "THROTTLE_THREADS_CHILD=1", "GOMAXPROCS=")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("child: %v %s", err, out)
	}
	threads, low := 0, []string{}
	for _, l := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(l, "tid=") {
			continue
		}
		threads++
		n, _ := strconv.Atoi(l[strings.LastIndex(l, "=")+1:])
		if n < Nice {
			low = append(low, l)
		}
	}
	if threads < 2 {
		t.Fatalf("want several threads, got %d: %q", threads, out)
	}
	if len(low) > 0 {
		t.Fatalf("%d of %d threads not niced to %d after Background: %v", len(low), threads, Nice, low)
	}
}
