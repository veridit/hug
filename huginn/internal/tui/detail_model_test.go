package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
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
	} {
		tk := base
		tk.Assignee, tk.Model = tc.assignee, tc.model
		got := ansi.Strip(strings.Join(m.detailHeader(tk), "\n"))
		if !strings.Contains(got, tc.want) {
			t.Fatalf("%v/%q: want %q in\n%s", tc.assignee, tc.model, tc.want, got)
		}
		if tc.model != "" && strings.Count(got, tc.model) != 1 {
			t.Fatalf("model shown more than once:\n%s", got)
		}
	}
}
