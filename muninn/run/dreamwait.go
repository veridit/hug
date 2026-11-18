package run

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/activity"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/store"
)

// The dream waiter (h00cs2). The capture hook starts it when a pass is due;
// it waits for the person to be away and the machine to be idle, then runs
// one pass with the hug installed at that moment.

// dreamStep is how often a waiter looks again.
var dreamStep = 5 * time.Minute

// dreamSleep waits; tests advance a fake clock instead.
var dreamSleep = time.Sleep

// dreamSelf is the executable this process runs.
var dreamSelf = func() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// dreamSpawn starts argv detached; a no-op in tests, which swap it.
var dreamSpawn = func(argv []string) error {
	if testing.Testing() {
		return nil
	}
	return spawnDetached(argv)
}

// dreamExec replaces this process with the installed hug (on Windows it
// starts it detached and this process exits).
var dreamExec = execInstalled

// waiterArgs is the waiter command for the hug at exe: the muninn
// compatibility link has no "mem" level.
func waiterArgs(exe string) []string {
	args := app.DreamWaiterArgs
	if strings.TrimSuffix(filepath.Base(exe), ".exe") == "muninn" {
		args = args[1:]
	}
	return append([]string{exe}, args...)
}

// nudgeDream starts a waiter when a pass is due and none is waiting: one
// meta read and one lock try, for the capture hook.
func nudgeDream() {
	db, err := store.Open(config.DBPath())
	if err != nil {
		return
	}
	due := app.DreamDue(db, app.DreamClock())
	db.Close()
	if !due {
		return
	}
	release, ok, err := app.TryDreamWait()
	if err != nil || !ok {
		return // a waiter holds it
	}
	release()
	if err := dreamSpawn(waiterArgs(app.HugExecutable())); err != nil {
		logHook("dream: could not start the waiter: %v", err)
	}
}

// fileStamp is a file's identity and version: the same file, unchanged.
type fileStamp struct{ info os.FileInfo }

func stampOf(p string) fileStamp {
	info, err := os.Stat(p)
	if err != nil {
		return fileStamp{}
	}
	return fileStamp{info}
}

func (a fileStamp) same(b fileStamp) bool {
	return a.info != nil && b.info != nil && os.SameFile(a.info, b.info) &&
		a.info.ModTime().Equal(b.info.ModTime()) && a.info.Size() == b.info.Size()
}

// waitAndDream is hug mem dream --when-quiet: wait for quiet and a low load,
// then run one pass, or give up after DreamWaitMax. Every outcome is one
// line in hook.log.
func waitAndDream(ctx context.Context) error {
	release, ok, err := app.TryDreamWait()
	if err != nil {
		return err
	}
	if !ok {
		return nil // another waiter
	}
	self := stampOf(dreamSelf())
	db, err := store.Open(config.DBPath())
	if err != nil {
		release()
		return err
	}
	start := app.DreamClock()
	// With a week of activity the pass waits for the day's quiet stretch,
	// which may begin more than DreamWaitMax away: wait until it has begun,
	// then DreamWaitMax more at most (vk9td0).
	deadline := start.Add(app.DreamWaitMax)
	if p := app.ReadDreamPlan(db, start); p.Ready && p.Start.After(start) {
		deadline = p.Start.Add(app.DreamWaitMax)
	}
	for {
		now := app.DreamClock()
		if !app.DreamDue(db, now) {
			db.Close()
			release()
			return nil
		}
		ok, why := app.DreamGate(db, now)
		if ok {
			break
		}
		if !now.Before(deadline) {
			waited := now.Sub(start)
			db.Close()
			release()
			logHook("dream: gave up after %s waiting: %s; the next turn_end starts another", waited.Round(time.Minute), why)
			return nil
		}
		dreamSleep(dreamStep)
	}
	waited := app.DreamClock().Sub(start)
	stretch := ""
	if p := app.ReadDreamPlan(db, app.DreamClock()); p.Ready && !p.Start.IsZero() && !app.DreamClock().Before(p.Start) {
		stretch = ", in your quiet stretch " + p.Name
	}
	db.Close()
	// A waiter that slept for hours may be an old hug: run the installed
	// one (crab 2026-11-18, the mbdfzs rule). It finds the gate open and
	// runs at once.
	if exe := app.HugExecutable(); !self.same(stampOf(exe)) {
		release()
		logHook("dream: hug changed while waiting (%s); handing the pass to %s", waited.Round(time.Minute), exe)
		return dreamExec(waiterArgs(exe))
	}
	defer release()
	run, err := app.DreamRun()
	if err != nil {
		return err
	}
	defer run()
	began := time.Now()
	var r app.DreamResult
	var last app.DreamLast
	err = withApp(func(ctx context.Context, a *app.App) error {
		if !app.DreamDue(a.DB, app.DreamClock()) {
			return nil // a pass by hand or by serve ran meanwhile
		}
		var err error
		if r, err = a.Dream(ctx, false); err != nil {
			app.RecordDreamFailed(a.DB, app.DreamClock(), err) // not again at every turn_end
			return err
		}
		last, err = app.RecordDream(a.DB, app.DreamClock(), "while quiet", r)
		return err
	})
	if err != nil {
		logHook("dream: %v", err)
		return err
	}
	if last.At.IsZero() {
		return nil
	}
	logHook("%s (%s, waited %s for quiet%s)", last.Line(), time.Since(began).Round(100*time.Millisecond), waited.Round(time.Minute), stretch)
	return nil
}

// dreamStatus is what --status and the dry-run header print.
func dreamStatus(a *app.App) []string {
	l := app.ReadDreamLast(a.DB)
	p := app.ReadDreamPlan(a.DB, app.DreamClock())
	gate := fmt.Sprintf("once you have been away %s and the load is below %.0f", app.QuietFor, app.DreamLoadBelow)
	if p.Ready && !p.Start.IsZero() {
		gate = fmt.Sprintf("at the start of your quiet stretch, load below %.0f", app.DreamLoadBelow)
	}
	var out []string
	if l.At.IsZero() {
		out = []string{"Last pass: never (dream runs by itself " + gate + ", after your next turn ends)"}
	} else {
		out = []string{"Last pass: " + strings.TrimPrefix(l.Line(), "dream ")}
		out = append(out, "Next pass: after "+l.At.Add(app.DreamEvery).Local().Format("2006-01-02 15:04")+", "+gate)
	}
	return append(out, planLine(p))
}

// planLine names the quiet stretch the pass waits for, or how much of a
// week is still to learn.
func planLine(p app.DreamPlan) string {
	switch {
	case !p.Ready:
		return "Your week: " + p.Learning + "; until then dream waits for 30 minutes away (hug mem dream --week)"
	case p.Start.IsZero():
		return "Your week: no quiet stretch found; dream waits for 30 minutes away (hug mem dream --week)"
	}
	loc := activity.Location()
	return fmt.Sprintf("Next quiet stretch: %s to %s; longest: %s (hug mem dream --week)",
		p.Start.In(loc).Format("Mon 15:04"), p.End.In(loc).Format("Mon 15:04"), p.Longest)
}

// dreamWeek is hug mem dream --week: the hour-of-week histogram, the
// longest quiet stretch, and each day's.
func dreamWeek(a *app.App) []string {
	p := app.ReadDreamPlan(a.DB, app.DreamClock())
	out := []string{strings.TrimRight(p.Hist.Grid(), "\n")}
	if !p.Ready {
		return append(out, p.Learning+": until then dream waits for 30 minutes away")
	}
	if p.Longest == "" {
		return append(out, "No quiet stretch: dream waits for 30 minutes away")
	}
	out = append(out, "Longest quiet stretch: "+p.Longest)
	var each []string
	for d := 0; d < 7; d++ {
		if s, ok := p.Hist.ForDay(d); ok {
			each = append(each, s.String())
		}
	}
	out = append(out, "Dream starts each day at: "+strings.Join(each, ", "))
	return append(out, planLine(p))
}
