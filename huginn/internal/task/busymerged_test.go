package task

import (
	"errors"
	"strings"
	"testing"
)

// y8we58 check 2 (cd63d1 r1, rooster: take refused a second ticket while
// cd63d1 sat green waiting for its merge; of the five fix: lines the one
// that fit, hug task wait ID "branch ID merged", was not named): when the
// active task has every check settled, the refusal names that wait first.
// With a check still open the order is unchanged (done first).
func TestBusyAllTickedNamesWaitMergedFirst(t *testing.T) {
	s, _ := newStore(t)
	s.AcceptanceMode = "warn"
	a := stagedTask(t, s, "aaaaaa", `state: "todo"`)
	b := stagedTask(t, s, "bbbbbb", `state: "todo"`)
	if _, e := s.TakeAs(a, "kestrel", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	// One open check: done is the first way out, as before.
	_, e := s.TakeAs(b, "kestrel", "claude-opus-5-5")
	wantBusy(t, e, a, b)
	if _, e = s.CheckWith(a, 1, ""); e != nil {
		t.Fatal(e)
	}
	_, e = s.TakeAs(b, "kestrel", "claude-opus-5-5")
	merged := "hug task wait " + a + ` "branch ` + a + ` merged"`
	te := wantBusyMerged(t, e, a, b)
	if te.Fix != merged || te.Fixes[0] != merged {
		t.Fatalf("fix %q, fixes %q: want %q first", te.Fix, te.Fixes, merged)
	}
	if first := strings.Index(te.Error(), "\nfix: "); first < 0 || !strings.HasPrefix(te.Error()[first:], "\nfix: "+merged+"\n") {
		t.Fatalf("first fix: line is not the wait on the merge:\n%s", te.Error())
	}
}

// wantBusyMerged is wantBusy for an active task with every check settled:
// the wait on the merge first, then done, back to todo, the detour and the
// take with (the generic wait "on what" is replaced by the specific one).
func wantBusyMerged(t *testing.T, e error, active, taken string) *Error {
	t.Helper()
	var te *Error
	if !errors.As(e, &te) || te.Kind != "busy" {
		t.Fatalf("want busy refusal naming %s, got %v", active, e)
	}
	want := []string{
		"hug task wait " + active + ` "branch ` + active + ` merged"`,
		"hug task done " + active,
		"hug task set " + active + " --state todo && hug task note " + active + ` "why it goes back"`,
		"hug task dep " + active + " --add " + taken + " && hug task note " + active + ` "where I was"`,
		"hug task take " + taken + " --with " + active + " --model claude-opus-5-5 --as kestrel",
	}
	if strings.Join(te.Fixes, "|") != strings.Join(want, "|") || te.Fix != want[0] {
		t.Fatalf("fixes %q (fix %q), want %q", te.Fixes, te.Fix, want)
	}
	if strings.Count(te.Error(), "\nfix: ") != len(want) {
		t.Fatalf("want %d fix: lines:\n%s", len(want), te.Error())
	}
	return te
}
