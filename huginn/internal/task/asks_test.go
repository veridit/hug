package task

import (
	"strings"
	"testing"
)

// q2adr8: a task asked before questions were numbered (asked_* only)
// adopts its question as question 1 when a second is asked; leaving asked
// withdraws the open ones, so a later ask does not bring them back.
func TestAskAdoptsLegacyAndWithdrawsOnResume(t *testing.T) {
	s, id := answerStore(t)
	if _, e := s.TakeAs(id, "nautilus", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	// A pre-q2adr8 ask: asked_* set, no ## Questions.
	d, _, e := s.load(id)
	if e != nil {
		t.Fatal(e)
	}
	for k, v := range map[string]string{"stage": "asked", "asked_what": "old question?", "asked_for": "jhf", "asked_by": "nautilus", "asked_since": "2026-11-18T10:00"} {
		if e := d.SetField(k, v); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.finish(d, true); e != nil {
		t.Fatal(e)
	}
	// One open legacy question: a bare answer still answers it.
	if _, e := s.Ask(id, Question{What: "new question?", For: "jhf"}, "nautilus"); e != nil {
		t.Fatal(e)
	}
	tk, raw := readTask(t, s, id)
	if !strings.Contains(raw, "- 2026-11-18T10:00 question 1 by nautilus to jhf:\n  old question?\n") || !strings.Contains(raw, " question 2 by nautilus to jhf:\n  new question?\n") {
		t.Fatalf("legacy question not adopted as 1:\n%s", raw)
	}
	if tk.Fields["asked_what"] != "old question?" || !strings.Contains(raw, "asked question 2 of jhf, 2 open") {
		t.Fatalf("stage not on the oldest, or no History line:\n%s", raw)
	}
	// Resume leaves asked: both are withdrawn; a new ask is question 3
	// alone, the stage on it.
	if _, e := s.Resume(id); e != nil {
		t.Fatal(e)
	}
	if _, raw = readTask(t, s, id); strings.Count(raw, "  withdrawn ") != 2 {
		t.Fatalf("resume did not withdraw both:\n%s", raw)
	}
	if _, e := s.Ask(id, Question{What: "third?", For: "jhf"}, "nautilus"); e != nil {
		t.Fatal(e)
	}
	tk, raw = readTask(t, s, id)
	if len(OpenQuestions(tk)) != 1 || tk.Fields["asked_what"] != "third?" {
		t.Fatalf("a withdrawn question came back: %+v\n%s", OpenQuestions(tk), raw)
	}
	if _, e := s.Answer(id, "yes", "jhf"); e != nil {
		t.Fatal(e)
	}
	if tk, _ = readTask(t, s, id); tk.Stage != "active" || tk.Questions[2].Answer == nil || tk.Questions[2].Answer.Text != "yes" {
		t.Fatalf("answer: %+v", tk.Questions)
	}
}

// Two clones each ask a question on the same task: the merge keeps both,
// the later renumbered past the board's; an answer on one side stays
// under its question.
func TestMergeQuestionsRenumbersOursAfterBoards(t *testing.T) {
	base := edit(mergeBase, "## Notes\n", "## Questions\n- 2026-11-18T04:00 question 1 by nautilus to jhf (check 1):\n  old?\n\n## Notes\n")
	ours := edit(string(base), "  old?\n", "  old?\n- 2026-11-18T05:20 question 2 by kestrel to jhf:\n  from B?\n")
	theirs := edit(string(base), "  old?\n", "  old?\n  answered 2026-11-18T05:05 by jhf:\n    yes\n- 2026-11-18T05:10 question 2 by sam to jhf:\n  from A?\n")
	got, kept, e := MergeTask("8/8q6v7b.md", base, ours, theirs, side)
	if e != nil || len(kept) != 0 {
		t.Fatalf("e=%v kept=%+v", e, kept)
	}
	want := edit(string(base), "  old?\n", "  old?\n  answered 2026-11-18T05:05 by jhf:\n    yes\n- 2026-11-18T05:10 question 2 by sam to jhf:\n  from A?\n- 2026-11-18T05:20 question 3 by kestrel to jhf:\n  from B?\n")
	if string(got) != string(want) {
		t.Fatalf("merged:\n%s\nwant:\n%s", got, want)
	}
	qs := Parse("8/8q6v7b.md", got).Task.Questions
	if len(qs) != 3 || qs[0].Answer == nil || qs[0].Answer.Text != "yes" || qs[2].N != 3 || qs[2].Text != "from B?" || !qs[2].Open() {
		t.Fatalf("parsed: %+v", qs)
	}
}
