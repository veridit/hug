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

// m5pbb0: a task whose questions stay open in ## Questions after their
// mirror is gone (a pre-q2adr8 hug answered question 1 into History and
// cleared design_*/asked_*, p87ngv 2026-11-19) still takes an answer to
// each: answer ID N answers N, and the mirror comes back on the oldest
// still open, so the stage again says who is asked. Todo (design) and
// doing (asked) alike.
func TestAnswerOpenQuestionWithoutMirror(t *testing.T) {
	for _, state := range []string{"todo", "doing"} {
		t.Run(state, func(t *testing.T) {
			s, id := answerStore(t)
			if state == "doing" {
				if _, e := s.TakeAs(id, "nautilus", "claude-opus-5-5"); e != nil {
					t.Fatal(e)
				}
			}
			for _, q := range []string{"first?", "second?", "third?"} {
				if _, e := s.Ask(id, Question{What: q, For: "jhf"}, "crab"); e != nil {
					t.Fatal(e)
				}
			}
			// What v0.7.69 left: the stage's fields gone, all three open.
			d, _, e := s.load(id)
			if e != nil {
				t.Fatal(e)
			}
			for _, k := range []string{"design_question", "design_for", "design_since", "design_asked_by", "asked_what", "asked_for", "asked_since", "asked_by"} {
				if e := d.RemoveField(k); e != nil {
					t.Fatal(e)
				}
			}
			if e := d.SetField("stage", map[string]string{"todo": "ready", "doing": "active"}[state]); e != nil {
				t.Fatal(e)
			}
			if _, e := s.finish(d, true); e != nil {
				t.Fatal(e)
			}
			if tk, _ := readTask(t, s, id); len(OpenQuestions(tk)) != 3 || ownHolds(tk, askStage(state)) {
				t.Fatalf("fixture: %d open, fields %v", len(OpenQuestions(tk)), tk.Fields)
			}
			if _, e := s.AnswerQuestion(id, 2, "two", "jhf"); e != nil {
				t.Fatalf("answer 2: %v", e)
			}
			tk, raw := readTask(t, s, id)
			if !strings.Contains(raw, "  second?\n  answered ") || len(OpenQuestions(tk)) != 2 {
				t.Fatalf("question 2 not answered:\n%s", raw)
			}
			what := map[string]string{"todo": "design_question", "doing": "asked_what"}[state]
			if tk.Fields[what] != "first?" || tk.Stage != askStage(state) {
				t.Fatalf("mirror not back on question 1 (stage %q, %s %q):\n%s", tk.Stage, what, tk.Fields[what], raw)
			}
			// A bare answer still needs the number while several are open.
			if _, e := s.Answer(id, "bare", "jhf"); e == nil || !strings.Contains(e.Error(), "say which one") {
				t.Fatalf("bare answer with 2 open: %v", e)
			}
		})
	}
}
