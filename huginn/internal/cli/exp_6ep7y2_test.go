package cli

// 6ep7y2 experiment E1, branch ci/6ep7y2-20261118 only: never lands on main.
// H1: on Windows a BELOW_NORMAL pass is starved while normal-priority work
// keeps every CPU busy, so it holds the run lock for minutes.

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/internal/throttle"
)

const (
	expOn    = "EXP_6EP7Y2"
	expSpin  = "EXP_6EP7Y2_SPIN"
	expChild = "EXP_6EP7Y2_CHILD"
	expFull  = "EXP_6EP7Y2_FULL"
)

// expProcs lists live git/sh/test processes for a stall (E2), bounded 20 s.
var expProcs = func() string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var c *osexec.Cmd
	if runtime.GOOS == "windows" {
		c = osexec.CommandContext(ctx, "powershell", "-NoProfile", "-Command",
			"Get-CimInstance Win32_Process | Where-Object { $_.Name -match 'git|sh|cli.test|ssh' } | ForEach-Object { '{0} ppid {1} start {2:o} {3}' -f $_.ProcessId, $_.ParentProcessId, $_.CreationDate, $_.CommandLine }")
	} else {
		c = osexec.CommandContext(ctx, "ps", "-eo", "pid,ppid,lstart,args")
	}
	out, e := c.CombinedOutput()
	return fmt.Sprintf("%s(err %v)", out, e)
}

// TestExp6ep7y2Spin is a spinner process: one busy thread until killed.
func TestExp6ep7y2Spin(t *testing.T) {
	if os.Getenv(expSpin) != "1" {
		t.Skip("spinner child only")
	}
	runtime.GOMAXPROCS(1)
	x := 0
	for {
		x++
	}
}

// TestExp6ep7y2GitChild lowers itself as the pass does (unless full speed),
// then times 20 git calls and a fixed CPU loop.
func TestExp6ep7y2GitChild(t *testing.T) {
	if os.Getenv(expChild) != "1" {
		t.Skip("git child only")
	}
	if os.Getenv(expFull) != "1" {
		throttle.Background()
	}
	start := time.Now()
	for i := 0; i < 20; i++ {
		if e := osexec.Command("git", "--version").Run(); e != nil {
			t.Fatal(e)
		}
	}
	gitD := time.Since(start)
	start = time.Now()
	x := 0
	for i := 0; i < 200_000_000; i++ {
		x += i & 3
	}
	cpuD := time.Since(start)
	fmt.Printf("EXPRESULT priority=%q git20=%v cpu=%v x=%d\n", throttle.Priority(), gitD.Round(time.Millisecond), cpuD.Round(time.Millisecond), x%7)
}

// spinners starts n normal-priority busy processes; the return kills them.
func spinners(t *testing.T, n int) func() {
	exe, _ := os.Executable()
	var cs []*osexec.Cmd
	for i := 0; i < n; i++ {
		c := osexec.Command(exe, "-test.run=^TestExp6ep7y2Spin$")
		c.Env = append(os.Environ(), expSpin+"=1")
		if e := c.Start(); e != nil {
			t.Fatal(e)
		}
		cs = append(cs, c)
	}
	time.Sleep(time.Second)
	return func() {
		for _, c := range cs {
			_ = c.Process.Kill()
			_ = c.Wait()
		}
	}
}

func expChildRun(t *testing.T, full bool) string {
	exe, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	c := osexec.CommandContext(ctx, exe, "-test.run=^TestExp6ep7y2GitChild$", "-test.v")
	c.Env = append(os.Environ(), expChild+"=1")
	if full {
		c.Env = append(c.Env, expFull+"=1")
	}
	start := time.Now()
	out, e := c.CombinedOutput()
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "EXPRESULT") {
			return fmt.Sprintf("%s wall=%v", strings.TrimSpace(l), time.Since(start).Round(time.Millisecond))
		}
	}
	return fmt.Sprintf("NO RESULT after %v (err %v)", time.Since(start).Round(time.Millisecond), e)
}

// E1a: a lowered child vs a normal child, idle and with NumCPU spinners.
func TestExp6ep7y2Priority(t *testing.T) {
	if os.Getenv(expOn) != "1" {
		t.Skip("experiment only")
	}
	n := runtime.NumCPU()
	t.Logf("E1a %s NumCPU %d", runtime.GOOS, n)
	t.Logf("E1a idle   lowered: %s", expChildRun(t, false))
	t.Logf("E1a idle   normal : %s", expChildRun(t, true))
	stop := spinners(t, n)
	defer stop()
	t.Logf("E1a loaded lowered: %s", expChildRun(t, false))
	t.Logf("E1a loaded normal : %s", expChildRun(t, true))
}

// E1b: one real background pass after a note, with NumCPU spinners, lowered
// (default) vs HUG_FULL_SPEED=1; how long the run lock is held.
func TestExp6ep7y2Pass(t *testing.T) {
	if os.Getenv(expOn) != "1" {
		t.Skip("experiment only")
	}
	for _, full := range []bool{false, true} {
		a, _, _ := clones(t)
		if full {
			a.Sync.Env = append(a.Sync.Env, "HUG_"+"FULL_SPEED=1")
		}
		id := execute(t, a, "new", "Exp", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
		settled(t, a)
		r, _ := gitbranch.Open(a.Cwd, TaskBranch, TaskDir)
		bg, _ := r.Background()
		stop := spinners(t, runtime.NumCPU())
		start := time.Now()
		execute(t, a, "note", id, "under load")
		noteD := time.Since(start)
		done := until(180*time.Second, func() bool { return !bg.Pending() && !bg.Busy() })
		d := time.Since(start)
		stop()
		t.Logf("E1b %s full=%v note=%v settled=%v (done %v, run held %v)\nphase log:\n%s", runtime.GOOS, full, noteD.Round(time.Millisecond), d.Round(time.Millisecond), done, bg.RunHeld(), syncLogOf(t, a))
		if !done {
			until(120*time.Second, func() bool { return !bg.Pending() && !bg.Busy() })
		}
	}
}
