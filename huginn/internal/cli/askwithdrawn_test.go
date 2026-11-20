package cli

import (
	"strings"
	"testing"
)

// nanvnt check 1 (hpqt99 2026-11-18: question 1 withdrawn for a wrong
// claim still read as open in show): hug task show prints a withdrawn
// question with "withdrawn <at> (<why>)" under it, and only the one asked
// again reads as open.
func TestShowMarksWithdrawnQuestion(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Imported data", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "piglet", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "Is 2kb574 a leak?", "--as", "piglet")
	execute(t, a, "resume", id)
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "Approve 1B-amended plus 2B?", "--as", "piglet")
	_, out := runText(a, "show", id)
	q1 := strings.Index(out, "question 1 to jhf (check 1): Is 2kb574 a leak?")
	q2 := strings.Index(out, "question 2 to jhf (check 1): Approve 1B-amended plus 2B?")
	if q1 < 0 || q2 < 0 {
		t.Fatalf("show lacks a question:\n%s", out)
	}
	between := out[q1:q2]
	if !strings.Contains(between, "\n  withdrawn ") || !strings.Contains(between, "(doing/asked -> doing/active)") {
		t.Fatalf("question 1 is withdrawn but show does not say so:\n%s", out)
	}
	if strings.Count(out, "withdrawn ") != 1 {
		t.Fatalf("only question 1 is withdrawn:\n%s", out)
	}
}

// 5vehza check 1 (hpqt99 2026-11-20: hug task answer on a withdrawn
// question panicked on its nil answer in pickQuestion): answering a
// withdrawn question is refused with exit 2, says when and why it was
// withdrawn, and ends with a fix: line to ask it again. Nothing is written.
func TestAnswerWithdrawnQuestionRefused(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Imported data", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "piglet", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "Is 2kb574 a leak?", "--as", "piglet")
	execute(t, a, "resume", id)
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "Approve 1B-amended plus 2B?", "--as", "piglet")
	before := taskFile(t, a, id)
	code, out := runText(a, "answer", id, "1", "yes", "--as", "jhf")
	if code != 2 || !strings.Contains(out, "question 1 on "+id+" was withdrawn at ") || !strings.Contains(out, " (doing/asked -> doing/active)") {
		t.Fatalf("exit %d, want 2 and question 1 was withdrawn at ... (why):\n%s", code, out)
	}
	if want := "fix: hug task ask " + id + " --to jhf --check 1 \"Is 2kb574 a leak?\"\n"; !strings.HasSuffix(out, want) {
		t.Fatalf("want it to end %q:\n%s", want, out)
	}
	if after := taskFile(t, a, id); after != before {
		t.Fatalf("a refused answer wrote the task:\n%s", after)
	}
	// Question 2 is still open and answers as before.
	if code, out := runText(a, "answer", id, "2", "yes", "--as", "jhf"); code != 0 {
		t.Fatalf("answer 2: exit %d\n%s", code, out)
	}
}
