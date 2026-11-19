package tui

// An amplifier for flakes under load (mq5ger, adqdfe; by kikazaru): it runs
// a test's body while N goroutines spin, so a wait that a busy machine
// stretches shows in hundreds of runs instead of one CI run in a thousand.
// It is skipped unless STARVE is set. Run it niced and bounded:
//
//	STARVE=8 GOMAXPROCS=4 nice -n 19 go test -p=1 -count=1000 \
//	  -run 'TestStarvedChangeBeforeWatch$' ./huginn/internal/tui/
//
// STARVE_STACKS=FILE writes every goroutine's stack on a failure: where the
// test waited. Add a TestStarved... for another test the same way.

import (
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
)

// starve spins STARVE goroutines until the test ends; it skips the test
// when STARVE is not set. On a failure, STARVE_STACKS gets the stacks.
func starve(t *testing.T) {
	t.Helper()
	n, _ := strconv.Atoi(os.Getenv("STARVE"))
	if n <= 0 {
		t.Skip("an amplifier: STARVE=N spins N goroutines while the test runs")
	}
	var stop atomic.Bool
	for i := 0; i < n; i++ {
		go func() {
			for !stop.Load() {
				runtime.Gosched()
			}
		}()
	}
	t.Cleanup(func() {
		stop.Store(true)
		if p := os.Getenv("STARVE_STACKS"); p != "" && t.Failed() {
			buf := make([]byte, 1<<22)
			_ = os.WriteFile(p, buf[:runtime.Stack(buf, true)], 0o644)
		}
	})
}

// TestStarvedChangeBeforeWatch is TestChangeBeforeWatchIsUpIsSeen under
// starvation: 4 of 1000 failed while a reload waited on git (adqdfe), and
// a watch closing its files twice (mq5ger) failed it before that.
func TestStarvedChangeBeforeWatch(t *testing.T) {
	starve(t)
	TestChangeBeforeWatchIsUpIsSeen(t)
}
