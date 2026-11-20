package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
)

const timeoutReportFixture = "HUG_TEST_SYNC_TIMEOUT_REPORT"
const timeoutReportPhase = "retained-phase-fixture: waiting for run lock"

// This test fails only in the subprocess selected by the parent below.
// It calls settled's actual failure path directly, without sleeping 30 s.
func TestSyncTimeoutFailureHelper(t *testing.T) {
	receipt := os.Getenv(timeoutReportFixture)
	if receipt == "" {
		return
	}
	common := t.TempDir()
	bg := gitbranch.BackgroundAt(common, "hug/tasks")
	release, err := bg.Exclusive()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	bg.LogPhase(timeoutReportPhase)
	if err := os.WriteFile(receipt, []byte(common), 0o600); err != nil {
		t.Fatal(err)
	}
	syncTimeoutFailure(t, bg, "fixture")
}

// 6zhyq2: settled waits for a pass that keeps going, however long it
// takes under its cap, and fails one that stops. A pass holds the run
// lock three stall windows long: logging a phase every 100 ms it is
// waited for; silent, settled gives up after one stall window; logging
// past the cap, it gives up at the cap. (CI 44522707343 attempt 1 failed
// a pass that was still starting git, at a fixed 30 s.) On a synctest
// bubble's clock: the stall windows are settled's own 30 s, and no wall
// clock is read.
func TestSettleWaitsOnProgressNotAClock(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		logging bool
		cap     time.Duration
		want    string
		took    time.Duration
	}{
		{"progressing", true, settleCap, "", 3 * settleStall},
		{"silent", false, settleCap, "no progress for 30s (30s in all)", settleStall},
		{"capped", true, 2 * settleStall, "still going after 1m0s", 2 * settleStall},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				bg := gitbranch.BackgroundAt(t.TempDir(), "hug/tasks")
				release, err := bg.Exclusive()
				if err != nil {
					t.Fatal(err)
				}
				done := make(chan struct{})
				go func() {
					defer close(done)
					defer release()
					for end := time.Now().Add(3 * settleStall); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
						if c.logging {
							bg.LogPhase("still integrating")
						}
					}
				}()
				start := time.Now()
				got := settleWait(bg, settleStall, c.cap)
				took := time.Since(start)
				<-done
				if got != c.want {
					t.Fatalf("settleWait = %q after %v, want %q", got, took, c.want)
				}
				if took < c.took || took > c.took+time.Second {
					t.Fatalf("settleWait returned %q after %v, want about %v", got, took, c.took)
				}
			})
		})
	}
}

// 6ep7y2: ordinary go test (without -v) prints a real Fatalf's phase log
// before TempDir cleanup removes the file. Check the captured child output
// and prove the source directory is already gone when the parent reads it.
// This is a deterministic failure-output check, not a timing assertion.
func TestSyncTimeoutFailureRetainsPhases(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(t.TempDir(), "common-path")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestSyncTimeoutFailureHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(), timeoutReportFixture+"="+receipt)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || ctx.Err() != nil {
		t.Fatalf("expected a deliberate test failure, got %v (context %v):\n%s", err, ctx.Err(), out)
	}
	for _, want := range []string{
		"--- FAIL: TestSyncTimeoutFailureHelper",
		"background pass still running, fixture:",
		"queued false, busy true, pending false, turn held false, run held true",
		timeoutReportPhase,
		"pid ",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("failed test output lost %q:\n%s", want, out)
		}
	}
	common, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(common)); !os.IsNotExist(err) {
		t.Fatalf("child's temporary phase directory survived cleanup: %s (error %v)", common, err)
	}
}

const earlyReturnPhase = "retained-early-return-fixture: queue taken before pass"
const earlyReturnHead = "hug task new early-return-fixture"

// The same assertion helper called by TestSettledWaitsForSlowChild, forced
// without a six-second child delay or a probabilistic background failure.
func TestSyncEarlyReturnFailureHelper(t *testing.T) {
	receipt := os.Getenv(timeoutReportFixture)
	if receipt == "" {
		return
	}
	a, _, _ := clones(t)
	settled(t, a)
	r, err := gitbranch.Open(a.Cwd, TaskBranch, TaskDir)
	if err != nil {
		t.Fatal(err)
	}
	bg, err := r.Background()
	if err != nil {
		t.Fatal(err)
	}
	bg.LogPhase(earlyReturnPhase)
	if err := os.WriteFile(receipt, []byte(a.Cwd), 0o600); err != nil {
		t.Fatal(err)
	}
	syncEarlyReturnFailure(t, a, earlyReturnHead)
}

func TestSyncEarlyReturnFailureRetainsPhases(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(t.TempDir(), "early-return-path")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestSyncEarlyReturnFailureHelper$", "-test.count=1")
	cmd.Env = append(os.Environ(), timeoutReportFixture+"="+receipt)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || ctx.Err() != nil {
		t.Fatalf("expected an early-return assertion failure, got %v (context %v):\n%s", err, ctx.Err(), out)
	}
	for _, want := range []string{
		"--- FAIL: TestSyncEarlyReturnFailureHelper",
		"settled returned before the pass pushed; remote head:",
		earlyReturnHead,
		"busy false, pending false",
		earlyReturnPhase,
		"pid ",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("early-return failure output lost %q:\n%s", want, out)
		}
	}
	common, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(common)); !os.IsNotExist(err) {
		t.Fatalf("child's early-return phase directory survived cleanup: %s (error %v)", common, err)
	}
}
