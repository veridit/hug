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
		"↳ " + epic.ID + " Epic [todo] open (epic: 1 open)",
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

// A condition that does not hold yet shows the stage depends, its cell
// counted in the bar, and when it holds the row is ready again (6ephqk
// check 5 with 16dxsr's conditions).
func TestRowStageCountsConditions(t *testing.T) {
	load := 140.0
	old := cond.LoadAverage
	cond.LoadAverage = func() (float64, bool) { return load, true }
	t.Cleanup(func() { cond.LoadAverage = old })
	s := &task.Store{Root: t.TempDir()}
	r, err := s.New(task.NewOptions{Title: "Heavy run", Goal: "G", Acceptance: []string{"C"}, Type: "chore"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddDependencies(r.Task.ID, []string{"when load below 120"}); err != nil {
		t.Fatal(err)
	}
	row := func() (string, string) {
		m := newModel(s, Options{User: "tester"})
		for _, x := range m.all {
			if x.ID == r.Task.ID {
				return m.rowStage(x)
			}
		}
		t.Fatal("no task")
		return "", ""
	}
	if w, text := row(); w != "depends" || text != "▱ 0/1  Heavy run" {
		t.Fatalf("load 140: %q %q", w, text)
	}
	load = 80
	if w, text := row(); w != "ready" || text != "" {
		t.Fatalf("load 80: %q %q", w, text)
	}
}
