package gitbranch

import "testing"

// mmm8c8: a probe (Queued, which settled and every Nudge call) took the
// queue lock for a moment to see whether a syncer held it. A syncer whose
// Run tried the queue in that moment took the probe for a queued syncer and
// exited without a pass, and the commit it was started for stayed unpushed
// (macOS CI 44227534600: "settled returned before the pass pushed").
// Counted, not timed: every Run while a probe loops must run its pass.
func TestProbeNeverCostsASyncerItsTurn(t *testing.T) {
	b := BackgroundAt(t.TempDir(), "hug/tasks")
	const runs = 300
	lost := 0
	for i := 0; i < runs; i++ {
		stop, done := make(chan struct{}), make(chan struct{})
		go func() {
			defer close(done)
			for {
				select {
				case <-stop:
					return
				default:
					b.Queued()
				}
			}
		}()
		ran := false
		e := b.Run(Settle{}, func(func() (func(), error)) State { ran = true; return State{} })
		close(stop)
		<-done
		if e != nil {
			t.Fatal(e)
		}
		if !ran {
			lost++
		}
	}
	if lost > 0 {
		t.Fatalf("%d of %d syncers gave up their turn to a probe and ran no pass", lost, runs)
	}
}
