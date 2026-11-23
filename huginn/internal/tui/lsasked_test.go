package tui

import (
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// fthher check 1: hug task ls --asked lists what ASKED counts. Asked is
// the rule '? N for you' and the ASKED rows use, so the CLI and the TUI
// agree on the same board: one row per open question asked of the user,
// on exactly the tasks the TUI counts, oldest first within the board.
func TestAskedAgreesWithForYou(t *testing.T) {
	for name, board := range map[string]func(*testing.T) model{"stage": stageBoard, "asked": askedBoard, "withdrawn": withdrawnBoard} {
		m := board(t)
		for _, user := range []string{"jhf", "Jørgen", "nautilus", "nobody"} {
			m.opts.User = user
			got := Asked(m.all, m.people, user, "")
			tasks := map[string]bool{}
			for _, q := range got {
				tasks[q.ID] = true
			}
			if n := m.countForYou(); len(tasks) != n {
				t.Errorf("%s board, %s: ls --asked names %d tasks %v, '? N for you' counts %d", name, user, len(tasks), got, n)
			}
			want := 0
			for _, tk := range m.all {
				want += len(m.askedAll(tk))
			}
			if len(got) != want {
				t.Errorf("%s board, %s: %d questions, the TUI's pages show %d", name, user, len(got), want)
			}
			for i := 1; i < len(got); i++ {
				if got[i].At < got[i-1].At {
					t.Errorf("%s board, %s: not oldest first: %v", name, user, got)
				}
			}
		}
	}
	// The asked board, for jhf: aaaaaa's check-2 question, cccccc's
	// long one (first line kept whole here; the CLI cuts it), eeeeee's
	// design question with no check.
	m := askedBoard(t)
	got := Asked(m.all, m.people, "jhf", "")
	byID := map[string]Question{}
	for _, q := range got {
		byID[q.ID] = q
	}
	if q := byID["aaaaaa"]; q.Check != 2 || q.N != 1 || q.By != "nautilus" || q.Text != "look at the state/stage rows on your board" {
		t.Errorf("aaaaaa: %+v", q)
	}
	if q := byID["eeeeee"]; q.Check != 0 || q.Text != "should type leave the rows, or stay as a column?" {
		t.Errorf("eeeeee: %+v", q)
	}
	// A person found by git email, as the TUI finds them.
	people := []task.Person{{Short: "jhf", Email: []string{"jorgen@example.invalid"}}}
	if byEmail := Asked(m.all, people, "", "jorgen@example.invalid"); len(byEmail) != len(got) {
		t.Errorf("by email: %d questions, by name %d", len(byEmail), len(got))
	}
}
