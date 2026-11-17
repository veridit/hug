package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// sfa8w4 check 3 (TUI): with a check focused, - asks for the reason and
// Enter strikes it; it is drawn crossed through with the decision; - again
// asks to restore it, Enter restores. Space on a struck check is refused.
func TestStrikeInTUI(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := fixture(t)
	m.width, m.height = 140, 40
	m = press(m, "enter") // open Alpha task, checks focused
	if m.detailPane != focusAcceptance {
		t.Fatal("checks not focused")
	}
	m = press(m, "j", "-")
	if m.prompt != strikePrompt {
		t.Fatalf("prompt %q", m.prompt)
	}
	// An empty reason strikes nothing.
	m = press(m, "enter")
	if !strings.Contains(m.message, "needs a reason") {
		t.Fatalf("empty reason: %q", m.message)
	}
	m = press(m, "-")
	for _, r := range "covered by the first" {
		m = press(m, string(r))
	}
	m = press(m, "enter")
	if !strings.HasPrefix(m.message, "strike ") {
		t.Fatalf("message %q", m.message)
	}
	var tk task.Task
	for _, x := range m.all {
		if x.Title == "Alpha task" {
			tk = x
		}
	}
	a := tk.Acceptance[1]
	if !a.Struck || a.Reason != "covered by the first" || a.StruckBy != "tester" {
		t.Fatalf("struck: %+v", a)
	}
	m = press(m, "k") // move off, so the line is drawn unselected
	view := ansi.Strip(m.detailText(tk))
	if !strings.Contains(view, "☒ 2. second check (not relevant: covered by the first, tester, "+a.StruckOn+")") {
		t.Fatalf("detail:\n%s", view)
	}
	m = press(m, "j", "-")
	if m.confirm != "-" || !strings.Contains(ansi.Strip(m.confirmLine()), "Restore struck check: ☒ 2. second check") {
		t.Fatalf("confirm %q: %s", m.confirm, ansi.Strip(m.confirmLine()))
	}
	m = press(m, "enter")
	for _, x := range m.all {
		if x.Title == "Alpha task" && x.Acceptance[1].Struck {
			t.Fatalf("not restored: %+v", x.Acceptance[1])
		}
	}
}
