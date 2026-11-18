package cli

import (
	"strings"
	"testing"
)

// q2adr8 check 4 (Jørgen on rxr4z8, 2026-11-18 19:27Z: "If you ask three
// questions, I should have three questions to answer in the ticket"): an
// ask whose text is a numbered list of questions is refused with a fix that
// asks each on its own. One question that lists options, or a list of
// facts under one question, is still one question.
func TestAskRefusesNumberedQuestions(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Many", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	several := []string{
		"(1) Is the header right? (2) Should search take tag:? (3) Ship today?",
		"1. Is the header right?\n2. Should search take tag:?\n3. Ship today?",
		"Three things:\n1) Is the header right?\n2) Should search take tag:?",
		"Q1: is the header right?\nQ2: ship today?",
		// rxr4z8, verbatim: three choices behind one question mark.
		"Before I build marks: is footer-mock.md (in FILES) what you want? Three choices in it: (1) ◆ as the mark, or another sign; (2) after an action, unmark the tasks that went through and keep the refused ones marked, or keep every mark; (3) Space on a grouped heading keeps folding, Space on a task marks.",
	}
	for _, q := range several {
		code, out := runText(a, "ask", id, "--to", "jhf", q, "--as", "nautilus")
		if code != 2 || !strings.Contains(out, "fix: ask each question on its own: hug task ask "+id+" --to jhf \"") {
			t.Errorf("ask %q: exit %d, want 2 with the fix\n%s", q, code, out)
		}
		if strings.Contains(taskFile(t, a, id), "asked_what") {
			t.Fatalf("a refused ask wrote the question:\n%s", taskFile(t, a, id))
		}
	}
	one := []string{
		"Column order A or B? Four sheets attached: 1. overview 2. list 3. board 4. view line.",
		"Which rows go first: (1) the open ones, (2) the newest, or (3) by rank?",
		"Does check 2 hold? It was 3 of 4 on CI 44380717065.",
		"Does the fix hold? Steps I ran: 1. hug task ls 2. hug task next 3. hug tui",
	}
	for _, q := range one {
		if code, out := runText(a, "ask", id, "--to", "jhf", q, "--as", "nautilus"); code != 0 {
			t.Errorf("one question %q refused: exit %d\n%s", q, code, out)
		}
	}
}
