package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// vzghj8 checks 1 and 2: / search and # jump find a task by an imported id
// (an alias, as hug task show accepts), case-insensitively; a row found only
// through an alias names it, dim, after the title.
func TestSearchAndJumpFindImportedAlias(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	for _, title := range []string{"Statbus import", "Unrelated"} {
		if _, err := s.New(task.NewOptions{Title: title, Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}}); err != nil {
			t.Fatal(err)
		}
	}
	ts, _ := s.Tasks()
	var target task.Task
	for _, x := range ts {
		if x.Title == "Statbus import" {
			target = x
		}
	}
	file := filepath.Join(s.Root, target.Path)
	b, _ := os.ReadFile(file)
	if err := os.WriteFile(file, []byte(strings.Replace(string(b), "\nlabels:", "\naliases: [\"STATBUS-402\"]\nlabels:", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 120, 24
	m.resize()
	m.setView("list")

	m = press(m, "/", "statbus-402")
	if len(m.visible) != 1 || m.visible[0].ID != target.ID {
		t.Fatalf("/ statbus-402 found %v", m.visible)
	}
	if sel, _ := m.selected(); sel.ID != target.ID {
		t.Fatalf("selected %s, want %s", sel.ID, target.ID)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Statbus import · STATBUS-402") {
		t.Fatalf("row does not name the alias:\n%s", view)
	}
	// The whole screen, ids as ######: the alias after the title, nothing
	// else moved (vzghj8 check 1, golden).
	golden(t, filepath.Join("testdata", "rows", "alias-search-120.txt"), "TestSearchAndJumpFindImportedAlias",
		strings.ReplaceAll(taskID.ReplaceAllString(view, "$1######"), target.ID, "######"))
	// Found by the title: no alias shown.
	m = press(newModel(s, Options{User: "tester"}), "/", "statbus imp")
	m.width, m.height = 120, 24
	m.resize()
	if v := ansi.Strip(m.View()); !strings.Contains(v, "Statbus import") || strings.Contains(v, "· STATBUS-402") {
		t.Fatalf("title match shows the alias:\n%s", v)
	}

	m = press(newModel(s, Options{User: "tester"}), "#", "STATBUS-402", "enter")
	if sel, _ := m.selected(); !m.detail || sel.ID != target.ID {
		t.Fatalf("# STATBUS-402: detail %v selected %s", m.detail, sel.ID)
	}
}

// golden compares text with the file at path; HUG_UPDATE_GOLDEN=1 writes it.
func golden(t *testing.T, path, test, text string) {
	t.Helper()
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run %s ./huginn/internal/tui/", err, test)
	}
	if text != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("screen differs from %s:\n%s\nfix: HUG_UPDATE_GOLDEN=1 go test -run %s ./huginn/internal/tui/ (then review the diff)", path, text, test)
	}
}
