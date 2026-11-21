package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// ht3cdx check 2 from inside a task: a filter typed while a task is open
// closes it to the rows of the view it was opened over, the overview,
// not the list (Esc's level, as a view switch closed it before).
func TestFilterFromATaskKeepsTheView(t *testing.T) {
	m := fixture(t) // Alpha task (bug), Beta task (docs), both NEXT UP
	m.setView("overview")
	m = press(m, "enter")
	if !m.detail {
		t.Fatal("Enter did not open the task")
	}
	m = commandLine(m, "type bug")
	if m.view != "overview" || m.detail || m.filter.Type != "bug" {
		t.Fatalf("': type bug' in a task: view %s, detail %v, filter %q", m.view, m.detail, m.filter.Type)
	}
	screen := ansi.Strip(m.View())
	if first, _, _ := strings.Cut(screen, "\n"); !strings.Contains(first, "HUG · overview ") || strings.Contains(first, "›") {
		t.Fatalf("not the overview's rows:\n%s", screen)
	}
	if !strings.Contains(screen, "IN PROGRESS · assigned to  [0] · None with type:bug") {
		t.Fatalf("the empty section does not name the filter:\n%s", screen)
	}
}

// ht3cdx check 5 with mcvfbz: # opens a task the filter hides over the
// overview; Esc goes back to the overview, says why it has no row and
// selects no other.
func TestJumpToHiddenTaskKeepsTheView(t *testing.T) {
	m := fixture(t)
	m.setView("overview")
	m = commandLine(m, "type bug")
	beta := ""
	for _, v := range m.all {
		if v.Title == "Beta task" {
			beta = v.ID
		}
	}
	m = press(m, "#", beta, "enter")
	if !m.detail || m.view != "overview" {
		t.Fatalf("# %s: detail %v, view %s", beta, m.detail, m.view)
	}
	m = press(m, "esc")
	if m.view != "overview" || m.detail {
		t.Fatalf("Esc: view %s, detail %v", m.view, m.detail)
	}
	if sel, ok := m.selected(); ok {
		t.Fatalf("selected %s in place of the hidden %s", sel.ID, beta)
	}
	if want := beta + " is hidden by the filter type:bug: F clears it"; !strings.Contains(ansi.Strip(m.View()), want) {
		t.Fatalf("footer lacks %q\n%s", want, ansi.Strip(m.View()))
	}
}
