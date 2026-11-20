package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// longBoard has long titles (plain, CJK, emoji), a long owner on a long
// branch, a closed task and a task with files, so every row kind and cell
// is at its widest.
func longBoard(t *testing.T) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	long := strings.Repeat("A very long title that goes on ", 8)
	titles := []string{
		long,
		"数据库迁移：把所有的表都迁移到新的模式里面去并且保持兼容性" + long,
		"🚀🔥 emoji rocket fire 🎉 " + long,
		"done " + long,
		"files " + long,
	}
	for i, title := range titles {
		kind, priority := "feature", "critical"
		if i == 4 {
			// One differs, so the list draws TYPE and PRIORITY (1bbe96: a
			// column every task shares is said once in the header).
			kind, priority = "bug", "high"
		}
		r, err := s.New(task.NewOptions{Title: title, Goal: "G", Acceptance: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}, Type: kind, Priority: priority})
		if err != nil {
			t.Fatal(err)
		}
		id := r.Task.ID
		switch i {
		case 0, 1:
			// Two agents, one active ticket each (n2hq8p), names of one length.
			name := [2]string{"an-agent-with-a-very-long-name-indeed", "an-agent-with-a-very-long-name-agreed"}[i]
			if _, err := s.TakeAt(id, name, "claude-opus-5-5", task.Place{Branch: "feature/a-very-long-branch-name-for-testing"}); err != nil {
				t.Fatal(err)
			}
		case 3:
			for n := 1; n <= 12; n++ {
				if _, err := s.Check(id, n); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.Take(id, "tester"); err != nil { // take before done (6ephqk)
				t.Fatal(err)
			}
			if _, err := s.Done(id); err != nil {
				t.Fatal(err)
			}
		case 4:
			dir := filepath.Join(s.Root, "files", id)
			if err := os.MkdirAll(dir, 0o755); err == nil {
				_ = os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644)
			}
		}
	}
	m := newModel(s, Options{User: "tester", MainBranch: "main"})
	return m
}

// viewAt is the screen of view at width w.
func viewAt(m model, view string, w int) string {
	m.width, m.height = w, 30
	m.resize()
	m.setView(view)
	return m.View()
}

// 102bc0 check 1: every row of every view is one line, no wider than the
// terminal, at 60, 80, 100, 140 and 200 columns; the screen has exactly
// the terminal's height, so nothing wrapped onto a line of its own.
func TestRowsFitOneLineAtEveryWidth(t *testing.T) {
	m := longBoard(t)
	for _, w := range []int{60, 80, 100, 140, 200} {
		for _, view := range []string{"overview", "list", "board", "lint"} {
			screen := viewAt(m, view, w)
			lines := strings.Split(screen, "\n")
			for n, line := range lines {
				if lw := ansi.StringWidth(line); lw > w {
					t.Errorf("%s at %d: line %d is %d wide: %q", view, w, n+1, lw, ansi.Strip(line))
				}
			}
			if len(lines) != 30 {
				t.Errorf("%s at %d: %d lines, want 30 (a row wrapped)", view, w, len(lines))
			}
			// Every task row still starts with its id after the gutter
			// and pie: the id never moves or drops.
			for _, v := range m.all {
				if view == "list" && !strings.Contains(ansi.Strip(screen), v.ID) {
					t.Errorf("%s at %d: %s not shown", view, w, v.ID)
				}
			}
			// Every task row keeps a piece of its title: columns and the
			// owner give way first (on main before 102bc0 the IN PROGRESS
			// row at 60 lost its whole title behind the owner column).
			for _, line := range strings.Split(ansi.Strip(screen), "\n") {
				for _, v := range m.all {
					at := strings.Index(line, v.ID+" ")
					if at < 0 {
						continue
					}
					word := string([]rune(v.Title)[:2])
					if !strings.Contains(line[at:], word) {
						t.Errorf("%s at %d: the row of %s has no title: %q", view, w, v.ID, line)
					}
				}
			}
		}
	}
}

// 102bc0 check 2: titles with wide runes are cut by display width: the
// cut title fills its cell to the column, never past it, and a wide rune
// is never split.
func TestWideTitlesCutByDisplayWidth(t *testing.T) {
	m := longBoard(t)
	for _, w := range []int{60, 80, 100, 140, 200} {
		screen := ansi.Strip(viewAt(m, "list", w))
		for _, line := range strings.Split(screen, "\n") {
			if !strings.Contains(line, "数据库") && !strings.Contains(line, "🚀") {
				continue
			}
			if lw := ansi.StringWidth(line); lw != w {
				t.Errorf("at %d the wide title row is %d cells: %q", w, lw, line)
			}
			if !strings.Contains(line, "…") {
				t.Errorf("at %d the wide title is not cut with …: %q", w, line)
			}
			if strings.ContainsRune(line, '\uFFFD') {
				t.Errorf("at %d a rune was split: %q", w, line)
			}
		}
	}
}

// 102bc0 check 3: at 60 columns the row keeps the pie, the id and a cut
// title, and drops whole columns in the fixed order: type first, then
// priority, then status.
func TestNarrowRowsDropColumnsInOrder(t *testing.T) {
	m := longBoard(t)
	m.width = 60
	m.resize()
	// The order, from wide to narrow, for a list row.
	var seen []string
	last := ""
	for w := 140; w >= 30; w-- {
		c := m.rowLayout(w, false, 0, 0)
		state := fmt.Sprintf("status=%v type=%v priority=%v", c.status, c.kind, c.priority)
		if state != last {
			seen = append(seen, fmt.Sprintf("%d: %s", w, state))
			last = state
		}
		if c.title < 0 {
			t.Fatalf("at %d the title is %d wide", w, c.title)
		}
	}
	got := strings.Join(seen, "\n")
	for _, want := range []string{"status=true type=true priority=true", "status=true type=false priority=true", "status=true type=false priority=false", "status=false type=false priority=false"} {
		if !strings.Contains(got, want) {
			t.Fatalf("drop order lacks %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "type=false priority=true") > strings.Index(got, "type=false priority=false") {
		t.Fatalf("priority dropped before type:\n%s", got)
	}
	screen := ansi.Strip(viewAt(m, "list", 60))
	header := ""
	for _, line := range strings.Split(screen, "\n") {
		if strings.Contains(line, "TITLE") {
			header = line
		}
	}
	if !strings.Contains(header, "STAGE") || strings.Contains(header, "TYPE") || !strings.Contains(header, "PRIORITY") {
		t.Fatalf("at 60 the list keeps stage and priority, drops type: %q", header)
	}
	for _, v := range m.all {
		if !strings.Contains(screen, v.ID) {
			t.Fatalf("at 60 the id %s is gone:\n%s", v.ID, screen)
		}
	}
	if !strings.Contains(screen, "○ 12 ") || !strings.Contains(screen, "A very long title t…") {
		t.Fatalf("at 60 the pie or the cut title is gone:\n%s", screen)
	}
}

// The IN PROGRESS owner cell gives way before any column: the branch is
// cut first, then the owner, and only then do columns drop.
func TestOwnerCellGivesWayBeforeColumns(t *testing.T) {
	m := longBoard(t)
	names := len("an-agent-with-a-very-long-name-indeed")
	full := names + len(" · feature/a-very-long-branch-name-for-testing")
	for _, tc := range []struct {
		w             int
		owner         string // the owner cell, bounds
		columnsAllKep bool
	}{
		{200, "branch", true},
		{100, "name", true},
		{80, "cut", true},
	} {
		c := m.rowLayout(tc.w, false, min(30, full), names)
		switch tc.owner {
		case "branch":
			if c.owner != 30 {
				t.Errorf("at %d owner %d, want 30", tc.w, c.owner)
			}
		case "name", "cut":
			if c.owner < ownerMin || c.owner > 30 {
				t.Errorf("at %d owner %d out of [%d,30]", tc.w, c.owner, ownerMin)
			}
		}
		if tc.columnsAllKep && (!c.status || !c.kind || !c.priority) {
			t.Errorf("at %d a column dropped while the owner cell could give way: %+v", tc.w, c)
		}
		if c.title < titleMin && c.owner > ownerMin {
			t.Errorf("at %d title %d < %d while the owner is %d", tc.w, c.title, titleMin, c.owner)
		}
	}
}

// 102bc0 check 3 (golden): the overview, list and board at 60 columns, ids
// as ######. Rows keep the pie, the id and a cut title; IN PROGRESS cuts
// the owner's branch and type before the title goes; the list drops type.
func TestNarrowRowsGolden(t *testing.T) {
	m := longBoard(t)
	var got []string
	for _, view := range []string{"overview", "list", "board"} {
		screen := ansi.Strip(viewAt(m, view, 60))
		got = append(got, "== "+view+" at 60", taskID.ReplaceAllString(screen, "$1######"))
	}
	text := strings.Join(got, "\n")
	for _, v := range m.all {
		text = strings.ReplaceAll(text, v.ID, "######")
	}
	path := filepath.Join("testdata", "rows", "narrow-60.txt")
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
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestNarrowRowsGolden ./huginn/internal/tui/", err)
	}
	if text != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("rows at 60 differ from %s:\n%s", path, text)
	}
}
