package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"

	"github.com/charmbracelet/x/ansi"
)

// From a task's detail, n and p open the next and previous task of the list
// it was opened from, and stop at either end (2026-11-16, user: { and } are
// hard to reach on a Norwegian keyboard).
func TestDetailStepsToNextAndPreviousTask(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	for _, title := range []string{"Alpha", "Bravo", "Charlie"} {
		if _, err := s.New(task.NewOptions{Title: title, Goal: "g", Acceptance: []string{"a"}}); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 30
	m.setView("list")
	m = press(m, "enter")
	title := func() string { tk, _ := m.selected(); return tk.Title }
	first := title()
	if !strings.Contains(ansi.Strip(m.View()), "Task 1 of 3") {
		t.Fatalf("footer lacks position:\n%s", ansi.Strip(m.View()))
	}
	m = press(m, "n")
	second := title()
	if second == first || !m.detail {
		t.Fatalf("n did not step: %q -> %q detail=%v", first, second, m.detail)
	}
	m = press(m, "n")
	m = press(m, "ctrl+n")
	if !strings.Contains(ansi.Strip(m.View()), "Task 3 of 3") {
		t.Fatalf("did not stop at the end:\n%s", ansi.Strip(m.View()))
	}
	m = press(m, "p")
	if title() != second {
		t.Fatalf("p went to %q, want %q", title(), second)
	}
	m = press(m, "esc")
	if tk, _ := m.selected(); m.detail || tk.Title != second {
		t.Fatalf("Esc should land on the task last shown: detail=%v %q", m.detail, tk.Title)
	}
}

// Board rows put the lint mark in the gutter and type and priority in one
// letter each, with a legend that spells them out.
func TestBoardRowsAreCompactWithLegend(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	if _, err := s.New(task.NewOptions{Title: "A reasonably long task title here", Goal: "g", Type: "spike", Priority: "high", Acceptance: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 120, 30
	m.setView("board")
	text := ansi.Strip(m.boardText())
	ts, _ := s.Tasks()
	if !regexp.MustCompile(ts[0].ID + `   A reasonably l… ○ 1   S H`).MatchString(text) { // 911j70: ID TITLE ✓ T P
		t.Fatalf("row not compact:\n%s", text)
	}
	if !strings.Contains(text, "S spike") || !strings.Contains(text, "H high") {
		t.Fatalf("legend missing:\n%s", text)
	}
}

// The palette is a tree: filters sit under "filter", then field, then value,
// and the list uses the screen's height (2026-11-16, user).
func TestPaletteIsATreeAndUsesTheScreen(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	if _, err := s.New(task.NewOptions{Title: "One", Goal: "g", Type: "spike", Acceptance: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 40
	m.setView("list")
	m = press(m, ":")
	top := ansi.Strip(m.paletteText())
	if strings.Contains(top, "Filter todo") || !strings.Contains(top, "filter ›") {
		t.Fatalf("top level should hold one filter branch:\n%s", top)
	}
	m.input.SetValue("filter ")
	if got := len(m.completions()); got != len(paletteFields) {
		t.Fatalf("filter level: %d choices, want %d", got, len(paletteFields))
	}
	m.input.SetValue("filter type ")
	text := ansi.Strip(m.paletteText())
	if !strings.Contains(text, "filter type spike") || !strings.Contains(text, "1 tasks") {
		t.Fatalf("value level lacks spike and its count:\n%s", text)
	}
	if n := strings.Count(text, "filter type "); n < 5 {
		t.Fatalf("only %d of the type values shown on a 40-line screen:\n%s", n, text)
	}
	m.input.SetValue("filter type spike")
	if !m.commandReady() || !m.executeCommand() || len(m.visible) != 1 {
		t.Fatalf("filter type spike did not run: visible=%d msg=%q", len(m.visible), m.message)
	}
	// Short forms still run when typed, and a field name finds its branch.
	m.input.SetValue("state todo")
	if !m.commandReady() {
		t.Fatal("state todo should still run")
	}
	m.input.SetValue("pri")
	if cs := m.completions(); len(cs) != 1 || cs[0].value != "filter priority " {
		t.Fatalf("pri should find filter priority: %+v", cs)
	}
}

// ": f done" and ": filter done" find the status without naming the field.
func TestPaletteFindsValueWithoutField(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	if _, err := s.New(task.NewOptions{Title: "One", Goal: "g", Acceptance: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	for _, typed := range []string{"f done", "filter done"} {
		m := newModel(s, Options{User: "tester"})
		m.width, m.height = 100, 30
		m = press(m, ":", typed)
		cs := m.completions()
		if len(cs) != 1 || cs[0].value != "filter state done" {
			t.Fatalf("%q: %+v", typed, cs)
		}
		m = press(m, "enter", "enter")
		if m.prompt != "" || m.filter.State != "done" {
			t.Fatalf("%q did not filter: prompt=%q status=%q", typed, m.prompt, m.filter.State)
		}
	}
}

// The tick prompt quotes the check as the ticket shows it, after a colon.
func TestTickPromptQuotesTheCheckAsShown(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	if _, err := s.New(task.NewOptions{Title: "One", Goal: "g", Acceptance: []string{"posts drafted for user review"}}); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 30
	m.setView("list")
	m = press(m, "enter", " ")
	if view := ansi.Strip(m.View()); !strings.Contains(view, "Tick: ☐ 1. posts drafted for user review") {
		t.Fatalf("prompt:\n%s", view)
	}
	m = press(m, "enter", " ")
	if view := ansi.Strip(m.View()); !strings.Contains(view, "Untick: ☑ 1. posts drafted for user review") {
		t.Fatalf("prompt:\n%s", view)
	}
}
