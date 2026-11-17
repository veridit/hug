package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// headerLine is the first line of view that holds every word, stripped.
func headerLine(t *testing.T, view string, words ...string) string {
	t.Helper()
	for _, l := range strings.Split(ansi.Strip(view), "\n") {
		ok := true
		for _, w := range words {
			ok = ok && strings.Contains(l, w)
		}
		if ok {
			return l
		}
	}
	t.Fatalf("no line with %q:\n%s", words, ansi.Strip(view))
	return ""
}

// bejd89 check 1: Tab and Shift-Tab move the column cursor (the header
// marks it, underlined) and h/l still change section. Golden at 80 and
// 140 columns.
func TestColumnCursorMarksHeader(t *testing.T) {
	var golden []string
	for _, w := range []int{80, 140} {
		m := manyModel(t)
		m.width, m.height = w, 30
		m.resize()
		if m.column != -1 {
			t.Fatalf("a column is marked before Tab: %d", m.column)
		}
		m = press(m, "tab")
		if gridColumns[m.column].label != "✓" || !strings.Contains(m.message, "o order by ✓") {
			t.Fatalf("first Tab marks %q, footer %q", gridColumns[m.column].label, m.message)
		}
		m = press(m, "tab", "tab", "tab") // ✓ id type priority
		if gridColumns[m.column].field != "priority" {
			t.Fatalf("at %d three more Tabs mark %s", w, gridColumns[m.column].field)
		}
		// The marked header starts with ▸, also without colour; no other does.
		if line := headerLine(t, m.View(), "ID", "TITLE"); !strings.Contains(line, "▸PRIOR") || strings.Count(line, "▸") != 1 {
			t.Fatalf("at %d the header does not mark priority: %q", w, line)
		}
		m = press(m, "shift+tab")
		if gridColumns[m.column].field != "type" {
			t.Fatalf("Shift-Tab marks %s", gridColumns[m.column].field)
		}
		// h/l change section, not the column.
		m.setView("overview")
		pane, col := m.pane, m.column
		m = press(m, "l")
		if m.pane == pane || m.column != col {
			t.Fatalf("l: pane %d -> %d, column %d -> %d", pane, m.pane, col, m.column)
		}
		m = press(m, "h")
		if m.pane != pane {
			t.Fatalf("h: pane %d, want %d", m.pane, pane)
		}
		golden = append(golden, "== list at "+strconv.Itoa(w), ansi.Strip(press(func() model { m.setView("list"); return m }(), "o").View()))
	}
	text := strings.Join(golden, "\n") + "\n"
	path := filepath.Join("testdata", "columns", "cursor.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestColumnCursorMarksHeader ./huginn/internal/tui/", err)
	}
	if text != string(want) {
		t.Fatalf("differs from %s:\n%s", path, text)
	}
}

// bejd89 check 2: o cycles ▲, ▼ and off on the marked column; in a
// section the rows reorder within it; off restores rank order. o on
// another column replaces the order.
func TestColumnOrderCycles(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	rank := ids(m)
	m = press(m, "o")
	if len(m.sort) != 0 || !strings.Contains(m.message, "Tab picks a column") {
		t.Fatalf("o without a column: sort %v, %q", m.sort, m.message)
	}
	m = press(m, "tab", "tab", "tab", "tab") // priority
	m = press(m, "o")
	if len(m.sort) != 1 || m.sort[0] != (sortKey{"priority", false}) || !strings.Contains(headerLine(t, m.View(), "ID", "TITLE"), "PRIOR ▲") {
		t.Fatalf("o once: %v\n%s", m.sort, headerLine(t, m.View(), "ID", "TITLE"))
	}
	asc := m.visible[0].Priority
	m = press(m, "o")
	if m.sort[0] != (sortKey{"priority", true}) || !strings.Contains(headerLine(t, m.View(), "ID", "TITLE"), "PRIOR ▼") || m.visible[0].Priority != "critical" || asc != "low" {
		t.Fatalf("o twice: %v first %s (asc first %s)", m.sort, m.visible[0].Priority, asc)
	}
	m = press(m, "o")
	if len(m.sort) != 0 || strings.Join(ids(m), " ") != strings.Join(rank, " ") || strings.Contains(headerLine(t, m.View(), "ID", "TITLE"), "▲") {
		t.Fatalf("o three times: sort %v, rank restored %v", m.sort, strings.Join(ids(m), " ") == strings.Join(rank, " "))
	}
	// Another column replaces the order.
	m = press(m, "o", "shift+tab", "o") // priority ▲, then type ▲
	if len(m.sort) != 1 || m.sort[0].field != "type" {
		t.Fatalf("o on another column: %v", m.sort)
	}
	// In the overview a section reorders within itself.
	m.applySort(nil)
	m.setView("overview")
	m.column = -1
	m = press(m, "tab", "tab", "tab", "tab", "o", "o") // priority ▼
	p := m.paneIndices(m.pane)
	for n := 1; n < len(p); n++ {
		if m.compareBy("priority", m.visible[p[n-1]], m.visible[p[n]]) < 0 {
			t.Fatalf("section rows not by priority ▼ at %d", n)
		}
	}
	if !strings.Contains(headerLine(t, m.View(), "ID", "TITLE"), "PRIOR ▼") {
		t.Fatalf("overview has no marked header:\n%s", ansi.Strip(m.View()))
	}
}

func ids(m model) []string {
	var out []string
	for _, t := range m.visible {
		out = append(out, t.ID)
	}
	return out
}

// bejd89 check 3: the order shows in the header and the footer, and
// survives a reload (the handoff) and a filter change.
func TestColumnOrderSurvives(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	m = press(m, "tab", "tab", "tab", "tab", "o")
	if !strings.Contains(m.message, "o order by priority (▼)") {
		t.Fatalf("footer: %q", m.message)
	}
	m.setFilter("type", "feature")
	m.applyFilter()
	if len(m.sort) != 1 || !strings.Contains(headerLine(t, m.View(), "ID", "TITLE"), "PRIOR ▲") {
		t.Fatalf("after a filter: %v", m.sort)
	}
	h := m.handoff()
	fresh := newModel(m.store, m.opts)
	fresh.width, fresh.height = 140, 40
	fresh.resize()
	fresh.resume(h)
	if fresh.column != m.column || len(fresh.sort) != 1 || fresh.sort[0] != m.sort[0] || !strings.Contains(headerLine(t, fresh.View(), "ID", "TITLE"), "PRIOR ▲") {
		t.Fatalf("after reload: column %d sort %v", fresh.column, fresh.sort)
	}
}

// bejd89 check 4: o no longer opens the overview; v o does (and v i, v b,
// v !), also from a task; help lists both.
func TestViewKeysMoveToV(t *testing.T) {
	m := manyModel(t)
	m = press(m, "o")
	if m.view != "list" {
		t.Fatalf("o switched view to %s", m.view)
	}
	m = press(m, "v", "o")
	if m.view != "overview" {
		t.Fatalf("v o: %s", m.view)
	}
	m = press(m, "v", "i")
	if m.view != "list" {
		t.Fatalf("v i: %s", m.view)
	}
	m = press(m, "v", "b")
	if m.view != "board" {
		t.Fatalf("v b: %s", m.view)
	}
	m = press(m, "enter", "v", "o")
	if m.detail || m.view != "overview" {
		t.Fatalf("v o from a task: detail %v view %s", m.detail, m.view)
	}
	help := ansi.Strip(m.helpText())
	for _, want := range []string{"v o/i/b/!", "views: overview / list / board / lint", "Tab, o", "o orders: ▲ ▼ off"} {
		if !strings.Contains(strings.Join(strings.Fields(help), " "), want) {
			t.Fatalf("help lacks %q:\n%s", want, help)
		}
	}
}
