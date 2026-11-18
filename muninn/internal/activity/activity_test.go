package activity_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/activity"
	"github.com/veridit/hug/muninn/internal/store"
)

func testDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// utc pins the hour-of-week to UTC, so the synthetic week means the same on
// every runner.
func utc(t *testing.T) {
	t.Helper()
	old := activity.Location
	activity.Location = func() *time.Location { return time.UTC }
	t.Cleanup(func() { activity.Location = old })
}

// monday is Monday 2026-11-02 00:00 UTC, the first day of the synthetic week.
var monday = time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)

// SyntheticWeek records one turn at hh:10 for every busy hour from `from`
// for `days` days. Busy: 07:00 to 21:59 every day, except that Saturday
// stops at 13:59 and Sunday starts at 09:00. So the nights are quiet from
// 22:00 for 9 h, and the longest quiet stretch is Saturday 14:00 to Sunday
// 09:00 (19 h).
func SyntheticWeek(t *testing.T, db *store.DB, from time.Time, days int) {
	t.Helper()
	for d := 0; d < days; d++ {
		day := from.AddDate(0, 0, d)
		for h := 0; h < 24; h++ {
			at := day.Add(time.Duration(h)*time.Hour + 10*time.Minute)
			if !busy(at) {
				continue
			}
			if err := activity.Record(db, at); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func busy(at time.Time) bool {
	h := at.Hour()
	switch at.Weekday() {
	case time.Saturday:
		return h >= 7 && h < 14
	case time.Sunday:
		return h >= 9 && h < 22
	}
	return h >= 7 && h < 22
}

// vk9td0 check 2: with 7+ days of activity, hug builds the 168-bucket
// hour-of-week histogram and names the longest quiet stretch.
func TestHourOfWeekHistogramNamesTheLongestQuietStretch(t *testing.T) {
	utc(t)
	db := testDB(t)
	SyntheticWeek(t, db, monday, 7)
	now := monday.AddDate(0, 0, 8).Add(15 * time.Hour) // Tue 2026-11-10 15:00
	h, err := activity.Read(db, now)
	if err != nil {
		t.Fatal(err)
	}
	if !h.Ready() {
		t.Fatalf("a full week of activity is not ready: %s", h.Learning())
	}
	if got := h.Turns[activity.HourOfWeek(monday.Add(9*time.Hour))]; got != 1 {
		t.Fatalf("Mon 09:00 bucket = %d turns, want 1", got)
	}
	if got := h.Turns[activity.HourOfWeek(monday.Add(3*time.Hour))]; got != 0 {
		t.Fatalf("Mon 03:00 bucket = %d turns, want 0", got)
	}
	l, ok := h.Longest()
	if !ok || l.String() != "Sat 14:00 for 19 h" {
		t.Fatalf("longest quiet stretch = %q %v, want Sat 14:00 for 19 h", l, ok)
	}
	tue, ok := h.ForDay(1)
	if !ok || tue.String() != "Tue 22:00 for 9 h" {
		t.Fatalf("Tuesday's stretch = %q %v, want Tue 22:00 for 9 h", tue, ok)
	}
	start, end, ok := h.Window(now)
	if !ok || !start.Equal(monday.AddDate(0, 0, 8).Add(22*time.Hour)) || end.Sub(start) != 9*time.Hour {
		t.Fatalf("next window from Tue 15:00 = %s..%s %v, want Tue 22:00 for 9 h", start, end, ok)
	}
	// Inside last night's stretch, the window is the one running now.
	start, _, ok = h.Window(monday.AddDate(0, 0, 8).Add(3 * time.Hour))
	if !ok || !start.Equal(monday.AddDate(0, 0, 7).Add(22*time.Hour)) {
		t.Fatalf("window at Tue 03:00 starts %s, want Mon 22:00", start)
	}
	if g := h.Grid(); !strings.Contains(g, "Sat") || strings.Count(g, "\n") < 8 {
		t.Fatalf("grid:\n%s", g)
	}
}

// Under a week the histogram is not trusted: no stretch, today's behaviour.
func TestUnderAWeekIsStillLearning(t *testing.T) {
	utc(t)
	db := testDB(t)
	SyntheticWeek(t, db, monday, 6)
	h, err := activity.Read(db, monday.AddDate(0, 0, 6).Add(8*time.Hour)) // Sun 08:00: 6 d 8 h of data
	if err != nil {
		t.Fatal(err)
	}
	if h.Ready() {
		t.Fatal("6 days of activity counted as a week")
	}
	if _, _, ok := h.Window(monday.AddDate(0, 0, 6)); ok {
		t.Fatal("a window before a week of data")
	}
	if got := h.Learning(); !strings.Contains(got, "6 of 7 days") {
		t.Fatalf("learning line: %q", got)
	}
}

// One stray late turn in four weeks does not break the stretch; in the only
// week there is, it does.
func TestAStrayTurnInFourWeeks(t *testing.T) {
	utc(t)
	db := testDB(t)
	SyntheticWeek(t, db, monday, 28)
	stray := monday.AddDate(0, 0, 9).Add(3 * time.Hour) // Wed 03:00, week 2
	if err := activity.Record(db, stray); err != nil {
		t.Fatal(err)
	}
	h, _ := activity.Read(db, monday.AddDate(0, 0, 29).Add(12*time.Hour))
	if wed, ok := h.ForDay(2); !ok || wed.String() != "Wed 22:00 for 9 h" {
		t.Fatalf("Wednesday after 4 weeks with one stray Wed 03:00 turn: %q", wed)
	}
	tue, _ := h.ForDay(1)
	if tue.String() != "Tue 22:00 for 9 h" {
		t.Fatalf("Tuesday night with one stray turn in 4 weeks = %q, want Tue 22:00 for 9 h", tue)
	}

	db = testDB(t)
	SyntheticWeek(t, db, monday, 7)
	activity.Record(db, stray.AddDate(0, 0, -7)) // Wed 03:00 of the only week
	h, _ = activity.Read(db, monday.AddDate(0, 0, 8))
	if tue, _ := h.ForDay(1); tue.String() != "Tue 22:00 for 5 h" {
		t.Fatalf("Tuesday night with a stray turn in the only week = %q, want Tue 22:00 for 5 h", tue)
	}
}

// hook.log already holds two weeks of jcode turns on a machine upgraded
// today: the first read seeds from it, once, skipping notices and agent DMs.
func TestSeedFromHookLog(t *testing.T) {
	utc(t)
	db := testDB(t)
	var b strings.Builder
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			at := monday.AddDate(0, 0, d).Add(time.Duration(h)*time.Hour + 5*time.Minute)
			if busy(at) {
				b.WriteString(at.Format(time.RFC3339) + ` jcode[global] session_crab_17937832816: msg="next step\n\nplease" hits=2 elapsed=5ms parked turn=m1 err=<nil>` + "\n")
			} else {
				b.WriteString(at.Format(time.RFC3339) + ` jcode session_crab_17937832816: msg="**Background task** done" hits=0 skipped="harness notice" elapsed=1ms` + "\n")
				b.WriteString(at.Format(time.RFC3339) + ` jcode session_fox_178697367042: msg="DM from crab: go on" hits=0 skipped="" elapsed=1ms` + "\n")
				b.WriteString(at.Format(time.RFC3339) + ` capture skipped=skip_dirs` + "\n")
			}
		}
	}
	log := filepath.Join(t.TempDir(), "hook.log")
	os.WriteFile(log, []byte(b.String()), 0o600)
	n, err := activity.SeedOnce(db, log)
	if err != nil || n != 95 {
		t.Fatalf("seeded %d turns (%v), want 95: 15 h x 5 weekdays + 7 + 13, no notices, no DMs", n, err)
	}
	if n, _ := activity.SeedOnce(db, log); n != 0 {
		t.Fatalf("second seed added %d", n)
	}
	h, _ := activity.Read(db, monday.AddDate(0, 0, 8))
	if l, ok := h.Longest(); !ok || l.String() != "Sat 14:00 for 19 h" {
		t.Fatalf("seeded longest = %q", l)
	}
}
