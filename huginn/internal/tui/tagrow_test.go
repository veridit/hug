package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// tagRowBoard is tagBoard plus a task whose tags run past the 22-column
// cap, typed and prioritised so every column has a value.
func tagRowBoard(t *testing.T, width int) (model, map[string]string) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	if err := os.WriteFile(filepath.Join(s.Root, task.ConfigName), []byte("tags = [\"parser\", \"compiler\", \"codegen\", \"runtime\", \"perceus\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, c := range []struct {
		title, kind, priority, dep string
		tags                       []string
	}{
		{"Bare", "chore", "low", "", nil},
		{"Lex the input", "bug", "high", "", []string{"parser"}},
		{"Both areas", "feature", "high", "", []string{"compiler", "parser"}}, // listed order: parser first
		{"Long tags", "feature", "medium", "", []string{"perceus", "runtime", "codegen", "compiler"}},
		{"Odd one", "feature", "high", "Bare", []string{"zzz"}}, // waits on Bare: another stage
	} {
		var deps []string
		if c.dep != "" {
			deps = []string{ids[c.dep]}
		}
		r, err := s.New(task.NewOptions{Title: c.title, Goal: "g", Acceptance: []string{"a"}, Labels: c.tags, Type: c.kind, Priority: c.priority, Deps: deps})
		if err != nil {
			t.Fatal(err)
		}
		ids[c.title] = r.Task.ID
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = width, 20
	m.resize()
	m.setView("list")
	return m, ids
}

func rowOf(t *testing.T, s, id string) string {
	t.Helper()
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, id) {
			return line
		}
	}
	t.Fatalf("no row for %s:\n%s", id, s)
	return ""
}

// txjkrm check 6 (Jørgen 19:35Z): the row reads pie, id, tags, type,
// priority, stage, text; the tags are dim, in hug.toml's order, and take
// at most 22 columns.
func TestTagsSitRightAfterTheID(t *testing.T) {
	m, ids := tagRowBoard(t, 140)
	s := screen(m)
	head := regexp.MustCompile(`(?m)^\s+✓?\s*ID\s+TAGS\s+TYPE\s+PRIORITY\s+STAGE\s+TITLE`)
	if !head.MatchString(s) {
		t.Fatalf("the column header is not ID TAGS TYPE PRIORITY STAGE TITLE:\n%s", s)
	}
	row := rowOf(t, s, ids["Both areas"])
	want := regexp.MustCompile(regexp.QuoteMeta(ids["Both areas"]) + `\s+parser compiler\s+feature\s+high\s+\S+\s+Both areas`)
	if !want.MatchString(row) {
		t.Fatalf("Both areas row is not id, tags (listed order), type, priority, stage, title:\n%q", row)
	}
	// The cap: the tag cell never passes 22 columns, cut with an ellipsis.
	long := rowOf(t, s, ids["Long tags"])
	header := ""
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "TAGS") {
			header = line
		}
	}
	from := lipgloss.Width(header[:strings.Index(header, "TAGS")])
	cell := ansi.Cut(long, from, from+tagMax+1)
	if !strings.HasPrefix(cell, "compiler codegen") || !strings.HasSuffix(cell, "… ") {
		t.Fatalf("long tags cell %q: want the listed order, cut with … inside 22 columns", cell)
	}
	if !strings.Contains(long, "… feature") {
		t.Fatalf("type does not follow the cut tags one space after the cap:\n%q", long)
	}
	if w := lipgloss.Width(strings.TrimSpace(cell)); w != tagMax {
		t.Fatalf("long tags cell is %d wide, want %d: %q", w, tagMax, cell)
	}
	// Dim: the tags are drawn in the dim style.
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	color := m.View()
	if dim := m.styles.dim.Render(fit("parser compiler", 22)); dim == fit("parser compiler", 22) || !strings.Contains(color, dim) {
		t.Fatalf("the tags are not dim:\n%q", rowOf(t, color, ids["Both areas"]))
	}
	// Every row is one line within the width (102bc0).
	for _, line := range strings.Split(s, "\n") {
		if w := lipgloss.Width(line); w > 140 {
			t.Fatalf("a row is %d wide:\n%q", w, line)
		}
	}
}

// txjkrm check 6 at 80 columns: the tags stay right after the id, cut
// before the type column drops (102bc0: what gives way, in order).
func TestTagsAt80Columns(t *testing.T) {
	m, ids := tagRowBoard(t, 80)
	s := screen(m)
	row := rowOf(t, s, ids["Both areas"])
	if !regexp.MustCompile(regexp.QuoteMeta(ids["Both areas"]) + `\s+parser\S*\s`).MatchString(row) {
		t.Fatalf("at 80 the tags do not follow the id:\n%q", row)
	}
	if !strings.Contains(row, "Both areas") {
		t.Fatalf("at 80 the title is gone:\n%q", row)
	}
	for _, line := range strings.Split(s, "\n") {
		if w := lipgloss.Width(line); w > 80 {
			t.Fatalf("a row is %d wide:\n%q", w, line)
		}
	}
}

// A board with no tags has no tags cell: rows read as before.
func TestNoTagsNoCell(t *testing.T) {
	m := stageBoard(t)
	m.width, m.height = 120, 30
	m.resize()
	m.setView("list")
	if s := screen(m); strings.Contains(s, "TAGS") {
		t.Fatalf("a board without tags shows a TAGS column:\n%s", s)
	}
}

// txjkrm check 6: the column cursor stands on TAGS between ID and TYPE,
// and o orders by the tags.
func TestTagsColumnCursor(t *testing.T) {
	m, _ := tagRowBoard(t, 140)
	m = press(m, "L", "L", "L")
	if got := gridColumns[m.markedColumn()].field; got != "tags" {
		t.Fatalf("the third column is %s, want tags", got)
	}
	m = press(m, "o")
	var got []string
	for _, tk := range m.visible {
		got = append(got, tk.Title)
	}
	// a to z by the row's tag text; a task without tags last.
	if strings.Join(got, ",") != "Long tags,Lex the input,Both areas,Odd one,Bare" {
		t.Fatalf("o on TAGS: %v", got)
	}
}

// Goldens at 80 and 140 columns (txjkrm checks 3 and 6).
func TestTagRowsGolden(t *testing.T) {
	for _, w := range []int{80, 140} {
		m, ids := tagRowBoard(t, w)
		s := screen(m)
		// Random ids become one letter six times, by title, so the golden
		// holds still.
		for title, letter := range map[string]string{"Lex the input": "a", "Both areas": "b", "Long tags": "c", "Odd one": "d", "Bare": "e"} {
			s = strings.ReplaceAll(s, ids[title], strings.Repeat(letter, len(ids[title])))
		}
		goldenFile(t, filepath.Join("testdata", "rows", fmt.Sprintf("tags-%d.txt", w)), s)
	}
}
