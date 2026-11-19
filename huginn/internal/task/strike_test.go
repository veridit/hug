package task

import (
	"strings"
	"testing"
	"time"
)

// sfa8w4 checks 1, 2, 5: strike records reason, name and date in standard
// Markdown; done counts struck checks as settled and still refuses an open
// one with a fix line; older files parse unchanged.
func TestStrikeOutCheck(t *testing.T) {
	old := Now
	Now = func() time.Time { return time.Date(2026, 1, 31, 23, 59, 59, 900_000_000, time.Local) } // the strike is dated by the board clock (jsyksd)
	t.Cleanup(func() { Now = old })
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "Strike me", Goal: "G", Acceptance: []string{"first holds", "second, no longer", "third"}, Type: "chore"})
	if e != nil {
		t.Fatal(e)
	}
	id := r.Task.ID
	if _, e := s.Strike(id, 2, "", "kestrel", false); e == nil || !strings.Contains(e.Error(), "needs a reason") || !strings.Contains(e.Error(), "fix: hug task strike "+id+" 2 --reason") {
		t.Fatalf("no reason: %v", e)
	}
	if _, e := s.Strike(id, 2, "replaced by check 3, see note", "kestrel", false); e != nil {
		t.Fatal(e)
	}
	d, _ := s.Get(id)
	today := Now().Format("2006-01-02")
	line := "- [ ] ~~second, no longer~~ (not relevant: replaced by check 3, see note, kestrel, " + today + ")\n"
	if !strings.Contains(string(d.Bytes()), "\n"+line) {
		t.Fatalf("file:\n%s", d.Bytes())
	}
	a := d.Task.Acceptance[1]
	if !a.Struck || a.Checked || a.Text != "second, no longer" || a.Reason != "replaced by check 3, see note" || a.StruckBy != "kestrel" || a.StruckOn != today || a.State() != "struck" || !a.Settled() {
		t.Fatalf("parsed: %+v", a)
	}
	// Ticking a struck check is refused: restore it first.
	if _, e := s.Check(id, 2); e == nil || !strings.Contains(e.Error(), "fix: hug task strike "+id+" 2 --undo") {
		t.Fatalf("check struck: %v", e)
	}
	if _, e := s.Take(id, "kestrel"); e != nil {
		t.Fatal(e)
	}
	s.Check(id, 1)
	_, e = s.Done(id)
	if e == nil || !strings.Contains(e.Error(), "3. third") || strings.Contains(e.Error(), "2. second") || !strings.Contains(e.Error(), "or strike one that no longer applies: hug task strike "+id+" 3") || !strings.Contains(e.Error(), "fix: hug task check "+id+" 3") {
		t.Fatalf("done with check 3 open: %v", e)
	}
	// A ticked check cannot be struck.
	if _, e := s.Strike(id, 1, "x", "kestrel", false); e == nil || !strings.Contains(e.Error(), "is ticked") {
		t.Fatalf("strike ticked: %v", e)
	}
	if _, e := s.Strike(id, 3, "obsolete", "kestrel", false); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Done(id); e != nil {
		t.Fatalf("done with ticked + struck: %v", e)
	}
	d, _ = s.Get(id)
	if fs := Lint([]*Document{d}); len(fs) != 0 {
		t.Fatalf("lint: %v", fs)
	}
	// Undo restores the text as an open check.
	if r, e := s.Strike(id, 3, "", "", true); e != nil || !r.Changed {
		t.Fatal(r, e)
	}
	d, _ = s.Get(id)
	if !strings.Contains(string(d.Bytes()), "\n- [ ] third\n") || d.Task.Acceptance[2].Struck {
		t.Fatalf("undo:\n%s", d.Bytes())
	}
}

// t01kqm: a done task with an open check (an import or a hand edit; done
// itself refuses one) is history, not a finding: lint judges open tasks.
func TestDoneWithOpenCheckIsNotAFinding(t *testing.T) {
	b := strings.Replace(sample, "state: todo", "state: done", 1)
	d := Parse("zzzzzz.k7m2qx.recall-gate.md", []byte(b))
	for _, f := range Lint([]*Document{d}) {
		if strings.Contains(f.Problem, "unchecked") {
			t.Fatalf("finding on a done task's open check: %s", f.Problem)
		}
	}
}

// sfa8w4 check 5: the stored form is a standard checklist line with
// strikethrough; old files (no strike) parse as before, byte-identical.
func TestStruckFormParsesAndOldFilesUnchanged(t *testing.T) {
	d := Parse("a0.k7m2qx.recall-gate.md", []byte(sample))
	if len(d.Task.Acceptance) != 2 || d.Task.Acceptance[0].Struck || d.Task.Acceptance[0].Text != "observable one" || !d.Task.Acceptance[1].Checked || string(d.Bytes()) != sample {
		t.Fatalf("old file: %+v", d.Task.Acceptance)
	}
	for in, want := range map[string]Acceptance{
		"~~a~~ (not relevant: why, bob, 2026-11-17)":        {Text: "a", Struck: true, Reason: "why", StruckBy: "bob", StruckOn: "2026-11-17"},
		"~~a, b~~ (not relevant: x, y, z, bob, 2026-11-17)": {Text: "a, b", Struck: true, Reason: "x, y, z", StruckBy: "bob", StruckOn: "2026-11-17"},
		"~~hand written~~ (not relevant: no longer needed)": {Text: "hand written", Struck: true, Reason: "no longer needed"},
		"~~just struck~~": {Text: "~~just struck~~"},
		"plain text ~~with~~ inline (not relevant: no)": {Text: "plain text ~~with~~ inline (not relevant: no)"},
	} {
		got := parseAcceptance(in, false, 1)
		got.Line = 0
		if got != want {
			t.Errorf("%q: %+v, want %+v", in, got, want)
		}
	}
	// A ticked line is ticked, whatever its text.
	if a := parseAcceptance("~~a~~ (not relevant: why)", true, 1); a.Struck || !a.Checked {
		t.Fatalf("ticked: %+v", a)
	}
}
