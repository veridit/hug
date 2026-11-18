package cli

// MEASURE ONLY (0n4rnx, branch ci/0n4rnx-measure-split, never lands):
// every 100 ms, each goroutine of a cli test is put in one bucket by what
// it is blocked in, and the counts per test are written to $ZZ_SAMPLE.

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var zzTest = regexp.MustCompile(`cli\.(Test[A-Za-z0-9_]+)`)

func init() {
	p := os.Getenv("ZZ_SAMPLE")
	if p == "" || len(os.Args) == 4 {
		return
	}
	go func() {
		counts := map[string]int{}
		buf := make([]byte, 16<<20)
		for i := 0; ; i++ {
			time.Sleep(100 * time.Millisecond)
			n := runtime.Stack(buf, true)
			for _, g := range strings.Split(string(buf[:n]), "\n\n") {
				m := zzTest.FindAllStringSubmatch(g, -1)
				if m == nil || strings.Contains(g, "testing.(*T).Parallel") || strings.Contains(g, "testing.tRunner.func1") || strings.Contains(g, "testing.runTests") {
					continue
				}
				test := m[len(m)-1][1]
				k := "run"
				switch {
				case strings.Contains(g, "os/exec.(*Cmd)") || strings.Contains(g, "os.(*Process)") || strings.Contains(g, "syscall.forkExec") || strings.Contains(g, "syscall.StartProcess"):
					k = "exec"
				case strings.Contains(g, "time.Sleep"):
					k = "sleep"
				case strings.Contains(g, "flock.") || strings.Contains(g, "LockFileEx") || strings.Contains(g, "syscall.Flock"):
					k = "lock"
				}
				counts[test+"\t"+k]++
			}
			if i%10 == 9 {
				var b strings.Builder
				for k, v := range counts {
					fmt.Fprintf(&b, "%s\t%d\n", k, v)
				}
				os.WriteFile(p, []byte(b.String()), 0o644)
			}
		}
	}()
}
