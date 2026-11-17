package tui

import (
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// moveBoard makes tasks in rank order from titles; "epic:" titles get the
// next task as a child, "done:" and "dropped:" titles are closed. Commits are
// counted.
func moveBoard(t *testing.T, titles ...string) (model, map[string]string, *int) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	ids := map[string]string{}
	var epic string
	for _, title := range titles {
		r, err := s.New(task.NewOptions{Title: title, Goal: "G", Acceptance: []string{"C"}})
		if err != nil {
			t.Fatal(err)
		}
		id := r.Task.ID
		ids[title] = id
		if epic != "" {
			if _, err := s.Set(id, "parent", epic); err != nil {
				t.Fatal(err)
			}
			epic = ""
		}
		switch {
		case strings.HasPrefix(title, "epic:"):
			epic = id
		case strings.HasPrefix(title, "done:"):
			if _, err := s.Check(id, 1); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Take(id, "tester"); err != nil { // take before done (6ephqk)
				t.Fatal(err)
			}
			if _, err := s.Done(id); err != nil {
				t.Fatal(err)
			}
		case strings.HasPrefix(title, "dropped:"):
			if _, err := s.Drop(id, "test reason"); err != nil {
				t.Fatal(err)
			}
		}
	}
	commits := 0
	m := newModel(s, Options{User: "tester", Commit: func(string) error { commits++; return nil }})
	m.width, m.height = 140, 40
	m.resize()
	return m, ids, &commits
}

// shown is the ids of the rows in the current section, top to bottom.
func shown(m model) []string {
	var out []string
	for _, i := range m.indices() {
		out = append(out, m.visible[i].ID)
	}
	return out
}

func selectRow(t *testing.T, m model, id string) model {
	t.Helper()
	for _, i := range m.indices() {
		if m.visible[i].ID == id {
			m.cursor = i
			return m
		}
	}
	t.Fatalf("%s not shown: %v", id, shown(m))
	return m
}

func nextUp(m model) model {
	m.setView("overview")
	m.pane = 1
	m.clamp()
	return m
}

// screenOrder is the order in which ids appear on screen.
func screenOrder(m model, ids ...string) []string {
	view := m.View()
	type at struct {
		id  string
		pos int
	}
	var found []at
	for _, id := range ids {
		if p := strings.Index(view, id); p >= 0 {
			found = append(found, at{id, p})
		}
	}
	for i := range found {
		for j := i + 1; j < len(found); j++ {
			if found[j].pos < found[i].pos {
				found[i], found[j] = found[j], found[i]
			}
		}
	}
	var out []string
	for _, f := range found {
		out = append(out, f.id)
	}
	return out
}

// Check 1 and 3: in NEXT UP, J swaps a row with the row shown below it, even
// with an epic (not shown in NEXT UP) ranked between them; one press moves
// it at once, with no confirm (3xcesq), and the result names both tasks.
func TestMoveJPastHiddenEpicInNextUp(t *testing.T) {
	m, ids, commits := moveBoard(t, "Alpha", "epic: Gem", "Xray", "Child")
	a, x, c, gem := ids["Alpha"], ids["Xray"], ids["Child"], ids["epic: Gem"]
	m = nextUp(m)
	if got := strings.Join(shown(m), " "); got != a+" "+x+" "+c {
		t.Fatalf("NEXT UP before: %s (epic %s must not be shown)", got, gem)
	}
	m = selectRow(t, m, a)
	m = press(m, "J")
	if m.confirm != "" {
		t.Fatal("J asked:", m.confirmLine())
	}
	if got := strings.Join(shown(m), " "); got != x+" "+a+" "+c {
		t.Fatalf("NEXT UP after J: %s", got)
	}
	if got := strings.Join(screenOrder(m, a, x, c), " "); got != x+" "+a+" "+c {
		t.Fatalf("screen after J: %s\n%s", got, m.View())
	}
	if want := "moved " + a + " below " + x + ", row 2 of 3 in NEXT UP"; m.message != want || *commits != 1 {
		t.Fatalf("message %q (want %q), commits %d", m.message, want, *commits)
	}
	if sel, _ := m.selected(); sel.ID != a {
		t.Fatal("selection did not follow the moved task", sel.ID)
	}
	// The epic keeps its place in the full order, between Alpha's old and new
	// place: only Alpha moved.
	var full []string
	for _, v := range m.all {
		full = append(full, v.ID)
	}
	if got := strings.Join(full, " "); got != gem+" "+x+" "+a+" "+c {
		t.Fatalf("full order: %s", got)
	}
}

// Check 2: K moves up past the row shown above; at the first or last row
// shown, the message says so, no confirm is asked and no commit is made.
func TestMoveKAndEdges(t *testing.T) {
	m, ids, commits := moveBoard(t, "Alpha", "epic: Gem", "Xray", "Child")
	a, x, c := ids["Alpha"], ids["Xray"], ids["Child"]
	m = selectRow(t, nextUp(m), c)
	m = press(m, "K")
	if m.confirm != "" {
		t.Fatal("K asked:", m.confirmLine())
	}
	if got := strings.Join(shown(m), " "); got != a+" "+c+" "+x || *commits != 1 {
		t.Fatalf("after K: %s, commits %d", got, *commits)
	}
	if want := "moved " + c + " above " + x + ", row 2 of 3 in NEXT UP"; m.message != want {
		t.Fatal(m.message)
	}
	m = selectRow(t, m, a)
	m = press(m, "K")
	if m.confirm != "" || m.message != a+" is already the first row in NEXT UP; nothing moved" {
		t.Fatalf("first row: confirm %q message %q", m.confirm, m.message)
	}
	m = selectRow(t, m, x)
	m = press(m, "J")
	if m.confirm != "" || m.message != x+" is already the last row in NEXT UP; nothing moved" {
		t.Fatalf("last row: confirm %q message %q", m.confirm, m.message)
	}
	if *commits != 1 {
		t.Fatal("an edge press made a commit", *commits)
	}
}

// Check 4: with a filter, J and K move relative to the rows the filter shows.
func TestMoveRelativeToFilteredRows(t *testing.T) {
	m, ids, commits := moveBoard(t, "Alpha one", "Beta", "Gamma", "Alpha two", "Delta")
	a1, a2 := ids["Alpha one"], ids["Alpha two"]
	m.setView("list")
	m = press(m, "/", "Alpha", "tab")
	if got := strings.Join(shown(m), " "); got != a1+" "+a2 {
		t.Fatalf("filtered: %s", got)
	}
	m = selectRow(t, m, a1)
	m = press(m, "J")
	if want := "moved " + a1 + " below " + a2 + ", row 2 of 2 in the list (filtered)"; m.message != want {
		t.Fatalf("message %q, want %q", m.message, want)
	}
	if got := strings.Join(shown(m), " "); got != a2+" "+a1 || *commits != 1 {
		t.Fatalf("filtered after J: %s, commits %d", got, *commits)
	}
	var full []string
	for _, v := range m.all {
		full = append(full, v.Title)
	}
	if got := strings.Join(full, ", "); got != "Beta, Gamma, Alpha two, Alpha one, Delta" {
		t.Fatal("full order", got)
	}
	m = press(m, "J")
	if m.confirm != "" || !strings.Contains(m.message, "already the last row in the list (filtered)") {
		t.Fatal(m.message)
	}
}

// Check 5, Jørgen's case: closed tasks and an epic ranked between the rows
// of NEXT UP. Before ksv323, each J moved past the next task in the full
// order, shown or not: three presses made commits that changed nothing on
// screen. Now every commit changes the order shown, and a press that cannot
// move says so and commits nothing.
func TestMoveThreePressesEachVisible(t *testing.T) {
	m, ids, commits := moveBoard(t, "Alpha", "epic: Gem", "Gem child", "done: Old", "dropped: Gone", "Xray", "Yoke")
	a, x, y, g := ids["Alpha"], ids["Xray"], ids["Yoke"], ids["Gem child"]
	m = selectRow(t, nextUp(m), a)
	want := [][]string{{g, a, x, y}, {g, x, a, y}, {g, x, y, a}}
	before := strings.Join(shown(m), " ")
	if before != a+" "+g+" "+x+" "+y {
		t.Fatalf("before: %s", before)
	}
	for press3 := 0; press3 < 3; press3++ {
		m = press(m, "J")
		after := strings.Join(shown(m), " ")
		if after == before || after != strings.Join(want[press3], " ") || *commits != press3+1 {
			t.Fatalf("press %d: %s -> %s, commits %d", press3+1, before, after, *commits)
		}
		if got := strings.Join(screenOrder(m, want[press3]...), " "); got != after {
			t.Fatalf("press %d: screen shows %s", press3+1, got)
		}
		before = after
	}
	m = press(m, "J")
	if m.confirm != "" || *commits != 3 || !strings.Contains(m.message, "already the last row in NEXT UP") {
		t.Fatalf("fourth press: %q, commits %d", m.message, *commits)
	}
}
