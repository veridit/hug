package task

import (
	"strings"
	"testing"
)

// qsqhpe check 1 (task half): the person asked ticking the check the
// question is about answers it: asked -> active, one answer block in
// History whose text is "answered by ticking check N", the asker sees it,
// and the Result says it answered (so the write paths deliver it).
func TestTickByAskedAnswers(t *testing.T) {
	s, id := tickAskStore(t)
	s.Actor = "jhf"
	r, e := s.Check(id, 2)
	if e != nil {
		t.Fatal(e)
	}
	if !r.Answered {
		t.Fatalf("the Result does not say the tick answered: %+v", r)
	}
	tk, raw := readTask(t, s, id)
	if tk.State != "doing" || tk.Stage != "active" || tk.Fields["asked_for"] != "" || tk.Fields["asked_check"] != "" {
		t.Fatalf("still asked: %s/%s %v\n%s", tk.State, tk.Stage, tk.Fields, raw)
	}
	want := "answered asked (check 2), by jhf, for nautilus (doing/asked -> doing/active):\n  > type stays, or drop?\n  answered by ticking check 2\n"
	if !strings.Contains(raw, want) {
		t.Fatalf("History lacks\n%s\nfile:\n%s", want, raw)
	}
	if !strings.Contains(raw, "- [x] b\n") {
		t.Fatalf("check 2 not ticked:\n%s", raw)
	}
	if !AnswerFor(tk, "nautilus") || tk.Answer.Text != "answered by ticking check 2" || tk.Answer.Check != 2 {
		t.Fatalf("the asker does not see it: %+v", tk.Answer)
	}
	if item, ok := NewestItem(tk, "answer"); !ok || item.Name() != "answer 1" {
		t.Fatalf("not an answer item: %+v %v", item, ok)
	}
	// Ticking it again is no second answer.
	if r, e := s.Check(id, 2); e != nil || r.Answered || r.Changed {
		t.Fatalf("second tick: %+v %v", r, e)
	}
}

// qsqhpe check 2: a tick by anyone but the person asked, or of another
// check, leaves the question open.
func TestTickByOthersLeavesQuestion(t *testing.T) {
	for _, c := range []struct {
		actor, why string
		n          int
	}{
		{"nautilus", "the asker", 2},
		{"wyvern", "another agent", 2},
		{"", "nobody named", 2},
		{"jhf", "the person asked, another check", 1},
	} {
		s, id := tickAskStore(t)
		s.Actor = c.actor
		r, e := s.Check(id, c.n)
		if e != nil {
			t.Fatal(e)
		}
		tk, raw := readTask(t, s, id)
		if r.Answered || tk.Stage != "asked" || tk.Fields["asked_for"] != "jhf" || strings.Contains(raw, "answered by ticking") {
			t.Fatalf("%s ticking check %d answered: %+v stage %s\n%s", c.why, c.n, r.Answered, tk.Stage, raw)
		}
	}
}

// pw0t19 and qsqhpe together: the person asked ticking the check with
// evidence answers too, and the evidence is noted in the same write.
func TestTickWithEvidenceAnswers(t *testing.T) {
	s, id := tickAskStore(t)
	s.Actor = "jhf"
	r, e := s.CheckWith(id, 2, "looked at it, keep the type")
	if e != nil || !r.Answered {
		t.Fatalf("tick with evidence did not answer: %+v %v", r, e)
	}
	tk, raw := readTask(t, s, id)
	if tk.Stage != "active" || !strings.Contains(raw, "answered by ticking check 2") || !strings.Contains(raw, "check 2: looked at it, keep the type") {
		t.Fatalf("stage %s:\n%s", tk.Stage, raw)
	}
	// Evidence on the already ticked check is noted, and is no second answer.
	if r, e := s.CheckWith(id, 2, "again"); e != nil || r.Answered {
		t.Fatalf("evidence on a ticked check answered again: %+v %v", r, e)
	}
	// Only a tick answers: when someone else ticked the check first, the
	// person asked adding evidence to it does not answer the question.
	s, id = tickAskStore(t)
	s.Actor = "nautilus"
	if _, e := s.Check(id, 2); e != nil {
		t.Fatal(e)
	}
	s.Actor = "jhf"
	if r, e := s.CheckWith(id, 2, "seen"); e != nil || r.Answered {
		t.Fatalf("evidence on a check another ticked answered: %+v %v", r, e)
	}
}

// tickAskStore: nautilus holds a task with checks a and b and asked jhf
// about check 2.
func tickAskStore(t *testing.T) (*Store, string) {
	t.Helper()
	s, _ := answerStore(t)
	r, e := s.New(NewOptions{Title: "Tags", Goal: "G", Acceptance: []string{"a", "b"}, Type: "chore"})
	if e != nil {
		t.Fatal(e)
	}
	id := r.Task.ID
	if _, e := s.TakeAs(id, "nautilus", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Ask(id, Question{What: "type stays, or drop?", For: "jhf", Check: 2}, "nautilus"); e != nil {
		t.Fatal(e)
	}
	return s, id
}
