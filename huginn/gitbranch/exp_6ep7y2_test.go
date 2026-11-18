package gitbranch

// 6ep7y2 experiment E5, branch ci/6ep7y2-h5 only: never lands on main.
// H5: on Windows a push from a background syncer (started by detach: hidden
// console, own process group) sporadically never returns because the
// remote's post-receive hook (#!/bin/sh + cat: MSYS processes) hangs.
// E3 pushed 7500 times with the hook from the test process itself (the
// runner's console) and saw no hang: E5 pushes from detached children, as
// the pass does, half to a remote with the fixture's hook, half without.

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	expWorker = "EXP_6EP7Y2_WORKER"
	expUntil  = "EXP_6EP7Y2_UNTIL"
)

// expProcs lists live git/sh/cat processes, bounded 20 s.
func expProcs() string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command",
			"Get-CimInstance Win32_Process | Where-Object { $_.Name -match '^(git.*|sh|bash|cat|gb\\.test)\\.exe$' } | ForEach-Object { '{0} ppid {1} start {2:o} {3}' -f $_.ProcessId, $_.ParentProcessId, $_.CreationDate, $_.CommandLine }")
	} else {
		c = exec.CommandContext(ctx, "ps", "-eo", "pid,ppid,etime,args")
	}
	out, e := c.CombinedOutput()
	return fmt.Sprintf("%s(err %v)", out, e)
}

// TestExp6ep7y2HookWorker is one detached child: commit and push in a loop
// until the deadline, one result line per push.
func TestExp6ep7y2HookWorker(t *testing.T) {
	base := os.Getenv(expWorker)
	if base == "" {
		t.Skip("worker child only")
	}
	until, _ := strconv.ParseInt(os.Getenv(expUntil), 10, 64)
	NetTimeout = 150 * time.Second // see a hang's whole length
	dir := filepath.Join(base, "one")
	r := &Repository{Dir: dir, Root: dir, Remote: "origin", Branch: "main"}
	res, e := os.OpenFile(filepath.Join(base, "result.txt"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Close()
	for i := 0; time.Now().Unix() < until; i++ {
		if _, e := r.git("commit", "-q", "--allow-empty", "-m", "n"+strconv.Itoa(i)); e != nil {
			fmt.Fprintf(res, "commitfail %q\n", firstLine(e))
			continue
		}
		start := time.Now()
		snap := make(chan string, 1)
		watch := time.AfterFunc(20*time.Second, func() { snap <- expProcs() })
		_, e := r.git("push", "origin", "HEAD:refs/heads/main")
		d := time.Since(start)
		fmt.Fprintf(res, "push %d %t\n", d.Milliseconds(), e != nil)
		if !watch.Stop() {
			fmt.Fprintf(res, "SLOW start %s took %v err %v\nprocesses at 20 s:\n%s\nENDSLOW\n", start.UTC().Format(time.RFC3339Nano), d.Round(time.Millisecond), e, <-snap)
		} else if e != nil {
			fmt.Fprintf(res, "FAILED took %v err %q\n", d.Round(time.Millisecond), firstLine(e))
		}
	}
}

func TestExp6ep7y2Hook(t *testing.T) {
	if os.Getenv("EXP_6EP7Y2") != "1" {
		t.Skip("experiment only")
	}
	dur, _ := time.ParseDuration(os.Getenv("EXP_6EP7Y2_FOR"))
	if dur == 0 {
		dur = 25 * time.Minute
	}
	sh := func(dir string, args ...string) {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if b, e := c.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, b, e)
		}
	}
	until := time.Now().Add(dur).Unix()
	type worker struct {
		arm  string
		base string
		cmd  *exec.Cmd
	}
	var ws []worker
	n, _ := strconv.Atoi(os.Getenv("EXP_6EP7Y2_WORKERS"))
	if n == 0 {
		n = 24
	}
	for w := 0; w < n; w++ {
		arm := "hook" // two of three: the arm under test
		if w%3 == 2 {
			arm = "nohook"
		}
		base := t.TempDir()
		bare := filepath.Join(base, "remote.git")
		sh(base, "init", "-q", "--bare", "-b", "main", bare)
		// Thousands of pushes per remote here (the fixture's remotes get
		// tens) would start auto maintenance, which Windows git runs in the
		// foreground (daemonize is ENOSYS there): a slow push in both arms.
		sh(bare, "config", "gc.auto", "0")
		if arm == "hook" {
			// The fixture's hook, byte for byte (huginn/internal/cli clones).
			hook := "#!/bin/sh\ncat >> \"$GIT_DIR/pushes\"\n"
			if e := os.WriteFile(filepath.Join(bare, "hooks", "post-receive"), []byte(hook), 0o755); e != nil {
				t.Fatal(e)
			}
		}
		dir := filepath.Join(base, "one")
		sh(base, "init", "-q", "-b", "main", dir)
		sh(dir, "config", "user.name", "T")
		sh(dir, "config", "user.email", "t@example.invalid")
		sh(dir, "config", "core.autocrlf", "false")
		sh(dir, "config", "gc.auto", "0")
		sh(dir, "commit", "-q", "--allow-empty", "-m", "init")
		sh(dir, "remote", "add", "origin", bare)
		sh(dir, "push", "-q", "origin", "main")
		out, e := os.Create(filepath.Join(base, "out.txt"))
		if e != nil {
			t.Fatal(e)
		}
		c := exec.Command(os.Args[0], "-test.run=^TestExp6ep7y2HookWorker$", "-test.v", "-test.timeout=0")
		c.Env = append(os.Environ(), expWorker+"="+base, expUntil+"="+strconv.FormatInt(until, 10), "EXP_6EP7Y2=")
		c.Stdout, c.Stderr = out, out
		detach(c) // the pass's own start: hidden console, own process group
		if e := c.Start(); e != nil {
			t.Fatal(e)
		}
		out.Close()
		ws = append(ws, worker{arm, base, c})
	}
	type arm struct {
		n, slow, failed int
		max             time.Duration
	}
	arms := map[string]*arm{"hook": {}, "nohook": {}}
	var notes []string
	for i, w := range ws {
		e := w.cmd.Wait()
		a := arms[w.arm]
		f, err := os.Open(filepath.Join(w.base, "result.txt"))
		if err != nil {
			notes = append(notes, fmt.Sprintf("worker %d %s: no results (%v), exit %v", i, w.arm, err, e))
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		var slow []string
		inSlow := false
		for sc.Scan() {
			l := sc.Text()
			switch {
			case inSlow:
				if l == "ENDSLOW" {
					inSlow = false
					notes = append(notes, fmt.Sprintf("E5 SLOW worker %d %s %s", i, w.arm, strings.Join(slow, "\n")))
					slow = nil
				} else {
					slow = append(slow, l)
				}
			case strings.HasPrefix(l, "push "):
				var ms int64
				var failed bool
				fmt.Sscanf(l, "push %d %t", &ms, &failed)
				d := time.Duration(ms) * time.Millisecond
				a.n++
				a.max = max(a.max, d)
				if failed {
					a.failed++
				}
				if d > 20*time.Second {
					a.slow++
				}
			case strings.HasPrefix(l, "SLOW "):
				inSlow, slow = true, []string{l}
			case strings.HasPrefix(l, "FAILED ") || strings.HasPrefix(l, "commitfail "):
				notes = append(notes, fmt.Sprintf("E5 worker %d %s %s", i, w.arm, l))
			}
		}
		f.Close()
		if e != nil {
			b, _ := os.ReadFile(filepath.Join(w.base, "out.txt"))
			notes = append(notes, fmt.Sprintf("worker %d %s exit %v:\n%s", i, w.arm, e, b))
		}
	}
	for _, n := range []string{"hook", "nohook"} {
		a := arms[n]
		t.Logf("E5 arm %-6s pushes %6d over20s %d failed %d max %v", n, a.n, a.slow, a.failed, a.max.Round(time.Millisecond))
	}
	t.Logf("E5 notes: %d", len(notes))
	for _, s := range notes {
		t.Log(s)
	}
}
