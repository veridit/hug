package gitbranch

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// rt80x4: exercise settled's real Busy-then-Pending observations across a
// real child's pending-to-queue handoff. The existing Settle.Probe callback
// gates the child while it owns queue, after removing pending, before run.
// Aging the marker replaces a slow startup without sleeping or a test seam.
func TestBusyPendingHandoffNeverLooksIdle(t *testing.T) {
	b := BackgroundAt(t.TempDir(), "hug/tasks")
	if err := os.MkdirAll(filepath.Dir(b.base), 0o700); err != nil {
		t.Fatal(err)
	}
	pending := b.base + ".pending"
	if err := os.WriteFile(pending, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-pendingFor - time.Second)
	if err := os.Chtimes(pending, old, old); err != nil {
		t.Fatal(err)
	}
	// Expiration must continue to allow Nudge to retry a dead startup.
	if b.Queued() || !b.Pending() {
		t.Fatal("fixture must have an expired pending marker and no queue holder")
	}
	busyBeforeHandoff := b.Busy()
	queued, proceed := make(chan struct{}), make(chan struct{})
	var announce, release sync.Once
	allow := func() { release.Do(func() { close(proceed) }) }
	defer allow()
	var passes atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- b.Run(Settle{
			Quiet: time.Nanosecond, Max: time.Nanosecond,
			Probe: func() string {
				announce.Do(func() { close(queued) })
				<-proceed
				return "unchanged"
			},
		}, func(func() (func(), error)) State {
			passes.Add(1)
			return State{}
		})
	}()
	<-queued
	// This is the second operand of the actual settled predicate. The
	// child cannot advance until all observations below are complete.
	pendingAfterHandoff := b.Pending()
	queueHeld, runHeld, passesBefore := b.Queued(), b.RunHeld(), passes.Load()
	falseIdle := !busyBeforeHandoff && !pendingAfterHandoff
	allow()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	t.Logf("handoff: busy before=%t pending after=%t queue held=%t run held=%t passes before=%d after=%d", busyBeforeHandoff, pendingAfterHandoff, queueHeld, runHeld, passesBefore, passes.Load())
	if !queueHeld || runHeld || pendingAfterHandoff || passesBefore != 0 || passes.Load() != 1 {
		t.Fatal("did not exercise exactly one pending-to-queue handoff before the pass")
	}
	if falseIdle {
		t.Fatal("Busy then Pending reported idle while a queued child had not run its pass")
	}
	if b.Busy() || b.Pending() {
		t.Fatal("completed child still reports busy or pending")
	}
}

// Busy deliberately retains an orphan marker, but Nudge must still attempt
// another startup after Queued's expiry. A missing executable proves Start
// was actually attempted, without a child process, polling or environment seam.
func TestExpiredPendingAllowsNudgeRetry(t *testing.T) {
	common := t.TempDir()
	b := BackgroundAt(common, "hug/tasks")
	if err := os.MkdirAll(filepath.Dir(b.base), 0o700); err != nil {
		t.Fatal(err)
	}
	pending := b.base + ".pending"
	if err := os.WriteFile(pending, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-pendingFor - time.Second)
	if err := os.Chtimes(pending, old, old); err != nil {
		t.Fatal(err)
	}
	if b.Queued() || !b.Busy() || !b.Pending() {
		t.Fatal("expired orphan must be busy for a waiter, but not queued for Nudge")
	}
	missing := filepath.Join(common, "missing-sync-child")
	err := b.Nudge(common, []string{missing})
	if !missingExecutableError(err) {
		t.Fatalf("expired marker suppressed retry: Nudge error %v, want attempted missing executable", err)
	}
	if b.Pending() || b.Busy() {
		t.Fatal("failed child startup left its pending marker behind")
	}
	t.Log("expired orphan: Busy true, Queued false; Nudge attempted missing executable and cleared pending on failure")
}

func missingExecutableError(err error) bool {
	return err != nil && (os.IsNotExist(err) || errors.Is(err, exec.ErrNotFound))
}

func TestMissingExecutableErrorClasses(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"path not exist", &os.PathError{Op: "fork/exec", Path: "missing-sync-child", Err: os.ErrNotExist}, true},
		{"wrapped Windows lookup", fmt.Errorf("start: %w", &exec.Error{Name: "missing-sync-child", Err: exec.ErrNotFound}), true},
		{"nil means no attempt", nil, false},
		{"unrelated error", errors.New("unrelated"), false},
		{"path permission error", &os.PathError{Op: "fork/exec", Path: "missing-sync-child", Err: os.ErrPermission}, false},
		{"lookup permission error", &exec.Error{Name: "missing-sync-child", Err: os.ErrPermission}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := missingExecutableError(tc.err); got != tc.want {
				t.Fatalf("missing executable classification = %t, want %t, error %v", got, tc.want, tc.err)
			}
		})
	}
}
