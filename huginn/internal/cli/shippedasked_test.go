package cli

import (
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// questionCodes maps each task id to its question finding code (at3qep).
func questionCodes(t *testing.T, a *App) map[string]string {
	t.Helper()
	got := map[string]string{}
	for _, f := range execute(t, a, "lint")["findings"].([]task.Finding) {
		id, _ := task.IDOf(f.Path)
		if c := task.FindingCode(f.Problem); task.QuestionCode(c) {
			got[id] = c
		}
	}
	return got
}

// yf39ya check 7 (txjkrm 2026-11-20: todo, question 1 to jhf open since
// 11-18, shipped in v0.7.71, still shipped-not-done on Jørgen's board): a
// task with an open question to a person never shows shipped-not-done,
// todo or doing. Its question says why it is not done. Once the question
// is answered, the finding is back.
func TestShippedWithOpenQuestionIsNotBroken(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no", "--name", "Jørgen H. Fjeld")
	todo := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q",
		"--accept", "built", "--accept", "Jørgen confirms the tags read right")["task"].(Reply)["id"].(string)
	git(t, a.Cwd, "tag", "v0.1.0")
	deduced(t, a)
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Tags ("+todo+")")
	git(t, a.Cwd, "tag", "v0.1.1")
	deduced(t, a)
	if got := questionCodes(t, a); got[todo] != "shipped-not-done" {
		t.Fatalf("before the ask: %v", got)
	}
	execute(t, a, "ask", todo, "--to", "jhf", "--check", "2", "Do the tags read right?", "--as", "crab")
	if tk := stateOf(t, a, todo); tk.State != "todo" || len(task.OpenQuestions(tk)) != 1 {
		t.Fatalf("after the ask: %s/%s\n%s", tk.State, tk.Stage, taskFile(t, a, todo))
	}
	if got := questionCodes(t, a); got[todo] != "" {
		t.Fatalf("%s has a question open to jhf, but lint says %s:\n%s", todo, got[todo], taskFile(t, a, todo))
	}
	execute(t, a, "answer", todo, "they do, but check 1 is not built", "--as", "jhf")
	if got := questionCodes(t, a); got[todo] != "shipped-not-done" {
		t.Fatalf("after the answer: %v\n%s", got, taskFile(t, a, todo))
	}
	if f := taskFile(t, a, todo); !strings.Contains(f, "answered") {
		t.Fatalf("no answer:\n%s", f)
	}
}
