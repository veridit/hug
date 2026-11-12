package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

func TestMoveRepairsTiesAndUsesGlobalOrder(t *testing.T) {
	m := fixture(t)
	for _, v := range m.all {
		dest := filepath.Join(m.store.Root, v.Group, task.BuildName("A", v.ID, v.Slug))
		if e := os.Rename(filepath.Join(m.store.Root, v.Path), dest); e != nil {
			t.Fatal(e)
		}
	}
	m.refresh("")
	first := m.visible[0].ID
	m = act(m, "J")
	if m.visible[1].ID != first || strings.Contains(m.message, "error") {
		t.Fatal(m.View())
	}
	m = act(m, "K")
	if m.visible[0].ID != first {
		t.Fatal(m.View())
	}
	m = press(m, "/", m.visible[0].Slug, "enter", "J", "enter", "c")
	if m.visible[1].ID != first {
		t.Fatal("filtered move did not use global order", m.View())
	}
}

func TestCombinedFiltersAndExternalRefresh(t *testing.T) {
	m := fixture(t)
	m = commandLine(m, "status todo")
	m = commandLine(m, "type bug")
	m = commandLine(m, "group auth")
	m = commandLine(m, "label urgent")
	m = press(m, "/", "Alpha", "enter")
	if len(m.visible) != 1 {
		t.Fatal(m.View())
	}
	id := m.visible[0].ID
	if _, err := m.store.Drop(id); err != nil {
		t.Fatal(err)
	}
	m = press(m, "r")
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
	m = press(m, "enter", "]", "esc", "j", " ", "enter")
	if !m.visible[1].Acceptance[0].Checked {
		t.Fatal(m.View())
	}
}

func TestErrorAndTerminalInputSafety(t *testing.T) {
	m := fixture(t)
	m.opts.Commit = func(string) error { return os.ErrPermission }
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
	next, _ := m.Update(editorMsg{os.ErrPermission})
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
