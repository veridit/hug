package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// q2adr8 check 3: a task asking jhf three questions shows each open one
// with its own answer action: two that name no check as blocks at the
// top, each its own stop with its own "a answer", and one on check 2
// under that check. a on the second block answers question 2 alone; the
// others stay, each still with its a. Golden at 100x24.
func TestSeveralQuestionsEachAnswered(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := stageBoard(t)
	m.opts.User = "jhf"
	for _, q := range []task.Question{
		{What: "Is the mark sign right?", For: "jhf"},
		{What: "Keep folding on grouped headings?", For: "jhf"},
		{What: "Ship it today?", For: "jhf", Check: 2},
	} {
		if _, err := m.store.Ask("aaaaaa", q, "nautilus"); err != nil {
			t.Fatal(err)
		}
	}
	m.refresh("")
	m.width, m.height = 100, 24
	m.resize()
	m = openTask(t, m, "aaaaaa")
	screen := ansi.Strip(m.View())
	// The whole page: the third question sits under check 2, below the
	// fold at 24 lines.
	page := ansi.Strip(m.detailText(mustSelected(t, m)))
	for _, want := range []string{
		"> ? Is the mark sign right?",
		"└ question 1 · nautilus",
		"  ? Keep folding on grouped headings?",
		"└ question 2 · nautilus",
		"? ☐ 2. b",
		"Ship it today?",
		"└ question 3 · nautilus",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("lacks %q:\n%s", want, page)
		}
	}
	if n := answerActions(page); n != 3 {
		t.Fatalf("%d answer actions for 3 questions:\n%s", n, page)
	}
	golden := []string{"== opened", screen}
	// The next stop is question 2's block; a answers it alone.
	m = press(m, "tab")
	if m.detailPane != focusQuestion || m.question != 1 {
		t.Fatalf("tab: pane %d question %d, want question block 2", m.detailPane, m.question)
	}
	m = press(m, "a")
	for _, r := range "yes, keep folding" {
		m = press(m, string(r))
	}
	typing := ansi.Strip(m.View())
	if q2 := strings.Index(typing, "Keep folding"); q2 < 0 || !strings.Contains(typing[q2:], "answer: yes, keep folding") || strings.Contains(typing[:q2], "answer: yes") || strings.Count(typing, "answer: ") != 1 {
		t.Fatalf("box not under question 2:\n%s", typing)
	}
	golden = append(golden, "== typing under question 2", typing)
	m = press(m, "enter")
	tk := mustSelected(t, m)
	b, _ := os.ReadFile(filepath.Join(m.store.Root, tk.Path))
	if !strings.Contains(string(b), "  Keep folding on grouped headings?\n  answered ") || !strings.Contains(string(b), "    yes, keep folding\n") || tk.Stage != "asked" {
		t.Fatalf("question 2 not answered alone (stage %q):\n%s", tk.Stage, b)
	}
	if open := task.OpenQuestions(tk); len(open) != 2 || open[0].N != 1 || open[1].N != 3 {
		t.Fatalf("open after answering 2: %+v", open)
	}
	after := ansi.Strip(m.View())
	// The answered question goes to the Questions record, a long section
	// folded to QUESTIONS (1) until opened (8a0ybb).
	if afterPage := ansi.Strip(m.withUnfolded("questions").detailText(tk)); strings.Contains(afterPage, "? Keep folding") || !strings.Contains(afterPage, "└ jhf: yes, keep folding") || !strings.Contains(afterPage, "Is the mark sign right?") || !strings.Contains(afterPage, "Ship it today?") || answerActions(afterPage) != 2 {
		t.Fatalf("after answering 2:\n%s", afterPage)
	}
	if !strings.Contains(after, "> ? Is the mark sign right?") || !strings.Contains(after, "✓ jhf answered question 2: yes, keep folding") {
		t.Fatalf("after answering 2, the > is not on question 1 or the header does not say question 2:\n%s", after)
	}
	golden = append(golden, "== question 2 answered", after)
	goldenFile(t, "testdata/detail/several-questions-100.txt", strings.Join(golden, "\n")+"\n")
}

// answerActions counts the lines ending in "· answer": one per question
// asked of you, each its own a.
func answerActions(page string) int {
	return len(regexp.MustCompile(`(?m)· answer *$`).FindAllString(page, -1))
}
