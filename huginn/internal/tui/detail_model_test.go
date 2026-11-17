package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// et85h1: the detail header names who holds a task and on which model.
func TestDetailHeaderShowsModel(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := fixture(t)
	m.width = 140
	base := m.visible[0]
	for _, tc := range []struct {
		assignee []string
		model    string
		want     string
	}{
		{nil, "", "unassigned"},
		{[]string{"poodle"}, "claude-opus-5-5", "assigned:poodle · claude-opus-5-5"},
		{nil, "gpt-6.1-sol", "assigned:gpt-6.1-sol"},
		{[]string{"gpt-6.1-sol"}, "gpt-6.1-sol", "assigned:gpt-6.1-sol"},
		{[]string{"jhf"}, "", "assigned:jhf"},
		// jr55s8: the effort it was taken at shows with the model.
		{[]string{"poodle"}, "claude-opus-5-5/high", "assigned:poodle · claude-opus-5-5/high"},
		{[]string{"gpt-6.1-sol"}, "gpt-6.1-sol/xhigh", "assigned:gpt-6.1-sol/xhigh"},
	} {
		tk := base
		tk.Assignee = tc.assignee
		tk.Model, tk.Effort = task.SplitModel(tc.model)
		got := ansi.Strip(strings.Join(m.detailHeader(tk), "\n"))
		if !strings.Contains(got, tc.want) {
			t.Fatalf("%v/%q: want %q in\n%s", tc.assignee, tc.model, tc.want, got)
		}
		if tc.model != "" && strings.Count(got, tc.model) != 1 {
			t.Fatalf("model shown more than once:\n%s", got)
		}
	}
}

// x1t1t1: the detail header names the branch the task was taken on.
func TestDetailHeaderShowsBranch(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := fixture(t)
	m.width = 160
	tk := m.visible[0]
	tk.Assignee, tk.Model, tk.Branch = []string{"dragon"}, "claude-opus-5-5", "feat/x"
	if got := ansi.Strip(strings.Join(m.detailHeader(tk), "\n")); !strings.Contains(got, "assigned:dragon · claude-opus-5-5 · feat/x") {
		t.Fatalf("branch missing:\n%s", got)
	}
	tk.Assignee, tk.Model = nil, ""
	if got := ansi.Strip(strings.Join(m.detailHeader(tk), "\n")); strings.Contains(got, "feat/x") {
		t.Fatalf("untaken task shows a branch:\n%s", got)
	}
}
