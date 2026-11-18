package task

import (
	"strings"
	"testing"
)

// hbgybn checks 1 and 2 (task half): a request is stored verbatim as a
// block under ## Requests (before Notes), numbered per task; resolve closes
// one by N with the account under it, verbatim; N stays the same request
// when another closes; an unknown or resolved N is refused with a fix; the
// stage does not change.
func TestRequestAndResolve(t *testing.T) {
	s, id := answerStore(t)
	if _, e := s.Note(id, "a note"); e != nil {
		t.Fatal(e)
	}
	before, _ := readTask(t, s, id)
	text := "the goal says \"tags\": make it per board\n\nand split the TUI part out"
	if _, e := s.Request(id, text+"\n", "jhf"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Request(id, "second: shorter title", "jhf"); e != nil {
		t.Fatal(e)
	}
	tk, raw := readTask(t, s, id)
	want := "## Requests\n- 2026-11-17T13:30 request 1 by jhf:\n  the goal says \"tags\": make it per board\n  \n  and split the TUI part out\n- 2026-11-17T13:30 request 2 by jhf:\n  second: shorter title\n\n## Notes\n"
	if !strings.Contains(raw, want) {
		t.Fatalf("file lacks the requests before Notes:\n%s\nfile:\n%s", want, raw)
	}
	if len(tk.Requests) != 2 || tk.Requests[0].Text != text || tk.Requests[0].By != "jhf" || tk.Requests[0].N != 1 || !tk.Requests[0].Open() || tk.Requests[1].Text != "second: shorter title" {
		t.Fatalf("parsed: %+v", tk.Requests)
	}
	if tk.Stage != before.Stage || tk.State != before.State {
		t.Fatalf("a request moved the stage: %s", tk.Stage)
	}
	// Resolve 2 first: 1 keeps its number.
	account := "set the title to \"Short\"\nno other change"
	if _, e := s.Resolve(id, 2, account, "nautilus"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Resolve(id, 1, "goal now per board; new task abc123 for the TUI", "nautilus"); e != nil {
		t.Fatal(e)
	}
	tk, raw = readTask(t, s, id)
	if !strings.Contains(raw, "- 2026-11-17T13:30 request 2 by jhf:\n  second: shorter title\n  resolved 2026-11-17T13:30 by nautilus:\n    set the title to \"Short\"\n    no other change\n") {
		t.Fatalf("resolve 2 not under request 2:\n%s", raw)
	}
	if !strings.Contains(raw, "  and split the TUI part out\n  resolved 2026-11-17T13:30 by nautilus:\n    goal now per board; new task abc123 for the TUI\n- 2026-11-17T13:30 request 2") {
		t.Fatalf("resolve 1 not under request 1:\n%s", raw)
	}
	if len(OpenRequests(tk)) != 0 || tk.Requests[1].Resolved == nil || tk.Requests[1].Resolved.Text != account || tk.Requests[0].Text != text {
		t.Fatalf("after resolve: %+v / %+v", tk.Requests[0], tk.Requests[1].Resolved)
	}
	// A third request is 3, never a reused number.
	if _, e := s.Request(id, "third", "jhf"); e != nil {
		t.Fatal(e)
	}
	if tk, _ = readTask(t, s, id); tk.Requests[2].N != 3 {
		t.Fatalf("third request numbered %d", tk.Requests[2].N)
	}
	for _, c := range []struct {
		n         int
		want, fix string
	}{
		{9, id + " has no request 9; its requests are 1, 2, 3 (open: 3)", "hug task requests " + id},
		{1, "request 1 on " + id + " is already resolved, by nautilus", "hug task show " + id},
	} {
		_, e := s.Resolve(id, c.n, "x", "nautilus")
		if e == nil || !strings.Contains(e.Error(), c.want) || !strings.Contains(e.Error(), "fix: "+c.fix) {
			t.Fatalf("resolve %d: %v", c.n, e)
		}
	}
	if _, e := s.Request(id, "  \n", "jhf"); e == nil || !strings.Contains(e.Error(), "request needs the request") {
		t.Fatalf("empty request: %v", e)
	}
	if !strings.Contains(raw, "request 1 by jhf\n") || !strings.Contains(raw, "resolved request 2 by nautilus\n") {
		t.Fatalf("History lacks the request lines:\n%s", raw)
	}
	for _, f := range Lint([]*Document{mustGet(t, s, id)}) {
		if FindingCode(f.Problem) != "missing-section" {
			t.Fatalf("lint on a task with requests: %v", f)
		}
	}
}

func mustGet(t *testing.T, s *Store, id string) *Document {
	t.Helper()
	d, e := s.Get(id)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
