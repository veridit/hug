package cli

import (
	"slices"
	"strings"
	"testing"
)

// 7zg9mf check 2 (the CLI half): a filter value that starts with ! keeps
// every task but those with that value, in ls, search, next and board.
// A bad value after the ! is refused with a fix: line that keeps the !.
func TestNegatedFilters(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	ids := map[string]string{}
	for _, tk := range []struct{ title, typ string }{{"Open quartz", "feature"}, {"Bug quartz", "bug"}, {"Done quartz", "feature"}, {"Done bug quartz", "bug"}} {
		ids[tk.title] = execute(t, a, "new", tk.title, "--goal", "g", "--purpose", "P", "--principle", "Q", "--accept", "a", "--type", tk.typ)["task"].(Reply)["id"].(string)
	}
	for _, title := range []string{"Done quartz", "Done bug quartz"} {
		execute(t, a, "check", ids[title], "1")
		execute(t, a, "take", ids[title])
		execute(t, a, "done", ids[title])
	}
	listed := func(r Reply, key string) []string {
		var out []string
		for _, x := range r[key].([]Reply) {
			out = append(out, x["id"].(string))
		}
		slices.Sort(out)
		return out
	}
	want := func(titles ...string) []string {
		var out []string
		for _, s := range titles {
			out = append(out, ids[s])
		}
		slices.Sort(out)
		return out
	}
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"ls", "--state", "!done"}, want("Open quartz", "Bug quartz")},
		{[]string{"ls", "--type", "!bug"}, want("Open quartz", "Done quartz")},
		{[]string{"ls", "--state", "!done", "--type", "!bug"}, want("Open quartz")},
		{[]string{"ls", "--state", "!todo"}, want("Done quartz", "Done bug quartz")},
	} {
		if got := listed(execute(t, a, tc.args...), "tasks"); !slices.Equal(got, tc.want) {
			t.Errorf("%v: got %v, want %v", tc.args, got, tc.want)
		}
	}
	if got := listed(execute(t, a, "search", "quartz", "--state", "!done"), "results"); !slices.Equal(got, want("Open quartz", "Bug quartz")) {
		t.Errorf("search quartz --state !done: got %v", got)
	}
	if r := execute(t, a, "next", "--type", "!feature"); r["task"].(Reply)["id"] != ids["Bug quartz"] {
		t.Errorf("next --type !feature: got %v, want %s", r["task"].(Reply)["id"], ids["Bug quartz"])
	}
	cols := execute(t, a, "board", "--type", "!bug")["board"].([]BoardColumn)
	if cols[0].Total != 1 || cols[2].Total != 1 {
		t.Errorf("board --type !bug: %+v", cols)
	}
	// A bad value keeps the ! in its fix, the last line of the error.
	for _, tc := range []struct {
		args []string
		fix  string
	}{
		{[]string{"ls", "--state", "!don"}, "fix: hug task ls --state '!done'"},
		{[]string{"ls", "--type", "!bugs"}, "fix: hug task ls --type '!bug'"},
	} {
		code, out := runText(a, tc.args...)
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if code != 2 || lines[len(lines)-1] != tc.fix {
			t.Errorf("%v: exit %d, want 2 ending %q:\n%s", tc.args, code, tc.fix, out)
		}
	}
}
