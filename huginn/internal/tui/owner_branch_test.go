package tui

import (
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// x1t1t1 check 4: an IN PROGRESS row names the branch the work is on,
// unless it is the main branch; the owner's name comes first.
func TestInProgressRowShowsBranch(t *testing.T) {
	m := newModel(&task.Store{Root: t.TempDir()}, Options{User: "tester", MainBranch: "main"})
	for _, c := range []struct {
		t    task.Task
		want string
	}{
		{task.Task{Assignee: []string{"poodle"}, Branch: "feat/x"}, "poodle · feat/x"},
		{task.Task{Assignee: []string{"poodle"}, Branch: "main"}, "poodle"},
		{task.Task{Assignee: []string{"poodle"}}, "poodle"},
		{task.Task{Model: "gpt-6.1-sol", Branch: "feat/y"}, "gpt-6.1-sol · feat/y"},
		// jr55s8: a model that holds the task shows the effort it runs at;
		// a named agent's row keeps the name alone.
		{task.Task{Model: "gpt-6.1-sol", Effort: "xhigh"}, "gpt-6.1-sol/xhigh"},
		{task.Task{Assignee: []string{"gpt-6.1-sol"}, Model: "gpt-6.1-sol", Effort: "xhigh"}, "gpt-6.1-sol/xhigh"},
		{task.Task{Assignee: []string{"poodle"}, Model: "claude-opus-5-5", Effort: "high"}, "poodle"},
		{task.Task{}, "unassigned"},
	} {
		if got := m.rowOwner(c.t); got != c.want {
			t.Errorf("rowOwner(%+v) = %q, want %q", c.t, got, c.want)
		}
	}

	m = spareBoard(t, 2, 0)
	m.opts.MainBranch = "main"
	m.width = 140
	m.visible[0].Assignee, m.visible[0].Branch = []string{"alice"}, "feat/terminal-work"
	m.visible[1].Assignee, m.visible[1].Branch = []string{"bob"}, "main"
	out := m.overviewText()
	if !strings.Contains(out, "alice · feat/terminal-work") {
		t.Fatalf("no branch on alice's row:\n%s", out)
	}
	if strings.Contains(out, "bob · main") {
		t.Fatalf("the main branch is shown:\n%s", out)
	}
	m.width = 80
	// The stage column (6ephqk) is 2 cells wider than status was: the
	// branch is cut further, the owner still comes first.
	if out = m.overviewText(); !strings.Contains(out, "alice · feat/…") {
		t.Fatalf("80 columns: owner first, branch cut:\n%s", out)
	}
}
