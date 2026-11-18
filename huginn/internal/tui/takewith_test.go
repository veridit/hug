package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// byj479 check 2 (Jørgen 20:37Z, sql_saga: bqtzce sat in NEXT UP "ready"
// with 2/2 ticked while sabertooth worked it with dee3fq): a task taken
// with the owner's active one is in IN PROGRESS beside it, marked
// "with ID", and the section counts both. Golden at 100x24.
func TestTakenWithShowsInProgress(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := stageBoard(t)
	if _, err := m.store.TakeWith("ffffff", "aaaaaa", "nautilus"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	m.width, m.height = 100, 24
	m.resize()
	m.setView("overview")
	view := ansi.Strip(m.View())
	doing := section(view, "IN PROGRESS")
	if !strings.Contains(doing, "[2]") || !strings.Contains(doing, "aaaaaa") || !strings.Contains(doing, "ffffff") {
		t.Fatalf("IN PROGRESS does not hold and count both:\n%s", view)
	}
	var row string
	for _, l := range strings.Split(doing, "\n") {
		if strings.Contains(l, "ffffff") {
			row = l
		}
	}
	if !strings.Contains(row, "with aaaaaa") {
		t.Fatalf("ffffff's row does not say with aaaaaa: %q\n%s", row, view)
	}
	if strings.Contains(section(view, "NEXT UP"), "ffffff") {
		t.Fatalf("ffffff still in NEXT UP:\n%s", view)
	}
	goldenFile(t, "testdata/overview/taken-with-100.txt", "== overview at 100x24\n"+view+"\n")
}
