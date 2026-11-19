package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/editor"
)

func TestHelpSearchFindsOneActionAndEscBacksOut(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 140, 40
	m.resize()
	m = press(m, "?")
	m = press(m, "/")
	for _, r := range "drop the" {
		m = press(m, string(r))
	}
	v := m.View()
	if !strings.Contains(v, "take / done / drop the task") || strings.Contains(v, "half page down") {
		t.Fatal("help search did not narrow to the drop row\n", v)
	}
	if !strings.Contains(v, "1 of ") {
		t.Fatal("help search must count matches\n", v)
	}
	m = press(m, "enter")
	if !strings.Contains(m.View(), "take / done / drop the task") || !m.help {
		t.Fatal("Enter must keep the search and stay in help")
	}
	m = press(m, "esc")
	if !m.help || m.helpQuery != "" || !strings.Contains(m.View(), "half page down") {
		t.Fatal("first Esc clears the search and keeps help open")
	}
	m = press(m, "esc")
	if m.help {
		t.Fatal("second Esc closes help")
	}
}

func TestTabsMarkShortcutWithoutExtraWidth(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := fixture(t)
	m.width, m.height = 140, 40
	m.styles = newTheme()
	m.resize()
	m = press(m, "v", "o")
	v := ansi.Strip(m.View()) // NO_COLOR keeps the view line's bold and underlines (r80svs)
	if !strings.Contains(v, "view: overview  list  board  !lint") {
		t.Fatal("the view line must read view: overview  list  board  !lint (qh9ep1)\n", v)
	}
	if m = press(m, "v", "l"); m.view != "list" {
		t.Fatal("v l opens the list\n", m.View())
	}
}

func TestEditorFollowsVisualEditorThenGit(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "zed --wait")
	if got := strings.Join(editor.Command(), " "); got != "zed --wait" {
		t.Fatalf("EDITOR: got %q", got)
	}
	t.Setenv("VISUAL", "code -w")
	if got := strings.Join(editor.Command(), " "); got != "code -w" {
		t.Fatalf("VISUAL wins: got %q", got)
	}
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// GIT_CONFIG_GLOBAL overrides $HOME/.gitconfig (set on CI hosts).
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[core]\n\teditor = nano -w\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(editor.Command(), " "); got != "nano -w" {
		t.Fatalf("git core.editor: got %q", got)
	}
}
