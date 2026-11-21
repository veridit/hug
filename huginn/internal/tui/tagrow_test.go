package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

// txjkrm check 6, as 911j70 check 5 has it (Jørgen 2026-11-20 08:26Z):
// the tags are a dim prefix of the TITLE cell, "parser compiler · Both
// areas", in hug.toml's order, at most 22 columns; there is no TAGS column.
// (Was: a TAGS column right after the id.)
func TestTagsSitRightAfterTheID(t *testing.T) {
	m, ids := tagRowBoard(t, 140)
	s := screen(m)
	head := regexp.MustCompile(`(?m)^\s+ID\s+TITLE\s+STAGE\s+✓\s+TYPE\s+PRIORITY\s*$`)
	if !head.MatchString(s) || strings.Contains(s, "TAGS") {
		t.Fatalf("the column header is not ID TITLE STAGE ✓ TYPE PRIORITY, or has TAGS:\n%s", s)
	}
	row := rowOf(t, s, ids["Both areas"])
	want := regexp.MustCompile(regexp.QuoteMeta(ids["Both areas"]) + `   parser compiler · Both areas\s+\S+\s+○ 1\s+feature\s+high`)
	if !want.MatchString(row) {
		t.Fatalf("Both areas row is not id, tags · title (listed order), stage, ✓, type, priority:\n%q", row)
	}
	// The cap: the tags never pass 22 columns, cut with an ellipsis, and
	// the title follows them.
	long := rowOf(t, s, ids["Long tags"])
	title := regexp.MustCompile(`   (compiler codegen[^·]*…) · Long tags`).FindStringSubmatch(long)
	if title == nil {
		t.Fatalf("long tags %q: want the listed order, cut with … inside 22 columns, then the title", long)
	}
	if w := lipgloss.Width(title[1]); w != tagMax {
		t.Fatalf("long tags prefix is %d wide, want %d: %q", w, tagMax, title[1])
	}
	// A task without tags starts its title where the tags would.
	if bare := rowOf(t, s, ids["Bare"]); !strings.Contains(bare, ids["Bare"]+"   Bare ") {
		t.Fatalf("Bare's title does not follow the id:\n%q", bare)
	}
	// Dim: the tags and their · are drawn in the dim style.
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	color := m.View()
	if dim := m.styles.dim.Render("parser compiler · "); dim == "parser compiler · " || !strings.Contains(color, dim) {
		t.Fatalf("the tags are not dim:\n%q", rowOf(t, color, ids["Both areas"]))
	}
	// Every row is one line within the width (102bc0).
	for _, line := range strings.Split(s, "\n") {
		if w := lipgloss.Width(line); w > 140 {
			t.Fatalf("a row is %d wide:\n%q", w, line)
		}
	}
}

// txjkrm check 6 at 80 columns, as 911j70 check 5 has it: the tags stay
// the title's prefix and the title text is what is cut.
func TestTagsAt80Columns(t *testing.T) {
	m, ids := tagRowBoard(t, 80)
	s := screen(m)
	row := rowOf(t, s, ids["Both areas"])
	if !regexp.MustCompile(regexp.QuoteMeta(ids["Both areas"]) + `   parser compiler · Both areas`).MatchString(row) {
		t.Fatalf("at 80 the tags are not the title's prefix:\n%q", row)
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

// txjkrm check 6, as 911j70 has it: tags are no column, so the column
// cursor never stands on them; from ID, L goes to TITLE, whose cell they
// open. (Was: the cursor stands on TAGS and o orders by them.) Past
// PRIORITY, L lands on no column, then on ID again (hktdgs).
func TestTagsColumnCursor(t *testing.T) {
	m, _ := tagRowBoard(t, 140)
	var marked []string
	for range len(gridColumns) + 1 {
		m = press(m, "L")
		if i := m.markedColumn(); i >= 0 {
			marked = append(marked, gridColumns[i].field)
		} else {
			marked = append(marked, "none")
		}
	}
	if got := strings.Join(marked, " "); got != "id title stage checks type priority none id" {
		t.Fatalf("L marks %s, want id title stage checks type priority, then none, then id again", got)
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
