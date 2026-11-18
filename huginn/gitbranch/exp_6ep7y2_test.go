package gitbranch

// 6ep7y2 experiment E3, branch ci/6ep7y2-20261118 only: never lands on main.
// H3: a network-class git call against a local bare remote occasionally
// hangs on Windows until runAt's NetTimeout kills it.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestExp6ep7y2NetOps(t *testing.T) {
	if os.Getenv("EXP_6EP7Y2") != "1" {
		t.Skip("experiment only")
	}
	dur, _ := time.ParseDuration(os.Getenv("EXP_6EP7Y2_FOR"))
	if dur == 0 {
		dur = 8 * time.Minute
	}
	sh := func(dir string, args ...string) {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if b, e := c.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, b, e)
		}
	}
	type stat struct {
		n   int
		max time.Duration
	}
	var mu sync.Mutex
	stats := map[string]*stat{}
	var slow []string
	record := func(w int, op string, start time.Time, d time.Duration, e error) {
		mu.Lock()
		defer mu.Unlock()
		s := stats[op]
		if s == nil {
			s = &stat{}
			stats[op] = s
		}
		s.n++
		s.max = max(s.max, d)
		if d > 5*time.Second || e != nil {
			slow = append(slow, fmt.Sprintf("worker %d %s start %s took %v err %v", w, op, start.UTC().Format(time.RFC3339Nano), d.Round(time.Millisecond), e))
		}
	}
	var wg sync.WaitGroup
	deadline := time.Now().Add(dur)
	for w := 0; w < 8; w++ {
		base := t.TempDir()
		bare := filepath.Join(base, "remote.git")
		sh(base, "init", "-q", "--bare", "-b", "main", bare)
		hook := "#!/bin/sh\ncat >> \"$GIT_DIR/pushes\"\n"
		if e := os.WriteFile(filepath.Join(bare, "hooks", "post-receive"), []byte(hook), 0o755); e != nil {
			t.Fatal(e)
		}
		dir := filepath.Join(base, "one")
		sh(base, "init", "-q", "-b", "main", dir)
		sh(dir, "config", "user.name", "T")
		sh(dir, "config", "user.email", "t@example.invalid")
		sh(dir, "config", "core.autocrlf", "false")
		sh(dir, "commit", "-q", "--allow-empty", "-m", "init")
		sh(dir, "remote", "add", "origin", bare)
		sh(dir, "push", "-q", "origin", "main")
		r := &Repository{Dir: dir, Root: dir, Remote: "origin", Branch: "main"}
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				for _, op := range [][]string{
					{"commit", "-q", "--allow-empty", "-m", "n" + strconv.Itoa(i)},
					{"ls-remote", "--heads", "origin", "refs/heads/main"},
					{"fetch", "origin", "+refs/heads/main:refs/remotes/origin/main"},
					{"status", "--porcelain"},
					{"rebase", "refs/remotes/origin/main"},
					{"push", "origin", "HEAD:refs/heads/main"},
				} {
					start := time.Now()
					_, e := r.git(op...)
					record(w, op[0], start, time.Since(start), e)
				}
			}
		}(w)
	}
	wg.Wait()
	ops := make([]string, 0, len(stats))
	for op := range stats {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	for _, op := range ops {
		t.Logf("E3 op %-9s n %5d max %v", op, stats[op].n, stats[op].max.Round(time.Millisecond))
	}
	t.Logf("E3 slow or failed ops: %d", len(slow))
	for _, s := range slow {
		t.Logf("E3 SLOW %s", s)
	}
}
