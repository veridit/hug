package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// sameBoard is Jørgen's IN PROGRESS shape (1bbe96): doing tasks, all
// active, all features, owned by two agents, with todo tasks after them.
// kinds and priorities name each doing task's type and priority by
// position; the rest are feature and medium.
func sameBoard(t *testing.T, doing, todo int, kinds, priorities map[int]string) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	for i := 0; i < doing+todo; i++ {
		kind, priority := "feature", "medium"
		if k, ok := kinds[i]; ok {
			kind = k
		}
		if p, ok := priorities[i]; ok {
			priority = p
		}
		r, err := s.New(task.NewOptions{Title: "Task " + strconv.Itoa(i+1) + " of the section", Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}, Type: kind, Priority: priority})
		if err != nil {
			t.Fatal(err)
		}
		if i < doing {
			path := filepath.Join(s.Root, r.Task.Path)
			b, _ := os.ReadFile(path)
			text := strings.Replace(string(b), "state: todo", "state: doing", 1)
			text = strings.Replace(text, "assignee: []", `assignee: ["`+[]string{"ann", "bob"}[i%2]+`"]`, 1)
			if err := os.WriteFile(path, []byte(text), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	m := newModel(s, Options{User: "tester"})
	m.setView("overview")
	m.pane = sectionDoing
	m.clamp()
	return m
}

// 1bbe96 check 1: a section whose tasks all share a stage draws no STAGE
// column, its heading ends '· all active' in the stage's colour, and the
// title gets the width (Jørgen's 13-active IN PROGRESS, at 80 and 140).
// Check 2: type too, so the heading reads '· all active · all feature'.
func TestSharedColumnMovesIntoHeading(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m := sameBoard(t, 13, 3, map[int]string{}, map[int]string{0: "high", 1: "low"})
		m.width, m.height = w, 30
		m.resize()
		view := ansi.Strip(m.View())
		head := ""
		for _, line := range strings.Split(view, "\n") {
			if strings.Contains(line, "IN PROGRESS") {
				head = strings.TrimRight(line, " ")
			}
		}
		if !strings.HasSuffix(head, "[13] · all active · all feature") {
			t.Fatalf("at %d the heading reads %q", w, head)
		}
		rows := section(view, "IN PROGRESS")
		rows = rows[strings.Index(rows, "\n")+1:] // the rows, not the heading
		if strings.Contains(rows, "active") || strings.Contains(rows, "feature") {
			t.Fatalf("at %d a shared column is still drawn:\n%s", w, rows)
		}
		if !strings.Contains(rows, "high") || !strings.Contains(rows, "low") {
			t.Fatalf("at %d PRIORITY, which differs, is gone:\n%s", w, rows)
		}
		// The title takes the freed width: the row's width less only the
		// cells still drawn (at 80 the full row would have cut TYPE and
		// still be narrower).
		full := m.rowLayout(w, false, m.ownerWidth(), m.ownerNames(m.paneIndices(sectionDoing)))
		got := m.sectionLayout(sectionDoing)
		want := w - rowHead - m.progressWidth() - idCell - on(got.priority, priorityCell) - got.owner - ownerGap
		if got.title != want || got.title <= full.title {
			t.Fatalf("at %d the title is %d wide, want %d (all columns: %d)", w, got.title, want, full.title)
		}
		// Ids are random per run: the golden names them by place.
		shot := strings.Join(strings.Split(view, "\n")[3:20], "\n")
		for _, i := range m.rawPane(sectionDoing) {
			shot = strings.Replace(shot, m.visible[i].ID, "id"+strings.Repeat("·", 4), 1)
		}
		golden = append(golden, "== "+strconv.Itoa(w), shot)
	}
	// The value in the stage's colour, the type in its own.
	m := sameBoard(t, 3, 0, nil, map[int]string{0: "high"})
	m.width = 140
	m.resize()
	same := m.sectionSame(sectionDoing)
	label := m.sameLabel(same, m.visible[m.rawPane(sectionDoing)[0]])
	if !strings.Contains(label, m.styles.status["doing"].Render("active")) || !strings.Contains(label, m.styles.kind["feature"].Render("feature")) {
		t.Fatalf("the heading's values are not in their colours: %q", label)
	}
	// A section of one task keeps its columns.
	one := sameBoard(t, 1, 0, nil, nil)
	if s := one.sectionSame(sectionDoing); s.any() {
		t.Fatalf("a section of one hides %+v", s)
	}
	goldenFile(t, "testdata/overview/same-columns.txt", strings.Join(golden, "\n")+"\n")
}

// 1bbe96 check 2: type, priority and owner move the same way; the list,
// one section, says them after TITLE.
func TestEverySharedColumnMoves(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := sameBoard(t, 4, 0, nil, nil)
	for i := range m.visible {
		m.visible[i].Assignee = []string{"ann"}
	}
	m.width, m.height = 140, 30
	m.resize()
	s := m.sectionSame(sectionDoing)
	if s.stage != "active" || s.kind != "feature" || s.priority != "medium" || s.owner != "ann" {
		t.Fatalf("shared %+v", s)
	}
	c := m.sectionLayout(sectionDoing)
	if c.status || c.kind || c.priority || c.owner != 0 {
		t.Fatalf("shared columns still drawn: %+v", c)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "· all active · all feature · all medium · all ann") {
		t.Fatalf("heading lacks the four values:\n%s", view)
	}
	// The list: TODO and DOING mixed, so STAGE stays; type and priority go.
	l := sameBoard(t, 2, 2, nil, nil)
	l.width, l.height = 100, 20
	l.resize()
	l.setView("list")
	screen := ansi.Strip(l.View())
	header := ""
	for _, line := range strings.Split(screen, "\n") {
		if strings.Contains(line, "TITLE") {
			header = strings.TrimRight(line, " ")
		}
	}
	if !strings.Contains(header, "STAGE") || strings.Contains(header, "TYPE") || strings.Contains(header, "PRIORITY") || !strings.HasSuffix(header, "· all feature · all medium") {
		t.Fatalf("list header %q", header)
	}
}

// 1bbe96 check 3: the choice uses every task of the section after
// filters, never the rows on screen: a 28-task section whose hidden rows
// differ keeps the column while it scrolls, and a filter that leaves
// only one value hides it.
func TestSharedColumnIsDecidedOnTheWholeSection(t *testing.T) {
	m := sameBoard(t, 28, 0, map[int]string{27: "bug"}, nil)
	m.width, m.height = 140, 24
	m.resize()
	if got := len(m.rawPane(sectionDoing)); got != 28 {
		t.Fatalf("IN PROGRESS holds %d tasks", got)
	}
	for step := 0; step < 28; step++ {
		c := m.sectionLayout(sectionDoing)
		if !c.kind {
			t.Fatalf("after %d steps TYPE vanished: the bug is off screen, not out of the section", step)
		}
		if strings.Contains(ansi.Strip(m.View()), "all feature") {
			t.Fatalf("after %d steps the heading says all feature, yet one is a bug", step)
		}
		m = press(m, "j")
	}
	// Filtered to features, the section is all feature: TYPE goes.
	f := commandLine(sameBoard(t, 28, 0, map[int]string{27: "bug"}, nil), "type feature")
	f.setView("overview")
	f.pane = sectionDoing
	if s := f.sectionSame(sectionDoing); s.kind != "feature" {
		t.Fatalf("filtered to features, shared %+v", s)
	}
}

// 1bbe96 check 4: H/L skip a hidden column and o never sorts by one; a
// marker on a column that becomes hidden moves to the next drawn column.
func TestMarkerSkipsSharedColumns(t *testing.T) {
	m := sameBoard(t, 5, 0, nil, map[int]string{0: "high", 1: "low"})
	m.width, m.height = 140, 30
	m.resize()
	c := m.columnLayout()
	if c.kind || c.status || !c.priority {
		t.Fatalf("layout %+v, want TYPE and STAGE hidden, PRIORITY drawn", c)
	}
	var marked []string
	for n := 0; n < len(gridColumns)+1; n++ {
		m = press(m, "L")
		if i := m.markedColumn(); i >= 0 {
			marked = append(marked, gridColumns[i].field)
		} else {
			marked = append(marked, "none") // the no-column stop past the last (hktdgs)
		}
	}
	for _, f := range marked {
		if f == "type" || f == "stage" {
			t.Fatalf("L stopped on the hidden %s: %v", f, marked)
		}
	}
	if !strings.Contains(strings.Join(marked, " "), "priority") {
		t.Fatalf("L never reached PRIORITY: %v", marked)
	}
	// A marker left on TYPE (from a section that drew it) orders by the
	// next drawn column, priority, never type (911j70: ✓ type priority).
	m.column = 4 // type
	if got := gridColumns[m.markedColumn()].field; got != "priority" {
		t.Fatalf("a marker on hidden TYPE stands on %s", got)
	}
	m = press(m, "o")
	if len(m.sort) != 1 || m.sort[0].field != "priority" {
		t.Fatalf("o on hidden TYPE sorted %+v", m.sort)
	}
	if hint := m.columnHint(); strings.Contains(hint, "type") {
		t.Fatalf("the hint names the hidden column: %q", hint)
	}
	if f, ok := m.currentColumn(); !ok || f.name != "priority" {
		t.Fatalf("sc would sort by %+v", f)
	}
}
