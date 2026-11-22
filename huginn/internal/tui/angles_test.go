package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Check 2 of heype5: a note with <dir> and <rev> shows both in the detail
// view, as written. Glamour read them as raw HTML and dropped them.
func TestDetailShowsAngleBracketsLiterally(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 140, 40
	m.resize()
	alpha := m.visible[0]
	note := "(git worktree add --detach <dir> <rev>)"
	if _, err := m.store.Note(alpha.ID, note); err != nil {
		t.Fatal(err)
	}
	m.refresh(alpha.Path)
	m = selectRow(t, m, alpha.ID)
	m = press(m, "enter")
	// Notes fold (8a0ybb): Shift-Tab to the NOTES heading, Enter opens it.
	for n := 0; n < 20 && !strings.Contains(detailPlain(m), "▸ NOTES"); n++ {
		m = press(m, "shift+tab")
	}
	m = press(m, "enter")
	view := ansi.Strip(m.View())
	if !m.detail || !strings.Contains(view, note) {
		t.Fatalf("note not shown as written: want %q in\n%s", note, view)
	}
}

// Only "<" that markdown would read as a tag is escaped; code is left alone.
func TestLiteralAngles(t *testing.T) {
	for in, want := range map[string]string{
		"add <dir> <rev>":         `add \<dir> \<rev>`,
		"a < b and </x> <!x <?y":  `a < b and \</x> \<!x \<?y`,
		"`code <dir>` then <dir>": "`code <dir>` then \\<dir>",
		"``a ` <b>`` x":           "``a ` <b>`` x",
		"already \\<e>":           "already \\<e>",
		"```\n<dir>\n```\n<x>":    "```\n<dir>\n```\n\\<x>",
		"unclosed ` <dir>":        "unclosed ` \\<dir>",
	} {
		if got := literalAngles(in); got != want {
			t.Errorf("literalAngles(%q) = %q, want %q", in, got, want)
		}
	}
}
