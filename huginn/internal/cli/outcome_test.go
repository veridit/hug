package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// 41hhgn check 11 (harness test, the daemon faked at its socket): after a
// person answers, the asker's next write on the task is the answer's
// outcome, and hug hands it to the coordinator's live session itself, as it
// does the answer (gmzd87): one line naming the task, who acted on whose
// answer and the write, with History "outcome 1 delivered to crab (SID)";
// crab's hook acknowledges it from the transcript; a second write by the
// asker sends nothing. Jørgen 00:26: he answered from the TUI and expected
// the outcome back through crab; nothing came.
func TestOutcomeReachesCoordinator(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "which rows?", "--as", "nautilus")
	execute(t, a, "answer", id, "the open ones", "--as", "jhf")
	if n := len(jc.messages()); n != 1 {
		t.Fatalf("the answer: daemon got %d messages", n)
	}
	if _, err := Acknowledge(context.Background(), jc.home, crab); err != nil {
		t.Fatal(err)
	}
	a.Env["HUG_USER"] = "nautilus"
	r := execute(t, a, "note", id, "rows now list the open ones first")
	if got := r["outcome_receipt"]; got != "pending, delivered to crab ("+crab+")" {
		t.Fatalf("outcome receipt %v; reply %v", got, r)
	}
	msgs := jc.messages()
	if len(msgs) != 2 {
		t.Fatalf("daemon got %d messages, want the answer and its outcome: %q", len(msgs), msgs)
	}
	sid, msg, _ := strings.Cut(msgs[1], "\x00")
	board := realPath(t, a.Cwd)
	want := "hug: nautilus acted on jhf's answer on " + id + " (Tags), check 2, on board " + filepath.Base(board) + " (" + board + "): hug task note " + id + ` "rows now list the open ones first" (the answer: "the open ones"). hug task show ` + id + " ["
	if sid != crab || !strings.HasPrefix(msg, want) || strings.Contains(strings.TrimSpace(msg), "\n") {
		t.Fatalf("outcome message to %s (want %s), want one line starting\n%s\ngot\n%s", sid, crab, want, msg)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " outcome 1 delivered to crab ("+crab+")\n") {
		t.Fatalf("History lacks the outcome's delivery:\n%s", f)
	}
	acked, err := Acknowledge(context.Background(), jc.home, crab)
	if err != nil || len(acked) != 1 || acked[0] != id+" outcome 1" {
		t.Fatalf("acknowledged %v, err %v", acked, err)
	}
	o, _ := execute(t, a, "show", id)["task"].(Reply)["outcome"].(map[string]any)
	if rc, _ := o["receipt"].(map[string]any); rc["state"] != "acknowledged" || rc["to"] != "crab" {
		t.Fatalf("show --json outcome receipt: %#v", o)
	}
	// Once: the asker's next write is no outcome.
	if r = execute(t, a, "note", id, "and again"); r["outcome_receipt"] != nil {
		t.Fatalf("a second write sent an outcome: %v", r)
	}
	if n := len(jc.messages()); n != 2 {
		t.Fatalf("daemon got %d messages after a second write", n)
	}
}

// The tick that answers a question (qsqhpe) has an outcome too: the
// asker's next write after jhf ticks the asked check goes to crab.
func TestOutcomeOfTickAnswerReachesCoordinator(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "does b hold?", "--as", "nautilus")
	a.Env["HUG_USER"] = "jhf"
	execute(t, a, "check", id, "2")
	a.Env["HUG_USER"] = "nautilus"
	r := execute(t, a, "check", id, "1")
	if got := r["outcome_receipt"]; got != "pending, delivered to crab ("+crab+")" {
		t.Fatalf("outcome receipt %v", got)
	}
	// nautilus's tick of check 1 was its last own check too: the task went
	// to review by crab (5yh7hr), one line, between the answer and the
	// outcome.
	msgs := jc.messages()
	if len(msgs) != 3 {
		t.Fatalf("daemon got %q", msgs)
	}
	if _, msg, _ := strings.Cut(msgs[1], "\x00"); !strings.HasPrefix(msg, "hug: nautilus ticked the last own check of "+id+" (Tags)") {
		t.Fatalf("review message:\n%s", msg)
	}
	if _, msg, _ := strings.Cut(msgs[2], "\x00"); !strings.Contains(msg, "hug: nautilus acted on jhf's answer on "+id+" (Tags), check 2,") || !strings.Contains(msg, ": hug task check "+id+" 1 (the answer: \"answered by ticking check 2\"). hug task show "+id) {
		t.Fatalf("outcome message:\n%s", msg)
	}
}

// Offline (bbxdd7): an outcome written while the coordinator has no live
// session waits for its next session, which receives it once at its start.
// The answer itself was pending too, but its asker has acted, so it is no
// longer owed (bbxdd7's rule); the outcome line carries the answer.
func TestOutcomeReachesCoordinatorsNextSession(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "which rows?", "--as", "nautilus")
	execute(t, a, "answer", id, "the open ones", "--as", "jhf")
	a.Env["HUG_USER"] = "nautilus"
	r := execute(t, a, "note", id, "rows now list the open ones first")
	if got, _ := r["outcome_receipt"].(string); got != "pending, not delivered: no jcode session named crab on this machine" {
		t.Fatalf("offline outcome receipt %q", got)
	}
	next := jc.session(t, "crab", 3000, true)
	none := func(string) string { return "" }
	got, err := DeliverPending(context.Background(), jc.home, next, none)
	if err != nil || strings.Join(got, ",") != id+" outcome 1" {
		t.Fatalf("crab's next session start delivered %v, err %v", got, err)
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("daemon got %q", msgs)
	}
	if _, msg, _ := strings.Cut(msgs[0], "\x00"); !strings.Contains(msg, "hug: nautilus acted on jhf's answer on "+id+" (Tags), check 2,") || !strings.Contains(msg, `(the answer: "the open ones")`) {
		t.Fatalf("outcome message:\n%s", msg)
	}
	if got, err := DeliverPending(context.Background(), jc.home, next, none); err != nil || len(got) != 0 || len(jc.messages()) != 1 {
		t.Fatalf("second start delivered %v, err %v", got, err)
	}
	if f := taskFile(t, a, id); strings.Count(f, " outcome 1 delivered to crab") != 1 {
		t.Fatalf("History:\n%s", f)
	}
}
