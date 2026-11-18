package cli

import (
	"strings"
	"testing"
)

// yphbra check 1 (piglet on ec6z6j, 2026-11-18: to ask Jørgen one design
// question on a todo task blocked by rxr4z8, take refused and ask refused):
// hug task ask on a todo task that depends on an unfinished task records
// the question in ## Questions, leaves the state todo, the assignee and
// the dependency as they were, and exits 0. Nobody takes it to ask.
func TestAskOnBlockedTodoTask(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	blocker := execute(t, a, "new", "Marks", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	id := execute(t, a, "new", "One request for the marked", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "mock reviewed", "--accept", "b", "--dep", blocker)["task"].(Reply)["id"].(string)
	execute(t, a, "take", blocker, "--as", "macaque", "--model", "m")
	before := taskFile(t, a, id)
	code, out := runText(a, "ask", id, "--to", "jhf", "--check", "1", "Box at the bottom of the list, or in the first marked task's page?", "--as", "piglet")
	if code != 0 {
		t.Fatalf("ask on a blocked todo task: exit %d\n%s", code, out)
	}
	f := taskFile(t, a, id)
	for _, want := range []string{
		"## Questions\n",
		" question 1 by piglet to jhf (check 1):\n  Box at the bottom of the list, or in the first marked task's page?\n",
		`state: "todo"`,
		`stage: "depends"`,
		`design_question: "Box at the bottom of the list, or in the first marked task's page?"`,
		`design_for: "jhf"`,
		`design_asked_by: "piglet"`,
	} {
		if !strings.Contains(f, want) {
			t.Fatalf("file lacks %q:\n%s", want, f)
		}
	}
	for _, keep := range []string{"assignee: []", blocker} {
		if strings.Contains(before, keep) && !strings.Contains(f, keep) {
			t.Fatalf("ask changed %q:\n%s", keep, f)
		}
	}
	if strings.Contains(f, "doing_since") || strings.Contains(f, "withdrawn") {
		t.Fatalf("ask took the task or withdrew its question:\n%s", f)
	}
	// A second question is question 2; the first stays open and mirrored.
	execute(t, a, "ask", id, "--to", "jhf", "Should t be offered with marks?", "--as", "piglet")
	f = taskFile(t, a, id)
	if !strings.Contains(f, " question 2 by piglet to jhf:\n  Should t be offered with marks?\n") || !strings.Contains(f, `design_question: "Box at the bottom`) || strings.Contains(f, "withdrawn") {
		t.Fatalf("second question:\n%s", f)
	}
}

// yphbra check 2: the person asked answers a todo task's question as they
// answer a doing one: answer N, or by ticking the check it is about; each
// answer goes under its question, the answer is for the asker (next
// --mine shows it to them first), and the mirror moves to the next open.
func TestAnswerOnTodoTaskGoesToAsker(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Todo", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "Diamond or star?", "--as", "piglet")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "Fold headings too?", "--as", "piglet")
	execute(t, a, "answer", id, "2", "yes, fold", "--as", "jhf")
	f := taskFile(t, a, id)
	for _, want := range []string{
		"  Fold headings too?\n  answered ",
		" by jhf:\n    yes, fold\n",
		"answered design (question 2, check 2), by jhf, for piglet",
		`design_question: "Diamond or star?"`,
		`stage: "design"`,
	} {
		if !strings.Contains(f, want) {
			t.Fatalf("file lacks %q:\n%s", want, f)
		}
	}
	ans, _ := execute(t, a, "show", id)["task"].(Reply)["answer"].(map[string]any)
	if ans["for"] != "piglet" || (ans["question_n"] != float64(2) && ans["question_n"] != 2) {
		t.Fatalf("the answer is not for the asker: %#v", ans)
	}
	// jhf ticks check 1: that answers question 1, the last open; the
	// design stage goes, the task is ready, still todo.
	a.Env["HUG_USER"] = "jhf"
	execute(t, a, "check", id, "1")
	a.Env["HUG_USER"] = "tester"
	f = taskFile(t, a, id)
	if !strings.Contains(f, "answered design (question 1, check 1), by jhf, for piglet") || strings.Contains(f, "design_question") || !strings.Contains(f, `state: "todo"`) || !strings.Contains(f, `stage: "ready"`) {
		t.Fatalf("tick did not answer question 1:\n%s", f)
	}
}

// yphbra, crab 21:59 ruling A: taking a todo task with an open question
// carries it over: the task is doing/asked, asked_* mirror the open
// question, and nothing is withdrawn (a question is never lost; the taker
// sees what he waits on).
func TestTakeCarriesOpenQuestion(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	blocker := execute(t, a, "new", "Marks", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	id := execute(t, a, "new", "One request for the marked", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "mock reviewed", "--dep", blocker)["task"].(Reply)["id"].(string)
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "A or B?", "--as", "piglet")
	execute(t, a, "dep", id, "--remove", blocker)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	f := taskFile(t, a, id)
	for _, want := range []string{`state: "doing"`, `stage: "asked"`, `asked_what: "A or B?"`, `asked_for: "jhf"`, `asked_check: "1"`, `asked_by: "piglet"`} {
		if !strings.Contains(f, want) {
			t.Fatalf("take lost the question: lacks %q:\n%s", want, f)
		}
	}
	if strings.Contains(f, "withdrawn") || strings.Contains(f, "design_question") {
		t.Fatalf("take withdrew the question or kept design:\n%s", f)
	}
	execute(t, a, "answer", id, "A", "--as", "jhf")
	if f = taskFile(t, a, id); !strings.Contains(f, `stage: "active"`) || !strings.Contains(f, "answered asked (check 1), by jhf, for piglet") {
		t.Fatalf("answer after take:\n%s", f)
	}
}

// yphbra (crab 21:59): hug task design keeps the 11-17 precedence, its
// question waits behind an open dependency; a hug task ask on a todo task
// is asked now. The help says which is which.
func TestAskHelpSaysTodoTasks(t *testing.T) {
	t.Parallel()
	a := app(t)
	_, out := runText(a, "ask", "--help")
	for _, want := range []string{"a todo task", "without taking it", "hug task design"} {
		if !strings.Contains(out, want) {
			t.Fatalf("ask help lacks %q:\n%s", want, out)
		}
	}
}
