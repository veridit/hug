package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/internal/cond"
)

// 16dxsr check 4 (TUI half): the detail view prints each condition with
// what hug saw, the list marks the task blocked while it does not hold,
// and an epic dependency says how many children are open.
func TestDetailShowsUnmetCondition(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	load := 140.0
	old := cond.LoadAverage
	cond.LoadAverage = func() (float64, bool) { return load, true }
	t.Cleanup(func() { cond.LoadAverage = old })
	s := &task.Store{Root: t.TempDir()}
	mk := func(title string) task.Task {
		r, err := s.New(task.NewOptions{Title: title, Goal: "G", Acceptance: []string{"C"}, Type: "chore"})
		if err != nil {
			t.Fatal(err)
		}
		return r.Task
	}
	heavy, epic, kid := mk("Heavy run"), mk("Epic"), mk("Kid")
	if _, err := s.Set(kid.ID, "parent", epic.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddDependencies(heavy.ID, []string{"when load below 120 wakes kestrel", epic.ID}); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 160, 50
	m.setView("list")
	var tk task.Task
	for _, x := range m.all {
		if x.ID == heavy.ID {
			tk = x
		}
	}
	if !m.blocked[tk.Path] {
		t.Fatal("waiting task not marked blocked")
	}
	got := ansi.Strip(m.detailText(tk))
	for _, want := range []string{
		"↳ " + epic.ID + " Epic [todo] BLOCKED (epic: 1 open)",
		"⧗ when load below 120 WAITING · load is 140.0 · wakes kestrel",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("detail missing %q:\n%s", want, got)
		}
	}
	load = 10
	if got := ansi.Strip(m.detailText(tk)); !strings.Contains(got, "⧗ when load below 120 met · load is 10.0 · wakes kestrel") {
		t.Fatalf("met:\n%s", got)
	}
}
