package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/huginn/internal/task"
)

func fixture(t *testing.T) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	for _, o := range []task.NewOptions{
		{Title: "Alpha task", Goal: "Goal alpha", Acceptance: []string{"first check", "second check"}, Type: "bug", Group: "auth", Labels: []string{"urgent"}},
		{Title: "Beta task", Goal: "Goal beta", Acceptance: []string{"verify beta"}, Type: "docs"},
	} {
		if _, err := s.New(o); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel(s, Options{User: "tester"})
	m.setView("list")
	return m
}
func key(m model, k string) (model, tea.Cmd) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	codes := map[string]tea.KeyType{"enter": tea.KeyEnter, "esc": tea.KeyEsc, "backspace": tea.KeyBackspace, "pgdown": tea.KeyPgDown, "pgup": tea.KeyPgUp, "home": tea.KeyHome, "end": tea.KeyEnd, "up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight, "tab": tea.KeyTab, "shift+tab": tea.KeyShiftTab, "ctrl+d": tea.KeyCtrlD, "ctrl+u": tea.KeyCtrlU, "ctrl+n": tea.KeyCtrlN, "ctrl+p": tea.KeyCtrlP, "ctrl+c": tea.KeyCtrlC}
	if code, ok := codes[k]; ok {
		msg = tea.KeyMsg{Type: code}
	}
	next, cmd := m.Update(msg)
	return next.(model), cmd
}
func press(m model, ks ...string) model {
	for _, k := range ks {
		m, _ = key(m, k)
	}
	return m
}
func act(m model, k string) model             { return press(m, k, "enter") }
func commandLine(m model, value string) model { return press(m, ":", value, "enter") }
func TestOverviewFirst(t *testing.T) {
	m := fixture(t)
	m = newModel(m.store, m.opts)
	for _, s := range []string{"HUGINN · overview", "todo 2", "doing 0", "done 0", "blocked 0", "IN PROGRESS", "NEXT UP", "BLOCKED", "LINT", "Alpha task"} {
		if !strings.Contains(m.View(), s) {
			t.Fatalf("missing %q\n%s", s, m.View())
		}
	}
}
func TestNavigationViews(t *testing.T) {
	m := fixture(t)
	m = press(m, "j")
	if m.cursor != 1 {
		t.Fatal(m.cursor)
	}
	m = press(m, "k", "enter")
	if !strings.Contains(m.View(), "Goal alpha") {
		t.Fatal(m.View())
	}
	m = press(m, "]")
	if m.acceptance != 1 {
		t.Fatal(m.acceptance)
	}
	m = press(m, "[", "esc", "b")
	if m.view != "board" {
		t.Fatal(m.View())
	}
	m = press(m, "?", "pgdown", "pgup")
	if !strings.Contains(m.helpText(), "$EDITOR") {
		t.Fatal(m.View())
	}
	_, cmd := key(m, "q")
	if cmd == nil {
		t.Fatal("quit missing")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("not quit")
	}
}
func TestActions(t *testing.T) {
	m := fixture(t)
	first := m.visible[0].ID
	commits := 0
	m.opts.Commit = func(msg string) error {
		commits++
		if !strings.Contains(msg, first) {
			t.Error(msg)
		}
		return nil
	}
	m = act(m, "d")
	if !strings.Contains(m.message, "huginn check "+first+" 1") {
		t.Fatal(m.message)
	}
	m = press(m, "enter")
	m = act(m, " ")
	if !m.visible[0].Acceptance[0].Checked {
		t.Fatal("not checked")
	}
	m = act(m, " ")
	if m.visible[0].Acceptance[0].Checked {
		t.Fatal("not unchecked")
	}
	m = act(m, " ")
	m = press(m, "]")
	m = act(m, " ")
	m = act(m, "t")
	d, e := m.store.Get(first)
	if e != nil || d.Task.Status != "doing" || len(d.Task.Assignee) != 1 {
		t.Fatal(d, e)
	}
	m = act(m, "d")
	d, _ = m.store.Get(first)
	if d.Task.Status != "done" {
		t.Fatal(m.message)
	}
	m = act(m, "x")
	d, _ = m.store.Get(first)
	if d.Task.Status != "dropped" {
		t.Fatal(m.message)
	}
	if commits != 7 {
		t.Fatalf("commits=%d", commits)
	}
	m.opts.Commit = func(string) error { return errors.New("commit failed") }
	m = act(m, "t")
	if m.message != "commit failed" {
		t.Fatal(m.message)
	}
}
func TestMoveAndNew(t *testing.T) {
	m := fixture(t)
	id := m.visible[0].ID
	m = act(m, "J")
	if m.visible[1].ID != id {
		t.Fatal(m.visible)
	}
	m = act(m, "K")
	if m.visible[0].ID != id {
		t.Fatal(m.visible)
	}
	m = press(m, "n", "Gamma", "backspace", "a", "enter", "enter")
	if len(m.visible) != 3 {
		t.Fatal(m.message)
	}
	d, e := m.store.Get("gamma")
	if e != nil || strings.TrimSpace(d.Task.Goal) != "Gamma" || len(d.Task.Acceptance) != 1 {
		t.Fatal(d, e)
	}
	m = press(m, "n", "esc")
	if m.prompt != "" {
		t.Fatal(m.prompt)
	}
}
func TestFilters(t *testing.T) {
	for _, tc := range []struct {
		cmd  string
		want int
	}{{"status todo", 2}, {"type bug", 1}, {"group auth", 1}, {"label urgent", 1}} {
		t.Run(tc.cmd, func(t *testing.T) {
			m := commandLine(fixture(t), tc.cmd)
			if len(m.visible) != tc.want {
				t.Fatal(m.View())
			}
			m = press(m, "c")
			if len(m.visible) != 2 {
				t.Fatal(m.View())
			}
		})
	}
	m := press(fixture(t), "/", "Alpha")
	if len(m.visible) != 1 || m.query != "Alpha" {
		t.Fatal("search is not live", m.View())
	}
	m = press(m, "enter")
	if m.prompt != "" || len(m.visible) != 1 {
		t.Fatal(m.View())
	}
	m = press(m, "esc")
	if len(m.visible) != 2 {
		t.Fatal(m.View())
	}
}
func TestMalformedAndEditorRefresh(t *testing.T) {
	m := fixture(t)
	if e := os.WriteFile(filepath.Join(m.store.Root, "broken.md"), []byte("---\nnot: [yaml\n---\n\x1b]52;c;BAD\a"), 0600); e != nil {
		t.Fatal(e)
	}
	m = press(m, "r")
	if len(m.visible) != 3 || len(m.findings) == 0 {
		t.Fatal(m.View())
	}
	for i, v := range m.visible {
		if v.Path == "broken.md" {
			m.cursor = i
		}
	}
	m = press(m, "enter")
	if !strings.Contains(strings.Join(m.detailHeader(m.visible[m.cursor]), "\n"), "⚠") {
		t.Fatal(m.View())
	}
	for _, k := range []string{"t", "d", "x", " ", "J", "K"} {
		m = act(m, k)
		_ = m.View()
	}
	m.cursor = 0
	m.detail = false
	t.Setenv("EDITOR", "true")
	m = press(m, "e")
	_, cmd := key(m, "enter")
	if cmd == nil {
		t.Fatal("editor missing")
	}
	updated, _ := m.Update(editorMsg{})
	m = updated.(model)
	if !strings.Contains(m.message, "re-linted") {
		t.Fatal(m.message)
	}
	if strings.Contains(m.View(), "\x1b]52") {
		t.Fatal("terminal injection")
	}
}
func TestDependenciesAndAllSections(t *testing.T) {
	m := fixture(t)
	a, b := m.visible[0], m.visible[1]
	d, e := m.store.Get(a.ID)
	if e != nil {
		t.Fatal(e)
	}
	raw := strings.Replace(string(d.Bytes()), "## Depends on\n", "## Depends on\n- "+b.ID+"\n", 1)
	raw += "\n## Description\nFull prose survives\n\n## History\n2026-11-10: created\n" + strings.Repeat("History line\n", 100)
	if e = task.AtomicWrite(filepath.Join(m.store.Root, a.Path), []byte(raw)); e != nil {
		t.Fatal(e)
	}
	m.refresh(a.Path)
	m = press(m, "enter")
	text := m.detailText(m.visible[m.cursor])
	for _, v := range []string{"BLOCKED", "Full prose survives", "2026-11-10: created", "History line"} {
		if !strings.Contains(text, v) {
			t.Fatal(text)
		}
	}
	m = press(m, "G")
	if !m.viewport.AtBottom() || !strings.Contains(m.View(), "History line") {
		t.Fatal(m.View())
	}
	m = press(m, "g", "g")
	if !m.viewport.AtTop() {
		t.Fatal("gg did not scroll detail")
	}
}
func TestEmptyAndErrors(t *testing.T) {
	m := newModel(&task.Store{Root: t.TempDir()}, Options{})
	for _, k := range []string{"j", "k", "enter", " ", "t", "d", "x", "J", "K", "r", "e", "G", "g", "g", "tab"} {
		m = press(m, k)
		_ = m.View()
	}
	m.store.Root = filepath.Join(t.TempDir(), "absent")
	m = press(m, "r")
	if m.message == "" {
		t.Fatal("missing IO error")
	}
	m.width = 1
	m.height = 1
	m.resize()
	_ = m.View()
}
func TestLayoutBounds(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {30, 10}, {80, 24}, {140, 40}, {200, 60}} {
		m := fixture(t)
		m.width = size[0]
		m.height = size[1]
		m.resize()
		m.visible[0].Title = strings.Repeat("界🤖 wide-title ", 70)
		for _, ks := range [][]string{{"o"}, {"v"}, {"b"}, {"enter"}, {"esc", ":"}, {"esc", "?"}} {
			m = press(m, ks...)
			lines := strings.Split(m.View(), "\n")
			if len(lines) != m.height {
				t.Fatalf("height %v = %d", size, len(lines))
			}
			for n, line := range lines {
				if lipgloss.Width(line) > m.width {
					t.Fatalf("width %v row %d=%d %q", size, n, lipgloss.Width(line), ansi.Strip(line))
				}
			}
		}
	}
}
