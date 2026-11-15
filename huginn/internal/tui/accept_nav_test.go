package tui

import (
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"

	"github.com/charmbracelet/x/ansi"
)

// In detail, ] and [ move through every acceptance item, and the pointer
// follows (2026-11-15: user could only reach item 1).
func TestDetailAcceptanceNavigation(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	// Long items wrap; the first is checked, as in the user's screenshot.
	long := strings.Repeat("word ", 40)
	if _, err := s.New(task.NewOptions{Title: "Layout", Goal: "g", Acceptance: []string{"one " + long, "two " + long, "three", "four " + long, "five", "six " + long}}); err != nil {
		t.Fatal(err)
	}
	ts, _ := s.Tasks()
	if _, err := s.Check(ts[0].ID, 1); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.setView("list")
	m = press(m, "enter")
	for want := 0; want < 3; want++ {
		if m.acceptance != want {
			t.Fatalf("after %d presses of ]: acceptance=%d", want, m.acceptance)
		}
		tk, _ := m.selected()
		text := ansi.Strip(m.detailText(tk))
		if !strings.Contains(text, "> ☐ "+itoa(want+1)+".") && !strings.Contains(text, "> ☑ "+itoa(want+1)+".") {
			t.Fatalf("pointer not on item %d (detail=%v acceptance=%d):\n%s", want+1, m.detail, m.acceptance, text)
		}
		m = press(m, "]")
	}
	m = press(m, "[")
	if m.acceptance < 1 {
		t.Fatalf("[ did not move back: %d", m.acceptance)
	}
}

func itoa(n int) string { return string(rune('0' + n)) }
