package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
)

// dreamWhenQuiet runs one dream pass from hug mem serve's ticker when one is
// due and the gate is open (away QuietFor, load below DreamLoadBelow): the
// same rule and the same run lock as the waiter the capture hook starts
// (h00cs2).
func dreamWhenQuiet(ctx context.Context, a *app.App, mu *sync.Mutex, logf func(string, ...any)) {
	now := app.DreamClock()
	if !app.DreamDue(a.DB, now) {
		return
	}
	if ok, _ := app.DreamGate(a.DB, now); !ok {
		return
	}
	release, err := app.DreamRun()
	if err != nil {
		logf("dream: %v", err)
		return
	}
	defer release()
	if !app.DreamDue(a.DB, app.DreamClock()) {
		return // a waiter ran it meanwhile
	}
	mu.Lock()
	defer mu.Unlock()
	began := time.Now()
	r, err := a.Dream(ctx, false)
	if err != nil {
		app.RecordDreamFailed(a.DB, app.DreamClock(), err) // not again every 5 minutes
		logf("dream: %v", err)
		return
	}
	last, err := app.RecordDream(a.DB, app.DreamClock(), "by hug mem serve", r)
	if err != nil {
		logf("dream: %v", err)
		return
	}
	logf("%s (%s)", last.Line(), time.Since(began).Round(100*time.Millisecond))
}
