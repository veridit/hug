package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// slotBoard is five todo tasks, Alpha..Echo (aaaaaa..eeeeee), for jhf;
// with ask, crab asks jhf about check (0: none) on each, so all five are
// in ASKED in that order.
func slotBoard(t *testing.T, ask bool, check int) model {
	t.Helper()
	old, oldTask := now, task.Now
	now = func() time.Time { return time.Date(2026, 11, 20, 9, 53, 0, 0, time.Local) }
	task.Now = func() time.Time { return time.Date(2026, 11, 20, 9, 50, 0, 0, time.Local) }
	t.Cleanup(func() { now, task.Now = old, oldTask })
	s := &task.Store{Root: t.TempDir()}
	for _, title := range []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo"} {
		id := strings.Repeat(strings.ToLower(title[:1]), 6)
		body := "---\nid: \"" + id + "\"\ntitle: \"" + title + " task\"\nstate: \"todo\"\ntype: \"feature\"\npriority: \"high\"\nlabels: []\nassignee: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n- [ ] one\n- [ ] two\n"
		if err := os.WriteFile(filepath.Join(s.Root, "0"+id[:1]+"."+id+".t.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if ask {
			if _, err := s.Ask(id, task.Question{What: "pick one?", For: "jhf", Check: check}, "crab"); err != nil {
				t.Fatal(err)
			}
		}
	}
	s.Actor = "jhf" // as hug tui sets it (cli/hooks.go)
	m := newModel(s, Options{User: "jhf"})
	m.width, m.height = 100, 30
	m.resize()
	m.setView("overview")
	return m
}

// openedID is the id of the open task, "" with none open.
func openedID(m model) string {
	if !m.detail {
		return ""
	}
	t, _ := m.selected()
	return t.ID
}

// rowID is the id on the row the cursor is on, "" with none.
func rowID(m model) string {
	t, ok := m.rowTask()
	if !ok {
		return ""
	}
	return t.ID
}

// 24403z checks 1 and 7, in the model: in an overview section, a
// filtered list and a board column with Alpha..Echo, an action that moves
// the open task out of the pane leaves the reader in it: n opens the
// task after it there, p the one before, Esc lands on that neighbour;
// after the last, n says so and stays. (TestTmuxActionKeepsTheSlot is
// the same in the real binary.)
func TestActionKeepsTheSlot(t *testing.T) {
	for _, c := range []struct {
		name  string
		ask   bool
		setup []string
		act   []string
		where string
	}{
		{"overview ASKED, answer", true, nil, []string{"a", "o", "k", "enter"}, "ASKED"},
		// Drop, not take: one ticket in doing at a time (n2hq8p).
		{"list filtered, drop", false, []string{"v", "l", "/", "state:todo", "tab"}, []string{"x", "d", "u", "p", "enter", "enter"}, "the list (filtered)"},
		{"board TODO, drop", false, []string{"v", "b"}, []string{"x", "d", "u", "p", "enter", "enter"}, "the TODO column"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := slotBoard(t, c.ask, 0)
			m = press(m, c.setup...)
			step := func(keys []string, open string, why string) {
				t.Helper()
				m = press(m, keys...)
				if got := openedID(m); got != open {
					t.Fatalf("%s: open %q, want %q (row %q, message %q)", why, got, open, rowID(m), m.message)
				}
			}
			step([]string{"j", "j", "enter"}, "cccccc", "opened the third row")
			step(c.act, "cccccc", "the action keeps the page")
			step([]string{"n"}, "dddddd", "n after Charlie moved")
			step([]string{"p"}, "bbbbbb", "p from Delta, Charlie gone")
			step(c.act, "bbbbbb", "the action keeps the page")
			step([]string{"p"}, "aaaaaa", "p after Bravo moved")
			step(c.act, "aaaaaa", "the action keeps the page")
			step([]string{"esc"}, "", "Esc after Alpha moved")
			if got := rowID(m); got != "dddddd" {
				t.Fatalf("Esc after Alpha moved landed on %q, want dddddd", got)
			}
			step([]string{"enter", "n"}, "eeeeee", "n from Delta")
			step(c.act, "eeeeee", "the action keeps the page")
			step([]string{"n"}, "eeeeee", "n after the last moved stays")
			if want := "eeeeee was the last in " + c.where + "; nothing after it"; m.message != want {
				t.Fatalf("n after the last: message %q, want %q", m.message, want)
			}
			step([]string{"esc"}, "", "Esc after Echo moved")
			if got := rowID(m); got != "dddddd" {
				t.Fatalf("Esc after Echo moved landed on %q, want dddddd", got)
			}
			// The first: p says so and stays.
			step([]string{"enter"}, "dddddd", "Delta, the only one left")
			step(c.act, "dddddd", "the action keeps the page")
			step([]string{"p"}, "dddddd", "p with nothing before")
			if want := "dddddd was the first in " + c.where + "; nothing before it"; m.message != want {
				t.Fatalf("p after the first moved: message %q, want %q", m.message, want)
			}
		})
	}
}

// A task that stays in its pane after an action keeps n and p as they
// were: from its own row (no gap is made).
func TestActionInPlaceStepsFromTheTask(t *testing.T) {
	m := slotBoard(t, false, 0)
	m = press(m, "v", "l", "j", "j", "enter")
	m = press(m, "t", "enter") // taken: still in the unfiltered list
	if openedID(m) != "cccccc" {
		t.Fatalf("open %q", openedID(m))
	}
	if m = press(m, "n"); openedID(m) != "dddddd" {
		t.Fatalf("n opened %q, want dddddd", openedID(m))
	}
	if m = press(m, "p", "p"); openedID(m) != "bbbbbb" {
		t.Fatalf("p p opened %q, want bbbbbb", openedID(m))
	}
	if m = press(m, "esc"); rowID(m) != "bbbbbb" {
		t.Fatalf("Esc landed on %q, want bbbbbb", rowID(m))
	}
}

// 24403z check 4: after an action from the TUI moves the task to another
// section, the message line reads '<ID> moved to <SECTION>' in the plain
// message style until the next key: answer and tick (ASKED to NEXT UP),
// take (NEXT UP to IN PROGRESS).
func TestActionSaysWhereTheTaskMoved(t *testing.T) {
	for _, c := range []struct {
		name  string
		ask   bool
		check int
		keys  []string
		want  string
	}{
		{"answer", true, 0, []string{"a", "o", "k", "enter"}, "bbbbbb moved to NEXT UP"},
		{"tick", true, 1, []string{" ", "enter"}, "bbbbbb moved to NEXT UP"},
		{"take", false, 0, []string{"t", "enter"}, "bbbbbb moved to IN PROGRESS"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := slotBoard(t, c.ask, c.check)
			m = press(m, "j", "enter")
			if openedID(m) != "bbbbbb" {
				t.Fatalf("open %q", openedID(m))
			}
			if c.name == "take" {
				m.detailPane = focusPage // t on the page takes the task
			}
			m = press(m, c.keys...)
			if m.message != c.want {
				t.Fatalf("message %q, want %q", m.message, c.want)
			}
			footer, _ := m.footLines()
			if footer != m.styles.dim.Render(c.want) {
				t.Fatalf("not in the plain message style: %q", footer)
			}
			if !strings.Contains(ansi.Strip(m.View()), c.want) {
				t.Fatalf("not on the screen:\n%s", ansi.Strip(m.View()))
			}
			if m = press(m, "J"); m.message != "" {
				t.Fatalf("the next key kept %q", m.message)
			}
		})
	}
}

// In a list, a task that left the filter says so (crab 20:07Z): its row
// is gone, so the message explains where.
func TestActionMovedOutsideTheFilter(t *testing.T) {
	m := slotBoard(t, false, 0)
	m = press(m, "v", "l", "/", "state:todo", "tab", "j", "enter")
	m = press(m, "t", "enter")
	if want := "bbbbbb moved to IN PROGRESS (outside this filter)"; m.message != want {
		t.Fatalf("message %q, want %q", m.message, want)
	}
	// In the unfiltered list the row stays: its section still changed.
	m = slotBoard(t, false, 0)
	m = press(m, "v", "l", "j", "enter", "t", "enter")
	if want := "bbbbbb moved to IN PROGRESS"; m.message != want {
		t.Fatalf("message %q, want %q", m.message, want)
	}
}

// 24403z check 5: the detail's breadcrumb names the pane it was opened
// from: the overview's section, the board's column, the list's filter
// (golden 100x30 per view).
func TestBreadcrumbNamesThePane(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, c := range []struct {
		name, crumb string
		ask         bool
		keys        []string
	}{
		{"overview", "HUG · overview › ASKED › bbbbbb", true, []string{"j", "enter"}},
		{"list", "HUG · list › /state:todo › bbbbbb", false, []string{"v", "l", "/", "state:todo", "tab", "j", "enter"}},
		{"board", "HUG · board › TODO › bbbbbb", false, []string{"v", "b", "j", "enter"}},
	} {
		m := slotBoard(t, c.ask, 0)
		m = press(m, c.keys...)
		screen := ansi.Strip(m.View())
		if first, _, _ := strings.Cut(screen, "\n"); !strings.HasPrefix(first, c.crumb) {
			t.Fatalf("%s: breadcrumb %q, want %q first", c.name, first, c.crumb)
		}
		golden = append(golden, "== "+c.name+" 100x30", screen)
	}
	goldenFile(t, "testdata/24403z/breadcrumb-100x30.txt", strings.Join(golden, "\n")+"\n")
}
