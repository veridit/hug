package cli

import (
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// lastTick is a tenth of a second before midnight on a day that is not
// today (jsyksd): a test that dates a note with the wall clock instead of
// the board clock fails on it every time, not only when CI crosses midnight.
var lastTick = time.Date(2026, 1, 31, 23, 59, 59, 900_000_000, time.Local)

// pinClock sets the board clock (task.Now) to at for the test and returns
// at's date. A test that calls it must not be parallel: task.Now is shared.
func pinClock(t *testing.T, at time.Time) string {
	t.Helper()
	old := task.Now
	task.Now = func() time.Time { return at }
	t.Cleanup(func() { task.Now = old })
	return at.Format("2006-01-02")
}
