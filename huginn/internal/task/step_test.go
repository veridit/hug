package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// erv680: the step due on a task, derived from its state alone. One
// command per step; a review is the reviewer's, ship it the coordinator's.
func TestStepOf(t *testing.T) {
	approved := &Approval{By: "crab", At: "2026-11-27T10:00", Text: "looks right"}
	doing := func(checks ...Acceptance) Task {
		return Task{ID: "abc123", Title: "Steps", State: "doing", Assignee: []string{"wolf"}, Acceptance: checks}
	}
	open, ticked, struck := Acceptance{Text: "a"}, Acceptance{Text: "a", Checked: true}, Acceptance{Text: "a", Struck: true}
	review := doing(ticked)
	review.Fields = map[string]string{"review_by": "crab"}
	review.Stage = "review"
	ship := doing(ticked)
	ship.Approval = approved
	shipOpen := doing(open)
	shipOpen.Approval = approved
	shipWaiting := doing(ticked)
	shipWaiting.Approval, shipWaiting.Stage, shipWaiting.Fields = approved, "waiting", map[string]string{"waiting_on": "CI"}
	asked := doing(ticked)
	asked.Questions = []AskedQuestion{{N: 1, By: "wolf", To: "jhf", Text: "ok?"}}
	cases := []struct {
		name string
		t    Task
		c    StepConfig
		kind string
		for_ string
		next string
	}{
		{"todo", Task{ID: "abc123", State: "todo"}, StepConfig{}, StepTake, "", "hug task take abc123"},
		{"no checks", doing(), StepConfig{}, StepChecks, "wolf", `hug task set abc123 --accept "observable check"`},
		{"open check", doing(ticked, open), StepConfig{}, StepCheck, "wolf", "hug task check abc123 2"},
		{"struck holds", doing(struck, ticked), StepConfig{}, StepDone, "wolf", "hug task done abc123"},
		{"in review", review, StepConfig{}, StepReview, "crab", `hug task answer abc123 "looks right"`},
		{"question open", asked, StepConfig{}, StepAnswer, "", "hug task next"},
		{"approved, no ship line", ship, StepConfig{Coordinator: "crab"}, StepShip, "crab", "ship it, as your AGENTS.md says; once it has shipped: hug task done abc123"},
		{"approved, ship line", ship, StepConfig{Coordinator: "crab", Ship: "deploy with make deploy"}, StepShip, "crab", "ship it, as hug.toml says: deploy with make deploy; once it has shipped: hug task done abc123"},
		{"approved, no coordinator", ship, StepConfig{}, StepShip, "wolf", "ship it, as your AGENTS.md says; once it has shipped: hug task done abc123"},
		{"approved, a check reopened", shipOpen, StepConfig{Coordinator: "crab"}, StepCheck, "wolf", "hug task check abc123 1"},
		{"approved, waiting on CI", shipWaiting, StepConfig{Coordinator: "crab"}, StepDone, "wolf", "hug task done abc123"},
		{"done", Task{ID: "abc123", State: "done"}, StepConfig{}, StepNone, "", "hug task next"},
		{"dropped", Task{ID: "abc123", State: "dropped"}, StepConfig{}, StepNone, "", "hug task next"},
	}
	for _, c := range cases {
		s := StepOf(c.t, c.c)
		if s.Kind != c.kind || s.For != c.for_ || s.Next != c.next {
			t.Errorf("%s: got %+v, want kind %s for %q next %q", c.name, s, c.kind, c.for_, c.next)
		}
		if n := strings.Count(s.Next, "hug task "); n != 1 {
			t.Errorf("%s: next %q names %d commands, want 1", c.name, s.Next, n)
		}
		if s.Cue != "" && (!strings.HasSuffix(s.Cue, s.Next) || strings.Count(s.Cue, "hug task ") != 1) {
			t.Errorf("%s: cue %q does not end in its one command %q", c.name, s.Cue, s.Next)
		}
		for _, word := range []string{"tag", "release", "tested sha", "land "} {
			if strings.Contains(strings.ToLower(s.Cue+" "+s.Next), word) {
				t.Errorf("%s: names a procedure (%q): %+v", c.name, word, s)
			}
		}
	}
	// NextFor: a step due for another role is not the owner's.
	if n := StepOf(review, StepConfig{}).NextFor("wolf"); n != "hug task next" {
		t.Errorf("owner of a task in review: %q", n)
	}
	if n := StepOf(review, StepConfig{}).NextFor("Crab"); n != `hug task answer abc123 "looks right"` {
		t.Errorf("reviewer: %q", n)
	}
	if n := StepOf(ship, StepConfig{Coordinator: "crab"}).NextFor("wolf"); n != "hug task next" {
		t.Errorf("owner after approval: %q", n)
	}
	if n := StepOf(doing(open), StepConfig{}).NextFor("crab"); n != "hug task check abc123 1" {
		t.Errorf("a check step is anyone's to see: %q", n)
	}
}

func TestApproves(t *testing.T) {
	for text, want := range map[string]bool{
		"looks right":                       true,
		"Looks right.":                      true,
		"looks  right, ship it":             true,
		"LGTM":                              true,
		"approved":                          true,
		"looks right but rename the flag":   false,
		"rename the flag, then looks right": false,
		"not yet":                           false,
		"":                                  false,
		"no":                                false,
	} {
		if got := Approves(text); got != want {
			t.Errorf("Approves(%q) = %v, want %v", text, got, want)
		}
	}
}

// The approval is the newest review answer, until a new review or the task
// is given back; a design or asked answer is no approval.
func TestReadApproval(t *testing.T) {
	block := func(stage, by, text string) []string {
		return []string{"- 2026-11-27T10:00 answered " + stage + ", by " + by + ", for wolf (doing/review -> doing/active):", "  > review by " + by, "  " + text}
	}
	cat := func(parts ...[]string) []string {
		var out []string
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	take := []string{"- 2026-11-27T09:00 todo -> doing/active (take)"}
	cases := []struct {
		name string
		h    []string
		by   string
	}{
		{"approved", cat(take, block("review", "crab", "looks right")), "crab"},
		{"changes asked", cat(take, block("review", "crab", "rename the flag")), ""},
		{"changes then approved", cat(take, block("review", "crab", "rename it"), block("review", "jhf", "looks right")), "jhf"},
		{"approved then changes", cat(take, block("review", "crab", "looks right"), block("review", "jhf", "no, rename it")), ""},
		{"a new review", cat(take, block("review", "crab", "looks right"), []string{"- 2026-11-27T11:00 doing/active -> doing/review (review)"}), ""},
		{"given back", cat(take, block("review", "crab", "looks right"), []string{"- 2026-11-27T11:00 doing/active -> todo/ready"}), ""},
		{"a design answer", cat(take, block("asked", "jhf", "looks right")), ""},
		{"old one-line answer", cat(take, []string{`- 2026-11-17T14:30 answered review: "review by crab" -> "looks right", by crab, for wolf (doing/review -> doing/active)`}), "crab"},
		{"a note after", cat(take, block("review", "crab", "looks right"), []string{"- 2026-11-27T11:00 crab read the answer: hug task note abc123 \"x\""}), "crab"},
	}
	for _, c := range cases {
		a := readApproval(c.h)
		got := ""
		if a != nil {
			got = a.By
		}
		if got != c.by {
			t.Errorf("%s: approval by %q, want %q (%+v)", c.name, got, c.by, a)
		}
	}
}

func TestShipLine(t *testing.T) {
	dir := t.TempDir()
	if s, e := ShipLine(dir); s != "" || e != nil {
		t.Fatalf("no hug.toml: %q %v", s, e)
	}
	os.WriteFile(filepath.Join(dir, ConfigName), []byte("layout = 3\nship = \"not in board\"\n\n[board]\ncoordinator = \"crab\"\nship = \"merge  the branch\"\n\n[people.jhf]\nship = \"no\"\n"), 0o644)
	if s, e := ShipLine(dir); s != "merge the branch" || e != nil {
		t.Fatalf("ship line: %q %v", s, e)
	}
	if c := StepConfigOf(dir); c.Coordinator != "crab" || c.Ship != "merge the branch" {
		t.Fatalf("config: %+v", c)
	}
}
