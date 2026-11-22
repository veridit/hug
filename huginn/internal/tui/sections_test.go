package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// overviewAt is the 60-task fixture in the overview at width x height.
func overviewAt(t *testing.T, width, height int) model {
	t.Helper()
	m := manyModel(t)
	m.width, m.height = width, height
	m.resize()
	m.setView("overview")
	return m
}

// shownRows counts the lines drawn under each section heading: a row
// and, for one that waits, its second line (89y0b7). The heading is on
// the section's titled rule, and its end rule is no row (v68n82).
func shownRows(m model) map[int]int {
	out := map[int]int{}
	p := -1
	for _, line := range strings.Split(ansi.Strip(m.overviewText()), "\n") {
		if endRule.MatchString(strings.TrimRight(line, " ")) {
			continue
		}
		trimmed := strings.TrimLeft(line, "↑↓─ ")
		for q, name := range overviewSections {
			if strings.HasPrefix(trimmed, name) {
				p = q
				out[p] = 0
			}
		}
		if p >= 0 && strings.TrimSpace(line) != "" && !strings.Contains(line, overviewSections[p]) {
			out[p]++
		}
	}
	return out
}

// h9zghj check 1: on a 40-row terminal the focused section takes every
// spare row; each other non-empty section shows at least min(count, 3), and
// rows the focused one does not need go out in spareOrder (z6kz20).
func TestOverviewFocusedSectionGrows(t *testing.T) {
	m := overviewAt(t, 140, 40)
	for _, focus := range []int{sectionDoing, sectionNext, sectionRecent} {
		m.pane = focus
		m.clamp()
		// Counted in lines: the selected row takes two when it waits
		// (89y0b7, tje4sx), every other row one.
		counts, least := map[int]int{}, map[int]int{}
		for p := range overviewSections {
			idx := m.paneIndices(p)
			counts[p] = m.linesOf(idx, m.selIn(p))
			least[p] = m.linesOf(idx[:min(len(idx), 3)], m.selIn(p))
		}
		if counts[sectionDoing] <= 3 || counts[sectionRecent] <= 3 {
			t.Fatalf("the fixture must overflow the focused sections: %v", counts)
		}
		rows, shown := m.overviewRows(), shownRows(m)
		budget := rowBudget(m)
		spare := budget
		for p := range overviewSections {
			if p != focus {
				spare -= least[p]
			}
		}
		if want := min(counts[focus], max(least[focus], spare)); rows[focus] != want || shown[focus] != want {
			t.Fatalf("focus %d: shows %d (drawn %d), want every spare row: %d", focus, rows[focus], shown[focus], want)
		}
		spare -= rows[focus]
		for _, p := range spareOrder {
			if p == focus {
				continue
			}
			want := least[p]
			want += min(spare, counts[p]-want)
			spare -= want - least[p]
			if rows[p] != want || shown[p] != want {
				t.Fatalf("focus %d: section %d shows %d (drawn %d), want %d of %d", focus, p, rows[p], shown[p], want, counts[p])
			}
		}
		if lines := strings.Split(m.overviewText(), "\n"); len(lines) > m.bodyHeight() {
			t.Fatalf("focus %d: %d lines in a body of %d", focus, len(lines), m.bodyHeight())
		}
	}
	// On a short terminal it still fits: RECENTLY DONE gives way first.
	m = overviewAt(t, 80, 24)
	if rows := m.overviewRows(); rows[sectionRecent] != 1 || rows[sectionDoing] < 1 {
		t.Fatalf("80x24: %v", rows)
	}
	if lines := strings.Split(m.overviewText(), "\n"); len(lines) > m.bodyHeight() {
		t.Fatalf("80x24: %d lines in a body of %d", len(lines), m.bodyHeight())
	}
}

// h9zghj check 2: focused, RECENTLY DONE scrolls through every closed task
// newest first; unfocused it shows 3.
func TestRecentlyDoneScrollsThroughAll(t *testing.T) {
	m := overviewAt(t, 140, 40)
	closed := 0
	for _, v := range m.visible {
		if v.State == "done" || v.State == "dropped" {
			closed++
		}
	}
	all := m.paneIndices(sectionRecent)
	if len(all) != closed || closed != 30 {
		t.Fatalf("RECENTLY DONE holds %d of %d closed tasks", len(all), closed)
	}
	for i := 1; i < len(all); i++ {
		if m.closedKey(m.visible[all[i-1]]) < m.closedKey(m.visible[all[i]]) {
			t.Fatalf("not newest first at %d", i)
		}
	}
	if shownRows(m)[sectionRecent] != 3 {
		t.Fatalf("unfocused RECENTLY DONE shows %d rows, want 3", shownRows(m)[sectionRecent])
	}
	m = press(m, "l", "l", "l")
	if m.pane != sectionRecent {
		t.Fatalf("pane %d", m.pane)
	}
	var seen []string
	for range all {
		sel, _ := m.selected()
		seen = append(seen, sel.ID)
		if !strings.Contains(ansi.Strip(m.overviewText()), sel.ID) {
			t.Fatalf("selected %s not on screen", sel.ID)
		}
		m = press(m, "j")
	}
	for i, idx := range all {
		if seen[i] != m.visible[idx].ID {
			t.Fatalf("j step %d selected %s, want %s", i, seen[i], m.visible[idx].ID)
		}
	}
}

// h9zghj check 3: Tab away from a scrolled section and back keeps its
// scroll and its selected row, in the overview and on the board.
func TestSectionsKeepScroll(t *testing.T) {
	m := overviewAt(t, 140, 30) // 4 sections since 6ephqk: 30 lines keep IN PROGRESS short of its 15
	for range 14 {
		m = press(m, "j")
	}
	before, _ := m.selected()
	first := m.scrollOf(sectionDoing)
	if first == 0 {
		t.Fatal("IN PROGRESS did not scroll; the check would prove nothing")
	}
	m = press(m, "l", "j", "l", "h", "h")
	if m.pane != sectionDoing {
		t.Fatalf("pane %d", m.pane)
	}
	if after, _ := m.selected(); after.ID != before.ID || m.scrollOf(sectionDoing) != first {
		t.Fatalf("back in IN PROGRESS: %s scroll %d, want %s scroll %d", after.ID, m.scrollOf(sectionDoing), before.ID, first)
	}
	// The unfocused section draws its kept window: the selected row is
	// still among IN PROGRESS's rows while NEXT UP is focused.
	m = press(m, "l")
	view := ansi.Strip(m.overviewText())
	inProgress := view[:strings.Index(view, "NEXT UP")]
	if !strings.Contains(inProgress, before.ID) {
		t.Fatalf("IN PROGRESS lost its row %s while unfocused:\n%s", before.ID, inProgress)
	}
	// The board: each column keeps its scroll.
	m = manyModel(t)
	m.width, m.height = 140, 14 // 3 rows per column: TODO's 8 tasks scroll
	m.resize()
	m.setView("board")
	for range 10 {
		m = press(m, "j")
	}
	before, _ = m.selected()
	first = m.scrollOf(0)
	if first == 0 {
		t.Fatal("TODO column did not scroll")
	}
	m = press(m, "l", "j", "h")
	if after, _ := m.selected(); after.ID != before.ID || m.scrollOf(0) != first {
		t.Fatalf("board: back in TODO: %s scroll %d, want %s scroll %d", after.ID, m.scrollOf(0), before.ID, first)
	}
}

// h9zghj check 4: an empty section is one line, None to the right of its
// title, in every overview section and board column.
func TestEmptySectionIsOneLine(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	if _, err := s.New(task.NewOptions{Title: "Only", Goal: "g", Acceptance: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 200, 40
	m.resize()
	m.setView("overview")
	view := ansi.Strip(m.overviewText())
	for _, name := range []string{"IN PROGRESS · assigned to", "RECENTLY DONE · last closed"} {
		want := name + "  [0] · None"
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q:\n%s", want, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if strings.TrimSpace(line) == "None" {
			t.Fatalf("None on a row of its own:\n%s", view)
		}
	}
	m.setView("board")
	board := ansi.Strip(m.boardText())
	for _, s := range []string{"DOING", "DEPENDS", "DONE", "DROPPED"} {
		if !strings.Contains(board, fmt.Sprintf("%s [0] · None", s)) {
			t.Fatalf("board column %s lacks None on its title:\n%s", s, board)
		}
	}
	if !strings.Contains(board, "TODO [1]") || strings.Contains(board, "TODO [1] · None") {
		t.Fatalf("a column with tasks says None:\n%s", board)
	}
}
