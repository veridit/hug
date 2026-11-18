package task

import (
	"errors"
	"strings"
	"testing"
)

// byj479: the store call the TUI uses per marked task (crab 20:43,
// macaque's rxr4z8: t on N marked tasks takes the first and the rest with
// it). TakeWith(ref, with, user): both doing, ref marked doing_with; ref
// does not count as active, so a third take is still refused and the
// active one stays the one to take with.
func TestStoreTakeWith(t *testing.T) {
	s, _ := newStore(t)
	a := stagedTask(t, s, "aaaaaa", `state: "todo"`)
	b := stagedTask(t, s, "bbbbbb", `state: "todo"`)
	c := stagedTask(t, s, "cccccc", `state: "todo"`)
	if _, e := s.Take(a, "macaque"); e != nil {
		t.Fatal(e)
	}
	r, e := s.TakeWith(b, a, "macaque")
	if e != nil || r.Task.State != "doing" || TakenWith(r.Task) != a {
		t.Fatalf("TakeWith: %v %+v", e, r.Task.Fields)
	}
	ts, _ := s.Tasks()
	if act, ok := Active("macaque", "", ts); !ok || act.ID != a {
		t.Fatalf("active is %v %v, want %s", act.ID, ok, a)
	}
	// With the one taken with is refused: it is not the active one.
	_, e = s.TakeWith(c, b, "macaque")
	var te *Error
	if !errors.As(e, &te) || te.Fix != "hug task take "+c+" --with "+a+" --model MODEL --as macaque" {
		t.Fatalf("with a task taken with: %v", e)
	}
	if _, e = s.TakeWith(c, a, "macaque"); e != nil {
		t.Fatal(e)
	}
}

// Once the task it was taken with is parked or done, the one taken with
// it is the work in hand again: a third take is refused naming it.
func TestTakenWithIsActiveOnceAloneAgain(t *testing.T) {
	s, _ := newStore(t)
	s.AcceptanceMode = "warn"
	a := stagedTask(t, s, "aaaaaa", `state: "todo"`)
	b := stagedTask(t, s, "bbbbbb", `state: "todo"`)
	c := stagedTask(t, s, "cccccc", `state: "todo"`)
	if _, e := s.Take(a, "macaque"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.TakeWith(b, a, "macaque"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Wait(a, "CI run 1", "macaque"); e != nil {
		t.Fatal(e)
	}
	ts, _ := s.Tasks()
	if act, ok := Active("macaque", "", ts); !ok || act.ID != b {
		t.Fatalf("with a parked, active is %q %v, want %s", act.ID, ok, b)
	}
	_, e := s.Take(c, "macaque")
	var te *Error
	if !errors.As(e, &te) || te.Kind != "busy" || !strings.Contains(te.Value, b) {
		t.Fatalf("third take with a parked: %v", e)
	}
}
