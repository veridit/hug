package task

import (
	"errors"
	"strings"
	"testing"
)

// 4tycwd check 2: a take refused for one ticket in doing offers a fourth
// way out, the detour: the active task depends on the one being taken, and
// a note keeps where it was. Following it lets the take through, and done
// on the detour brings the first task back to doing/active by itself.
func TestBusyOffersTheDetourFix(t *testing.T) {
	s, _ := newStore(t)
	s.AcceptanceMode = "warn"
	a := stagedTask(t, s, "aaaaaa", `state: "todo"`)
	b := stagedTask(t, s, "bbbbbb", `state: "todo"`)
	if _, e := s.TakeAs(a, "kestrel", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	_, e := s.TakeAs(b, "kestrel", "claude-opus-5-5")
	var te *Error
	if !errors.As(e, &te) || te.Kind != "busy" {
		t.Fatalf("want busy, got %v", e)
	}
	want := "hug task dep " + a + " --add " + b + " && hug task note " + a + ` "where I was"`
	// byj479: the take-with is the fifth, after the detour.
	if len(te.Fixes) != 5 || te.Fixes[3] != want || te.Fixes[4] != "hug task take "+b+" --with "+a+" --model claude-opus-5-5 --as kestrel" || te.Fix != te.Fixes[0] {
		t.Fatalf("fixes %q, want a fourth %q and the take-with fifth", te.Fixes, want)
	}
	if _, e = s.AddDependencies(a, []string{b}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Note(a, "where I was: step 2"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.TakeAs(b, "kestrel", "claude-opus-5-5"); e != nil {
		t.Fatalf("take after the detour fix: %v", e)
	}
	if _, e = s.Done(b); e != nil {
		t.Fatal(e)
	}
	if tk, _ := load(t, s, a); Stage(tk, all(t, s)) != "active" {
		t.Fatalf("after the detour: stage %q, want active", Stage(tk, all(t, s)))
	}
	if r, e := Next(all(t, s), Filter{Mine: "kestrel"}); e != nil || r.Task.ID != a {
		t.Fatalf("next --mine after the detour: %s, %v", r.Task.ID, e)
	}
}

// 4tycwd checks 3 and 4: next means what to work on now, so my doing task
// in stage waiting is never next during its wait (crab 17:57). With nothing
// else, next --mine says so (exit 4) and lists the waits, one line each
// with what they wait on. A woken wait is active and next again.
func TestNextMineSkipsWaiting(t *testing.T) {
	setClock(t, 2026, 11, 18)
	s := &Store{Root: t.TempDir(), AcceptanceMode: "warn"}
	// aaaaaa ranks first: before the fix it came back as next work.
	a := doingTask(t, s, "aaaaaa", "kestrel")
	if _, e := s.Wait(a, "CI run 7", "kestrel"); e != nil {
		t.Fatal(e)
	}
	b := doingTask(t, s, "bbbbbb", "kestrel")
	if r, e := Next(all(t, s), Filter{Mine: "kestrel"}); e != nil || r.Task.ID != b {
		t.Fatalf("next --mine with %s waiting: %s, %v; want %s", a, r.Task.ID, e, b)
	}
	if _, e := s.Done(b); e != nil {
		t.Fatal(e)
	}
	// A timed wait too, ranked after a.
	c := doingTask(t, s, "cccccc", "kestrel")
	if _, e := s.WaitUntil(c, "a week of use", "2026-11-24", "kestrel"); e != nil {
		t.Fatal(e)
	}
	_, e := Next(all(t, s), Filter{Mine: "kestrel"})
	var te *Error
	if !errors.As(e, &te) || te.Kind != "nothing-matches" || te.ExitCode() != 4 {
		t.Fatalf("next --mine with only waits: want nothing-matches (exit 4), got %v", e)
	}
	wantWaits := []string{a + " waiting on CI run 7", c + " waiting until 2026-11-24 · on a week of use"} // atex3d: the date first
	if strings.Join(te.Waiting, "|") != strings.Join(wantWaits, "|") {
		t.Fatalf("waiting lines %q, want %q", te.Waiting, wantWaits)
	}
	if !strings.Contains(te.Error(), "\n"+wantWaits[0]+"\n"+wantWaits[1]) {
		t.Fatalf("message must list the waits one per line:\n%s", te.Error())
	}
	// Plain next (not --mine) is unchanged: no waiting list.
	if _, e := Next(all(t, s), Filter{}); !errors.As(e, &te) || len(te.Waiting) != 0 {
		t.Fatalf("plain next: %v", e)
	}
	// The day comes: c wakes, is active and is next.
	setClock(t, 2026, 11, 24)
	if _, e := s.Wake(c); e != nil {
		t.Fatal(e)
	}
	if r, e := Next(all(t, s), Filter{Mine: "kestrel"}); e != nil || r.Task.ID != c {
		t.Fatalf("next --mine after the wake: %s, %v; want %s", r.Task.ID, e, c)
	}
}

// 4tycwd check 5: resume says once what the task was waiting on, since
// afterwards that text is only in ## History, which show does not print.
func TestResumeSaysWhatItWaitedOn(t *testing.T) {
	setClock(t, 2026, 11, 18)
	s := &Store{Root: t.TempDir()}
	a := doingTask(t, s, "aaaaaa", "kestrel")
	if _, e := s.Wait(a, "detour to bbbbbb; was at step 2", "kestrel"); e != nil {
		t.Fatal(e)
	}
	r, e := s.Resume(a)
	if e != nil {
		t.Fatal(e)
	}
	want := "was waiting on: detour to bbbbbb; was at step 2"
	if len(r.Warnings) != 0 || r.Said != want {
		t.Fatalf("resume said %q (warnings %q), want %q", r.Said, r.Warnings, want)
	}
}
