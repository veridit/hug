package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// yphbra check 2 (crab 21:59): a question asked with hug task ask on a
// todo task blocked by another shows in the asked person's ASKED, because
// the asker chose to ask now; a answers it there and the task stays todo.
// A hug task design question behind a dependency keeps the 11-17 rule: it
// waits for the dependency and is not in ASKED.
func TestAskedTodoQuestionInAsked(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := stageBoard(t)
	// cccccc is doing, so a task depending on it is blocked; eeeeee's
	// design question is outranked once it depends too.
	for id, dep := range map[string]string{"ffffff": "cccccc", "eeeeee": "cccccc"} {
		if _, err := m.store.AddDependencies(id, []string{dep}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.store.Ask("ffffff", task.Question{What: "Box at the bottom of the list, or in the page?", For: "jhf", Check: 2}, "piglet"); err != nil {
		t.Fatal(err)
	}
	m.opts.User = "jhf"
	m.refresh("")
	m.width, m.height = 100, 30
	m.resize()
	m.setView("overview")
	var asked []string
	for _, i := range m.paneIndices(sectionAsked) {
		asked = append(asked, m.visible[i].ID)
	}
	if strings.Join(asked, ",") != "ffffff" {
		t.Fatalf("ASKED holds %v, want only ffffff (eeeeee's design question waits on its dependency):\n%s", asked, ansi.Strip(m.View()))
	}
	m = openTask(t, m, "ffffff")
	page := ansi.Strip(m.detailText(mustSelected(t, m)))
	if !strings.Contains(page, "Box at the bottom of the list, or in the page?") || answerActions(page) != 1 {
		t.Fatalf("the question has no answer action:\n%s", page)
	}
	m.detailPane, m.acceptance = focusAcceptance, 1
	m = press(m, "a")
	for _, r := range "at the bottom" {
		m = press(m, string(r))
	}
	m = press(m, "enter")
	tk := mustSelected(t, m)
	b, _ := os.ReadFile(filepath.Join(m.store.Root, tk.Path))
	if tk.State != "todo" || len(task.OpenQuestions(tk)) != 0 || !strings.Contains(string(b), "    at the bottom\n") || !strings.Contains(string(b), "by jhf, for piglet") {
		t.Fatalf("a did not answer it on the todo task (state %s):\n%s", tk.State, b)
	}
}
