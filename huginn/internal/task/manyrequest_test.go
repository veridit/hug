package task

import (
	"errors"
	"strings"
	"testing"
)

// manyStore is a board with three tasks, a, b and c, for one request
// over several of them (ec6z6j).
func manyStore(t *testing.T) (*Store, []string) {
	t.Helper()
	s, a := answerStore(t)
	ids := []string{a}
	for _, title := range []string{"Rows", "Header"} {
		r, e := s.New(NewOptions{Title: title, Goal: "G", Acceptance: []string{"C"}, Type: "chore"})
		if e != nil {
			t.Fatal(e)
		}
		ids = append(ids, r.Task.ID)
	}
	return s, ids
}

// ec6z6j check 2 (task half): one request for several tasks is written
// once, verbatim, on the first (its home), with the exact selected IDs;
// every other selected task gets a link to it, with no copy of the text.
func TestRequestForSeveralTasks(t *testing.T) {
	s, ids := manyStore(t)
	a, b, c := ids[0], ids[1], ids[2]
	text := "the goal says \"tags\": make it per board\n\nsame fix on each"
	// b is named twice: the selection is a set.
	r, e := s.RequestFor([]string{a, b, c, b}, text+"\n", "jhf")
	if e != nil {
		t.Fatal(e)
	}
	if r.Task.ID != a {
		t.Fatalf("the reply is on %s, want the home %s", r.Task.ID, a)
	}
	home, raw := readTask(t, s, a)
	want := "## Requests\n- 2026-11-17T13:30 request 1 by jhf (for " + a + ", " + b + ", " + c + "):\n  the goal says \"tags\": make it per board\n  \n  same fix on each\n"
	if !strings.Contains(raw, want) {
		t.Fatalf("home lacks\n%s\nfile:\n%s", want, raw)
	}
	if strings.Count(raw, "same fix on each") != 1 {
		t.Fatalf("text written more than once on the home:\n%s", raw)
	}
	rq := home.Requests[0]
	if rq.By != "jhf" || rq.Text != text || rq.Home != nil || len(rq.Targets) != 3 {
		t.Fatalf("home request: %+v", rq)
	}
	for i, tg := range rq.Targets {
		if tg.ID != ids[i] || tg.Resolved {
			t.Fatalf("target %d: %+v", i, tg)
		}
	}
	if !rq.Open() || rq.Closed() {
		t.Fatalf("a new request is closed: %+v", rq)
	}
	if !strings.Contains(raw, " request 1 by jhf (for "+a+", "+b+", "+c+")\n") {
		t.Fatalf("home History lacks the request line:\n%s", raw)
	}
	for _, id := range []string{b, c} {
		tk, raw := readTask(t, s, id)
		link := "## Requests\n- 2026-11-17T13:30 request 1 by jhf (see " + a + " request 1):\n\n"
		if !strings.Contains(raw, link) || strings.Contains(raw, "same fix on each") {
			t.Fatalf("%s: link block wrong (want %q, no copy of the text):\n%s", id, link, raw)
		}
		l := tk.Requests[0]
		if l.By != "jhf" || l.Home == nil || l.Home.ID != a || l.Home.N != 1 || l.Text != "" || !l.Open() {
			t.Fatalf("%s link: %+v", id, l)
		}
	}
	// One task named is a plain request, as before.
	r, e = s.RequestFor([]string{c}, "just this one", "jhf")
	if e != nil {
		t.Fatal(e)
	}
	if tk, raw := readTask(t, s, c); len(tk.Requests) != 2 || tk.Requests[1].Targets != nil || !strings.Contains(raw, "- 2026-11-17T13:30 request 2 by jhf:\n  just this one\n") {
		t.Fatalf("single request: %+v\n%s", tk.Requests, raw)
	}
}

// ec6z6j (crab 15:48, ruling 3): an unknown ID refuses the whole request
// before anything is written, exit 2, with the request on the known IDs
// as the fix.
func TestRequestForUnknownWritesNothing(t *testing.T) {
	s, ids := manyStore(t)
	before := map[string]string{}
	for _, id := range ids {
		_, before[id] = readTask(t, s, id)
	}
	_, e := s.RequestFor([]string{ids[0], "zzzzzz", ids[2]}, "fix it", "jhf")
	if e == nil {
		t.Fatal("unknown id accepted")
	}
	var te *Error
	if !errors.As(e, &te) || te.ExitCode() != 2 {
		t.Fatalf("exit code: %v", e)
	}
	for _, want := range []string{"no task zzzzzz", "nothing was written", "fix: hug task request " + ids[0] + "," + ids[2] + ` "fix it"`} {
		if !strings.Contains(e.Error(), want) {
			t.Fatalf("error lacks %q: %v", want, e)
		}
	}
	for _, id := range ids {
		if _, raw := readTask(t, s, id); raw != before[id] {
			t.Fatalf("%s was written:\n%s", id, raw)
		}
	}
	// Empty text and no author refuse before any write too.
	for _, c := range []struct{ text, by, want string }{{" \n", "jhf", "request needs the request"}, {"x", "", "request needs who asks"}} {
		if _, e := s.RequestFor(ids, c.text, c.by); e == nil || !strings.Contains(e.Error(), c.want) {
			t.Fatalf("%q by %q: %v", c.text, c.by, e)
		}
	}
	for _, id := range ids {
		if _, raw := readTask(t, s, id); raw != before[id] {
			t.Fatalf("%s was written by a refused request:\n%s", id, raw)
		}
	}
}

// ec6z6j check 3 (crab 15:50): each target resolves on its own, in any
// order, the home too; the request closes only when the last target
// resolves, with "request N closed: all targets resolved" on the home's
// History, so a partial fix never closes unhandled scope.
func TestTargetsResolveInAnyOrder(t *testing.T) {
	s, ids := manyStore(t)
	a, b, c := ids[0], ids[1], ids[2]
	if _, e := s.RequestFor([]string{a, b, c}, "same fix on each", "jhf"); e != nil {
		t.Fatal(e)
	}
	state := func() (Request, string) {
		tk, raw := readTask(t, s, a)
		return tk.Requests[0], raw
	}
	// The home's own part first: the request stays open.
	if _, e := s.Resolve(a, 1, "fixed on a", "nautilus"); e != nil {
		t.Fatal(e)
	}
	rq, raw := state()
	if rq.Closed() || rq.Receipt.State == ReceiptAnswered || !rq.Targets[0].Resolved || rq.Targets[1].Resolved || rq.Targets[2].Resolved {
		t.Fatalf("after the home's part: %+v", rq)
	}
	if strings.Contains(raw, "closed: all targets resolved") {
		t.Fatalf("closed with two targets open:\n%s", raw)
	}
	// The home's part cannot be resolved twice.
	if _, e := s.Resolve(a, 1, "again", "nautilus"); e == nil || !strings.Contains(e.Error(), "already resolved") {
		t.Fatalf("second home resolve: %v", e)
	}
	// c, a link: its own file gets the account, the home a target line.
	if _, e := s.Resolve(c, 1, "fixed on c", "nautilus"); e != nil {
		t.Fatal(e)
	}
	if tk, craw := readTask(t, s, c); tk.Requests[0].Open() || !strings.Contains(craw, "  resolved 2026-11-17T13:30 by nautilus:\n    fixed on c\n") {
		t.Fatalf("c's account:\n%s", craw)
	}
	rq, raw = state()
	if rq.Closed() || !rq.Targets[2].Resolved || rq.Targets[1].Resolved || !strings.Contains(raw, " request 1 target "+c+" resolved by nautilus\n") {
		t.Fatalf("after c: %+v\n%s", rq, raw)
	}
	// b, the last: the request closes, on the home.
	if _, e := s.Resolve(b, 1, "fixed on b", "wyvern"); e != nil {
		t.Fatal(e)
	}
	rq, raw = state()
	if !rq.Closed() || rq.Receipt.State != ReceiptAnswered || !strings.Contains(raw, " request 1 target "+b+" resolved by wyvern\n") || !strings.Contains(raw, " request 1 closed: all targets resolved\n") {
		t.Fatalf("after the last target: %+v\n%s", rq, raw)
	}
	if strings.Count(raw, "closed: all targets resolved") != 1 {
		t.Fatalf("closed twice:\n%s", raw)
	}
}

// ec6z6j: links first, the home last: the home's own resolve closes it.
func TestHomeResolvedLastCloses(t *testing.T) {
	s, ids := manyStore(t)
	a, b := ids[0], ids[1]
	if _, e := s.RequestFor([]string{a, b}, "one fix", "jhf"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Resolve(b, 1, "done on b", "nautilus"); e != nil {
		t.Fatal(e)
	}
	if tk, _ := readTask(t, s, a); tk.Requests[0].Closed() {
		t.Fatal("closed with the home open")
	}
	if _, e := s.Resolve(a, 1, "done on a", "nautilus"); e != nil {
		t.Fatal(e)
	}
	tk, raw := readTask(t, s, a)
	if !tk.Requests[0].Closed() || !strings.Contains(raw, " request 1 closed: all targets resolved\n") {
		t.Fatalf("home last did not close:\n%s", raw)
	}
}
