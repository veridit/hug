package task

import (
	"strings"
	"testing"
	"time"
)

// z25pb2 check 1: hug task new dates created and updated by the board
// clock (Now), as done_at, reflections and strikes are: one clock for
// every date hug writes. Pinned a tenth of a second before midnight on a
// day that is not today, so the wall clock cannot pass by chance.
func TestNewDatesByTheBoardClock(t *testing.T) {
	old := Now
	Now = func() time.Time { return time.Date(2026, 1, 31, 23, 59, 59, 900_000_000, time.Local) }
	t.Cleanup(func() { Now = old })
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "Dated", Goal: "G", Acceptance: []string{"c"}})
	if e != nil {
		t.Fatal(e)
	}
	if r.Task.Created != "2026-01-31" || r.Task.Updated != "2026-01-31" {
		t.Fatalf("created %q updated %q, want 2026-01-31 (the board clock)", r.Task.Created, r.Task.Updated)
	}
	d, _ := s.Get(r.Task.ID)
	if b := string(d.Bytes()); !strings.Contains(b, "\ncreated: 2026-01-31\nupdated: 2026-01-31\n") {
		t.Fatalf("file:\n%s", b)
	}
}
