package task

import (
	"strings"
	"testing"
	"time"
)

// setClock sets Now to y-m-d 09:00 local.
func setClock(t *testing.T, y int, m time.Month, d int) {
	t.Helper()
	old := Now
	Now = func() time.Time { return time.Date(y, m, d, 9, 0, 0, 0, time.Local) }
	t.Cleanup(func() { Now = old })
}

// doingTask is a doing/active task of owner with three checks.
func doingTask(t *testing.T, s *Store, id, owner string) string {
	t.Helper()
	stagedTask(t, s, id, "state: \"todo\"\nstage: \"ready\"")
	if _, e := s.TakeAs(id, owner, "m"); e != nil {
		t.Fatal(e)
	}
	return id
}

// z2vmte check 1: wait --until DATE parks the task in WAITING with the date;
// before the date a read changes nothing; on the first read on or after it,
// the task is doing/active, first in its owner's next, with one History line.
func TestWaitUntilWakesOnTheDate(t *testing.T) {
	setClock(t, 2026, 11, 18)
	s := &Store{Root: t.TempDir()}
	a := doingTask(t, s, "zzzzzz", "kestrel")
	if _, e := s.WaitUntil(a, "a week of use, then look again", "2026-11-24", "kestrel"); e != nil {
		t.Fatal(e)
	}
	// Another task of the owner, ranked before a (id order), active.
	other := doingTask(t, s, "aaaaaa", "kestrel")
	if _, e := s.Wait(other, "CI run 1", "kestrel"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Resume(other); e != nil {
		t.Fatal(e)
	}
	tk, raw := load(t, s, a)
	if node(tk) != "doing/waiting" || tk.Fields["waiting_until"] != "2026-11-24" || tk.Fields["waiting_on"] != "a week of use, then look again" {
		t.Fatalf("wait --until: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	if !strings.Contains(raw, "doing/active -> doing/waiting (wait until 2026-11-24)") {
		t.Fatalf("history:\n%s", raw)
	}
	// The day before: nothing is due, Wake changes nothing.
	setClock(t, 2026, 11, 23)
	ts, _ := s.Tasks()
	if due := Due(ts); len(due) != 0 {
		t.Fatalf("due on 23 Nov: %v", due)
	}
	if r, e := s.Wake(a); e != nil || r.Changed {
		t.Fatalf("wake before the date: changed %v, %v", r.Changed, e)
	}
	// The day: due, and the wake makes it active with one History line.
	setClock(t, 2026, 11, 24)
	ts, _ = s.Tasks()
	if due := Due(ts); len(due) != 1 || due[0] != a {
		t.Fatalf("due on 24 Nov: %v", due)
	}
	if r, e := s.Wake(a); e != nil || !r.Changed {
		t.Fatalf("wake on the date: changed %v, %v", r.Changed, e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "doing/active" || tk.Fields["waiting_until"] != "" || tk.Fields["waiting_on"] != "" {
		t.Fatalf("after the wake: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	if n := strings.Count(raw, "woke: the date 2026-11-24 has come"); n != 1 {
		t.Fatalf("%d wake lines:\n%s", n, raw)
	}
	if !strings.Contains(raw, "doing/waiting -> doing/active (woke: the date 2026-11-24 has come); was waiting_asked_by: kestrel; waiting_on: a week of use, then look again; waiting_since: 2026-11-18T09:00; waiting_until: 2026-11-24") {
		t.Fatalf("wake line:\n%s", raw)
	}
	// First in the owner's next, ahead of aaaaaa, which is active and
	// would come first otherwise (doing before todo, then rank: here they
	// are the owner's two doing tasks).
	ts, _ = s.Tasks()
	if r, e := Next(ts, Filter{Mine: "kestrel"}); e != nil || r.Task.ID != a {
		t.Fatalf("next --mine: %s, %v", r.Task.ID, e)
	}
	// A second wake is a no-op.
	if r, e := s.Wake(a); e != nil || r.Changed {
		t.Fatalf("second wake: changed %v, %v", r.Changed, e)
	}
	// A note by someone else leaves it woken; once the owner writes on it,
	// the wake is no longer news ("back on it after the wait").
	s.Actor = "crab"
	if _, e := s.Note(a, "seen"); e != nil {
		t.Fatal(e)
	}
	if tk, _ = load(t, s, a); !tk.Woken {
		t.Fatalf("a note by crab ended the wake")
	}
	s.Actor = "kestrel"
	if _, e := s.Note(a, "looked again"); e != nil {
		t.Fatal(e)
	}
	if tk, raw = load(t, s, a); tk.Woken || strings.Count(raw, "kestrel back on it after the wait") != 1 {
		t.Fatalf("still woken after the owner's note:\n%s", raw)
	}
	ts, _ = s.Tasks()
	if r, e := Next(ts, Filter{Mine: "kestrel"}); e != nil || r.Task.ID != "aaaaaa" {
		t.Fatalf("next --mine after: %s, %v", r.Task.ID, e)
	}
}

// z2vmte check 2: ask --after DATE: in nobody's ASKED before the date (a
// wait, no asked_* field); on the date the question is asked verbatim with
// its check, by who asked.
func TestAskAfterAsksOnTheDate(t *testing.T) {
	setClock(t, 2026, 11, 18)
	s := &Store{Root: t.TempDir()}
	a := doingTask(t, s, "aaaaaa", "crab")
	q := Question{What: "a week of use: has he had to remind an agent\nto update a ticket?", For: "jhf", Check: 1}
	if _, e := s.AskAfter(a, q, "2026-11-24", "crab"); e != nil {
		t.Fatal(e)
	}
	tk, raw := load(t, s, a)
	if node(tk) != "doing/waiting" || tk.Fields["asked_what"] != "" || tk.Fields["asked_for"] != "" {
		t.Fatalf("before the date: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	if tk.Fields["waiting_ask_what"] != q.What || tk.Fields["waiting_ask_for"] != "jhf" || tk.Fields["waiting_ask_check"] != "1" || tk.Fields["waiting_until"] != "2026-11-24" {
		t.Fatalf("held question: %v", tk.Fields)
	}
	if strings.Contains(tk.Fields["waiting_on"], "jhf:") {
		t.Fatalf("waiting_on reads as a question to an old hug: %q", tk.Fields["waiting_on"])
	}
	// No stage-two finding (3nhp67) and no unknown-person for a held ask.
	ts, _ := s.Tasks()
	for _, f := range derivedFindings(ts) {
		if f.Path == tk.Path {
			t.Fatalf("finding before the date: %s", f.Problem)
		}
	}
	setClock(t, 2026, 11, 24)
	if r, e := s.Wake(a); e != nil || !r.Changed {
		t.Fatalf("wake: %v %v", r.Changed, e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "doing/asked" || tk.Fields["asked_what"] != q.What || tk.Fields["asked_for"] != "jhf" || tk.Fields["asked_check"] != "1" || tk.Fields["asked_by"] != "crab" || tk.Fields["asked_since"] != "2026-11-24T09:00" {
		t.Fatalf("on the date: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	for k := range tk.Fields {
		if strings.HasPrefix(k, "waiting_") {
			t.Fatalf("waiting field left: %s\n%s", k, raw)
		}
	}
	if !strings.Contains(raw, "doing/waiting -> doing/asked (woke: the date 2026-11-24 has come, asked jhf)") {
		t.Fatalf("history:\n%s", raw)
	}
	if tk.Woken {
		t.Fatalf("an ask is not woken back to its owner")
	}
}

// A plain wait on a task holding a question for a date drops the question
// (to History): one wait at a time (3nhp67).
func TestWaitReplacesHeldAsk(t *testing.T) {
	setClock(t, 2026, 11, 18)
	s := &Store{Root: t.TempDir()}
	a := doingTask(t, s, "aaaaaa", "crab")
	if _, e := s.AskAfter(a, Question{What: "ok?", For: "jhf"}, "2026-11-24", "crab"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Wait(a, "CI run 9", "crab"); e != nil {
		t.Fatal(e)
	}
	tk, raw := load(t, s, a)
	for k := range tk.Fields {
		if strings.HasPrefix(k, "waiting_ask_") || k == "waiting_until" {
			t.Fatalf("%s kept after a plain wait:\n%s", k, raw)
		}
	}
	if !strings.Contains(raw, "waiting_ask_what: ok?") {
		t.Fatalf("held question not in History:\n%s", raw)
	}
	setClock(t, 2026, 11, 24)
	if r, _ := s.Wake(a); r.Changed {
		t.Fatalf("a plain wait woke on the old date")
	}
}

// A wake is hug's write, not the reader's: an answer the owner has not seen
// stays unseen through it (3jtfw9), and the wake is not "back on it".
func TestWakeMarksNothingRead(t *testing.T) {
	setClock(t, 2026, 11, 18)
	s := &Store{Root: t.TempDir()}
	a := doingTask(t, s, "aaaaaa", "crab")
	if _, e := s.Ask(a, Question{What: "ok?", For: "jhf"}, "crab"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Answer(a, "yes", "jhf"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.WaitUntil(a, "look again", "2026-11-20", ""); e != nil {
		t.Fatal(e)
	}
	if tk, _ := load(t, s, a); !AnswerFor(tk, "crab") {
		t.Fatalf("answer seen before the wake")
	}
	setClock(t, 2026, 11, 20)
	s.Actor = "crab" // the reader whose hug wakes it
	if r, e := s.Wake(a); e != nil || !r.Changed {
		t.Fatalf("wake: %v %v", r.Changed, e)
	}
	tk, raw := load(t, s, a)
	if !AnswerFor(tk, "crab") || strings.Contains(raw, "read the answer") || strings.Contains(raw, "back on it") {
		t.Fatalf("the wake marked it:\n%s", raw)
	}
}

// Dates: not a date, or one that has come, is refused with a fix that runs.
func TestTimedWaitRefusesBadDates(t *testing.T) {
	setClock(t, 2026, 11, 18)
	s := &Store{Root: t.TempDir()}
	a := doingTask(t, s, "aaaaaa", "crab")
	_, e := s.WaitUntil(a, "why", "24 Nov", "crab")
	wantRefused(t, e, `hug task wait aaaaaa --until 2026-11-19 "why"`)
	_, e = s.WaitUntil(a, "why", "2026-11-18", "crab")
	wantRefused(t, e, `hug task wait aaaaaa --until 2026-11-19 "why"`)
	_, e = s.AskAfter(a, Question{What: "ok?", For: "jhf", Check: 1}, "2026-11-17", "crab")
	wantRefused(t, e, `hug task ask aaaaaa --to jhf --after 2026-11-19 --check 1 "ok?"`)
	if tk, _ := load(t, s, a); node(tk) != "doing/active" {
		t.Fatalf("a refused wait wrote: %s", node(tk))
	}
}
