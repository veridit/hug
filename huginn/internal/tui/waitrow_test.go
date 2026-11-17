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

// waitBoard has n doing tasks, every other one waiting on CI (two lines).
func waitBoard(t *testing.T, n int) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	for i := 0; i < n; i++ {
		front := "status: \"doing\"\nstage: \"active\"\ndoing_since: \"2026-11-17T08:00:00Z\""
		if i%2 == 1 {
			front = "status: \"doing\"\nstage: \"waiting\"\ndoing_since: \"2026-11-17T08:00:00Z\"\nwaiting_on: \"CI run " + fmt.Sprint(i) + "\"\nwaiting_since: \"2026-11-17T08:30:00Z\""
		}
		id := fmt.Sprintf("w%05d", i)
		body := "---\nid: \"" + id + "\"\ntitle: \"Task " + fmt.Sprint(i) + "\"\n" + front + "\ntype: \"chore\"\npriority: \"low\"\nlabels: []\nassignee: [\"a" + fmt.Sprint(i) + "\"]\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n- [ ] a\n"
		if e := os.WriteFile(filepath.Join(s.Root, fmt.Sprintf("%03d.%s.t.md", i, id)), []byte(body), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 20
	m.resize()
	return m
}

// 89y0b7 check 3: j/k move by task over two-line rows; the window counts
// lines, never shows half a row, keeps the cursor's row (both its lines)
// on screen, and the screen keeps its height.
func TestTwoLineRowsScrollByTask(t *testing.T) {
	m := waitBoard(t, 30)
	for _, view := range []string{"list", "overview", "board"} {
		m.setView(view)
		if view == "board" {
			m.pane = 1 // doing
		}
		m.clamp()
		m.move(0, true)
		seen := map[string]bool{}
		for step := 0; step < 40; step++ {
			sel, ok := m.selected()
			if !ok {
				t.Fatalf("%s step %d: nothing selected", view, step)
			}
			seen[sel.ID] = true
			screen := ansi.Strip(m.View())
			lines := strings.Split(screen, "\n")
			if len(lines) != m.height {
				t.Fatalf("%s step %d: %d lines, want %d", view, step, len(lines), m.height)
			}
			at := -1
			for i, l := range lines {
				if strings.Contains(l, ">") && strings.Contains(l, sel.ID) {
					at = i
				}
				// A note line always sits under its own row.
				if strings.Contains(l, "└─") && (i == 0 || !strings.Contains(lines[i-1], "w0")) {
					t.Fatalf("%s step %d: a second line with no row above it:\n%s", view, step, screen)
				}
			}
			if at < 0 {
				t.Fatalf("%s step %d: selected %s is not on screen:\n%s", view, step, sel.ID, screen)
			}
			if m.rowHeight(sel) == 2 && (at+1 >= len(lines) || !strings.Contains(lines[at+1], "└─ waiting on CI run")) {
				t.Fatalf("%s step %d: %s shows without its second line:\n%s", view, step, sel.ID, screen)
			}
			m = press(m, "j")
		}
		want := min(len(m.indices()), 40)
		if len(seen) != want {
			t.Fatalf("%s: j visited %d tasks, want all %d", view, len(seen), want)
		}
		// k comes back one task at a time.
		before, _ := m.selected()
		m = press(m, "k")
		if after, _ := m.selected(); after.ID == before.ID {
			t.Fatalf("%s: k did not move", view)
		}
	}
}

// 89y0b7: the detail says what the task waits on under its header, with
// the person's long name; the row keeps the short one.
func TestDetailNamesTheLongName(t *testing.T) {
	m := stageBoard(t)
	if e := os.WriteFile(filepath.Join(m.store.Root, "hug.toml"), []byte("[people.jhf]\nname = \"Jørgen H. Fjeld\"\nemail = [\"jorgen@veridit.no\"]\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	m.refresh("")
	m.width, m.height = 140, 40
	m.resize()
	m.setView("list")
	for i, tk := range m.visible {
		if tk.ID == "eeeeee" {
			m.cursor = i
		}
	}
	list := ansi.Strip(m.View())
	if !strings.Contains(list, "└─ question for jhf: should type") || strings.Contains(list, "Fjeld") {
		t.Fatalf("row:\n%s", list)
	}
	m = press(m, "enter")
	if v := ansi.Strip(m.View()); !strings.Contains(v, "└─ question for jhf (Jørgen H. Fjeld): should type leave the rows") {
		t.Fatalf("detail:\n%s", v)
	}
}
