package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func answerStore(t *testing.T) (*Store, string) {
	t.Helper()
	old := Now
	Now = func() time.Time { return time.Date(2026, 11, 17, 13, 30, 0, 0, time.UTC) }
	t.Cleanup(func() { Now = old })
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "Tags", Goal: "G", Acceptance: []string{"C"}, Type: "chore"})
	if e != nil {
		t.Fatal(e)
	}
	return s, r.Task.ID
}

func readTask(t *testing.T, s *Store, id string) (Task, string) {
	t.Helper()
	d, e := s.Get(id)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(filepath.Join(s.Root, d.Task.Path))
	return d.Task, string(b)
}

// 3jtfw9 checks 2 and 4 (task half): answering a design question writes one
// History line with the question, the answer, who and for whom; the design
// fields go, the stage re-derives, no new field is written; the asker sees
// it until it writes on the task.
func TestAnswerDesign(t *testing.T) {
	s, id := answerStore(t)
	if _, e := s.Design(id, `type stays, or "drop"?`, "nautilus", "jhf"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Answer(id, "", "jhf"); e == nil || !strings.Contains(e.Error(), "fix: hug task answer "+id) {
		t.Fatalf("empty answer: %v", e)
	}
	if _, e := s.Answer(id, "drop it", ""); e == nil || !strings.Contains(e.Error(), "--as NAME") {
		t.Fatalf("no name: %v", e)
	}
	if _, e := s.Answer(id, "drop it", "jhf"); e != nil {
		t.Fatal(e)
	}
	tk, raw := readTask(t, s, id)
	want := "answered design, by jhf, for nautilus (todo/design -> todo/ready):\n  > type stays, or \"drop\"?\n  drop it\n"
	if !strings.Contains(raw, want) {
		t.Fatalf("history lacks %q:\n%s", want, raw)
	}
	if tk.Stage != "ready" || len(tk.Fields) != 0 || strings.Contains(raw, "\ndesign_") || strings.Contains(raw, "\nanswer") {
		t.Fatalf("stage %q fields %v:\n%s", tk.Stage, tk.Fields, raw)
	}
	if !AnswerFor(tk, "nautilus") || AnswerFor(tk, "jhf") || tk.Answer.Text != "drop it" || tk.Answer.Question != `type stays, or "drop"?` || tk.Answer.By != "jhf" {
		t.Fatalf("answer: %+v", tk.Answer)
	}
	// Someone else writing does not mark it read.
	s.Actor = "crab"
	if _, e := s.Note(id, "crab looked"); e != nil {
		t.Fatal(e)
	}
	if tk, _ = readTask(t, s, id); !AnswerFor(tk, "nautilus") {
		t.Fatal("a write by crab marked nautilus's answer read")
	}
	// The asker's write does.
	s.Actor = "nautilus"
	if _, e := s.Note(id, "dropping type"); e != nil {
		t.Fatal(e)
	}
	tk, raw = readTask(t, s, id)
	if tk.Answer != nil || !strings.Contains(raw, "nautilus read the answer") {
		t.Fatalf("not read: %+v\n%s", tk.Answer, raw)
	}
	// Once: the next write adds nothing.
	if _, e := s.Note(id, "more"); e != nil {
		t.Fatal(e)
	}
	if _, raw = readTask(t, s, id); strings.Count(raw, "read the answer") != 1 {
		t.Fatalf("read twice:\n%s", raw)
	}
}

func TestAnswerAskedAndReview(t *testing.T) {
	s, id := answerStore(t)
	if _, e := s.TakeAs(id, "nautilus", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Ask(id, Question{What: "which colour?", For: "jhf"}, "nautilus"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Answer(id, "green", "jhf"); e != nil {
		t.Fatal(e)
	}
	tk, raw := readTask(t, s, id)
	if tk.Stage != "active" || len(tk.Fields) != 1 || !strings.Contains(raw, "answered asked, by jhf, for nautilus (doing/asked -> doing/active):\n  > which colour?\n  green\n") {
		t.Fatalf("asked: %v\n%s", tk.Fields, raw)
	}
	// Nothing asked: refused, with a fix.
	if _, e := s.Answer(id, "again", "jhf"); e == nil || !strings.Contains(e.Error(), "asks nobody") || !strings.Contains(e.Error(), "fix: hug task show "+id) {
		t.Fatalf("active: %v", e)
	}
	// A wait asks nobody: resume ends it.
	if _, e := s.Wait(id, "CI run 1", "nautilus"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Answer(id, "green", "jhf"); e == nil || !strings.Contains(e.Error(), "fix: hug task resume "+id) {
		t.Fatalf("answer on a wait: %v", e)
	}
	// A review: the reviewer's verdict ends the review, done stays its own step.
	// nautilus owns it, jhf reviews it (doing/review).
	s2, id2 := answerStore(t)
	if _, e := s2.TakeAs(id2, "nautilus", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	d, _ := s2.Get(id2)
	if e := d.SetField("review_by", "jhf"); e != nil {
		t.Fatal(e)
	}
	if e := d.SetField("stage", "review"); e != nil {
		t.Fatal(e)
	}
	if e := AtomicWrite(filepath.Join(s2.Root, d.Task.Path), d.Bytes()); e != nil {
		t.Fatal(e)
	}
	s2.Invalidate()
	if _, e := s2.Answer(id2, "looks right", "jhf"); e != nil {
		t.Fatal(e)
	}
	tk, raw = readTask(t, s2, id2)
	if tk.State != "doing" || tk.Stage != "active" || tk.Fields["review_by"] != "" || !strings.Contains(raw, "answered review, by jhf, for nautilus (doing/review -> doing/active):\n  > review by jhf\n  looks right\n") {
		t.Fatalf("review: %s/%s\n%s", tk.State, tk.Stage, raw)
	}
}

// next --mine puts the task with my unseen answer first.
func TestNextMineAnswerFirst(t *testing.T) {
	s, id := answerStore(t)
	other, e := s.New(NewOptions{Title: "Mine, doing", Goal: "G", Acceptance: []string{"C"}, Type: "chore"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e := s.TakeAs(other.Task.ID, "nautilus", "m"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Design(id, "q?", "nautilus", "jhf"); e != nil {
		t.Fatal(e)
	}
	if r, e := s.Next(Filter{Mine: "nautilus"}); e != nil || r.Task.ID != other.Task.ID {
		t.Fatalf("before: %v %v", r.Task.ID, e)
	}
	if _, e := s.Answer(id, "a", "jhf"); e != nil {
		t.Fatal(e)
	}
	r, e := s.Next(Filter{Mine: "nautilus"})
	if e != nil || r.Task.ID != id || r.Next != "hug task show "+id {
		t.Fatalf("after: %v %q %v", r.Task.ID, r.Next, e)
	}
}

func TestUnseenAnswerParse(t *testing.T) {
	h := []string{
		`- 2026-11-17T13:30 answered design: "q1" -> "a1", by jhf, for nautilus (todo/design -> todo/ready)` + "\n",
		"- 2026-11-17T13:31 nautilus read the answer\n",
		`- 2026-11-17T13:40Z answered waiting: "q2" -> "a2", by jhf, for kestrel (doing/waiting -> doing/active)` + "\r\n",
	}
	a := unseenAnswer(h)
	if a == nil || a.For != "kestrel" || a.Text != "a2" || a.At != "2026-11-17T13:40Z" {
		t.Fatalf("%+v", a)
	}
	if unseenAnswer(h[:2]) != nil {
		t.Fatal("read answer still unseen")
	}
	// An answer without an asker, or by the asker, needs no reading.
	if unseenAnswer([]string{`- t answered design: "q" -> "a", by jhf (todo/design -> todo/ready)`}) != nil {
		t.Fatal("no asker")
	}
}
