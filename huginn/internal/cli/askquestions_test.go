package cli

import (
	"strings"
	"testing"
)

// q2adr8 check 1 (Jørgen on rxr4z8: "If you ask three questions, I should
// have three questions to answer in the ticket"): two asks on one task keep
// both questions, numbered once in ## Questions; show lists them; the stage
// stays asked until both are answered. asked_* mirror the oldest open one.
func TestTwoAsksKeepBothQuestions(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Two", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b", "--accept", "c")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "Is the mark sign right?", "--as", "nautilus")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "3", "Keep folding on grouped headings?", "--as", "nautilus")
	f := taskFile(t, a, id)
	for _, want := range []string{
		"## Questions\n",
		" question 1 by nautilus to jhf (check 1):\n  Is the mark sign right?\n",
		" question 2 by nautilus to jhf (check 3):\n  Keep folding on grouped headings?\n",
		`asked_what: "Is the mark sign right?"`, `asked_check: "1"`,
	} {
		if !strings.Contains(f, want) {
			t.Fatalf("file lacks %q:\n%s", want, f)
		}
	}
	_, out := runText(a, "show", id)
	if !strings.Contains(out, "question 1 to jhf (check 1): Is the mark sign right?") || !strings.Contains(out, "question 2 to jhf (check 3): Keep folding on grouped headings?") {
		t.Fatalf("show does not list both questions:\n%s", out)
	}
	qs, _ := execute(t, a, "show", id)["task"].(Reply)["questions"].([]any)
	if len(qs) != 2 {
		t.Fatalf("show --json questions: %v", qs)
	}
	// Answering question 1 leaves question 2 open: still asked, and
	// asked_* now mirror question 2.
	execute(t, a, "answer", id, "1", "keep the diamond", "--as", "jhf")
	f = taskFile(t, a, id)
	if !strings.Contains(f, `stage: "asked"`) || !strings.Contains(f, `asked_what: "Keep folding on grouped headings?"`) || !strings.Contains(f, `asked_check: "3"`) {
		t.Fatalf("after answering 1 of 2, not asked on question 2:\n%s", f)
	}
	execute(t, a, "answer", id, "2", "yes, keep folding", "--as", "jhf")
	f = taskFile(t, a, id)
	if strings.Contains(f, `stage: "asked"`) || strings.Contains(f, "asked_what") {
		t.Fatalf("both answered, still asked:\n%s", f)
	}
	// A third ask after both are answered is question 3, never a reused 1.
	execute(t, a, "ask", id, "--to", "jhf", "Ship it today?", "--as", "nautilus")
	if f = taskFile(t, a, id); !strings.Contains(f, " question 3 by nautilus to jhf:\n  Ship it today?\n") {
		t.Fatalf("third question not numbered 3:\n%s", f)
	}
}

// q2adr8 check 2: hug task answer ID N answers question N only; its answer
// sits under it in ## Questions and History says "(question N)"; the
// asker sees which. Ticking the check a question is about still answers
// that question, and only that one.
func TestAnswerNAnswersQuestionN(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Two", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b", "--accept", "c")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "Is the mark sign right?", "--as", "nautilus")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "3", "Keep folding on grouped headings?", "--as", "nautilus")
	execute(t, a, "answer", id, "2", "yes, keep folding", "--as", "jhf")
	f := taskFile(t, a, id)
	for _, want := range []string{
		" question 2 by nautilus to jhf (check 3):\n  Keep folding on grouped headings?\n  answered ",
		" by jhf:\n    yes, keep folding\n",
		"answered asked (question 2, check 3), by jhf, for nautilus",
		`asked_what: "Is the mark sign right?"`,
	} {
		if !strings.Contains(f, want) {
			t.Fatalf("file lacks %q:\n%s", want, f)
		}
	}
	if strings.Contains(strings.SplitN(f, " question 2 ", 2)[0], "answered") {
		t.Fatalf("question 1 got an answer:\n%s", f)
	}
	ans, _ := execute(t, a, "show", id, "--as", "nautilus")["task"].(Reply)["answer"].(map[string]any)
	if ans["question_n"] != float64(2) && ans["question_n"] != 2 {
		t.Fatalf("the asker does not see which question was answered: %#v", ans)
	}
	// Answering it again, or a question that does not exist, is refused.
	if code, out := runText(a, "answer", id, "2", "again", "--as", "jhf"); code != 2 || !strings.Contains(out, "question 2 on "+id+" is already answered") {
		t.Fatalf("answer 2 twice: exit %d\n%s", code, out)
	}
	if code, out := runText(a, "answer", id, "7", "x", "--as", "jhf"); code != 2 || !strings.Contains(out, "no question 7") || !strings.Contains(out, "open: 1") {
		t.Fatalf("answer 7: exit %d\n%s", code, out)
	}
	// With open questions left, a bare answer must say which.
	if code, out := runText(a, "answer", id, "plain", "--as", "jhf"); code != 0 {
		t.Fatalf("one open question left, a bare answer answers it: exit %d\n%s", code, out)
	}
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "Diamond or star?", "--as", "nautilus")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "Folding too?", "--as", "nautilus")
	if code, out := runText(a, "answer", id, "plain", "--as", "jhf"); code != 2 || !strings.Contains(out, "2 questions are open") || !strings.Contains(out, "fix: hug task answer "+id+" 3 \"plain\"") {
		t.Fatalf("bare answer with 2 open: exit %d\n%s", code, out)
	}
	// The tick on check 2 answers question 4 (about check 2), not 3.
	a.Env["HUG_USER"] = "jhf"
	execute(t, a, "check", id, "2")
	a.Env["HUG_USER"] = "tester"
	f = taskFile(t, a, id)
	if !strings.Contains(f, "answered asked (question 4, check 2), by jhf, for nautilus") || !strings.Contains(f, `asked_what: "Diamond or star?"`) {
		t.Fatalf("tick on check 2 did not answer question 4 alone:\n%s", f)
	}
}
