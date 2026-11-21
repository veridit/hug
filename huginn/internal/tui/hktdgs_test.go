package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// hktdgs (Jørgen 2026-11-20 12:07Z): H/L step through a ring of
// (no column), col1 ... colN. L from the last shown column and H from
// the first land on no column, the marker gone as when the view opens;
// the next press goes on to the first (L) or the last (H) column.
func TestColumnMarkerHasANoColumnStop(t *testing.T) {
	for _, view := range []string{"overview", "list", "board"} {
		t.Run(view, func(t *testing.T) { noColumnStop(t, view) })
	}
}

// noColumnStop is the check in one view.
func noColumnStop(t *testing.T, view string) {
	m := manyModel(t)
	m.width, m.height = 140, 30
	m.resize()
	m.setView(view)
	if m.markedColumn() != -1 {
		t.Fatalf("%s: a column is marked as the view opens: %d", view, m.markedColumn())
	}
	opened := body(m)
	c := m.columnLayout()
	var shown []int
	for i, col := range gridColumns {
		if col.shown(c) {
			shown = append(shown, i)
		}
	}
	if len(shown) < 2 {
		t.Fatalf("%s: %d columns shown, want at least 2", view, len(shown))
	}
	first, last := shown[0], shown[len(shown)-1]

	// L walks col1 ... colN, then lands on no column, then col1.
	for n, want := range shown {
		m = press(m, "L")
		if got := m.markedColumn(); got != want {
			t.Fatalf("%s: L %d marks %d, want %d", view, n+1, got, want)
		}
	}
	m = press(m, "L")
	if got := m.markedColumn(); got != -1 {
		t.Fatalf("%s: L from the last column (%s) marks %s, want no column", view, gridColumns[last].field, gridColumns[got].field)
	}
	if got := body(m); got != opened {
		t.Fatalf("%s: no column after L does not read as the view opened:\n%s\nwant\n%s", view, got, opened)
	}
	if f := footerLine(m); !strings.Contains(f, "H/L pick a column") {
		t.Fatalf("%s: the footer on no column says %q", view, f)
	}
	if _, ok := m.currentColumn(); ok {
		t.Fatalf("%s: the sort picker's c still has a column on no column", view)
	}
	m = press(m, "L")
	if got := m.markedColumn(); got != first {
		t.Fatalf("%s: L from no column marks %d, want the first %d", view, got, first)
	}

	// H from the first lands on no column, then on the last.
	m = press(m, "H")
	if got := m.markedColumn(); got != -1 {
		t.Fatalf("%s: H from the first column (%s) marks %s, want no column", view, gridColumns[first].field, gridColumns[got].field)
	}
	if got := body(m); got != opened {
		t.Fatalf("%s: no column after H does not read as the view opened:\n%s\nwant\n%s", view, got, opened)
	}
	m = press(m, "H")
	if got := m.markedColumn(); got != last {
		t.Fatalf("%s: H from no column marks %d, want the last %d", view, got, last)
	}
	for n := len(shown) - 2; n >= 0; n-- {
		m = press(m, "H")
		if got := m.markedColumn(); got != shown[n] {
			t.Fatalf("%s: H back to %d marks %d", view, shown[n], got)
		}
	}
	if m = press(m, "H"); m.markedColumn() != -1 {
		t.Fatalf("%s: H past the first again marks %d", view, m.markedColumn())
	}
}

// body is the view stripped, without its footer and the two lines under
// it: on no column the footer says what H/L do, as the hint does after
// any H/L.
func body(m model) string {
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	return strings.Join(lines[:len(lines)-3], "\n")
}
