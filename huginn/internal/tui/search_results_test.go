package tui

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/task"
)

// resultsBoard is the statbus case of 5cjy88 (Jørgen 2026-11-21 08:34Z:
// '/ k5q120' selected hmgadt in RECENTLY DONE, whose notes mention
// k5q120, while k5q120 itself sat in LINT). Target is the task whose id
// is typed: assigned and todo, with a lint finding, so in the overview it
// is in LINT only. Two done tasks (RECENTLY DONE) mention its id, one in
// its title (so, as among statbus's 599 tasks, every prefix typed on the
// way to the id matches in RECENTLY DONE) and one in its notes; one todo
// task (NEXT UP) mentions it in its notes; three fillers mention nothing.
// / ranks through the task cache's index, as hug tui does, so a word in
// the notes finds its task. 120x30, the overview, the cursor in RECENTLY
// DONE.
func resultsBoard(t *testing.T) (model, map[string]string) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	ids := map[string]string{}
	add := func(title, goal string) string {
		t.Helper()
		r, err := s.New(task.NewOptions{Title: title, Goal: goal, Purpose: "P", Principle: "Q", Acceptance: []string{"C"}})
		if err != nil {
			t.Fatal(err)
		}
		ids[title] = r.Task.ID
		return r.Task.ID
	}
	for _, f := range []string{"Filler one", "Filler two", "Filler three"} {
		add(f, "G")
	}
	// Two sentences in the Goal: goal-too-long advice, a LINT row.
	target := add("Target", "One sentence. Then another.")
	if _, err := s.Set(target, "assignee", "someone"); err != nil {
		t.Fatal(err)
	}
	next := add("Mentions next", "G")
	for _, title := range []string{"Mentions done A", "Mentions done B"} {
		id := add(title, "G")
		if title == "Mentions done A" {
			if _, err := s.Set(id, "title", "Follow-up to "+target); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := s.Note(id, "follow-up in "+target); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Check(id, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Take(id, "tester"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Done(id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Note(next, "after "+target+" lands"); err != nil {
		t.Fatal(err)
	}
	c, err := cache.Open(s.Root, filepath.Join(t.TempDir(), "cache.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	s.Refresh = func() ([]task.Task, error) { return c.List(context.Background()) }
	search := func(q string) ([]string, error) { return c.Rank(context.Background(), q) }
	m := newModel(s, Options{User: "tester", NoWatch: true, NoReload: true, Search: search})
	m.width, m.height = 120, 30
	m.resize()
	m.setView("overview")
	m.pane = sectionRecent
	m.clamp()
	if sel, ok := m.selected(); !ok || m.visible[m.cursor].State != "done" {
		t.Fatalf("fixture: cursor not in RECENTLY DONE: %v %s", ok, sel.Title)
	}
	if !m.linted[m.visible[slices.IndexFunc(m.visible, func(v task.Task) bool { return v.ID == target })].Path] {
		t.Fatal("fixture: Target has no lint finding")
	}
	return m, ids
}

// selID is the id of the selected task, or "".
func selID(m model) string {
	sel, ok := m.selected()
	if !ok {
		return ""
	}
	return sel.ID
}

// 5cjy88 check 1: typing a task's exact id selects that task, in whatever
// section or column it is, before tasks that only mention it. On main the
// current section kept the cursor because it had a match (searchTyped).
func TestSearchExactIDFirst(t *testing.T) {
	for _, view := range []string{"overview", "board", "list"} {
		t.Run(view, func(t *testing.T) {
			m, ids := resultsBoard(t)
			if view != "overview" {
				m.setView(view)
				if view == "board" {
					m.pane = slices.Index(boardStates, "done") // the mentions' column
					m.clamp()
				}
			}
			m = press(m, "/", ids["Target"])
			if got := selID(m); got != ids["Target"] {
				t.Fatalf("typing %s selected %q, want the task with that id\n%s", ids["Target"], got, ansi.Strip(m.View()))
			}
			// Upper case is the same id.
			m = press(m, "esc", "/", strings.ToUpper(ids["Target"]))
			if got := selID(m); got != ids["Target"] {
				t.Fatalf("typing %s selected %q", strings.ToUpper(ids["Target"]), got)
			}
		})
	}
}

// matchOrder is every match in walk order: each section top to bottom,
// the sections in screen order.
func matchOrder(m model) []string {
	var out []string
	for p := 0; p < m.paneCount(); p++ {
		for _, i := range m.paneIndices(p) {
			out = append(out, m.visible[i].ID)
		}
	}
	return out
}

// 5cjy88 check 2: in the / prompt ↑↓ and ctrl-n/p move between the
// matches, across sections, never through search history. ↑ in the empty
// prompt still recalls (xfv9pw).
func TestSearchPromptArrowsPickMatches(t *testing.T) {
	m, ids := resultsBoard(t)
	target := ids["Target"]
	m = press(m, "/", target)
	order := matchOrder(m)
	if len(order) != 4 || order[len(order)-1] != target {
		t.Fatalf("matches %v, want 4 ending in Target %s", order, target)
	}
	for _, k := range []string{"up", "ctrl+p"} {
		before := slices.Index(order, selID(m))
		m = press(m, k)
		if got := selID(m); got != order[before-1] || m.prompt != "/" || m.input.Value() != target {
			t.Fatalf("%s in the prompt: selected %q value %q, want %s (the match before)", k, got, m.input.Value(), order[before-1])
		}
	}
	for _, k := range []string{"down", "ctrl+n"} {
		before := slices.Index(order, selID(m))
		m = press(m, k)
		if got := selID(m); got != order[before+1] || m.input.Value() != target {
			t.Fatalf("%s in the prompt: selected %q value %q, want %s (the match after)", k, got, m.input.Value(), order[before+1])
		}
	}
	// ↓ is never history: with a search recalled, it picks.
	m = press(m, "enter", "esc", "/", "up")
	if m.input.Value() != target {
		t.Fatalf("↑ in the empty prompt recalls %q, want %s", m.input.Value(), target)
	}
	m = press(m, "down")
	if m.input.Value() != target {
		t.Fatalf("↓ on a recalled search changed the prompt to %q", m.input.Value())
	}
	// After a pick ↑ picks too.
	at := selID(m)
	m = press(m, "up")
	if m.input.Value() != target || selID(m) == at {
		t.Fatalf("↑ after a pick: value %q, selected %s (was %s)", m.input.Value(), selID(m), at)
	}
}

// 5cjy88 checks 3 and 6: Enter closes the prompt and leaves the results on
// screen. j/k walk every match in order across sections (RECENTLY DONE
// down into LINT), < and > go to the first and last, Enter opens the task
// and Esc there comes back to the results.
func TestSearchEnterShowsResults(t *testing.T) {
	m, ids := resultsBoard(t)
	target := ids["Target"]
	m = press(m, "/", target, "enter")
	if m.prompt != "" || m.query != target || len(m.visible) != 4 {
		t.Fatalf("Enter: prompt %q query %q, %d shown; want the results", m.prompt, m.query, len(m.visible))
	}
	if got := selID(m); got != target {
		t.Fatalf("Enter: selected %q, want %s", got, target)
	}
	screen := ansi.Strip(m.View())
	for _, want := range []string{"found: / " + target + " · 4 matches · Esc all tasks", "j/k match", "Enter open", "Esc all tasks"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("results screen lacks %q\n%s", want, screen)
		}
	}
	if strings.Contains(screen, "[/ "+target+"]") {
		t.Fatalf("results drawn as a kept filter chip\n%s", screen)
	}
	order := matchOrder(m)
	m = press(m, "<")
	walked := []string{selID(m)}
	for range order[1:] {
		m = press(m, "j")
		walked = append(walked, selID(m))
	}
	if !slices.Equal(walked, order) {
		t.Fatalf("j from the top walked %v, want every match %v", walked, order)
	}
	if m.pane != sectionLint {
		t.Fatalf("the last match is in pane %d, want LINT", m.pane)
	}
	// RECENTLY DONE then LINT: j crossed into LINT without h/l.
	m = press(m, "k")
	if m.pane != sectionRecent {
		t.Fatalf("k from LINT: pane %d, want RECENTLY DONE", m.pane)
	}
	m = press(m, ">")
	if selID(m) != target {
		t.Fatalf("> selected %s, want the last match %s", selID(m), target)
	}
	m = press(m, "K")
	if selID(m) != order[0] {
		t.Fatalf("K selected %s, want a page up: the first match %s", selID(m), order[0])
	}
	m = press(m, "J", "enter")
	if !m.detail || selID(m) != target {
		t.Fatalf("J Enter: detail %v selected %s, want %s open", m.detail, selID(m), target)
	}
	// In the task n and p step every match too, across sections.
	if m = press(m, "p"); selID(m) != order[2] || !m.detail {
		t.Fatalf("p in the task opened %s, want the match before: %s", selID(m), order[2])
	}
	if m = press(m, "n"); selID(m) != target {
		t.Fatalf("n in the task opened %s, want %s", selID(m), target)
	}
	m = press(m, "esc")
	if m.detail || m.query != target || selID(m) != target || !strings.Contains(ansi.Strip(m.View()), "found: / "+target) {
		t.Fatalf("Esc from the task: detail %v query %q selected %s; want the results again", m.detail, m.query, selID(m))
	}
}

// 5cjy88 check 3: m on the results leaves them first, as Esc does, and
// grabs in the full view, so the task moves among its real neighbours.
func TestSearchResultsMoveAmongAll(t *testing.T) {
	m, ids, commits := searchBoard(t)
	m.setView("list")
	m = press(m, "/", "Zebra", "enter")
	if m.query != "Zebra" || m.prompt != "" {
		t.Fatalf("Enter: query %q prompt %q", m.query, m.prompt)
	}
	m = press(m, "m")
	if !m.grab.active || m.query != "" || len(m.visible) != 60 {
		t.Fatalf("m on the results: grab %v, query %q, %d shown; want a grab among all 60", m.grab.active, m.query, len(m.visible))
	}
	m = press(m, "j", "enter")
	if want := "moved " + ids["Zebra"] + " below " + ids["Task 51"] + ", row 51 of 60 in the list"; m.message != want || *commits != 1 {
		t.Fatalf("m j Enter: %q (commits %d), want %q", m.message, *commits, want)
	}
}

// 5cjy88 check 4: Esc on the results clears the search, keeping the
// picked task selected and in view; Esc while typing cancels: the task
// selected before / and the search before it come back.
func TestSearchEscStepsBack(t *testing.T) {
	m, ids := resultsBoard(t)
	target := ids["Target"]
	m = press(m, "/", target, "up", "enter") // pick the match before Target
	picked := selID(m)
	if picked == target || picked == "" {
		t.Fatalf("↑ did not pick another match: %q", picked)
	}
	m = press(m, "esc")
	if m.query != "" || m.prompt != "" || len(m.visible) != len(m.all) {
		t.Fatalf("Esc on the results: query %q prompt %q, %d of %d shown", m.query, m.prompt, len(m.visible), len(m.all))
	}
	if got := selID(m); got != picked {
		t.Fatalf("Esc on the results: selected %q, want the pick %s", got, picked)
	}
	screen := ansi.Strip(m.View())
	if !strings.Contains(screen, picked) || !strings.Contains(screen, "found "+picked+", row") {
		t.Fatalf("the pick is not in view, or the footer does not say where\n%s", screen)
	}

	// Esc while typing: back to the task and section before /.
	m, ids = resultsBoard(t)
	target = ids["Target"]
	m = press(m, "j")
	before, pane := selID(m), m.pane
	m = press(m, "/", target, "esc")
	if m.query != "" || m.prompt != "" || selID(m) != before || m.pane != pane {
		t.Fatalf("Esc while typing: query %q prompt %q selected %s pane %d; want %s in pane %d", m.query, m.prompt, selID(m), m.pane, before, pane)
	}
	if strings.Contains(m.message, "found") {
		t.Fatalf("a cancelled search says %q", m.message)
	}
	// From the results, / refines; Esc while typing restores the results.
	m = press(m, "/", target, "enter", "/")
	if m.input.Value() != target {
		t.Fatalf("/ on the results opens with %q, want %s to refine", m.input.Value(), target)
	}
	m = press(m, "backspace", "esc")
	if m.query != target || selID(m) != target || !strings.Contains(ansi.Strip(m.View()), "found: / "+target) {
		t.Fatalf("Esc refining: query %q selected %s; want the results as they were", m.query, selID(m))
	}
}

// Enter with no match keeps the prompt open and says so.
func TestSearchEnterNoMatch(t *testing.T) {
	m, _ := resultsBoard(t)
	m = press(m, "/", "qqqqqq", "enter")
	if m.prompt != "/" || m.message != "no match for qqqqqq: edit it, or Esc" {
		t.Fatalf("Enter with no match: prompt %q message %q", m.prompt, m.message)
	}
}

// 5cjy88 check 7: Tab keeps the search as a filter chip that lasts across
// opening a task and switching view, until F clears it; Esc does not clear
// it. j/k walk its matches across sections too.
func TestSearchTabKeepsChip(t *testing.T) {
	m, ids := resultsBoard(t)
	target := ids["Target"]
	chip := "filter: [/ " + target + "]"
	m = press(m, "/", target, "tab")
	if m.query != target || !strings.Contains(ansi.Strip(m.View()), chip) {
		t.Fatalf("Tab: query %q\n%s", m.query, ansi.Strip(m.View()))
	}
	order := matchOrder(m)
	m = press(m, "<")
	walked := []string{selID(m)}
	for range order[1:] {
		m = press(m, "j")
		walked = append(walked, selID(m))
	}
	if !slices.Equal(walked, order) {
		t.Fatalf("j on the chip walked %v, want %v", walked, order)
	}
	m = press(m, "enter", "esc")
	if m.query != target || !strings.Contains(ansi.Strip(m.View()), chip) {
		t.Fatalf("after opening a task: query %q", m.query)
	}
	for _, view := range []string{"l", "b", "o"} {
		m = press(m, "v", view)
		if m.query != target || !strings.Contains(ansi.Strip(m.View()), chip) {
			t.Fatalf("after v %s: query %q\n%s", view, m.query, ansi.Strip(m.View()))
		}
	}
	m = press(m, "v", "l", "esc")
	if m.view != "overview" || m.query != target || !strings.Contains(ansi.Strip(m.View()), chip) {
		t.Fatalf("Esc on the chip: view %s query %q; want the overview, the chip kept", m.view, m.query)
	}
	m = press(m, "esc")
	if m.query != target {
		t.Fatalf("Esc in the overview cleared the chip")
	}
	m.follow(target)
	m = press(m, "F")
	if m.query != "" || selID(m) != target || strings.Contains(ansi.Strip(m.View()), chip) {
		t.Fatalf("F: query %q selected %s", m.query, selID(m))
	}
}

// An f chip is kept on Esc too (crab 09:16Z: one rule, a chip lasts until
// F or ⌫): Esc goes up a level, the chip line still drawn.
func TestEscKeepsFilterChips(t *testing.T) {
	m, _ := resultsBoard(t)
	m.setView("list")
	m = commandLine(m, "state todo")
	m = press(m, "esc")
	if m.view != "overview" || m.filter.State != "todo" || !strings.Contains(ansi.Strip(m.View()), "[state:todo ×]") {
		t.Fatalf("Esc with a chip: view %s state %q\n%s", m.view, m.filter.State, ansi.Strip(m.View()))
	}
	m = press(m, "backspace")
	if m.filter.State != "" {
		t.Fatalf("⌫ left %q", m.filter.State)
	}
}

// A view switch ends the Enter results and keeps the picked task; a Tab
// chip stays (above).
func TestSearchResultsEndOnViewSwitch(t *testing.T) {
	m, ids := resultsBoard(t)
	target := ids["Target"]
	m = press(m, "/", target, "enter", "v", "l")
	if m.query != "" || len(m.visible) != len(m.all) || selID(m) != target {
		t.Fatalf("v l on the results: query %q, %d of %d shown, selected %s; want all tasks, %s selected", m.query, len(m.visible), len(m.all), selID(m), target)
	}
}

// 5cjy88 check 5 (the footer and help half): the prompt and the results
// name their keys.
func TestSearchFooterAndHelp(t *testing.T) {
	m, ids := resultsBoard(t)
	typing := press(m, "/", ids["Target"])
	bar := ansi.Strip(typing.keyBar())
	for _, want := range []string{"pick", "Enter results", "Tab keep as filter", "Esc cancel"} {
		if !strings.Contains(bar, want) {
			t.Fatalf("prompt footer lacks %q: %s", want, bar)
		}
	}
	help := ansi.Strip(press(typing, "f1").helpText())
	for _, want := range []string{"↑↓ ctrl-n/p", "Enter", "show the results: j/k walk them", "Esc", "cancel: back where / was pressed"} {
		if !strings.Contains(help, want) {
			t.Fatalf("/ help lacks %q\n%s", want, help)
		}
	}
	results := press(typing, "enter")
	help = ansi.Strip(press(results, "?").helpText())
	for _, want := range []string{"IN THE SEARCH RESULTS", "next / previous match", "all tasks, keeping the task"} {
		if !strings.Contains(help, want) {
			t.Fatalf("results help lacks %q\n%s", want, help)
		}
	}
}
