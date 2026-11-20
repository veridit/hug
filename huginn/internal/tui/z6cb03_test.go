package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// lintBoard is a board of advice tasks, one finding each (a Goal of two
// sentences: goal-too-long, as on statbus since 0pr03h), with one broken
// task, last in rank (a dependency on no task), at 100x30.
func lintBoard(t *testing.T, advice int) (model, task.Task) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	for i := 0; i < advice; i++ {
		r, err := s.New(task.NewOptions{Title: fmt.Sprintf("Advice %02d", i+1), Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}})
		if err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(s.Root, r.Task.Path)
		b, _ := os.ReadFile(file)
		text := strings.Replace(string(b), "## Goal\nG\n", "## Goal\nOne sentence. Then another.\n", 1)
		if text == string(b) {
			t.Fatalf("no Goal G:\n%s", b)
		}
		if err := os.WriteFile(file, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := s.New(task.NewOptions{Title: "Broken last", Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(s.Root, r.Task.Path)
	b, _ := os.ReadFile(file)
	text := strings.Replace(string(b), "## Depends on\n", "## Depends on\n- zzzzzz\n", 1)
	if text == string(b) {
		t.Fatalf("no Depends on section:\n%s", b)
	}
	if err := os.WriteFile(file, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 30
	m.resize()
	m.setView("overview")
	return m, r.Task
}

// lintRowID is a row's id after its progress and lint mark: random per
// run, so the golden holds ###### in its place.
var lintRowID = regexp.MustCompile(`(?m)^([>↑↓ ] [○◔◑◕●] \d+(?:/\d+)? +[·!]?)[0-9a-z]{6}\b`)

// z6cb03 check 1: 30 advice findings and 1 broken: the broken row is the
// first in LINT at 100x30, focused or not (golden).
func TestOverviewLintBrokenFirstGolden(t *testing.T) {
	m, broken := lintBoard(t, 30)
	if !m.broken[broken.Path] || m.brokenCount() != 1 || len(m.findings) != 31 {
		t.Fatalf("board: %d findings, %d broken, %s broken %v", len(m.findings), m.brokenCount(), broken.ID, m.broken[broken.Path])
	}
	for _, focus := range []int{sectionNext, sectionLint} {
		m.pane = focus
		m.clamp()
		rows := m.paneRows(sectionLint)
		if len(rows) == 0 || m.visible[rows[0]].Path != broken.Path {
			t.Fatalf("focus %d: LINT does not start with the broken %s:\n%s", focus, broken.ID, ansi.Strip(m.overviewText()))
		}
	}
	m.pane = sectionNext
	m.clamp()
	got := ansi.Strip(m.overviewText())
	if !strings.Contains(got, broken.ID) {
		t.Fatalf("broken %s not in view:\n%s", broken.ID, got)
	}
	got = lintRowID.ReplaceAllString(got, "$1######")
	goldenFile(t, filepath.Join("testdata", "overview", "lint-broken-first-100x30.txt"), got)

	// The lint view lists them the same way.
	m.setView("lint")
	if rows := m.indices(); len(rows) == 0 || m.visible[rows[0]].Path != broken.Path {
		t.Fatalf("lint view does not start with the broken %s", broken.ID)
	}
}
