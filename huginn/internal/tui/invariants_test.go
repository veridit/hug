package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMoveRepairsTiesAndUsesGlobalOrder(t *testing.T) {
	m := fixture(t)
	for _, v := range m.all {
		// Every task at rank A, a tie the move must repair: in layout 3
		// (w40cmk) the rank is the file's rank: line.
		p := filepath.Join(m.store.Root, filepath.FromSlash(v.Path))
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		var out []string
		for _, l := range strings.SplitAfter(string(b), "\n") {
			if strings.HasPrefix(l, "rank: ") {
				l = "rank: \"A\"\n"
			}
			out = append(out, l)
		}
		if e := os.WriteFile(p, []byte(strings.Join(out, "")), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	m.refresh("")
	first := m.visible[0].ID
	m = press(m, "m", "j", "enter")
	if m.visible[1].ID != first || strings.Contains(m.message, "error") {
		t.Fatal(m.View())
	}
	m = press(m, "m", "k", "enter")
	if m.visible[0].ID != first {
		t.Fatal(m.View())
	}
	// With a filter showing one row, there is no shown row to move past: the
	// move is refused with a message, never made past a hidden task (ksv323;
	// this used to move past the next task in the global order).
	m = press(m, "/", m.visible[0].Slug, "tab", "m")
	if m.confirm != "" || m.grab.active || !strings.Contains(m.message, "is the only row in the list (filtered): nothing to move past") {
		t.Fatal("filtered move past a hidden row", m.message)
	}
	m = press(m, "c")
	if m.visible[0].ID != first {
		t.Fatal("refused move changed the order", m.View())
	}
}

func TestCombinedFiltersAndExternalRefresh(t *testing.T) {
	m := fixture(t)
	m = commandLine(m, "state todo")
	m = commandLine(m, "type bug")
	m = commandLine(m, "group auth")
	m = commandLine(m, "tag urgent")
	m = press(m, "/", "Alpha", "tab")
	if len(m.visible) != 1 {
		t.Fatal(m.View())
	}
	id := m.visible[0].ID
	if _, err := m.store.Drop(id, "test reason"); err != nil {
		t.Fatal(err)
	}
	m.refresh("") // r is request now (hbgybn); the board updates live
	if len(m.visible) != 0 {
		t.Fatal("external status edit not reflected", m.View())
	}
	m = press(m, "c")
	if len(m.visible) != 2 {
		t.Fatal(m.View())
	}
}

func TestAcceptanceSelectionClampedAcrossTasks(t *testing.T) {
	m := fixture(t)
	m = press(m, "enter", "]", "esc", "j", "enter", " ", "enter")
	if !m.visible[1].Acceptance[0].Checked {
		t.Fatal(m.View())
	}
}

func TestErrorAndTerminalInputSafety(t *testing.T) {
	m := fixture(t)
	m.opts.Commit = func(string) error { return os.ErrPermission }
	if got := press(m, " "); got.confirm != "" || len(got.marks) != 1 {
		// Space in a list marks the task (rxr4z8), never ticks a check.
		t.Fatalf("Space in a list must mark, not tick: confirm=%q marks=%v", got.confirm, got.marks)
	}
	m = press(m, "enter")
	m = act(m, " ")
	if m.message != os.ErrPermission.Error() {
		t.Fatal(m.message)
	}
	// A blank $EDITOR falls through to git's editor (2026-11-11), so the
	// action still launches an editor instead of refusing.
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "   ")
	m = press(m, "e")
	m, cmd := key(m, "enter")
	if cmd == nil {
		t.Fatal("blank EDITOR must fall back to an editor", m.message)
	}
	next, _ := m.Update(editorMsg{err: os.ErrPermission})
	if next.(model).message != os.ErrPermission.Error() {
		t.Fatal("editor error lost")
	}
	next, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	m = next.(model)
	_ = m.View()
	if strings.Contains(safe("\x1b[31mhello\a"), "\x1b") {
		t.Fatal("escape not sanitized")
	}
}
