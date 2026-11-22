package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// groupBoard is a board with an epic, two groups and a plain task:
//
//	Launch (epic, no group)
//	Launch posts, Launch mail (group later, children of Launch)
//	Ops one, Ops two (group ops)
//	Plain
func groupBoard(t *testing.T) (model, map[string]string) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	ids := map[string]string{}
	mk := func(title, group string) {
		r, err := s.New(task.NewOptions{Title: title, Goal: "g", Acceptance: []string{"a"}, Group: group})
		if err != nil {
			t.Fatal(err)
		}
		ids[title] = r.Task.ID
	}
	mk("Launch", "")
	mk("Launch posts", "later")
	mk("Ops one", "ops")
	mk("Launch mail", "later")
	mk("Ops two", "ops")
	mk("Plain", "")
	for _, c := range []string{"Launch posts", "Launch mail"} {
		if _, err := s.Set(ids[c], "parent", ids["Launch"]); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 80, 24
	m.resize()
	m.setView("list")
	m.move(0, true)
	return m, ids
}

// kx2ndm check 1: list and overview rows show each task's group and epic,
// "later·↳EPIC", before the text, with no line of their own; the overview
// footer says what ↳ means.
func TestRowsShowGroupAndEpic(t *testing.T) {
	m, ids := groupBoard(t)
	for _, view := range []string{"overview", "list"} {
		for _, w := range []int{80, 140} {
			m.width, m.height = w, 30
			m.resize()
			m.setView(view)
			s := screen(m)
			if n := len(strings.Split(s, "\n")); n != 30 {
				t.Fatalf("%s at %d: %d lines", view, w, n)
			}
			for _, want := range []string{"later·↳" + ids["Launch"] + " Launch posts", "ops Ops one"} {
				if !strings.Contains(s, want) {
					t.Fatalf("%s at %d lacks %q:\n%s", view, w, want, s)
				}
			}
			for _, line := range strings.Split(s, "\n") {
				if strings.Contains(line, ids["Plain"]) && (strings.Contains(line, "↳") || strings.Contains(line, "·Plain")) {
					t.Fatalf("%s at %d: a task with no group or epic shows one: %q", view, w, line)
				}
				if strings.Contains(line, ids["Plain"]) && !strings.Contains(line, " Plain") {
					t.Fatalf("%s at %d: %q", view, w, line)
				}
			}
			if view == "overview" && !strings.Contains(s, "! lint finding · ↳ epic") {
				t.Fatalf("overview at %d: no legend for ↳ in the footer:\n%s", w, s)
			}
		}
	}
}

// kx2ndm check 2: g (or : grouping) cycles flat → group → epic → tag
// (txjkrm) → flat, and the header says which.
func TestGroupingCyclesAndHeaderSays(t *testing.T) {
	m, ids := groupBoard(t)
	if strings.Contains(screen(m), "grouped by") {
		t.Fatalf("flat rows say grouped:\n%s", screen(m))
	}
	m = commandLine(m, "grouping")
	s := screen(m)
	header := strings.Split(s, "\n")[0]
	if !strings.Contains(header, "grouped by group") {
		t.Fatalf("header %q", header)
	}
	for _, want := range []string{"▾ later  [2]", "▾ ops  [2]", "▾ no group  [2]"} {
		if !strings.Contains(s, want) {
			t.Fatalf("by group lacks %q:\n%s", want, s)
		}
	}
	// The heading names the group, so the cell leaves it out.
	if strings.Contains(s, "later·") {
		t.Fatalf("grouped by group, rows repeat the group:\n%s", s)
	}
	m = commandLine(m, "grouping")
	s = screen(m)
	if h := strings.Split(s, "\n")[0]; !strings.Contains(h, "grouped by epic") {
		t.Fatalf("header %q", h)
	}
	for _, want := range []string{"▾ ↳" + ids["Launch"] + " Launch  [3]", "▾ no epic  [3]", "later Launch posts"} {
		if !strings.Contains(s, want) {
			t.Fatalf("by epic lacks %q:\n%s", want, s)
		}
	}
	m = press(m, "g")
	if s := screen(m); !strings.Contains(strings.Split(s, "\n")[0], "grouped by tag") {
		t.Fatalf("g after epic is not by tag (txjkrm):\n%s", s)
	}
	m = press(m, "g")
	if s := screen(m); strings.Contains(s, "grouped by") || strings.Contains(s, "▾") {
		t.Fatalf("g after tag is not flat:\n%s", s)
	}
	// g alone cycles too (Jørgen 2026-11-17 23:37): flat → group.
	if m = press(m, "g"); m.grouping != "group" {
		t.Fatalf("g: grouping %q", m.grouping)
	}
	m = commandLine(m, "grouping epic")
	if !strings.Contains(strings.Split(screen(m), "\n")[0], "grouped by epic") {
		t.Fatalf(": grouping epic:\n%s", screen(m))
	}
	m = commandLine(m, "grouping sideways")
	if !strings.Contains(m.message, "fix: : grouping flat") {
		t.Fatalf("bad grouping: %q", m.message)
	}
}

// kx2ndm check 3: j/k stop on headings; Space folds and unfolds the one
// under the cursor, its count stays, its tasks leave the rows and the
// cursor stays on it; the fold holds in every section.
func TestSpaceFoldsAHeading(t *testing.T) {
	m, ids := groupBoard(t)
	m = commandLine(m, "grouping")
	m.move(0, true)
	if m.head != foldKey("group", "") {
		t.Fatalf("top row is %q, want the no group heading (Launch leads the rank)", m.head)
	}
	// Down to the later heading: no group [2] holds Launch and Plain.
	for range 3 {
		m = press(m, "j")
	}
	if m.head != foldKey("group", "later") {
		t.Fatalf("after 3 j: head %q cursor %d", m.head, m.cursor)
	}
	if !strings.Contains(screen(m), "> ▾ later  [2]") {
		t.Fatalf("the heading is not marked:\n%s", screen(m))
	}
	m = press(m, " ")
	s := screen(m)
	if !strings.Contains(s, "> ▸ later  [2]") || strings.Contains(s, "Launch posts") || strings.Contains(s, "Launch mail") {
		t.Fatalf("folded: want '▸ later  [2]' and no later tasks:\n%s", s)
	}
	if !strings.Contains(s, "TASKS  [6]") { // the list's title rule (v68n82)
		t.Fatalf("folded tasks still count in the list's total:\n%s", s)
	}
	// Actions refuse on a heading.
	m = press(m, "t")
	if m.confirm != "" || !strings.Contains(m.message, "heading is not a task") {
		t.Fatalf("t on a heading: confirm %q message %q", m.confirm, m.message)
	}
	// The overview folds it too, keeping NEXT UP's count.
	o := m
	o.setView("overview")
	if s := screen(o); !strings.Contains(s, "▸ later  [2]") || strings.Contains(s, "Launch posts") || !strings.Contains(s, "NEXT UP · rank order  [5]") {
		t.Fatalf("overview with later folded:\n%s", s)
	}
	m = press(m, "enter")
	if s := screen(m); !strings.Contains(s, "> ▾ later  [2]") || !strings.Contains(s, "Launch posts") {
		t.Fatalf("Enter did not unfold:\n%s", s)
	}
	// j from the heading goes into its first task.
	m = press(m, "j")
	if tk, ok := m.selected(); !ok || tk.ID != ids["Launch posts"] || m.head != "" {
		t.Fatalf("j from the heading: %v %q head %q", ok, tk.ID, m.head)
	}
	// Space on a task row marks it (rxr4z8); it never ticks a check
	// outside the task. Esc unmarks it again.
	m = press(m, " ")
	if len(m.marks) != 1 || m.confirm != "" {
		t.Fatalf("Space on a row: marks %v confirm %q", m.marks, m.confirm)
	}
	m = press(m, "esc")
	// A task found under a folded heading is shown: the heading unfolds.
	m = press(m, "k", "k", " ")
	if !m.folded[foldKey("group", "later")] {
		t.Fatalf("later not folded again")
	}
	if !m.follow(ids["Launch mail"]) {
		t.Fatalf("follow refused a task under a folded heading: %q", m.message)
	}
	if tk, _ := m.selected(); tk.ID != ids["Launch mail"] || m.folded[foldKey("group", "later")] {
		t.Fatalf("follow: selected %q, later folded %v", tk.ID, m.folded[foldKey("group", "later")])
	}
	// A reload (a change on disk) keeps the heading under the cursor.
	m.move(0, true)
	m = press(m, "j", "j", "j")
	m.applyKeeping(loadSnapshot(m.store, &loadCounter{}, &closeLog{}, nil))
	if m.head != foldKey("group", "later") || m.cursor != -1 {
		t.Fatalf("reload lost the heading: head %q cursor %d", m.head, m.cursor)
	}
}

// kx2ndm check 5: at 80 and 140 columns every grouped view keeps its
// height and no line runs wider than the terminal; golden reviewed.
func TestGroupedViewsFitAndGolden(t *testing.T) {
	m, ids := groupBoard(t)
	var out strings.Builder
	for _, g := range []string{"group", "epic"} {
		for _, view := range []string{"list", "overview", "board"} {
			for _, w := range []int{80, 140} {
				mm := commandLine(m, "grouping "+g)
				mm.width, mm.height = w, 24
				mm.resize()
				mm.setView(view)
				mm.move(0, true)
				if g == "group" {
					mm = press(mm, "j", "j", "j") // the later heading
				}
				mm = press(mm, " ") // by epic: the top heading, the epic's

				lines := strings.Split(screen(mm), "\n")
				if len(lines) != 24 {
					t.Fatalf("%s %s at %d: %d lines", g, view, w, len(lines))
				}
				for _, l := range lines {
					if ansi.StringWidth(l) > w {
						t.Fatalf("%s %s at %d: %d wide: %q", g, view, w, ansi.StringWidth(l), l)
					}
				}
				fmt.Fprintf(&out, "== %s · %s · %dx24\n%s\n", g, view, w, strings.Join(lines, "\n"))
			}
		}
	}
	// Ids are random: each becomes a name as wide, so columns stay.
	text := out.String()
	names := map[string]string{"Launch": "LAUNCH", "Launch posts": "POSTS0", "Launch mail": "MAIL00", "Ops one": "OPS001", "Ops two": "OPS002", "Plain": "PLAIN0"}
	for title, id := range ids {
		text = strings.ReplaceAll(text, id, names[title])
	}
	golden(t, "testdata/group/views.txt", "TestGroupedViewsFitAndGolden", text)
}
