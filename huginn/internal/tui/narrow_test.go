package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// No rendered line is wider than the terminal, in any view, down to 50
// columns (measured in cells, not bytes: "·" and "─" are multi-byte).
func TestNoLineWiderThanTerminal(t *testing.T) {
	for _, w := range []int{50, 60, 70, 80} {
		m := fixture(t)
		next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 20})
		m = next.(model)
		for _, view := range []string{"overview", "list", "board", "lint"} {
			m.setView(view)
			for _, line := range strings.Split(m.View(), "\n") {
				if n := lipgloss.Width(ansi.Strip(line)); n > w {
					t.Errorf("width %d, %s: line is %d wide: %q", w, view, n, ansi.Strip(line))
				}
			}
		}
	}
}
