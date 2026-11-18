package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// zhvkmjBoard is a copy of zhvkmj from hug's own board as Jørgen had it at
// 19:51 on 2026-11-18 (testdata/zhvkmj-board, from hug/tasks 75320b0): asked
// of jhf on check 4 with asked_show colours-overview.ans, so the page opens
// with that capture, an overview screen whose rows carry their own "> ".
func zhvkmjBoard(t *testing.T, w, h int) model {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(filepath.Join("testdata", "zhvkmj-board"))); err != nil {
		t.Fatal(err)
	}
	m := newModel(&task.Store{Root: root}, Options{User: "jhf"})
	m.width, m.height = w, h
	m.resize()
	return openTask(t, m, "zhvkmj")
}

// zhvkmj check 5 (Jørgen 2026-11-18 19:52: "I can't navigate and put focus
// into the files section"): from the top of zhvkmj's detail, Tab reaches
// Files and the page shows the focused heading; j picks the second capture
// and Space shows it. Before the fix the stop was reached but the page
// scrolled to the asked capture's own "> " rows, so Files was never seen.
func TestZhvkmjFilesReachableOnItself(t *testing.T) {
	for _, size := range [][2]int{{140, 40}, {80, 24}} {
		m := zhvkmjBoard(t, size[0], size[1])
		m = press(m, "g") // the top of the detail
		seen := false
		for i := 0; i < 8 && m.detailPane != focusFiles; i++ {
			m = press(m, "tab")
		}
		if m.detailPane != focusFiles {
			t.Fatalf("%dx%d: Tab never focused Files", size[0], size[1])
		}
		screen := screenOf(m)
		seen = strings.Contains(screen, "▸ FILES (2)") && strings.Contains(screen, "> ⎘ colours-overview.ans")
		if !seen {
			t.Fatalf("%dx%d: Files has the focus but is not on screen:\n%s", size[0], size[1], screen)
		}
		m = press(m, "j")
		if screen := screenOf(m); !strings.Contains(screen, "> ⎘ screen-1821.ans") {
			t.Fatalf("%dx%d: j did not mark the second capture on screen:\n%s", size[0], size[1], screen)
		}
		m = press(m, " ")
		if !m.glance || !strings.Contains(strings.SplitN(screenOf(m), "\n", 2)[0], "screen-1821.ans · ") {
			t.Fatalf("%dx%d: Space did not show screen-1821.ans:\n%s", size[0], size[1], screenOf(m))
		}
	}
}
