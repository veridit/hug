package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Version corner (Jørgen, 2026-11-16): every screen's key bar ends with
// "hug vX.Y.Z", right-aligned; a narrow bar drops it before any hint.
func TestVersionCorner(t *testing.T) {
	m := manyModel(t)
	m.opts.Version = "v9.9.9"
	for _, size := range [][2]int{{140, 40}, {100, 30}, {80, 24}} {
		m.width, m.height = size[0], size[1]
		m.resize()
		screens := map[string]model{
			"list":    m,
			"help":    press(m, "?"),
			"palette": press(m, ":"),
			"detail":  press(m, "enter"),
			"sort":    press(m, ":", "sort", "enter"),
		}
		for name, s := range screens {
			lines := strings.Split(s.View(), "\n")
			last := ansi.Strip(lines[len(lines)-1])
			bare := s
			bare.opts.Version = ""
			hints := ansi.Strip(bare.keyBar())
			if w := lipgloss.Width(last); w > s.width {
				t.Fatalf("%dx%d %s: key bar %d wide", s.width, s.height, name, w)
			}
			if !strings.HasPrefix(last, strings.TrimRight(hints, " ")) {
				t.Fatalf("%dx%d %s: the version pushed a hint off:\n%q\n%q", s.width, s.height, name, last, hints)
			}
			room := s.width-lipgloss.Width(hints) >= len("hug v9.9.9")+2
			if got := strings.HasSuffix(strings.TrimRight(last, " "), "hug v9.9.9"); got != room {
				t.Fatalf("%dx%d %s: room %v, version shown %v:\n%q", s.width, s.height, name, room, got, last)
			}
		}
	}
	// Far too narrow: the hints keep the line, the version goes.
	m.width = 40
	if strings.Contains(ansi.Strip(m.keyBar()), "hug v9.9.9") {
		t.Fatalf("width 40: %q", ansi.Strip(m.keyBar()))
	}
}

// At 140 columns every list view's key bar leaves room for the version
// corner, a long one too (rxr4z8, crab 21:26): the next hint that crosses
// it fails here instead of silently dropping the version.
func TestWideBarKeepsTheVersion(t *testing.T) {
	m := manyModel(t)
	m.opts.Version = "v0.10.100"
	m.width, m.height = 140, 40
	m.resize()
	for _, view := range []string{"overview", "list", "board", "lint"} {
		v := m
		v.setView(view)
		if bar := strings.TrimRight(ansi.Strip(v.keyBar()), " "); !strings.HasSuffix(bar, "hug v0.10.100") {
			t.Fatalf("%s at 140: the key bar has no room for the version (%d wide):\n%q", view, lipgloss.Width(bar), bar)
		}
	}
}
