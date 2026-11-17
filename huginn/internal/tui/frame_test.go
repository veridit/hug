package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// bigOverview is an overview of n tasks, a third doing, the rest todo,
// some waiting on a person and some with checks ticked: Seid's mix.
func bigOverview(t *testing.T, n int) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	for i := 0; i < n; i++ {
		front := "state: \"todo\""
		switch i % 3 {
		case 1:
			front = "state: \"doing\"\nstage: \"active\"\ndoing_since: \"2026-11-17T08:00:00Z\""
		case 2:
			front = "state: \"doing\"\nstage: \"waiting\"\ndoing_since: \"2026-11-17T08:00:00Z\"\nwaiting_on: \"tester: look\"\nwaiting_for: \"tester\"\nwaiting_since: \"2026-11-17T08:30:00Z\""
		}
		id := fmt.Sprintf("b%05d", i)
		check := "- [ ] a\n- [x] b\n"
		body := "---\nid: \"" + id + "\"\ntitle: \"Task " + fmt.Sprint(i) + "\"\n" + front + "\ntype: \"chore\"\npriority: \"low\"\nlabels: []\nassignee: [\"a" + fmt.Sprint(i%7) + "\"]\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n" + check
		if e := os.WriteFile(filepath.Join(s.Root, fmt.Sprintf("%05d.%s.t.md", i, id)), []byte(body), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 160, 45
	m.resize()
	m.setView("overview")
	return m
}

// 46ek1f check 4: a frame's work does not grow with the board. On Seid
// (2,520 tasks) a View took about 40 ms: every frame computed every task's
// stage, several times (pane rows, row heights, the "for you" count), and
// read every task's checks for the progress column. Stages are the
// snapshot's: computed once per load, none per frame.
func TestFrameComputesNoStages(t *testing.T) {
	m := bigOverview(t, 2000)
	if len(m.all) != 2000 {
		t.Fatalf("board has %d tasks", len(m.all))
	}
	m.View() // the first frame after the load
	for _, view := range []string{"overview", "list", "board"} {
		m.setView(view)
		m.clamp()
		before := stageComputes.Load()
		for range 3 {
			m = press(m, "j")
			m.View()
		}
		if n := stageComputes.Load() - before; n != 0 {
			t.Fatalf("%s: 3 frames on a 2,000-task board computed %d task stages, want 0 (the snapshot's)", view, n)
		}
	}
	// The overview still finds what waits on the TUI user.
	m.setView("overview")
	if n := m.forYouCount(); n != 666 {
		t.Fatalf("for you: %d, want 666", n)
	}
}

// 46ek1f check 4: what a frame measures does not grow with the board. Row
// heights are summed only as far as the screen reaches: the same frame on
// a board ten times larger looks up the same number of row heights.
func TestFrameRowWorkDoesNotGrowWithBoard(t *testing.T) {
	per := func(n int) map[string]int64 {
		m := bigOverview(t, n)
		m.View()
		out := map[string]int64{}
		for _, view := range []string{"overview", "list", "board"} {
			m.setView(view)
			m.clamp()
			m = press(m, "j")
			before := heightLookups.Load()
			m = press(m, "j")
			m.View()
			out[view] = heightLookups.Load() - before
		}
		return out
	}
	small, big := per(300), per(3000)
	for view, n := range big {
		if n > small[view] {
			t.Errorf("%s: a key and a frame looked up %d row heights on 3,000 tasks, %d on 300: the work grows with the board", view, n, small[view])
		}
	}
}
