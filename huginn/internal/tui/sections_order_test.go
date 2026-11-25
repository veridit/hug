package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// jnaspy check 1, 8a0ybb check 11, hya1st: the detail of a task with
// notes shows Goal, Purpose, Principle and a short Details, open, above
// Acceptance (no DESCRIPTION heading), and last Notes and History folded
// to one line each. The whole detail text is the golden, beyond one
// screen.
func TestDetailSectionOrderGolden(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 100, 40
	m.resize()
	m = press(m, "#", "000001", "enter")
	sel, ok := m.selected()
	if !ok || sel.ID != "000001" || !m.detail {
		t.Fatalf("000001 not open: %q", m.message)
	}
	text := ansi.Strip(m.detailText(sel))
	var heads []string
	for _, line := range strings.Split(text, "\n") {
		for _, h := range []string{"DESCRIPTION", "PURPOSE", "PRINCIPLE", "GOAL", "RELEVANT DETAILS", "DETAILS", "ACCEPTANCE", "DEPENDENCIES", "FILES (2)", "NOTES", "HISTORY"} {
			if strings.HasPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "▸ "), "› "), h) {
				heads = append(heads, h)
			}
		}
	}
	want := "GOAL|PURPOSE|PRINCIPLE|DETAILS|ACCEPTANCE|FILES (2)|NOTES|HISTORY"
	if got := strings.Join(heads, "|"); got != want {
		t.Fatalf("sections %s, want %s\n%s", got, want, text)
	}
	path := filepath.Join("testdata", "detail", "sections-000001-100.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	golden, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestDetailSectionOrderGolden ./huginn/internal/tui/", err)
	}
	if text != strings.ReplaceAll(string(golden), "\r\n", "\n") {
		t.Fatalf("detail differs from %s:\n%s", path, text)
	}
}
