package gitbranch

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/flock"
)

// 6ep7y2: Busy alone reported true for a queued child and hid whether a
// running pass prevented it proceeding. Count queue-only, run-only and idle
// observations with real locks, without waiting for an arbitrary sleep.
func TestRunHeldDistinguishesQueueFromRun(t *testing.T) {
	b := BackgroundAt(t.TempDir(), "hug/tasks")
	for i := 0; i < 20; i++ {
		queue, err := flock.Acquire(b.base + ".queue")
		if err != nil {
			t.Fatal(err)
		}
		queued, running := b.Queued(), b.RunHeld()
		queue()
		if !queued || running {
			t.Fatalf("queue-only round %d: queued %v run held %v", i, queued, running)
		}
		run, err := b.Exclusive()
		if err != nil {
			t.Fatal(err)
		}
		queued, running = b.Queued(), b.RunHeld()
		run()
		if queued || !running {
			t.Fatalf("run-only round %d: queued %v run held %v", i, queued, running)
		}
		if b.RunHeld() || b.Busy() {
			t.Fatalf("round %d: idle locks reported held", i)
		}
	}
}

// Each pass must log all its waits in order, before entering the callback,
// and completion before releasing the run lock. Assertions in the callback
// prove the progress lines are not buffered until a stuck pass finishes.
func TestBackgroundLogsEveryPassPhase(t *testing.T) {
	b := BackgroundAt(t.TempDir(), "hug/tasks")
	const rounds = 20
	phases := []string{
		"queue taken; settling",
		"settled; waiting for run lock",
		"run taken; queue released; starting pass",
		"pass finished; releasing run lock",
	}
	read := func() string {
		t.Helper()
		data, err := os.ReadFile(b.Log())
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	passes := 0
	for i := 0; i < rounds; i++ {
		err := b.Run(Settle{}, func(func() (func(), error)) State {
			passes++
			log := read()
			for _, phase := range phases[:3] {
				if got := strings.Count(log, phase); got != passes {
					t.Errorf("pass %d: %q logged %d times", passes, phase, got)
				}
			}
			if got := strings.Count(log, phases[3]); got != passes-1 {
				t.Errorf("pass %d: completion logged prematurely %d times", passes, got)
			}
			if !b.RunHeld() || b.Queued() {
				t.Error("callback does not hold only the run lock")
			}
			return State{}
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	lines := strings.Split(strings.TrimSpace(read()), "\n")
	if passes != rounds || len(lines) != rounds*len(phases) {
		t.Fatalf("passes %d, log lines %d, want %d and %d", passes, len(lines), rounds, rounds*len(phases))
	}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			t.Fatalf("malformed log line %q", line)
		}
		if _, err := time.Parse(time.RFC3339Nano, fields[0]); err != nil {
			t.Fatalf("timestamp in %q: %v", line, err)
		}
		if !strings.Contains(line, fmt.Sprintf("pid %d: ", os.Getpid())) || !strings.Contains(line, phases[i%len(phases)]) {
			t.Fatalf("line %d missing pid or ordered phase: %q", i, line)
		}
	}
}
