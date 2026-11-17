package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stagedTask writes a task file that already has stages (as hug upgrade
// will write them), with deps, and returns its id.
func stagedTask(t *testing.T, s *Store, id, front string, deps ...string) string {
	t.Helper()
	body := "---\nid: \"" + id + "\"\ntitle: \"T " + id + "\"\n" + front + "\ntype: \"feature\"\npriority: \"medium\"\nlabels: []\nassignee: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n"
	if len(deps) > 0 {
		body += "\n## Depends on\n"
		for _, d := range deps {
			body += "- " + d + "\n"
		}
	}
	body += "\n## Acceptance\n- [ ] a\n"
	if e := os.WriteFile(filepath.Join(s.Root, id+".t.md"), []byte(body), 0o644); e != nil {
		t.Fatal(e)
	}
	return id
}

func load(t *testing.T, s *Store, id string) (Task, string) {
	t.Helper()
	d, e := s.Get(id)
	if e != nil {
		t.Fatal(e)
	}
	return d.Task, string(d.Bytes())
}

func node(t Task) string {
	if t.Stage == "" {
		return t.State
	}
	return t.State + "/" + t.Stage
}

func fixedClock(t *testing.T) {
	old := Now
	Now = func() time.Time { return time.Date(2026, 11, 17, 9, 41, 0, 0, time.UTC) }
	t.Cleanup(func() { Now = old })
}

func wantRefused(t *testing.T, e error, fix string) {
	t.Helper()
	var te *Error
	if !errors.As(e, &te) || te.Fix != fix {
		t.Fatalf("want refusal with fix %q, got %v", fix, e)
	}
	if strings.Contains(te.Fix, "hug task edit") {
		t.Fatalf("a fix must run without an editor: %q", te.Fix)
	}
}

// 6ephqk check 3: take, wait, resume, design, done and drop move along
// the allowed transitions and write the node fields; leaving a node moves
// its fields to History; other moves are refused with a fix: line.
func TestMovesWriteNodeFields(t *testing.T) {
	fixedClock(t)
	s := &Store{Root: t.TempDir()}
	a := stagedTask(t, s, "aaaaaa", "state: \"todo\"\nstage: \"ready\"")

	// todo -> done is refused: take before done (Jørgen 11:37).
	_, e := s.Done(a)
	wantRefused(t, e, "hug task take aaaaaa --model MODEL")

	// design: todo/design with the question, when and who.
	_, e = s.Design(a, "keep type on rows?", "crab", "")
	wantRefused(t, e, `hug task design aaaaaa "keep type on rows?" --for NAME`)
	if _, e = s.Design(a, "keep type on rows?", "crab", "jhf"); e != nil {
		t.Fatal(e)
	}
	tk, raw := load(t, s, a)
	if node(tk) != "todo/design" || tk.Fields["design_question"] != "keep type on rows?" || tk.Fields["design_asked_by"] != "crab" || tk.Fields["design_for"] != "jhf" || tk.Fields["design_since"] != "2026-11-17T09:41" {
		t.Fatalf("design: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	if !strings.Contains(raw, "## History\n- 2026-11-17T09:41 todo/ready -> todo/design (design)\n") {
		t.Fatalf("design history:\n%s", raw)
	}
	// design --clear: back to ready, the question to History.
	if _, e = s.Design(a, "", "", ""); e != nil {
		t.Fatal(e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "todo/ready" || len(tk.Fields) != 0 || !strings.Contains(raw, "todo/design -> todo/ready (design answered); was design_asked_by: crab; design_for: jhf; design_question: keep type on rows?; design_since: 2026-11-17T09:41") {
		t.Fatalf("design --clear: %s %v\n%s", node(tk), tk.Fields, raw)
	}

	// wait and resume are for a doing task only.
	_, e = s.Wait(a, "CI", "")
	wantRefused(t, e, "hug task take aaaaaa --model MODEL")

	// take: doing/active, doing_since.
	if _, e = s.TakeAs(a, "nautilus", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	tk, _ = load(t, s, a)
	if node(tk) != "doing/active" || tk.Fields["doing_since"] != "2026-11-17T09:41" {
		t.Fatalf("take: %s %v", node(tk), tk.Fields)
	}
	_, e = s.Design(a, "q", "x", "jhf")
	wantRefused(t, e, "hug task set aaaaaa --state todo")
	_, e = s.Resume(a)
	wantRefused(t, e, "hug task show aaaaaa")

	// wait: doing/waiting with what and since; resume: back to active.
	if _, e = s.Wait(a, "CI run 44057678727", "nautilus"); e != nil {
		t.Fatal(e)
	}
	tk, _ = load(t, s, a)
	if node(tk) != "doing/waiting" || tk.Fields["waiting_on"] != "CI run 44057678727" || tk.Fields["waiting_since"] == "" || tk.Fields["waiting_asked_by"] != "nautilus" {
		t.Fatalf("wait: %s %v", node(tk), tk.Fields)
	}
	if _, e = s.Resume(a); e != nil {
		t.Fatal(e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "doing/active" || tk.Fields["waiting_on"] != "" || !strings.Contains(raw, "doing/waiting -> doing/active (resume); was waiting_asked_by: nautilus; waiting_on: CI run 44057678727; waiting_since: 2026-11-17T09:41") {
		t.Fatalf("resume: %s %v\n%s", node(tk), tk.Fields, raw)
	}

	// done: the doing fields go to History, done_at is written, no stage.
	if _, e = s.Check(a, 1); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Done(a); e != nil {
		t.Fatal(e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "done" || tk.Fields["done_at"] != "2026-11-17T09:41" || tk.Fields["doing_since"] != "" || strings.Contains(raw, "\nstage:") {
		t.Fatalf("done: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	if d := Parse(tk.Path, []byte(raw)); len(d.Findings) != 0 {
		t.Fatal(d.Findings)
	}

	// Reopen (Jørgen 11:40): done -> todo, done_at to History, the stage is
	// derived again.
	if _, e = s.Set(a, "state", "todo"); e != nil {
		t.Fatal(e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "todo/ready" || tk.Fields["done_at"] != "" || !strings.Contains(raw, "done -> todo/ready (reopen); was done_at: 2026-11-17T09:41") {
		t.Fatalf("reopen: %s %v\n%s", node(tk), tk.Fields, raw)
	}

	// drop needs a reason; with one it writes dropped_reason and _at.
	_, e = s.Drop(a, " ")
	wantRefused(t, e, `hug task drop aaaaaa "duplicate of OTHER"`)
	_, e = s.Set(a, "state", "dropped")
	wantRefused(t, e, `hug task drop aaaaaa "duplicate of OTHER"`)
	if _, e = s.Drop(a, "duplicate of 805p6y"); e != nil {
		t.Fatal(e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "dropped" || tk.Fields["dropped_reason"] != "duplicate of 805p6y" || tk.Fields["dropped_at"] == "" {
		t.Fatalf("drop: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	_, e = s.Set(a, "state", "done")
	wantRefused(t, e, "hug task show aaaaaa")
	// Reopen from dropped too; the reason goes to History.
	if _, e = s.Set(a, "state", "todo"); e != nil {
		t.Fatal(e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "todo/ready" || !strings.Contains(raw, "was dropped_at: 2026-11-17T09:41; dropped_reason: duplicate of 805p6y") {
		t.Fatalf("reopen dropped: %s\n%s", node(tk), raw)
	}
	if d := Parse(tk.Path, []byte(raw)); len(d.Findings) != 0 {
		t.Fatal(d.Findings)
	}
}

// Taking todo/review makes it doing/review with review_by = who takes;
// giving it back keeps the review stage (6ephqk check 6, store side).
func TestTakeForReviewAndGiveBack(t *testing.T) {
	fixedClock(t)
	s := &Store{Root: t.TempDir()}
	a := stagedTask(t, s, "aaaaaa", "state: \"todo\"\nstage: \"review\"\nreview_since: \"2026-11-17T08:00\"")
	if _, e := s.TakeAs(a, "Jørgen", "human"); e != nil {
		t.Fatal(e)
	}
	tk, raw := load(t, s, a)
	if node(tk) != "doing/review" || tk.Fields["review_by"] != "Jørgen" || tk.Fields["doing_since"] == "" {
		t.Fatalf("take for review: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	if _, e := s.Set(a, "state", "todo"); e != nil {
		t.Fatal(e)
	}
	tk, raw = load(t, s, a)
	if node(tk) != "todo/review" || tk.Fields["doing_since"] != "" || tk.Fields["review_by"] != "Jørgen" || len(tk.Assignee) != 0 || !strings.Contains(raw, "(give back; was assignee: Jørgen)") {
		t.Fatalf("give back: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	if d := Parse(tk.Path, []byte(raw)); len(d.Findings) != 0 {
		t.Fatal(d.Findings)
	}
}

// A dependency's state change re-settles its direct dependents (crab
// 11:37: bounded; a chain 10 deep and a cycle).
func TestStageCascadeChainAndCycle(t *testing.T) {
	fixedClock(t)
	s := &Store{Root: t.TempDir()}
	// t0 <- t1 <- ... <- t9: each depends on the one before.
	ids := make([]string, 10)
	for i := range ids {
		ids[i] = fmt.Sprintf("c%05d", i)
		front := "state: \"todo\"\nstage: \"depends\""
		if i == 0 {
			stagedTask(t, s, ids[i], "state: \"doing\"\nstage: \"active\"\ndoing_since: \"2026-11-17T08:00\"")
			continue
		}
		stagedTask(t, s, ids[i], front, ids[i-1])
	}
	if _, e := s.Check(ids[0], 1); e != nil {
		t.Fatal(e)
	}
	r, e := s.Done(ids[0])
	if e != nil {
		t.Fatal(e)
	}
	// Only the direct dependent moves: the others still depend on an
	// open task.
	if tk, _ := load(t, s, ids[1]); node(tk) != "todo/ready" {
		t.Fatalf("direct dependent: %s (%v)", node(tk), r.Warnings)
	}
	if !strings.Contains(strings.Join(r.Warnings, "\n"), "stage of "+ids[1]+" ready") {
		t.Fatal(r.Warnings)
	}
	for _, id := range ids[2:] {
		if tk, _ := load(t, s, id); node(tk) != "todo/depends" {
			t.Fatalf("%s: %s", id, node(tk))
		}
	}
	// Walking the chain: each done frees the next, 9 more links.
	for i := 1; i < len(ids)-1; i++ {
		if _, e = s.TakeAs(ids[i], "x", "human"); e != nil {
			t.Fatal(i, e)
		}
		if _, e = s.Check(ids[i], 1); e != nil {
			t.Fatal(e)
		}
		if _, e = s.Done(ids[i]); e != nil {
			t.Fatal(i, e)
		}
		if tk, _ := load(t, s, ids[i+1]); node(tk) != "todo/ready" {
			t.Fatalf("link %d: %s", i+1, node(tk))
		}
	}

	// A cycle: x <-> y. Dropping x settles y once and stops.
	x := stagedTask(t, s, "xxxxxx", "state: \"todo\"\nstage: \"depends\"", "yyyyyy")
	y := stagedTask(t, s, "yyyyyy", "state: \"todo\"\nstage: \"depends\"", "xxxxxx")
	if _, e = s.Drop(x, "cycle test"); e != nil {
		t.Fatal(e)
	}
	if tk, _ := load(t, s, y); node(tk) != "todo/ready" {
		t.Fatalf("cycle: y %s", node(tk))
	}
	if tk, _ := load(t, s, x); node(tk) != "dropped" {
		t.Fatalf("cycle: x %s", node(tk))
	}
}

// A write on a staged file keeps its stage right; an unstaged file (before
// hug upgrade) gets no stage.
func TestEveryWriteSettlesStagedFiles(t *testing.T) {
	fixedClock(t)
	s := &Store{Root: t.TempDir()}
	a := stagedTask(t, s, "aaaaaa", "state: \"todo\"\nstage: \"ready\"")
	b := stagedTask(t, s, "bbbbbb", "state: \"todo\"\nstage: \"ready\"")
	if _, e := s.AddDependencies(a, []string{b}); e != nil {
		t.Fatal(e)
	}
	if tk, _ := load(t, s, a); node(tk) != "todo/depends" {
		t.Fatalf("dep added: %s", node(tk))
	}
	// A file from before hug upgrade: a note leaves it unstaged; its first
	// move gives it its stage and the node's fields (crab 11:10 (a): only
	// status, stage and node fields are new).
	plain := newTest(t, s, "plain")
	if _, e := s.Note(plain.ID, "n"); e != nil {
		t.Fatal(e)
	}
	if tk, _ := load(t, s, plain.ID); Staged(tk) {
		t.Fatalf("a note staged the file: %s %v", node(tk), tk.Fields)
	}
	if _, e := s.TakeAs(plain.ID, "x", "human"); e != nil {
		t.Fatal(e)
	}
	tk, raw := load(t, s, plain.ID)
	if node(tk) != "doing/active" || tk.Fields["doing_since"] == "" || !strings.Contains(raw, "todo -> doing/active (take)") {
		t.Fatalf("first move: %s %v\n%s", node(tk), tk.Fields, raw)
	}
	if d := Parse(tk.Path, []byte(raw)); len(d.Findings) != 0 {
		t.Fatal(d.Findings)
	}
}
