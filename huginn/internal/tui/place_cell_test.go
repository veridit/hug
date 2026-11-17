package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// kx2ndm check 1: list and overview rows show each task's group and epic,
// "later·↳EPIC", before the title, without a line of their own.
func TestRowsShowGroupAndEpic(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	epic, _ := s.New(task.NewOptions{Title: "Launch", Goal: "g", Acceptance: []string{"a"}})
	child, _ := s.New(task.NewOptions{Title: "Launch posts", Goal: "g", Acceptance: []string{"a"}, Group: "later"})
	plain, _ := s.New(task.NewOptions{Title: "Plain", Goal: "g", Acceptance: []string{"a"}})
	if _, err := s.Set(child.Task.ID, "parent", epic.Task.ID); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	for _, view := range []string{"overview", "list"} {
		for _, w := range []int{80, 140} {
			m.width, m.height = w, 30
			m.resize()
			m.setView(view)
			screen := ansi.Strip(m.View())
			if n := len(strings.Split(screen, "\n")); n != 30 {
				t.Fatalf("%s at %d: %d lines", view, w, n)
			}
			want := "later·↳" + epic.Task.ID + " Launch posts"
			if !strings.Contains(screen, want) {
				t.Fatalf("%s at %d lacks %q:\n%s", view, w, want, screen)
			}
			for _, line := range strings.Split(screen, "\n") {
				if strings.Contains(line, plain.Task.ID) && strings.Contains(line, "↳") {
					t.Fatalf("%s at %d: a task without epic shows one: %q", view, w, line)
				}
			}
		}
	}
	if got := m.placeCell(m.all[0]); strings.Contains(got, "·") && m.all[0].Group == "" {
		t.Fatalf("placeCell with no group: %q", got)
	}
}
