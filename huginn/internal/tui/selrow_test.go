package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// The selected row has no bar (911j70, Jørgen 2026-11-18: the row bar and
// column band are too intrusive): its id, the row header, takes the one
// selection tint, its cells keep their colours, and nothing else on the
// row has a background. (Was: the tint spans the whole row, 2026-11-16.)
func TestSelectedRowTintFullWidth(t *testing.T) {
	m := fixture(t)
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.styles.renderer.SetHasDarkBackground(true)
	tk := m.visible[0]
	row := m.taskRow(tk, 100, true, false)
	text := ansi.Strip(row)
	id := strings.TrimRight(fit(oneLine(tk.ID), idCell-1), " ") // as the row cuts it
	if !strings.HasPrefix(string([]rune(text)[rowHead:]), id) {
		t.Fatalf("the id does not stand at %d: %q", rowHead, text)
	}
	for col := 0; col < lipgloss.Width(text); col++ {
		in := col >= rowHead && col < rowHead+lipgloss.Width(id)
		if c := sgrAt(row, col); c.bg != in || c.underline {
			t.Fatalf("column %d %q: background %v underline %v, want background only on the id: %q", col, cellAt(row, col), c.bg, c.underline, row)
		}
	}
}
