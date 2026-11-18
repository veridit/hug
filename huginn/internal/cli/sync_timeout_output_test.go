package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	syncTimeoutFailure(t, bg)
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
		"background pass still running after 30 s:",
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
