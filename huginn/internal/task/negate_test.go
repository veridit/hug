package task

import "testing"

// 7zg9mf check 2 (the model): a Filter value that starts with ! keeps
// every task but those with that value; next takes it as well.
func TestFilterNegation(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	open := newTest(t, s, "Open")
	feature, e := s.New(NewOptions{Title: "Feature", Goal: "observable outcome", Acceptance: []string{"passes"}, Type: "feature", Labels: []string{"ui"}})
	if e != nil {
		t.Fatal(e)
	}
	done := newTest(t, s, "Done")
	if _, e := s.Check(done.ID, 1); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Take(done.ID, "tester"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Done(done.ID); e != nil {
		t.Fatal(e)
	}
	ts, e := s.Tasks()
	if e != nil {
		t.Fatal(e)
	}
	kept := func(f Filter) []string {
		var out []string
		for _, x := range ts {
			if f.Keeps(x, ts) {
				out = append(out, x.Title)
			}
		}
		return out
	}
	for _, tc := range []struct {
		f    Filter
		want string
	}{
		{Filter{State: "!done"}, "[Open Feature]"},
		{Filter{State: "done"}, "[Done]"},
		{Filter{Type: "!bug"}, "[Feature]"},
		{Filter{Label: "!core"}, "[Feature]"},
		{Filter{State: "!done", Type: "!feature"}, "[Open]"},
		{Filter{Priority: "!high"}, "[Open Feature Done]"},
	} {
		if got := fmtList(kept(tc.f)); got != tc.want {
			t.Errorf("%+v kept %s, want %s", tc.f, got, tc.want)
		}
	}
	r, e := Next(ts, Filter{Type: "!bug"})
	if e != nil || r.Task.ID != feature.Task.ID {
		t.Fatalf("next --type !bug = %s %v, want %s (not %s)", r.Task.ID, e, feature.Task.ID, open.ID)
	}
	if _, e := Next(ts, Filter{Type: "!bugs"}); e == nil {
		t.Fatal("next --type !bugs accepted")
	}
	for _, v := range []string{"!done", "done", "!", ""} {
		base, not := Not(v)
		if v == "!done" && (base != "done" || !not) || v == "done" && (base != "done" || not) || v == "!" && (base != "" || !not) {
			t.Errorf("Not(%q) = %q %v", v, base, not)
		}
	}
}

func fmtList(s []string) string {
	out := "["
	for i, x := range s {
		if i > 0 {
			out += " "
		}
		out += x
	}
	return out + "]"
}
