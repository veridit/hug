package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// searchBoard is 60 tasks; "Zebra" is the 50th, below the first screen of
// every view at 140x40.
func searchBoard(t *testing.T) (model, map[string]string, *int) {
	t.Helper()
	var titles []string
	for i := 1; i <= 60; i++ {
		title := fmt.Sprintf("Task %02d", i)
		if i == 50 {
			title = "Zebra"
		}
		titles = append(titles, title)
	}
	return moveBoard(t, titles...)
}

// mcvfbz check 1: search, then leave it: the found task is selected in the
// full view and on screen, in the overview, the list and the board. Since
// 5cjy88 Enter shows the results and Esc on them leaves (Esc while typing
// cancels, TestSearchEscStepsBack); a view switch from the results leaves
// too, keeping the task.
func TestLeavingSearchKeepsFoundTask(t *testing.T) {
	for _, view := range []string{"overview", "list", "board"} {
		for _, leave := range []string{"enter esc", "ctrl+n ctrl+p enter esc"} {
			t.Run(view+" "+leave, func(t *testing.T) {
				m, ids, _ := searchBoard(t)
				zebra := ids["Zebra"]
				m.setView(view)
				if view == "overview" {
					m = nextUp(m)
				}
				m.move(0, true)
				if strings.Contains(ansi.Strip(m.View()), zebra) {
					t.Fatal("Zebra is on the first screen; the test needs it below", m.View())
				}
				m = press(m, "/", "Zebra")
				if sel, ok := m.selected(); !ok || sel.ID != zebra {
					t.Fatalf("while typing: selected %v", sel.ID)
				}
				m = press(m, strings.Fields(leave)...)
				sel, ok := m.selected()
				if m.prompt != "" || m.query != "" || len(m.visible) != 60 {
					t.Fatalf("search not left: prompt %q query %q, %d shown", m.prompt, m.query, len(m.visible))
				}
				if !ok || sel.ID != zebra {
					t.Fatalf("selected %q, want %s (Zebra)", sel.ID, zebra)
				}
				screen := ansi.Strip(m.View())
				if !strings.Contains(screen, zebra) {
					t.Fatal("Zebra selected but not on screen\n" + screen)
				}
				if !strings.Contains(screen, "found "+zebra+", row 50 of 60") {
					t.Fatal("footer does not say where it is\n" + screen)
				}
			})
		}
	}
}

// ↑↓ pick another match while typing; leaving keeps the one picked
// (Enter, then Esc on the results, 5cjy88).
func TestSearchPickKeepsPick(t *testing.T) {
	m, ids, _ := searchBoard(t)
	m.setView("list")
	m = press(m, "/", "Task 1", "down", "down", "enter", "esc")
	if sel, _ := m.selected(); sel.ID != ids["Task 12"] {
		t.Fatalf("selected %s, want Task 12 (%s)", sel.Title, ids["Task 12"])
	}
}

// mcvfbz check 2: search, Enter on the match, then m (grab) and J: the
// found task moves, among the full list's rows.
func TestFoundTaskMovesAtOnce(t *testing.T) {
	m, ids, commits := searchBoard(t)
	m.setView("list")
	m = press(m, "/", "Zebra", "enter", "m", "j", "enter")
	if want := "moved " + ids["Zebra"] + " below " + ids["Task 51"] + ", row 51 of 60 in the list"; m.message != want || *commits != 1 {
		t.Fatalf("m j Enter: %q (commits %d), want %q", m.message, *commits, want)
	}
	m = press(m, "/", "Zebra", "enter", "m", "k", "k", "enter")
	if *commits != 2 || !strings.Contains(m.message, ids["Zebra"]) {
		t.Fatalf("grab: %q (commits %d)", m.message, *commits)
	}
	if got := shown(m)[48]; got != ids["Zebra"] {
		t.Fatalf("row 49 is %s, want Zebra %s: %v", got, ids["Zebra"], shown(m))
	}
	// In NEXT UP too, where the move is relative to that section's rows;
	// Esc on the results first (5cjy88), then m.
	m = press(nextUp(m), "/", "Zebra", "enter", "esc", "m", "k", "enter")
	if *commits != 3 || !strings.HasPrefix(m.message, "moved "+ids["Zebra"]+" above ") {
		t.Fatalf("NEXT UP K: %q (commits %d)", m.message, *commits)
	}
}

// With a sort other than rank, the footer says moves need rank order.
func TestFoundTaskSortedSaysSortRank(t *testing.T) {
	m, ids, _ := searchBoard(t)
	m.setView("list")
	m = press(sorted(m, "t"), "enter")
	m = press(m, "/", "Zebra", "enter", "esc") // leaving the results says where it is (5cjy88)
	if sel, _ := m.selected(); sel.ID != ids["Zebra"] || !strings.Contains(m.message, sortRefusal) {
		t.Fatalf("selected %s, footer %q", sel.ID, m.message)
	}
}

// mcvfbz check 3: a found task the active filter hides (opened with #,
// which finds every task) is not swapped for another: the footer says so
// and no row is selected until j or k.
func TestFoundTaskHiddenByFilterSaysSo(t *testing.T) {
	m := fixture(t) // Alpha task (bug), Beta task (docs)
	m = commandLine(m, "type bug")
	beta := ""
	for _, v := range m.all {
		if v.Title == "Beta task" {
			beta = v.ID
		}
	}
	m = press(m, "#", beta, "enter")
	if !m.detail || m.detailPath == "" {
		t.Fatal("# did not open Beta", m.View())
	}
	m = press(m, "esc")
	if _, ok := m.selected(); ok {
		sel, _ := m.selected()
		t.Fatalf("selected %s instead of nothing", sel.Title)
	}
	screen := ansi.Strip(m.View())
	if want := beta + " is hidden by the filter type:bug: F clears it"; !strings.Contains(screen, want) {
		t.Fatalf("footer lacks %q\n%s", want, screen)
	}
	// Nothing else is marked selected on screen.
	if strings.Contains(screen, "> ! ") || strings.Contains(screen, ">  ") {
		t.Fatal("a row is drawn selected\n" + screen)
	}
	// Clearing the filter brings Beta back, selected.
	m = press(m, "F")
	if sel, ok := m.selected(); !ok || sel.ID != beta {
		t.Fatalf("after F: %v %v", sel.ID, ok)
	}
	// From nothing selected, j selects the first row.
	m = commandLine(m, "type bug")
	m = press(m, "#", beta, "enter", "esc", "j")
	if sel, ok := m.selected(); !ok || sel.Title != "Alpha task" {
		t.Fatalf("j from no selection: %v", sel.Title)
	}
}

// A kept search (Tab) cleared with F keeps the task too. Esc does not
// clear it (5cjy88, crab 09:16Z: a chip lasts until F or ⌫).
func TestKeptSearchClearedKeepsTask(t *testing.T) {
	m, ids, _ := searchBoard(t)
	m.setView("list")
	m = press(m, "/", "Zebra", "tab")
	if m.query != "Zebra" || len(m.visible) != 1 {
		t.Fatalf("Tab did not keep the search: %q %d", m.query, len(m.visible))
	}
	m = press(m, "esc")
	if m.query != "Zebra" {
		t.Fatalf("Esc cleared the kept search")
	}
	m.setView("list")
	m.follow(ids["Zebra"])
	m = press(m, "F")
	if sel, _ := m.selected(); m.query != "" || sel.ID != ids["Zebra"] {
		t.Fatalf("F: query %q, selected %s", m.query, sel.ID)
	}
}

// "/" then Enter with nothing typed searched nothing, so it says nothing
// and leaves the cursor where it was (Jørgen 2026-11-17 23:56: it said
// "found kx2ndm, row 1 of 22 in WAITING").
func TestEmptySearchSaysNothing(t *testing.T) {
	for _, leave := range []string{"enter", "esc", "tab"} {
		t.Run(leave, func(t *testing.T) {
			m, _, _ := searchBoard(t)
			m.setView("list")
			m = press(m, "j", "j")
			before, _ := m.selected()
			m.message = ""
			m = press(m, "/", leave)
			after, _ := m.selected()
			if m.prompt != "" || m.query != "" {
				t.Fatalf("search not left: prompt %q query %q", m.prompt, m.query)
			}
			if after.ID != before.ID {
				t.Fatalf("selected %s, want %s", after.ID, before.ID)
			}
			if strings.Contains(m.message, "found") {
				t.Fatalf("empty search says %q", m.message)
			}
		})
	}
}
