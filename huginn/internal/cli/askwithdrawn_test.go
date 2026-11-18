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
