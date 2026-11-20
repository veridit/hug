package cli

import (
	"context"
	"strings"
	"testing"
)

// 72n8m4 check 1 (crab 2026-11-20: done on ddpxs8, every check ticked and
// a question open to jhf, went through and withdrew the question,
// "doing/asked -> done"): hug task done on a task with an open question to
// a person exits non-zero with a fix: line and leaves the question open.
// Withdrawing it stays its own act (hug task resume); after that, done
// goes through.
func TestDoneRefusesOpenQuestion(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no", "--name", "Jørgen H. Fjeld")
	id := newTicket(t, a, "Release ticket")
	execute(t, a, "take", id, "--as", "crab", "--model", "m")
	execute(t, a, "check", id, "1")
	execute(t, a, "ask", id, "--to", "jhf", "Does the release read right on your board?", "--as", "crab")
	_, e := a.Execute(context.Background(), []string{"done", id})
	if e == nil {
		t.Fatalf("done went through with a question open to jhf:\n%s", taskFile(t, a, id))
	}
	code, problem, fix := ErrorDetails(e)
	if code == 0 || fix != "hug task resume "+id || !strings.Contains(problem, "question 1") || !strings.Contains(problem, "jhf") {
		t.Fatalf("done refusal: code %d problem %q fix %q", code, problem, fix)
	}
	if tk := stateOf(t, a, id); tk.State != "doing" || tk.Stage != "asked" {
		t.Fatalf("after the refusal: %s/%s", tk.State, tk.Stage)
	}
	if f := taskFile(t, a, id); strings.Contains(f, "withdrawn") {
		t.Fatalf("the refusal withdrew the question:\n%s", f)
	}
	// Withdrawn on purpose, done goes through.
	execute(t, a, "resume", id)
	execute(t, a, "done", id)
	if tk := stateOf(t, a, id); tk.State != "done" {
		t.Fatalf("after resume and done: %s", tk.State)
	}
}
