package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// nch9d4 check 1 and 2: the sort picker offers rank, the default board
// order, as its own row; r from a multi-field sort clears every key and
// puts the rows back in rank order, at once, as c does.
func TestSortPickerRankRowRestoresBoardOrder(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	rank := strings.Join(listIDs(m), " ")
	sortedM := press(m, "s", "p", "t", "u", "enter")
	if len(sortedM.sort) != 3 || strings.Join(listIDs(sortedM), " ") == rank {
		t.Fatalf("setup: p t u did not sort: %v", sortedM.sort)
	}
	panel := ansi.Strip(press(sortedM, "s").View())
	// The highlight starts on r, the first row (17w9r6): "> r  rank".
	if !strings.Contains(panel, " r  rank      board order, the default: moves work only in it") {
		t.Fatalf("the sort picker has no rank row:\n%s", panel)
	}
	for _, way := range [][]string{{"s", "r"}, {"s", "n", "a", "r"}} {
		got := press(sortedM, way...)
		if got.prompt != "" || len(got.sort) != 0 || len(got.sortDraft) != 0 {
			t.Fatalf("%v: prompt %q sort %v draft %v", way, got.prompt, got.sort, got.sortDraft)
		}
		if ids := strings.Join(listIDs(got), " "); ids != rank {
			t.Fatalf("%v: rows are not in rank order:\n got %s\nwant %s", way, ids, rank)
		}
		if line := strings.Split(ansi.Strip(got.View()), "\n")[2]; !strings.Contains(line, "sort: rank") {
			t.Fatalf("%v: filter line %q", way, line)
		}
	}
}

// nch9d4 check 3: the picker names the order in force and the one being
// built apart, so an empty draft does not read as the current order.
func TestSortPickerShowsCurrentAndDraft(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := manyModel(t)
	lines := func(m model) []string { return strings.Split(ansi.Strip(m.sortText()), "\n") }
	fresh := lines(press(m, "s"))
	if fresh[1] != "now: rank (board order)" || fresh[2] != "new: nothing picked yet, Enter or r gives rank" {
		t.Fatalf("at rank: %q / %q", fresh[1], fresh[2])
	}
	applied := press(m, "s", "p", "enter")
	open := lines(press(applied, "s"))
	if open[1] != "now: priority ↓" || open[2] != "new: nothing picked yet, Enter or r gives rank" {
		t.Fatalf("sorted by priority: %q / %q", open[1], open[2])
	}
	for _, l := range open {
		if l == "sort: rank" {
			t.Fatalf("the empty draft still reads as the order: %q", l)
		}
	}
	built := lines(press(applied, "s", "t", "u"))
	if built[1] != "now: priority ↓" || built[2] != "new: type · updated ↓" {
		t.Fatalf("building t u: %q / %q", built[1], built[2])
	}
}

// nch9d4 (Jørgen 11:50Z): with no column marked the c row cannot be
// picked, so the whole row is dim, letter and name too, and c only says
// why; with a column marked the row reads as the others do.
func TestSortPickerColumnRowDimWithoutColumn(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	cRow := func(m model) string {
		for _, l := range strings.Split(m.sortText(), "\n") {
			if strings.Contains(ansi.Strip(l), "c  current column") {
				return l
			}
		}
		t.Fatalf("no c row:\n%s", ansi.Strip(m.sortText()))
		return ""
	}
	open := press(m, "s")
	if got, want := cRow(open), open.styles.dim.Render("  c  current column mark one with H/L first"); got != want {
		t.Fatalf("no column marked: the c row is not dim as a whole:\n got %q\nwant %q", got, want)
	}
	pressed := press(open, "c")
	if pressed.prompt != "sort" || len(pressed.sort) != 0 || len(pressed.sortDraft) != 0 || !strings.Contains(pressed.message, "mark one with H/L first") {
		t.Fatalf("c with no column: prompt %q sort %v draft %v message %q", pressed.prompt, pressed.sort, pressed.sortDraft, pressed.message)
	}
	marked := press(m, "L", "L", "L")
	col, _ := marked.currentColumn()
	row := cRow(press(marked, "s"))
	if !strings.HasPrefix(row, "  c  current column ("+col.name+") ") {
		t.Fatalf("column marked: the c row's letter and name are styled: %q", row)
	}
}
