package tui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// findBoard is Jørgen's 6ephqk shape (4s3g5q): a task with long notes,
// 'row layout' three times in its text (a check, a note, and a note line
// a wrap splits), and a second task, so a find that became a list filter
// would hide it.
func findBoard(t *testing.T) (model, string) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	// The board clock dates the tasks (z25pb2): pinned, so the page's
	// created and updated are the same on every run.
	oldNow := task.Now
	task.Now = func() time.Time { return time.Date(2026, 1, 31, 23, 59, 59, 900_000_000, time.Local) }
	t.Cleanup(func() { task.Now = oldNow })
	s := &task.Store{Root: t.TempDir()}
	r, err := s.New(task.NewOptions{Title: "Rows on the board", Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"The row layout keeps its columns"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.New(task.NewOptions{Title: "Another task", Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}}); err != nil {
		t.Fatal(err)
	}
	for _, note := range []string{
		"First pass: the Row Layout is measured once per section.",
		"Second pass, at 80 columns the wrap takes the words of this note a long way along, past the edge of the screen it is drawn on, and a few words more, row layout",
		"Third: nothing to find here.",
	} {
		if _, err := s.Note(r.Task.ID, note); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 80, 24
	m.resize()
	m.setView("list")
	return openTask(t, m, r.Task.ID), r.Task.ID
}

func typeFind(m model, text string) model {
	m = press(m, "/")
	for _, r := range text {
		m = press(m, string(r))
	}
	return m
}

// 4s3g5q check 1: '/ row lay' in a task tints each match, the one a wrap
// splits in both its parts, and scrolls the first into view.
func TestFindInTaskTintsEveryMatch(t *testing.T) {
	m, id := findBoard(t)
	m = typeFind(m, "row lay")
	if m.prompt != findPrompt {
		t.Fatalf("/ in a task opened %q, not the find", m.prompt)
	}
	// Three matches, the wrapped one in two pieces on two lines.
	if len(m.find.hits) != 3 || len(m.find.hits[2].spans) != 2 {
		t.Fatalf("hits = %+v, want 3 matches, the last in two pieces", m.find.hits)
	}
	if got := m.find.hits[2].spans[1].line - m.find.hits[2].line; got != 1 {
		t.Fatalf("the wrapped match is not on two following lines: %+v", m.find.hits)
	}
	screen := ansi.Strip(m.View())
	if !strings.Contains(screen, "match 1 of 3") {
		t.Fatalf("footer does not count the matches:\n%s", screen)
	}
	// Each piece is drawn in the find's tint, the first in the strong one
	// (colour on, so the tint is in the bytes).
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { m.styles.renderer.SetColorProfile(termenv.Ascii) })
	page := m.tintHits(m.pageText())
	for i, piece := range []string{"row lay", "Row Lay", "row", "lay"} {
		style := m.styles.column
		if i == 0 {
			style = m.styles.columnCursor
		}
		if !strings.Contains(page, style.Render(piece)) {
			t.Fatalf("piece %d %q is not tinted:\n%q", i, piece, page)
		}
	}
	goldenFile(t, "testdata/detail/find.txt", strings.ReplaceAll(screen, id, "TASKID")+"\n")
}

func TestFindIncludesTitle(t *testing.T) {
	m, _ := findBoard(t)
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { m.styles.renderer.SetColorProfile(termenv.Ascii) })
	m = typeFind(m, "Rows on the board")
	if len(m.find.hits) != 1 || m.find.hits[0].line != -1 {
		t.Fatalf("title was not found: %+v", m.find.hits)
	}
	sel, _ := m.selected()
	if !strings.Contains(strings.Join(m.detailHeader(sel), "\n"), m.styles.columnCursor.Render("Rows on the board")) {
		t.Fatal("title match not highlighted in its fixed band")
	}
	m = press(m, "enter", "n", "N")
	if !strings.Contains(m.message, "match 1 of 1") || m.viewport.YOffset != 0 {
		t.Fatalf("title navigation: %q, offset %d", m.message, m.viewport.YOffset)
	}
}

func TestFindUsesDisplayCells(t *testing.T) {
	hits := findIn("界 row  \nlayout", "row lay")
	if len(hits) != 1 || len(hits[0].spans) != 2 {
		t.Fatalf("wrapped wide-character match: %+v", hits)
	}
	if got := hits[0].spans[0]; got.from != 3 || got.to != 6 {
		t.Fatalf("first span = %+v, want cells 3..6", got)
	}
}

// 4s3g5q check 2: n and N step through the matches and wrap, and the
// footer says which one.
func TestFindStepsWithNAndN(t *testing.T) {
	m, _ := findBoard(t)
	m = press(typeFind(m, "row lay"), "enter")
	if m.prompt != "" || !m.detail {
		t.Fatalf("Enter did not keep the matches in the task: prompt %q", m.prompt)
	}
	want := []string{"match 2 of 3", "match 3 of 3", "match 1 of 3"}
	for _, w := range want {
		m = press(m, "n")
		if !strings.Contains(m.message, w) {
			t.Fatalf("n: footer %q, want %q", m.message, w)
		}
		if h := m.find.hits[m.find.at]; h.line < m.viewport.YOffset || h.line >= m.viewport.YOffset+m.viewport.Height {
			t.Fatalf("%s is off the page: line %d, page %d+%d", w, h.line, m.viewport.YOffset, m.viewport.Height)
		}
	}
	m = press(m, "N")
	if !strings.Contains(m.message, "match 3 of 3") {
		t.Fatalf("N from the first did not wrap to the last: %q", m.message)
	}
	if !strings.Contains(plainMarks(ansi.Strip(m.View())), "next-, N previous match · / find again · Esc clear") {
		t.Fatalf("the key bar does not name n while a find is on:\n%s", ansi.Strip(m.View()))
	}
	// Smart case: an upper-case letter matches exactly.
	m = press(m, "esc")
	m = typeFind(m, "Row Layout")
	if len(m.find.hits) != 1 {
		t.Fatalf("'Row Layout' found %d, want only the exact one", len(m.find.hits))
	}
}

// 4s3g5q check 3: no match says so and Enter runs today's task search.
func TestFindNoMatchOffersTaskSearch(t *testing.T) {
	m, id := findBoard(t)
	m = typeFind(m, "another")
	if want := "0 matches in " + id + " · Enter searches all tasks"; !strings.Contains(ansi.Strip(m.View()), want) {
		t.Fatalf("footer lacks %q:\n%s", want, ansi.Strip(m.View()))
	}
	m = press(m, "enter")
	if m.prompt != "/" || m.detail {
		t.Fatalf("Enter did not run the task search: prompt %q detail %v", m.prompt, m.detail)
	}
	if sel, ok := m.selected(); !ok || sel.Title != "Another task" {
		t.Fatalf("the task search did not find 'Another task': %+v %q", sel, m.message)
	}
}

// 4s3g5q check 4: Esc clears the tints and keeps the page where it is;
// the next Esc goes back.
func TestFindEscClearsAndStays(t *testing.T) {
	m, id := findBoard(t)
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { m.styles.renderer.SetColorProfile(termenv.Ascii) })
	m = press(typeFind(m, "row lay"), "enter", "n", "n")
	y := m.viewport.YOffset
	if v := m.viewport.View(); !strings.Contains(v, m.styles.column.Render("row lay")) && !strings.Contains(v, m.styles.columnCursor.Render("row lay")) {
		t.Fatal("no tint before Esc")
	}
	m = press(m, "esc")
	if m.find.text != "" || !m.detail {
		t.Fatalf("Esc did not clear the find in the task: %+v detail %v", m.find, m.detail)
	}
	if page := m.viewport.View(); len(m.find.hits) != 0 || strings.Contains(page, m.styles.column.Render("row")) || strings.Contains(page, m.styles.columnCursor.Render("row")) {
		t.Fatal("tints left after Esc")
	}
	if m.viewport.YOffset != y {
		t.Fatalf("Esc moved the page: %d → %d", y, m.viewport.YOffset)
	}
	if sel, _ := m.selected(); sel.ID != id {
		t.Fatalf("Esc moved off the task: %s", sel.ID)
	}
	// Esc in the prompt clears too.
	m = press(typeFind(m, "row"), "esc")
	if m.prompt != "" || m.find.text != "" || !m.detail {
		t.Fatalf("Esc in the find prompt: prompt %q find %+v detail %v", m.prompt, m.find, m.detail)
	}
	m = press(m, "esc")
	if m.detail {
		t.Fatal("the second Esc did not go back")
	}
}

// 4s3g5q check 7: a find never becomes a list filter. After u the list
// shows every task with no '[/ …]' line (Jørgen 23:30).
func TestFindNeverFiltersTheList(t *testing.T) {
	m, _ := findBoard(t)
	// Tab was the list's 'keep as filter' (Jørgen's 23:30 screenshot).
	for _, keys := range [][]string{{"tab"}, {"tab", "enter"}, {"enter"}, {"esc"}, {"enter", "n", "u"}} {
		m2 := press(typeFind(m, "row lay"), keys...)
		if m2.prompt == findPrompt {
			m2 = press(m2, "esc") // u in the prompt is a letter
		}
		if m2.detail {
			m2 = press(m2, "u")
		}
		if m2.query != "" {
			t.Fatalf("after %v the list query is %q", keys, m2.query)
		}
		screen := ansi.Strip(m2.View())
		if strings.Contains(screen, "[/") || !strings.Contains(screen, "Another task") || !strings.Contains(screen, "Rows on the board") {
			t.Fatalf("after %v the list is filtered:\n%s", keys, screen)
		}
	}
	// In the list, / still finds tasks.
	m = press(m, "u", "/")
	if m.prompt != "/" {
		t.Fatalf("/ in the list opened %q", m.prompt)
	}
}

// Jørgen 23:29: / finds in a file the viewer shows too; n / N step its
// matches, and with no find on n still steps files.
func TestFindInGlancedFile(t *testing.T) {
	m, _ := glanceModel(t, 120, 30)
	m = press(toFiles(t, openTask(t, m, "000001")), " ", "n", "n", "n") // escapes.log
	if !m.glance {
		t.Fatal("no file shown")
	}
	m = typeFind(m, "long")
	if m.prompt != findPrompt || len(m.find.hits) < 2 {
		t.Fatalf("/ in the file: prompt %q hits %d", m.prompt, len(m.find.hits))
	}
	total := len(m.find.hits)
	m = press(m, "enter", "n")
	if !m.glance || !strings.Contains(m.message, "match 2 of ") {
		t.Fatalf("n in the file did not step the match: glance %v %q", m.glance, m.message)
	}
	m = press(m, "N", "N")
	if want := "match " + strconv.Itoa(total) + " of "; !strings.Contains(m.message, want) {
		t.Fatalf("N did not wrap back: %q", m.message)
	}
	head := m.glanceHead
	m = press(m, "esc")
	if !m.glance || m.find.text != "" || m.glanceHead != head {
		t.Fatalf("Esc did not just clear the find in the file: glance %v find %+v", m.glance, m.find)
	}
	// With no find on, n steps files again.
	if m = press(m, "n"); m.glanceHead == head {
		t.Fatal("n did not step to the next file once the find was cleared")
	}
	if m.query != "" {
		t.Fatalf("the file's find became a list query: %q", m.query)
	}
}

// 4s3g5q check 5: ? in a task and in the find name the find keys.
func TestFindHelp(t *testing.T) {
	m, _ := findBoard(t)
	if s := ansi.Strip(press(m, "?").View()); !strings.Contains(s, "find in this task") {
		t.Fatalf("task help lacks the find:\n%s", s)
	}
	if s := ansi.Strip(press(typeFind(m, "row"), "f1").View()); !strings.Contains(s, "IN / FIND IN THIS TASK") {
		t.Fatalf("find help lacks its section:\n%s", s)
	}
}
