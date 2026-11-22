package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// hxqqdp: a file with both "## Details" and an older hug's "## Relevant
// details" (inserted before it) shows both texts on the page, the
// Details text first, as hug task show prints them (crab 2026-11-22).
func TestPageShowsBothDetailsTexts(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	s := &task.Store{Root: t.TempDir()}
	r, err := s.New(task.NewOptions{Title: "Both", Goal: "Aim.", Purpose: "Why.", Principle: "How.", Details: "New hug wrote this.", Acceptance: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.Root, filepath.FromSlash(r.Task.Path))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	both := strings.Replace(string(b), "\n## Details\n", "\n## Relevant details\nOld hug wrote this.\n\n## Details\n", 1)
	if err := os.WriteFile(path, []byte(both), 0o644); err != nil {
		t.Fatal(err)
	}
	s.Invalidate()
	m := newModel(s, Options{User: "jhf"})
	m.width, m.height = 100, 30
	m.resize()
	m = openTask(t, m, r.Task.ID)
	if page := detailPlain(m); !strings.Contains(page, "› DETAILS (2)") {
		t.Fatalf("the Details fold does not count both texts:\n%s", page)
	}
	page := detailPlain(m.withUnfolded("details"))
	n, o := strings.Index(page, "New hug wrote this."), strings.Index(page, "Old hug wrote this.")
	if n < 0 || o < 0 || o < n {
		t.Fatalf("Details text at %d, Relevant details text at %d: want both, Details first:\n%s", n, o, page)
	}
}
