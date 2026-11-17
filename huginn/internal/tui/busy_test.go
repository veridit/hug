package tui

import (
	"strings"
	"testing"
)

// n2hq8p: t on a second task while the TUI user has an active one is
// refused like the CLI's take; the message names the active task and the
// fixes, and the second task stays todo.
func TestTakeSecondTaskRefusedInTUI(t *testing.T) {
	m := fixture(t)
	first, second := m.visible[0].ID, m.visible[1].ID
	if _, e := m.store.Take(first, "tester"); e != nil {
		t.Fatal(e)
	}
	m.refresh("")
	m.setView("list")
	for i, v := range m.visible {
		if v.ID == second {
			m.cursor = i
		}
	}
	m = act(m, "t")
	for _, want := range []string{"busy " + second, "tester is on " + first, "fix: hug task done " + first, "fix: hug task wait " + first} {
		if !strings.Contains(m.message, want) {
			t.Fatalf("message misses %q:\n%s", want, m.message)
		}
	}
	for _, v := range m.all {
		if v.ID == second && v.Status != "todo" {
			t.Fatalf("refused task is %s", v.Status)
		}
	}
}
