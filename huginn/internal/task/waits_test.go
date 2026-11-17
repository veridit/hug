package task

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/internal/cond"
)

func nextID(t *testing.T, s *Store) string {
	t.Helper()
	ts, _ := s.Tasks()
	r, e := Next(ts, Filter{})
	if e != nil {
		return ""
	}
	return r.Task.ID
}

func setStatus(t *testing.T, s *Store, id, status string) {
	t.Helper()
	switch status {
	case "done":
		// done only from doing (6ephqk gap 4): take it first.
		if _, e := s.Check(id, 1); e != nil {
			t.Fatal(e)
		}
		if _, e := s.Take(id, "x"); e != nil {
			t.Fatal(e)
		}
	case "dropped":
		// A drop needs a reason (6ephqk).
		if _, e := s.Drop(id, "not needed"); e != nil {
			t.Fatal(e)
		}
		return
	}
	if _, e := s.Set(id, "status", status); e != nil {
		t.Fatal(e)
	}
}

// 16dxsr check 1: a task depending on a group is blocked until every task
// in the group (and its subgroups) is done or dropped.
func TestDependOnGroup(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	waiter := newTest(t, s, "Waiter")
	a := newTest(t, s, "Auth one")
	b := newTest(t, s, "Auth two")
	c := newTest(t, s, "Auth deep")
	for id, g := range map[string]string{a.ID: "auth", b.ID: "auth", c.ID: "auth/deep"} {
		if _, e := s.MoveGroup(id, g); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.AddDependencies(waiter.ID, []string{"group:auth"}); e != nil {
		t.Fatal(e)
	}
	d, _ := s.Get(waiter.ID)
	if !strings.Contains(string(d.Bytes()), "\n- when group auth closed\n") {
		t.Fatalf("file:\n%s", d.Bytes())
	}
	// A group with no task cannot be checked: refused.
	if _, e := s.AddDependencies(waiter.ID, []string{"group:nosuch"}); e == nil || !strings.Contains(e.Error(), "no task is in group nosuch") {
		t.Fatalf("empty group: %v", e)
	}
	ts, _ := s.Tasks()
	byID := func(id string) Task {
		for _, x := range ts {
			if x.ID == id {
				return x
			}
		}
		t.Fatal("no", id)
		return Task{}
	}
	w := byID(waiter.ID)
	if got := Unmet(w, ts, false); len(got) != 1 || got[0].Seen != "3 open in group auth" || !got[0].Board {
		t.Fatalf("unmet: %+v", got)
	}
	if nextID(t, s) == waiter.ID {
		t.Fatal("next offered the waiting task")
	}
	if _, e := s.Take(waiter.ID, "x"); e == nil || !strings.Contains(e.Error(), "when group auth closed (3 open in group auth)") || !strings.Contains(e.Error(), "fix: hug task show "+waiter.ID) {
		t.Fatalf("take while waiting: %v", e)
	}
	setStatus(t, s, a.ID, "done")
	setStatus(t, s, b.ID, "dropped")
	ts, _ = s.Tasks()
	if got := Unmet(byID(waiter.ID), ts, false); len(got) != 1 || got[0].Seen != "1 open in group auth" {
		t.Fatalf("subgroup task still open, want waiting: %+v", got)
	}
	setStatus(t, s, c.ID, "done")
	ts, _ = s.Tasks()
	if got := Waits(byID(waiter.ID), ts); len(got) != 1 || !got[0].Met || got[0].Seen != "every task in group auth is closed" {
		t.Fatalf("waits: %+v", got)
	}
	if id := nextID(t, s); id != waiter.ID {
		t.Fatalf("next = %s, want %s", id, waiter.ID)
	}
	// Removed by what it says.
	if r, e := s.RemoveDependencies(waiter.ID, []string{"when group auth closed"}); e != nil || !r.Changed {
		t.Fatal(r, e)
	}
	if d, _ := s.Get(waiter.ID); len(d.Task.Waits) != 0 {
		t.Fatalf("still waits: %+v", d.Task.Waits)
	}
}

// A task in the group it waits on does not wait on itself.
func TestGroupWaitIgnoresItself(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	a := newTest(t, s, "Last in auth")
	if _, e := s.MoveGroup(a.ID, "auth"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.AddDependencies(a.ID, []string{"group:auth"}); e != nil {
		t.Fatal(e)
	}
	ts, _ := s.Tasks()
	if got := Unmet(ts[0], ts, false); len(got) != 0 {
		t.Fatalf("waits on itself: %+v", got)
	}
}

// 16dxsr check 2: a task depending on an epic is blocked until all its
// children are closed, whatever the epic's own status.
func TestDependOnEpicWaitsForChildren(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	epic := newTest(t, s, "Epic")
	k1 := newTest(t, s, "Kid one")
	k2 := newTest(t, s, "Kid two")
	for _, k := range []string{k1.ID, k2.ID} {
		if _, e := s.Set(k, "parent", epic.ID); e != nil {
			t.Fatal(e)
		}
	}
	waiter := newTest(t, s, "After the epic", epic.ID)
	blocked := func() []Task {
		ts, _ := s.Tasks()
		for _, x := range ts {
			if x.ID == waiter.ID {
				return Blocked(x, ts)
			}
		}
		return nil
	}
	if b := blocked(); len(b) != 1 || b[0].ID != epic.ID {
		t.Fatalf("blocked: %+v", b)
	}
	ts, _ := s.Tasks()
	for _, x := range ts {
		if x.ID == waiter.ID {
			if dd := DependencyDetails(x, ts); len(dd) != 1 || !dd[0].Blocking || dd[0].OpenChildren != 2 {
				t.Fatalf("details: %+v", dd)
			}
		}
	}
	setStatus(t, s, k1.ID, "done")
	if b := blocked(); len(b) != 1 {
		t.Fatal("one child open, want blocked")
	}
	setStatus(t, s, k2.ID, "dropped")
	// The epic itself is still todo: its children are all closed, so the
	// dependency is settled.
	if b := blocked(); len(b) != 0 {
		t.Fatalf("children closed, still blocked by %+v", b)
	}
	if _, e := s.Take(waiter.ID, "x"); e != nil {
		t.Fatal(e)
	}
}

// 16dxsr check 3: 'load below 120' blocks next while the 5-minute load is
// above 120 and frees it when below (injected load).
func TestLoadConditionBlocksNext(t *testing.T) {
	load := 140.0
	old := cond.LoadAverage
	cond.LoadAverage = func() (float64, bool) { return load, true }
	t.Cleanup(func() { cond.LoadAverage = old })
	s := &Store{Root: t.TempDir()}
	w := newTest(t, s, "Heavy benchmark")
	other := newTest(t, s, "Light work")
	if _, e := s.AddDependencies(w.ID, []string{"when load below 120 wakes kestrel"}); e != nil {
		t.Fatal(e)
	}
	d, _ := s.Get(w.ID)
	if !strings.Contains(string(d.Bytes()), "\n- when load below 120 wakes kestrel\n") {
		t.Fatalf("file:\n%s", d.Bytes())
	}
	if id := nextID(t, s); id != other.ID {
		t.Fatalf("load 140: next = %s, want %s", id, other.ID)
	}
	ts, _ := s.Tasks()
	if got := Unmet(ts[0], ts, false); len(got) != 1 || got[0].Seen != "load is 140.0" || got[0].Wakes != "kestrel" || got[0].Board {
		t.Fatalf("unmet: %+v", got)
	}
	load = 80
	if id := nextID(t, s); id != w.ID {
		t.Fatalf("load 80: next = %s, want %s", id, w.ID)
	}
	// Load only says when to start: once taken, done does not wait on it.
	if _, e := s.Take(w.ID, "x"); e != nil {
		t.Fatal(e)
	}
	load = 500
	if _, e := s.Check(w.ID, 1); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Done(w.ID); e != nil {
		t.Fatalf("done waited on load: %v", e)
	}
	// Unknown load (Windows): never met.
	cond.LoadAverage = func() (float64, bool) { return 0, false }
	x := newTest(t, s, "Needs a load average")
	if _, e := s.AddDependencies(x.ID, []string{"when load < 10"}); e != nil {
		t.Fatal(e)
	}
	ts, _ = s.Tasks()
	for _, y := range ts {
		if y.ID == x.ID {
			if ws := Waits(y, ts); len(ws) != 1 || ws[0].Met || !ws[0].Unknown || ws[0].Text != "load below 10" {
				t.Fatalf("unknown load: %+v", ws)
			}
			fs := WaitFindings(ts)
			if len(fs) != 1 || fs[0].Code != "unknown-condition" || !strings.Contains(fs[0].Problem, "no load average on this system") || fs[0].Fix != "hug task dep "+x.ID+" --remove 'when load below 10'" {
				t.Fatalf("findings: %+v", fs)
			}
		}
	}
}

// 16dxsr check 4 (lint half): a when line hug cannot read is a wait that
// never holds, and lint names it with a fix: line; dep --add refuses it.
func TestUnknownConditionLintAndRefusal(t *testing.T) {
	d := Parse("zzzzzz.k7m2qx.recall-gate.md", withDeps("- when the moon is full\n"))
	if len(d.Task.Waits) != 1 || len(d.Task.Dependencies) != 0 {
		t.Fatalf("waits %+v deps %+v", d.Task.Waits, d.Task.Dependencies)
	}
	fs := Lint([]*Document{d})
	if len(fs) != 1 {
		t.Fatalf("findings: %v", fs)
	}
	f := fs[0]
	if FindingCode(f.Problem) != "unknown-condition" || f.Line != 15 || f.FixLine() != "hug task dep k7m2qx --remove 'when the moon is full'" ||
		!strings.HasPrefix(f.Problem, `unknown condition: "when the moon is full": "the moon is full" is not a condition hug can check (it can check: date YYYY-MM-DD, load below N`) {
		t.Fatalf("finding: %s\nfix: %s", f, f.FixLine())
	}
	if w := Waits(d.Task, []Task{d.Task}); len(w) != 1 || w[0].Met || !w[0].Unknown {
		t.Fatalf("unknown condition must not hold: %+v", w)
	}
	s := &Store{Root: t.TempDir()}
	x := newTest(t, s, "X")
	if _, e := s.AddDependencies(x.ID, []string{"when pigs fly"}); e == nil || !strings.Contains(e.Error(), "invalid-condition") || !strings.Contains(e.Error(), "fix: hug task dep "+x.ID+" --add 'when load below 120'") {
		t.Fatalf("add unknown: %v", e)
	}
}

func TestWaitLineForms(t *testing.T) {
	for in, want := range map[string]string{
		"- when load below 120":                      "load below 120",
		"when load under 8.5":                        "load below 8.5",
		"* when 2026-12-01":                          "date 2026-12-01",
		"- when date 2026-12-01 wakes crab":          "date 2026-12-01",
		"- when branch feat/x merged":                "branch feat/x merged",
		"- when branch feat/x is merged at 1a2b3c4d": "branch feat/x merged at 1a2b3c4d",
		"- when release v0.7.40 exists":              "release v0.7.40 exists",
		"- when tag v1.0 exists":                     "tag v1.0 exists",
		"- when task:k7m2qx closed":                  "task:k7m2qx closed",
		"- group:auth/deep":                          "group auth/deep closed",
		"- When Group auth closed":                   "group auth closed",
	} {
		w, err, ok := parseWaitLine(in)
		if !ok || err != nil || w.Text != want {
			t.Errorf("%q: %+v %v %v, want %q", in, w, err, ok, want)
		}
	}
	for _, in := range []string{"- k7m2qx", "- TASK-12 when it lands", "whenever", "- groups are fun"} {
		if w, _, ok := parseWaitLine(in); ok {
			t.Errorf("%q read as wait %+v", in, w)
		}
	}
}

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@x.invalid", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@x.invalid")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// Branch and tag conditions read the checkout: a branch merged then
// deleted still reads as merged through its recorded tip.
func TestBranchAndTagConditions(t *testing.T) {
	repo := t.TempDir()
	gitT(t, repo, "init", "-b", "main")
	gitT(t, repo, "commit", "--allow-empty", "-m", "init")
	gitT(t, repo, "checkout", "-b", "feat/x")
	gitT(t, repo, "commit", "--allow-empty", "-m", "work")
	gitT(t, repo, "checkout", "main")
	st := cond.Local(repo, func() string { return "main" })
	cond.MemoFor = 0
	t.Cleanup(func() { cond.MemoFor = 10e9 })
	check := func(text string) (bool, string) {
		c, err := cond.Parse(text)
		if err != nil {
			t.Fatal(err)
		}
		return c.Met(st)
	}
	if met, seen := check("branch feat/x merged"); met || seen != "branch feat/x is not merged into main" {
		t.Fatal(met, seen)
	}
	tip := cond.Tip(repo, "feat/x")
	if len(tip) != 12 {
		t.Fatalf("tip %q", tip)
	}
	if met, _ := check("tag v1 exists"); met {
		t.Fatal("no tag yet")
	}
	gitT(t, repo, "merge", "--ff-only", "feat/x")
	gitT(t, repo, "branch", "-d", "feat/x")
	gitT(t, repo, "tag", "v1")
	if met, seen := check("branch feat/x merged at " + tip); !met || seen != "branch feat/x is merged into main" {
		t.Fatal(met, seen)
	}
	if met, seen := check("branch feat/x merged"); met || seen != "unknown: no branch feat/x in this clone" {
		t.Fatal("deleted without a tip must be unknown:", met, seen)
	}
	if met, _ := check("release v1 exists"); !met {
		t.Fatal("tag v1 exists")
	}
	_ = filepath.Join
}
