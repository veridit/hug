package task

import (
	"strings"
	"testing"
)

// 5yh7hr: when a tick hands a task to review (ReviewDue), and the step
// after every check holds on a board with a coordinator.
func TestReviewDue(t *testing.T) {
	people := []Person{{Short: "jhf", Name: "Jørgen"}}
	open, ticked := Acceptance{Text: "a holds"}, Acceptance{Text: "a holds", Checked: true}
	jhfs := Acceptance{Text: "jhf confirms on his board after the release"}
	doing := func(who string, checks ...Acceptance) Task {
		return Task{ID: "abc123", State: "doing", Stage: "active", Assignee: []string{who}, Acceptance: checks}
	}
	approved := doing("wolf", ticked)
	approved.Approval = &Approval{By: "crab"}
	asked := doing("wolf", ticked)
	asked.Questions = []AskedQuestion{{N: 1, By: "wolf", To: "jhf", Text: "ok?"}}
	waiting := doing("wolf", ticked)
	waiting.Stage = "waiting"
	both := doing("wolf", ticked)
	both.Assignee = []string{"wolf", "crab"}
	cases := []struct {
		name  string
		t     Task
		who   string
		coord string
		want  bool
	}{
		{"last own check", doing("wolf", ticked, ticked), "wolf", "crab", true},
		{"only a person's open", doing("wolf", ticked, jhfs), "wolf", "crab", true},
		{"case and @ aside", doing("Wolf", ticked), "@wolf", "crab", true},
		{"an own check open", doing("wolf", ticked, open), "wolf", "crab", false},
		{"the coordinator ticks its own", doing("crab", ticked), "crab", "crab", false},
		{"the coordinator does it too", both, "wolf", "crab", false},
		{"not the assignee", doing("wolf", ticked), "otter", "crab", false},
		{"no coordinator", doing("wolf", ticked), "wolf", "", false},
		{"approved", approved, "wolf", "crab", false},
		{"a question open", asked, "wolf", "crab", false},
		{"waiting", waiting, "wolf", "crab", false},
		{"no checks", doing("wolf"), "wolf", "crab", false},
		{"todo", Task{ID: "abc123", State: "todo", Assignee: []string{"wolf"}, Acceptance: []Acceptance{ticked}}, "wolf", "crab", false},
	}
	for _, c := range cases {
		by, ok := ReviewDue(c.t, people, c.who, c.coord)
		if ok != c.want || ok && by != c.coord {
			t.Errorf("%s: ReviewDue = %q, %v; want %v", c.name, by, ok, c.want)
		}
	}
}

// A doing task whose every check holds, not approved, on a board with a
// coordinator who does not do it: the owner hands it to review, never
// done (5yh7hr); with no coordinator, or the coordinator's own, done.
func TestStepHandsToReview(t *testing.T) {
	ticked := Acceptance{Text: "a", Checked: true}
	wolfs := Task{ID: "abc123", Title: "Steps", State: "doing", Assignee: []string{"wolf"}, Acceptance: []Acceptance{ticked}}
	s := StepOf(wolfs, StepConfig{Coordinator: "crab"})
	if s.Kind != StepDone || s.For != "wolf" || s.Next != "hug task review abc123 --by crab" || s.Cue != "every check of abc123 holds (Steps). Hand it to crab for review: hug task review abc123 --by crab" {
		t.Errorf("with a coordinator: %+v", s)
	}
	if s := StepOf(wolfs, StepConfig{}); s.Next != "hug task done abc123" {
		t.Errorf("no coordinator: %+v", s)
	}
	crabs := wolfs
	crabs.Assignee = []string{"crab"}
	if s := StepOf(crabs, StepConfig{Coordinator: "crab"}); s.Next != "hug task done abc123" {
		t.Errorf("the coordinator's own: %+v", s)
	}
	// StepAt reads hug.toml for a doing task whose checks all hold.
	dir := t.TempDir()
	if e := SetCoordinator(dir, "crab"); e != nil {
		t.Fatal(e)
	}
	if s := StepAt(wolfs, dir); s.Next != "hug task review abc123 --by crab" {
		t.Errorf("StepAt with hug.toml: %+v", s)
	}
	open := wolfs
	open.Acceptance = []Acceptance{{Text: "a"}}
	if s := StepAt(open, dir); s.Next != "hug task check abc123 1" {
		t.Errorf("StepAt with a check open: %+v", s)
	}
}

// OwedTo: what a pending delivery still owes name's next session.
func TestOwedTo(t *testing.T) {
	doc := func(history string) *Document {
		src := "---\nid: abc123\ntitle: T\nstate: doing\nassignee: [\"wolf\"]\n---\n\n## Goal\n\ng\n\n## Requests\n\n- 2026-11-27T10:00 request 1 by jhf:\n  rename it\n\n## History\n\n" + history
		return Parse("a/abc123.md", []byte(src))
	}
	req := Item{Kind: "request", N: 1}
	notDelivered := "- 2026-11-27T10:00 request 1 not delivered to wolf: no jcode session named wolf on this machine\n- 2026-11-27T10:00 request 1 delivered to crab (session_crab_1_00ab)\n"
	d := doc(notDelivered)
	if len(d.Task.Requests) != 1 {
		t.Fatalf("requests: %+v", d.Task.Requests)
	}
	if !d.OwedTo(req, "wolf") {
		t.Errorf("wolf, doing it and not delivered: not owed")
	}
	if d.OwedTo(req, "crab") {
		t.Errorf("crab, delivered: owed")
	}
	acked := doc(notDelivered + "- 2026-11-27T10:01 request 1 acknowledged by crab\n")
	if !acked.OwedTo(req, "wolf") {
		t.Errorf("wolf after crab's ack: not owed (the doer is owed it until it is resolved)")
	}
	if acked.OwedTo(req, "otter") {
		t.Errorf("otter, not doing it, after crab's ack: owed")
	}
	if doc(notDelivered+"- 2026-11-27T10:02 request 1 delivered to wolf (session_wolf_2_00ab)\n").OwedTo(req, "wolf") {
		t.Errorf("wolf after its delivery: owed")
	}
	if doc(notDelivered+"- 2026-11-27T10:02 request 1 acknowledged by wolf\n").OwedTo(req, "wolf") {
		t.Errorf("wolf after its own ack: owed")
	}
	if d.OwedTo(Item{Kind: "request", N: 2}, "wolf") {
		t.Errorf("a request the task does not have: owed")
	}
	// A review is owed to its reviewer while the task is in it.
	inReview := func(extra string) *Document {
		src := "---\nid: abc123\ntitle: T\nstate: doing\nstage: review\nassignee: [\"wolf\"]\nreview_by: crab\n---\n\n## Goal\n\ng\n\n## Acceptance\n\n- [x] a\n\n## History\n\n- 2026-11-27T10:00 doing/active -> doing/review (review: wolf ticked the last own check)\n- 2026-11-27T10:00 review 1 not delivered to crab: no jcode session named crab on this machine\n" + extra
		return Parse("a/abc123.md", []byte(src))
	}
	rv := Item{Kind: "review", N: 1}
	if r := inReview(""); !r.OwedTo(rv, "crab") || r.OwedTo(rv, "otter") {
		t.Errorf("review 1: owed crab %v, otter %v", r.OwedTo(rv, "crab"), r.OwedTo(rv, "otter"))
	}
	if inReview("- 2026-11-27T10:05 review 1 delivered to crab (session_crab_3_00ab)\n").OwedTo(rv, "crab") {
		t.Errorf("review 1 after its delivery: owed")
	}
	if inReview("").OwedTo(Item{Kind: "review", N: 2}, "crab") {
		t.Errorf("review 2, never moved into: owed")
	}
	left := inReview("")
	left.Task.Fields["review_by"] = ""
	if left.OwedTo(rv, "crab") {
		t.Errorf("review answered (no review_by): owed")
	}
	if it, ok := inReview("").NewestItem("review"); !ok || it.Name() != "review 1" {
		t.Errorf("NewestItem(review) = %+v %v", it, ok)
	}
	if d := inReview(""); d.Task.Review == nil || d.Task.Review.Receipt.NotDelivered == "" {
		t.Errorf("Task.Review: %+v", d.Task.Review)
	}
}

// StartReview moves the doer's task to doing/review by the coordinator
// once, with the History line; a shipped task is the release step's
// (AfterTick), so it stays out of review.
func TestStartReview(t *testing.T) {
	s, id := newStore(t)
	if e := SetCoordinator(s.Root, "crab"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Take(id, "wolf"); e != nil {
		t.Fatal(e)
	}
	if by, e := s.StartReview(id, "wolf"); by != "" || e != nil {
		t.Fatalf("with its check open: %q %v", by, e)
	}
	if _, e := s.CheckWith(id, 1, ""); e != nil {
		t.Fatal(e)
	}
	shipped, _ := s.New(NewOptions{Title: "Shipped", Goal: "g", Acceptance: []string{"a"}})
	if _, e := s.Take(shipped.Task.ID, "otter"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Note(shipped.Task.ID, "shipped in v0.7.1 (1111111)"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.CheckWith(shipped.Task.ID, 1, ""); e != nil {
		t.Fatal(e)
	}
	if by, e := s.StartReview(shipped.Task.ID, "otter"); by != "" || e != nil {
		t.Fatalf("a shipped task entered review: %q %v", by, e)
	}
	by, e := s.StartReview(id, "wolf")
	if by != "crab" || e != nil {
		t.Fatalf("StartReview = %q %v", by, e)
	}
	d, _ := s.Get(id)
	if d.Task.Stage != "review" || d.Task.Fields["review_by"] != "crab" || !strings.Contains(string(d.Bytes()), " doing/active -> doing/review (review: wolf ticked the last own check)") {
		t.Fatalf("after StartReview:\n%s", d.Bytes())
	}
	if by, _ := s.StartReview(id, "wolf"); by != "" {
		t.Fatalf("a second StartReview moved it again: %q", by)
	}
}
