package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// spareBoard is an overview with doing in-progress tasks and todo tasks
// next up and nothing else (z6kz20: Jørgen's 2 and 32).
func spareBoard(t *testing.T, doing, todo int) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	for i := 0; i < doing+todo; i++ {
		r, err := s.New(task.NewOptions{Title: fmt.Sprintf("Task %02d", i+1), Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}})
		if err != nil {
			t.Fatal(err)
		}
		if i < doing {
			path := filepath.Join(s.Root, r.Task.Path)
			b, _ := os.ReadFile(path)
			text := strings.Replace(string(b), "state: todo", "state: doing", 1)
			text = strings.Replace(text, "assignee: []", `assignee: ["ann"]`, 1)
			if err := os.WriteFile(path, []byte(text), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 140, 40
	m.resize()
	m.setView("overview")
	return m
}

var taskID = regexp.MustCompile(`(?m)^([>↑↓ ] [ ·!])[0-9a-z]{6}\b`) // the id leads the row (911j70)

// rowBudget is the overview's lines for rows: the body less a title and
// a blank per section with rows, a title alone per empty one (41hhgn).
func rowBudget(m model) int {
	b := m.bodyHeight()
	for p := range overviewSections {
		b-- // its titled rule
		if n := len(m.paneRows(p)); n > 0 && endRuleUnder(p, len(overviewSections), m.pane) {
			b-- // its end rule (v68n82)
		}
	}
	return b
}

// overviewBlank counts blank rows after the last drawn line of the body.
func overviewBlank(m model) int {
	lines := strings.Split(ansi.Strip(m.overviewText()), "\n")
	return m.bodyHeight() - len(lines)
}

// z6kz20 check 1: IN PROGRESS focused with 2 tasks and NEXT UP 32 on a
// 140x40 terminal: NEXT UP fills every spare row (golden).
func TestOverviewSpareRowsGolden(t *testing.T) {
	m := spareBoard(t, 2, 32)
	m.pane = sectionDoing
	m.clamp()
	rows := m.overviewRows()
	budget := rowBudget(m)
	if rows[sectionDoing] != 2 || rows[sectionNext] != budget-2 {
		t.Fatalf("rows %v, want IN PROGRESS 2 and NEXT UP %d", rows, budget-2)
	}
	// Ids are random per run: the golden holds ###### in their place.
	got := taskID.ReplaceAllString(ansi.Strip(m.overviewText()), "$1######")
	path := filepath.Join("testdata", "overview", "spare-rows-140x40.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestOverviewSpareRowsGolden ./huginn/internal/tui/", err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("overview differs from %s:\n%s", path, got)
	}
}

// z6kz20 check 2: with a small section focused, IN PROGRESS shows all its
// tasks before NEXT UP grows past 3.
func TestOverviewSpareRowsInProgressFirst(t *testing.T) {
	m := spareBoard(t, 12, 40)
	m.pane = sectionRecent // empty: the smallest there is
	m.clamp()
	budget := rowBudget(m)
	rows := m.overviewRows()
	if rows[sectionDoing] != 12 || rows[sectionNext] != budget-12 {
		t.Fatalf("rows %v, want IN PROGRESS 12 and NEXT UP %d", rows, budget-12)
	}
	// Not enough for IN PROGRESS: NEXT UP keeps its 3, no more.
	m = spareBoard(t, 40, 40)
	m.pane = sectionRecent
	m.clamp()
	rows = m.overviewRows()
	if rows[sectionNext] != 3 || rows[sectionDoing] != budget-3 {
		t.Fatalf("rows %v, want NEXT UP 3 and IN PROGRESS %d", rows, budget-3)
	}
	if shown := shownRows(m); shown[sectionDoing] != rows[sectionDoing] || shown[sectionNext] != 3 {
		t.Fatalf("drawn %v, rows %v", shown, rows)
	}
}

// z6kz20 check 3: no blank rows below the last section while any section
// has hidden rows, for every focus, on the seeded board and on Jørgen's.
func TestOverviewNoBlankWhileHidden(t *testing.T) {
	boards := map[string]model{"seeded": overviewAt(t, 140, 40), "2+32": spareBoard(t, 2, 32), "12+40": spareBoard(t, 12, 40)}
	for name, m := range boards {
		for _, size := range [][2]int{{140, 40}, {100, 30}, {140, 60}} {
			m.width, m.height = size[0], size[1]
			m.resize()
			for focus := range overviewSections {
				m.pane = focus
				m.clamp()
				rows := m.overviewRows()
				hidden := false
				for p := range overviewSections {
					if rows[p] < len(m.paneIndices(p)) {
						hidden = true
					}
				}
				// The last section's trailing blank line is the one kept.
				// Rows are whole, one or two lines (89y0b7): a blank line
				// is kept only when no hidden row fits in it.
				if blank := overviewBlank(m); hidden && blank > 0 && blank >= nextHidden(m, rows) {
					t.Fatalf("%s %dx%d focus %d: %d blank rows with rows hidden: %v", name, m.width, m.height, focus, blank, rows)
				}
				if blank := overviewBlank(m); blank < 0 {
					t.Fatalf("%s %dx%d focus %d: %d lines too many", name, m.width, m.height, focus, -blank)
				}
			}
		}
	}
}

// nextHidden is the fewest lines the next hidden row of any section takes
// (1 or 2), or 3 when no row is hidden.
func nextHidden(m model, rows []int) int {
	least := 3
	for p := range overviewSections {
		idx := m.paneIndices(p)
		cursor := m.memoCursor(p)
		if p == m.pane {
			cursor = m.cursor
		}
		start, end := m.window(idx, cursor, m.scrollOf(p), rows[p], m.selIn(p))
		if end < len(idx) {
			least = min(least, m.heightOf(idx[end], m.selIn(p)))
		}
		if start > 0 {
			least = min(least, m.heightOf(idx[start-1], m.selIn(p)))
		}
	}
	return least
}
