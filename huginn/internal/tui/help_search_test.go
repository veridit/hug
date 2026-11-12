package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/editor"
)

func TestHelpSearchFindsOneActionAndEscBacksOut(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 140, 40
	m.resize()
	m = press(m, "?")
	m = press(m, "/")
	for _, r := range "drop" {
		m = press(m, string(r))
	}
	v := m.View()
	if !strings.Contains(v, "take / done / drop task") || strings.Contains(v, "half page down") {
		t.Fatal("help search did not narrow to the drop row\n", v)
	}
	if !strings.Contains(v, "1 of ") {
		t.Fatal("help search must count matches\n", v)
	}
	m = press(m, "enter")
	if !strings.Contains(m.View(), "take / done / drop task") || !m.help {
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
	m = press(m, "o")
	v := m.View()
	if !strings.Contains(v, "[overview]") || !strings.Contains(v, " list ") || !strings.Contains(v, " board ") || !strings.Contains(v, " !lint ") {
		t.Fatal("tabs must read [overview] list board !lint with no separate key column\n", v)
	}
	m = press(m, "i")
	if !strings.Contains(m.View(), "[list]") {
		t.Fatal("i opens list\n", m.View())
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
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[core]\n\teditor = nano -w\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(editor.Command(), " "); got != "nano -w" {
		t.Fatalf("git core.editor: got %q", got)
	}
}
