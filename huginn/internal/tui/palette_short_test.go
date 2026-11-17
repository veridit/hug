package tui

import (
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// After "f", a field can be shortened: "f g " lists the groups and
// "f s todo" filters by status (2026-11-16: "f g " said "No completions").
func TestPaletteShortFieldAfterFilter(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	if _, err := s.New(task.NewOptions{Title: "One", Goal: "g", Type: "spike", Acceptance: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 30
	m = press(m, ":", "f g ")
	cs := m.completions()
	if len(cs) == 0 {
		t.Fatal("f g + space: no completions")
	}
	for _, c := range cs {
		if !strings.HasPrefix(c.value, "filter group ") {
			t.Fatalf("f g + space offered %q, want only groups", c.value)
		}
	}
	for short, field := range map[string]string{"s": "state", "t": "type", "p": "priority", "g": "group", "l": "label", "e": "epic", "a": "assignee"} {
		if got := expandField(short); got != field {
			t.Fatalf("expandField(%q) = %q, want %q", short, got, field)
		}
	}
	if got := expandField("zz"); got != "zz" {
		t.Fatalf("an unknown name must stay as typed, got %q", got)
	}
	m = press(newModel(s, Options{User: "tester"}), ":", "f t spike", "enter")
	if m.prompt != "" || m.filter.Type != "spike" {
		t.Fatalf("f t spike did not filter: prompt=%q type=%q message=%q", m.prompt, m.filter.Type, m.message)
	}
}
