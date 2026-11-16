package task

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// et85h1: TakeAs requires the model; without a name the assignee is the
// model; both are written to the file and read back.
func TestTakeAsModelRequiredAndRecorded(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	a := newTest(t, s, "A")
	var te *Error
	if _, e := s.TakeAs(a.ID, "poodle", " "); !errors.As(e, &te) || te.Kind != "no-model" || !strings.Contains(te.Fix, "--model MODEL") {
		t.Fatalf("no model: %v", e)
	}
	r, e := s.TakeAs(a.ID, "", "gpt-6.1-sol")
	if e != nil || !r.Changed || len(r.Task.Assignee) != 1 || r.Task.Assignee[0] != "gpt-6.1-sol" || r.Task.Model != "gpt-6.1-sol" {
		t.Fatal(r.Task, e)
	}
	// Repeating is a no-op; another model under the same name updates the model.
	if r, e = s.TakeAs(a.ID, "", "gpt-6.1-sol"); e != nil || r.Changed {
		t.Fatal("repeat", r, e)
	}
	b := newTest(t, s, "B")
	if r, e = s.TakeAs(b.ID, "poodle", "claude-opus-5-5"); e != nil || r.Task.Assignee[0] != "poodle" || r.Task.Model != "claude-opus-5-5" {
		t.Fatal(r.Task, e)
	}
	raw, _ := os.ReadFile(filepath.Join(s.Root, r.Task.Path))
	if !strings.Contains(string(raw), "model: \"claude-opus-5-5\"\n") || !strings.Contains(string(raw), "assignee: [\"poodle\"]\n") {
		t.Fatal(string(raw))
	}
	d := Parse(r.Task.Path, raw)
	if d.Task.Model != "claude-opus-5-5" || len(d.Findings) != 0 {
		t.Fatal(d.Task.Model, d.Findings)
	}
	if _, e = s.TakeAs(b.ID, "crab", "claude-opus-5-5"); !errors.As(e, &te) || te.Kind != "taken" {
		t.Fatalf("stolen: %v", e)
	}
}

// x1t1t1: TakeAt records the branch and worktree next to the model; taking
// again from another worktree moves them; an empty place leaves them.
func TestTakeAtRecordsBranchAndWorktree(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	a := newTest(t, s, "A")
	r, e := s.TakeAt(a.ID, "dragon", "claude-opus-5-5", Place{Branch: "feat/x", Worktree: "/w/feat-x"})
	if e != nil || r.Task.Branch != "feat/x" || r.Task.Worktree != "/w/feat-x" || r.Task.Model != "claude-opus-5-5" {
		t.Fatal(r.Task, e)
	}
	raw, _ := os.ReadFile(filepath.Join(s.Root, r.Task.Path))
	if !strings.Contains(string(raw), "branch: \"feat/x\"\n") || !strings.Contains(string(raw), "worktree: \"/w/feat-x\"\n") {
		t.Fatal(string(raw))
	}
	if d := Parse(r.Task.Path, raw); d.Task.Branch != "feat/x" || len(d.Findings) != 0 {
		t.Fatal(d.Task.Branch, d.Findings)
	}
	if r, e = s.TakeAt(a.ID, "dragon", "claude-opus-5-5", Place{Branch: "feat/x", Worktree: "/w/feat-x"}); e != nil || r.Changed {
		t.Fatal("repeat", r, e)
	}
	if r, e = s.TakeAt(a.ID, "dragon", "claude-opus-5-5", Place{Branch: "feat/y", Worktree: "/w/feat-y"}); e != nil || !r.Changed || r.Task.Branch != "feat/y" {
		t.Fatal("move", r.Task, e)
	}
	if r, e = s.TakeAs(a.ID, "dragon", "claude-opus-5-5"); e != nil || r.Task.Branch != "feat/y" {
		t.Fatal("no place keeps it", r.Task, e)
	}
}
