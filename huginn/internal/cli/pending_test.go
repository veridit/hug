package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bbxdd7 check 1 (gmzd87 open questions 2 and 3): a request written while
// the coordinator has no live jcode session says "pending, not delivered";
// the coordinator's next session, at its start (hug's turn_start hook),
// receives it through the daemon, once: History says delivered to that
// session, its hook acknowledges it from the transcript, and neither a
// second start nor a second jcode home holding the same pending item
// delivers it again. No poller: nothing runs until a session starts.
func TestPendingReachesCoordinatorsNextSession(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	jc2 := newFakeJcode(t)
	gone := jc.session(t, "crab", 1000, false)
	a := app(t)
	a.JcodeHomes = []string{jc.home, jc2.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	r := execute(t, a, "request", id, "split the TUI part", "--as", "jhf")
	if got := r["receipt"].(string); got != "pending, not delivered: crab's jcode session "+gone+" is not live in a jcode daemon" {
		t.Fatalf("coordinator offline: receipt %q", got)
	}
	if n := len(jc.messages()); n != 0 {
		t.Fatalf("daemon got %d messages while crab was offline", n)
	}
	none := func(string) string { return "" }

	// crab's next session starts: its hook hands the request over.
	next := jc.session(t, "crab", 3000, true)
	got, err := DeliverPending(context.Background(), jc.home, next, none)
	if err != nil || len(got) != 1 || got[0] != id+" request 1" {
		t.Fatalf("crab's next session start delivered %v, err %v; want [%s request 1]", got, err, id)
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("daemon got %d messages, want 1: %q", len(msgs), msgs)
	}
	sid, msg, _ := strings.Cut(msgs[0], "\x00")
	if sid != next || !strings.Contains(msg, "hug: jhf wrote request 1 on "+id+" (Tags)") || !strings.Contains(msg, "  split the TUI part\n") || !strings.Contains(msg, "hug task resolve "+id+" 1") {
		t.Fatalf("message to %s (want %s):\n%s", sid, next, msg)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " request 1 delivered to crab ("+next+")") {
		t.Fatalf("History lacks the delivery to %s:\n%s", next, f)
	}
	// The session's hook acknowledges it from its transcript, as gmzd87's.
	acked, err := Acknowledge(context.Background(), jc.home, next)
	if err != nil || len(acked) != 1 || acked[0] != id+" request 1" {
		t.Fatalf("acknowledged %v, err %v", acked, err)
	}
	if _, out := runText(a, "requests", id); !strings.Contains(out, "  receipt: acknowledged by crab\n") {
		t.Fatalf("requests after the ack:\n%s", out)
	}

	// Once: the same session's next turn, a later session, and a session
	// in the other jcode home that kept a copy deliver nothing.
	later := jc.session(t, "crab", 4000, true)
	other := jc2.session(t, "crab", 5000, true)
	for _, s := range []struct{ home, sid string }{{jc.home, next}, {jc.home, later}, {jc2.home, other}} {
		if got, err := DeliverPending(context.Background(), s.home, s.sid, none); err != nil || len(got) != 0 {
			t.Fatalf("second delivery in %s: %v, err %v", s.sid, got, err)
		}
	}
	if n := len(jc.messages()) + len(jc2.messages()); n != 1 {
		t.Fatalf("daemons got %d messages in all, want 1", n)
	}
	if f := taskFile(t, a, id); strings.Count(f, " request 1 delivered to crab") != 1 {
		t.Fatalf("History has more than one delivery:\n%s", f)
	}
	for _, h := range []string{jc.home, jc2.home} {
		if ents, _ := os.ReadDir(filepath.Join(h, "hug-inbox", "pending", "crab")); len(ents) != 0 {
			t.Fatalf("%s keeps %d pending items after the delivery", h, len(ents))
		}
	}
}

// bbxdd7: a session of another name finds nothing for it, and a request
// whose delivery worked leaves nothing pending.
func TestPendingOnlyForItsCoordinator(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "request", id, "first", "--as", "jhf")
	none := func(string) string { return "" }
	duck := jc.session(t, "duckling", 2000, true)
	if got, err := DeliverPending(context.Background(), jc.home, duck, none); err != nil || len(got) != 0 {
		t.Fatalf("duckling's session took crab's pending item: %v %v", got, err)
	}
	crab := jc.session(t, "crab", 3000, true)
	execute(t, a, "request", id, "second", "--as", "jhf")
	if got, err := DeliverPending(context.Background(), jc.home, crab, none); err != nil || len(got) != 1 || got[0] != id+" request 1" {
		t.Fatalf("crab's start: %v %v; want only the undelivered request 1", got, err)
	}
	if n := len(jc.messages()); n != 2 {
		t.Fatalf("daemon got %d messages, want 2 (request 2 live, request 1 at start)", n)
	}
}

// bbxdd7: an answer written while the coordinator was offline reaches its
// next session the same way, with the question and the answer verbatim.
func TestPendingAnswerReachesCoordinatorsNextSession(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "type stays, or drop?", "--as", "nautilus")
	r := execute(t, a, "answer", id, "drop it\nthe column says it", "--as", "jhf")
	// Neither nautilus, doing it, nor crab has a session (5yh7hr): both
	// are owed it at their next session; this test follows crab's.
	if got := r["receipt"].(string); got != "pending, not delivered: no jcode session named crab on this machine; not delivered live to nautilus, who is doing it: no jcode session named nautilus on this machine" {
		t.Fatalf("offline answer: receipt %q", got)
	}
	crab := jc.session(t, "crab", 3000, true)
	got, err := DeliverPending(context.Background(), jc.home, crab, func(string) string { return "" })
	if err != nil || len(got) != 1 || got[0] != id+" answer 1" {
		t.Fatalf("delivered %v, err %v", got, err)
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("daemon got %q", msgs)
	}
	_, msg, _ := strings.Cut(msgs[0], "\x00")
	for _, want := range []string{"hug: jhf answered on " + id + " (Tags), check 2, the question nautilus asked", "> type stays, or drop?\n", "  drop it\n  the column says it\n"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message lacks %q:\n%s", want, msg)
		}
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " answer 1 delivered to crab ("+crab+")") {
		t.Fatalf("History:\n%s", f)
	}
}
