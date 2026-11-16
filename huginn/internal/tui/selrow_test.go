package tui

import (
	"strings"
	"testing"

	"github.com/muesli/termenv"
)

// The selected row's tint spans the whole row: no reset code before the end
// (2026-11-16: cell resets cut the tint after the first coloured cell).
func TestSelectedRowTintFullWidth(t *testing.T) {
	m := fixture(t)
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	row := m.taskRow(m.visible[0], 100, true, false)
	body := strings.TrimSuffix(row, "\x1b[0m")
	if strings.Contains(body, "\x1b[0m") {
		t.Fatalf("reset inside selected row: %q", row)
	}
	if !strings.Contains(row, "48;2;22;54;74") && !strings.Contains(row, "48;2;") {
		t.Fatalf("no background on selected row: %q", row)
	}
}
