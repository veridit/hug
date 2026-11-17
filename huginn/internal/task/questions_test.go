package task

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func strconvItoa(n int64) string { return strconv.FormatInt(n, 10) }

// question returns the question finding (code) Lint gives task id on the
// store's board, or nil.
func questionOf(t *testing.T, s *Store, id string) *Finding {
	t.Helper()
	ts, err := s.Tasks()
	if err != nil {
		t.Fatal(err)
	}
	var docs []*Document
	for _, tk := range ts {
		b, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(tk.Path)))
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, Parse(tk.Path, b))
	}
	for _, f := range TopQuestions(Lint(docs)) {
		if _, fid, _, _ := ParseName(f.Path); fid == id && QuestionCode(FindingCode(f.Problem)) {
			return &f
		}
	}
	return nil
}

// TestQuestionsFromNotes (at3qep check 1, cases a, c, d): each state gives
// one question naming what was observed and from what, with a fix that
// clears it; a ticket raising two asks only the first.
func TestQuestionsFromNotes(t *testing.T) {
	s := &Store{Root: t.TempDir(), DepsMode: "warn"}
	newTask := func(title string) string {
		r, err := s.New(NewOptions{Title: title, Goal: "g", Purpose: "p", Principle: "q", Acceptance: []string{"one", "two"}})
		if err != nil {
			t.Fatal(err)
		}
		return r.Task.ID
	}
	want := func(id, code string, parts ...string) *Finding {
		t.Helper()
		q := questionOf(t, s, id)
		if q == nil || FindingCode(q.Problem) != code {
			t.Fatalf("%s: want a %s question, got %+v", id, code, q)
		}
		for _, p := range append(parts, id) {
			if !strings.Contains(q.Problem+" | "+q.FixLine(), p) {
				t.Fatalf("%s question %q (fix %q) lacks %q", id, q.Problem, q.FixLine(), p)
			}
		}
		return q
	}
	none := func(id, why string) {
		t.Helper()
		if q := questionOf(t, s, id); q != nil {
			t.Fatalf("%s: %s, but asked %q", id, why, q.Problem)
		}
	}

	// (a) doing, a commit naming it reached main, 0 of 2 checks ticked.
	a := newTask("Landed unchecked")
	if _, err := s.Take(a, "kestrel"); err != nil {
		t.Fatal(err)
	}
	none(a, "nothing landed yet")
	s.Note(a, LandedNote(Commit{Hash: "61bfb21aaaa", Subject: "hug: a fix (" + a + ")", Date: "2026-11-17"}, "main"))
	q := want(a, "landed-unchecked", "commit 61bfb21", "reached main", "0 of 2", "kestrel", "hug task check "+a+" 1")
	if !strings.Contains(q.Problem, "note says so") {
		t.Fatalf("question does not say where it was seen: %q", q.Problem)
	}
	if _, err := s.Check(a, 1); err != nil {
		t.Fatal(err)
	}
	none(a, "a check is ticked")

	// (c) shipped in a release, still doing; it outranks (a) on one ticket.
	c := newTask("Shipped open")
	if _, err := s.Take(c, "wyvern"); err != nil {
		t.Fatal(err)
	}
	s.Note(c, LandedNote(Commit{Hash: "d74af5daaaa", Subject: "x (" + c + ")", Date: "2026-11-17"}, "main"))
	s.Note(c, ShippedNote("v0.7.38", Commit{Hash: "d74af5daaaa"}))
	want(c, "landed-unchecked", "d74af5d") // (a) first: 0 checks ticked
	for _, n := range []int{1, 2} {
		if _, err := s.Check(c, n); err != nil {
			t.Fatal(err)
		}
	}
	want(c, "shipped-not-done", "release v0.7.38", "commit d74af5d", "wyvern", "hug task done "+c)
	if _, err := s.Done(c); err != nil {
		t.Fatal(err)
	}
	none(c, "it is done")

	// (d) todo, but a commit naming it reached main.
	d := newTask("Todo with commit")
	none(d, "no commit")
	s.Note(d, LandedNote(Commit{Hash: "2c27e42aaaa", Subject: "y (" + d + ")", Date: "2026-11-17"}, "main"))
	want(d, "todo-has-commit", "commit 2c27e42", "hug task take "+d)
	if _, err := s.Take(d, "dragon"); err != nil {
		t.Fatal(err)
	}
	// Taken: now doing with 0 checks and a landed note, which is (a).
	want(d, "landed-unchecked")
}

// TestIdleQuestion (at3qep check 1, case b): a doing, active ticket with
// no commit touching it for IdleAfter is asked about; a later commit on its
// file or a main subject naming it clears it, and a ticket that already has
// a question is not asked twice.
func TestIdleQuestion(t *testing.T) {
	now := time.Date(2026, 11, 17, 18, 0, 0, 0, time.Local)
	tk := Task{ID: "b1b1b1", Path: "x.b1b1b1.idle.md", State: "doing", Stage: "active", Assignee: []string{"kestrel"},
		Fields: map[string]string{"doing_since": now.Add(-5 * time.Hour).Format("2006-01-02T15:04")}}
	ask := func(last map[string]time.Time, asked map[string]bool) []Finding {
		return IdleFindings([]Task{tk}, last, asked, now)
	}
	if fs := ask(nil, nil); len(fs) != 0 {
		t.Fatalf("no history readable: want nothing, got %v", fs)
	}
	lines := []string{
		strconvItoa(now.Add(-4*time.Hour).Unix()) + "\thug task note b1b1b1\tzz.b1b1b1.idle.md",
		strconvItoa(now.Add(-3*time.Hour).Unix()) + "\thug task take c2c2c2",
	}
	fs := ask(TouchedSince(lines), nil)
	if len(fs) != 1 || FindingCode(fs[0].Problem) != "doing-idle" || !strings.Contains(fs[0].Problem, "4 hours") || !strings.Contains(fs[0].Problem, "kestrel") || fs[0].FixLine() != `hug task note b1b1b1 "where it stands"` {
		t.Fatalf("idle question: %+v", fs)
	}
	// A commit on main naming it an hour ago clears it.
	lines = append(lines, strconvItoa(now.Add(-time.Hour).Unix())+"\thug mem: something (b1b1b1)")
	if fs := ask(TouchedSince(lines), nil); len(fs) != 0 {
		t.Fatalf("a commit an hour ago: want nothing, got %v", fs)
	}
	// Waiting is not idle; already asked is not asked twice.
	if fs := ask(TouchedSince(lines[:1]), map[string]bool{"b1b1b1": true}); len(fs) != 0 {
		t.Fatalf("already asked: %v", fs)
	}
	tk.Stage = "waiting"
	if fs := ask(TouchedSince(lines[:1]), nil); len(fs) != 0 {
		t.Fatalf("waiting: %v", fs)
	}
}

// TestTopQuestionsOnePerTicket: of several questions on one ticket only the
// highest ranked stays (a > c > e > d > b); other findings stay.
func TestTopQuestionsOnePerTicket(t *testing.T) {
	fs := []Finding{
		{Path: "p.aaaaaa.x.md", Problem: "doing, but nothing on it for 3 hours"},
		{Path: "p.aaaaaa.x.md", Problem: "branch x is deleted, but aaaaaa is still doing", Code: "branch-gone"},
		{Path: "p.aaaaaa.x.md", Problem: "missing Goal"},
		{Path: "p.bbbbbb.y.md", Problem: "doing, but nothing on it for 3 hours"},
	}
	got := TopQuestions(fs)
	var s []string
	for _, f := range got {
		s = append(s, f.Path[2:8]+":"+FindingCode(f.Problem))
	}
	if strings.Join(s, " ") != "aaaaaa:branch-gone aaaaaa:missing-goal bbbbbb:doing-idle" {
		t.Fatalf("top questions: %v", s)
	}
}
