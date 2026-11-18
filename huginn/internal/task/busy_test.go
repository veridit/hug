package task

import (
	"errors"
	"strings"
	"testing"
)

// wantBusy checks the refusal naming active. taken is the task a take was
// refused for (4tycwd: the detour to it is a fourth fix); "" for a resume,
// which has no detour.
func wantBusy(t *testing.T, e error, active, taken string, as ...string) *Error {
	t.Helper()
	var te *Error
	if !errors.As(e, &te) || te.Kind != "busy" {
		t.Fatalf("want busy refusal naming %s, got %v", active, e)
	}
	want := []string{
		"hug task done " + active,
		"hug task wait " + active + ` "on what"`,
		"hug task set " + active + " --state todo && hug task note " + active + ` "why it goes back"`,
	}
	if taken != "" {
		want = append(want, "hug task dep "+active+" --add "+taken+" && hug task note "+active+` "where I was"`)
		// byj479: and take it with the active one, the fifth, as the
		// taker wrote it (default kestrel on claude-opus-5-5).
		who := " --model claude-opus-5-5 --as kestrel"
		if len(as) > 0 {
			who = as[0]
		}
		want = append(want, "hug task take "+taken+" --with "+active+who)
	}
	if strings.Join(te.Fixes, "|") != strings.Join(want, "|") || te.Fix != want[0] {
		t.Fatalf("fixes %q (fix %q), want %q", te.Fixes, te.Fix, want)
	}
	if !strings.Contains(te.Value, active) || strings.Count(te.Error(), "\nfix: ") != len(want) {
		t.Fatalf("refusal must name %s and print %d fix: lines:\n%s", active, len(want), te.Error())
	}
	return te
}

// n2hq8p check 1: the repro from 2026-11-17 13:10 (a second take went
// through while the first ticket was active) is refused with the three fix
// lines, and the refused task stays untouched in todo.
func TestSecondTakeWhileActiveIsRefused(t *testing.T) {
	s, _ := newStore(t)
	a := stagedTask(t, s, "aaaaaa", `state: "todo"`)
	b := stagedTask(t, s, "bbbbbb", `state: "todo"`)
	if _, e := s.TakeAs(a, "kestrel", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	_, e := s.TakeAs(b, "kestrel", "claude-opus-5-5")
	wantBusy(t, e, a, b)
	if tk, _ := load(t, s, b); tk.State != "todo" || len(tk.Assignee) != 0 {
		t.Fatalf("refused task changed: %s %v", node(tk), tk.Assignee)
	}
	// Someone else may still take it, and taking your own again is a no-op.
	if _, e = s.TakeAs(b, "nautilus", "claude-opus-5-5"); e != nil {
		t.Fatalf("another agent: %v", e)
	}
	if r, e := s.TakeAs(a, "kestrel", "claude-opus-5-5"); e != nil || r.Changed {
		t.Fatalf("retaking your own active task: %v changed=%v", e, r.Changed)
	}
	// A person is held to the same rule.
	c := stagedTask(t, s, "cccccc", `state: "todo"`)
	d := stagedTask(t, s, "dddddd", `state: "todo"`)
	if _, e = s.TakeAs(c, "Jørgen", "human"); e != nil {
		t.Fatal(e)
	}
	_, e = s.TakeAs(d, "Jørgen", "human")
	wantBusy(t, e, c, d, " --model human --as Jørgen")
	// Taking a review (todo/review -> doing/review) is not work of one's
	// own: it is allowed beside the active ticket.
	r := stagedTask(t, s, "rrrrrr", "state: \"todo\"\nstage: \"review\"\nreview_since: \"2026-11-17T08:00\"")
	if _, e = s.TakeAs(r, "Jørgen", "human"); e != nil {
		t.Fatalf("review take beside active work: %v", e)
	}
	if tk, _ := load(t, s, r); node(tk) != "doing/review" {
		t.Fatalf("review take: %s", node(tk))
	}
}

// n2hq8p check 2: an earlier ticket that is waiting, in review or depending
// on another task is not work in hand, so the next take goes through; each
// of the three fixes frees the taker.
func TestTakeAllowedWhenEarlierIsNotActive(t *testing.T) {
	fixedClock(t)
	for _, tc := range []struct {
		name  string
		front string
		deps  []string
	}{
		{"waiting", "state: \"doing\"\nstage: \"waiting\"\nwaiting_on: \"landing turn: after hatchling\"\nwaiting_since: \"2026-11-17T13:33\"\ndoing_since: \"2026-11-17T12:51\"", nil},
		{"review", "state: \"doing\"\nstage: \"review\"\nreview_by: \"kestrel\"\ndoing_since: \"2026-11-17T12:51\"", nil},
		{"depends", "state: \"doing\"\ndoing_since: \"2026-11-17T12:51\"", []string{"zzzzzz"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newStore(t)
			stagedTask(t, s, "zzzzzz", `state: "todo"`)
			first := stagedTask(t, s, "aaaaaa", tc.front+"\nmodel: \"claude-opus-5-5\"", tc.deps...)
			mark(t, s, first, "kestrel")
			if tk, _ := load(t, s, first); Stage(tk, all(t, s)) != tc.name {
				t.Fatalf("setup: stage %q, want %q", Stage(tk, all(t, s)), tc.name)
			}
			b := stagedTask(t, s, "bbbbbb", `state: "todo"`)
			if _, e := s.TakeAs(b, "kestrel", "claude-opus-5-5"); e != nil {
				t.Fatalf("take with the earlier one %s: %v", tc.name, e)
			}
			// Now b is active: resuming a waiting first one would make two.
			if tc.name == "waiting" {
				_, e := s.Resume(first)
				wantBusy(t, e, b, "")
			}
		})
	}
	// Each fix frees the taker: done, wait, back to todo.
	for _, free := range []string{"done", "wait", "todo"} {
		t.Run("after "+free, func(t *testing.T) {
			s, _ := newStore(t)
			s.AcceptanceMode = "warn"
			a := stagedTask(t, s, "aaaaaa", `state: "todo"`)
			b := stagedTask(t, s, "bbbbbb", `state: "todo"`)
			if _, e := s.TakeAs(a, "kestrel", "claude-opus-5-5"); e != nil {
				t.Fatal(e)
			}
			var e error
			switch free {
			case "done":
				_, e = s.Done(a)
			case "wait":
				_, e = s.Wait(a, "CI run 1", "")
			case "todo":
				_, e = s.Set(a, "state", "todo")
			}
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.TakeAs(b, "kestrel", "claude-opus-5-5"); e != nil {
				t.Fatalf("take after %s: %v", free, e)
			}
		})
	}
}

// mark sets the assignee of id without a take (a fixture already in doing).
func mark(t *testing.T, s *Store, id, who string) {
	t.Helper()
	if _, e := s.Set(id, "assignee", who); e != nil {
		t.Fatal(e)
	}
}

func all(t *testing.T, s *Store) []Task {
	t.Helper()
	ts, e := s.Tasks()
	if e != nil {
		t.Fatal(e)
	}
	return ts
}
