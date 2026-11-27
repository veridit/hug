package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// 5yh7hr: review starts by itself. A worker's last own tick moves its task
// to doing/review with review_by the board coordinator, and the
// coordinator's live session gets one line naming the task; History says
// delivered or not delivered, with the reason. Ticks before the last send
// nothing, and a task the coordinator does itself does not enter review.

// reviewBoard is a board with crab as coordinator, jhf as a person, and a
// task wolf is doing with three checks, the last one jhf's.
func reviewBoard(t *testing.T, jc *fakeJcode) (*App, string) {
	t.Helper()
	a := app(t)
	if jc != nil {
		a.JcodeHomes = []string{jc.home}
	} else {
		a.JcodeHomes = []string{t.TempDir()}
	}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	if code, out := runText(a, "people", "add", "jhf", "jorgen@veridit.no"); code != 0 {
		t.Fatalf("people add: %d\n%s", code, out)
	}
	id := execute(t, a, "new", "Review by itself", "--goal", "g", "--purpose", "p", "--principle", "q",
		"--accept", "one holds", "--accept", "two holds", "--accept", "jhf confirms on his board after the release")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "wolf", "--model", "m")
	return a, id
}

// TestLastTickStartsReview: checks 4, 6 and 7. wolf's first tick sends
// nothing and keeps the task active; its second, the last own check (the
// third is jhf's), moves it to doing/review by crab without any other
// command, and crab's live session gets exactly one line naming the task
// and the answer that approves it. The next: is wolf's next task; crab's
// is the review answer.
func TestLastTickStartsReview(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	a, id := reviewBoard(t, jc)

	r := execute(t, a, "check", id, "1", "evidence: test one", "--as", "wolf")
	if n := len(jc.messages()); n != 0 {
		t.Fatalf("the first tick sent %d messages: %q", n, jc.messages())
	}
	if _, ok := r["review_receipt"]; ok {
		t.Fatalf("the first tick has a review receipt: %v", r["review_receipt"])
	}
	if st := r["task"].(Reply)["stage"]; st != "active" {
		t.Fatalf("after the first tick: stage %v, want active", st)
	}
	if strings.Contains(taskFile(t, a, id), "review_by") {
		t.Fatalf("the first tick wrote review_by:\n%s", taskFile(t, a, id))
	}

	r = execute(t, a, "check", id, "2", "evidence: test two", "--as", "wolf")
	tk := r["task"].(Reply)
	if tk["state"] != "doing" || tk["stage"] != "review" {
		t.Fatalf("after the last own tick: %v/%v, want doing/review", tk["state"], tk["stage"])
	}
	if rv, _ := tk["review"].(Reply)["by"].(string); rv != "crab" {
		t.Fatalf("review_by %q, want crab: %v", rv, tk["review"])
	}
	if r["review_receipt"] != "pending, delivered to crab ("+crab+")" {
		t.Fatalf("review receipt %v", r["review_receipt"])
	}
	if r["next"] != "hug task next" {
		t.Fatalf("wolf's next after handing it to review: %v", r["next"])
	}
	if m, _ := r["message"].(string); m != "every own check holds: "+id+" is in review by crab (doing/review); crab answers it" {
		t.Fatalf("message %q", m)
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("daemon got %d messages, want 1: %q", len(msgs), msgs)
	}
	sid, msg, _ := strings.Cut(msgs[0], "\x00")
	board := realPath(t, a.Cwd)
	want := "hug: wolf ticked the last own check of " + id + " (Review by itself) on board " + filepath.Base(board) + " (" + board + "): it is in review by you. Read it (hug task show " + id + "), then answer: hug task answer " + id + ` "looks right" (or what to change) [hug ` + id + " review 1 "
	if sid != crab || !strings.HasPrefix(msg, want) || strings.Contains(msg, "\n") {
		t.Fatalf("message to %s:\n%s\nwant one line starting\n%s", sid, msg, want)
	}
	f := taskFile(t, a, id)
	if !strings.Contains(f, " doing/active -> doing/review (review: wolf ticked the last own check)") {
		t.Fatalf("History lacks the move:\n%s", f)
	}
	if !strings.Contains(f, " review 1 delivered to crab ("+crab+")") {
		t.Fatalf("History lacks the delivery:\n%s", f)
	}
	// crab's step is the review answer.
	a.Env["HUG_USER"] = "crab"
	if n := execute(t, a, "show", id)["next"]; n != "hug task answer "+id+` "looks right"` {
		t.Fatalf("crab's next: %v", n)
	}
	a.Env["HUG_USER"] = "tester"
	// A tick on a task already in review (jhf's check) sends nothing more.
	a.Env["HUG_USER"] = "jhf"
	execute(t, a, "check", id, "3", "evidence: jhf looked")
	a.Env["HUG_USER"] = "tester"
	if n := len(jc.messages()); n != 1 {
		t.Fatalf("a tick in review sent more: %d messages", n)
	}
}

// TestReviewNotDelivered: check 2. With crab's session not live, the task
// still enters review, and History says "review 1 not delivered to crab:"
// with the daemon's reason; the reply's receipt says the same. crab's next
// session gets it at its start, once.
func TestReviewNotDelivered(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	gone := jc.session(t, "crab", 1000, false)
	a, id := reviewBoard(t, jc)
	execute(t, a, "check", id, "1", "evidence: test one", "--as", "wolf")
	r := execute(t, a, "check", id, "2", "evidence: test two", "--as", "wolf")
	reason := "crab's jcode session " + gone + " is not live in a jcode daemon"
	if r["review_receipt"] != "pending, not delivered: "+reason {
		t.Fatalf("review receipt %v", r["review_receipt"])
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " review 1 not delivered to crab: "+reason) {
		t.Fatalf("History lacks the reason:\n%s", f)
	}
	next := jc.session(t, "crab", 3000, true)
	got, err := DeliverPending(context.Background(), jc.home, next, func(string) string { return "" })
	if err != nil || len(got) != 1 || got[0] != id+" review 1" {
		t.Fatalf("crab's next session start: %v %v", got, err)
	}
	if got, _ := DeliverPending(context.Background(), jc.home, next, func(string) string { return "" }); len(got) != 0 {
		t.Fatalf("delivered twice: %v", got)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " review 1 delivered to crab ("+next+")") {
		t.Fatalf("History lacks the later delivery:\n%s", f)
	}
	// No session of any kind on the machine: the reason says so.
	b, bid := reviewBoard(t, nil)
	execute(t, b, "check", bid, "1", "evidence: test one", "--as", "wolf")
	if r := execute(t, b, "check", bid, "2", "evidence: test two", "--as", "wolf"); r["review_receipt"] != "pending, not delivered: no jcode session named crab on this machine" {
		t.Fatalf("no session: %v", r["review_receipt"])
	}
	if f := taskFile(t, b, bid); !strings.Contains(f, " review 1 not delivered to crab: no jcode session named crab on this machine") {
		t.Fatalf("History:\n%s", f)
	}
}

// TestCoordinatorsTaskNoReview: check 5. A task crab does itself does not
// enter review by itself when crab ticks its last check, nor does a tick
// by someone who is not doing it; with no coordinator nothing does.
func TestCoordinatorsTaskNoReview(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	jc.session(t, "crab", 2000, true)
	a, _ := reviewBoard(t, jc)
	id := execute(t, a, "new", "Crab's own", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "one holds")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "crab", "--model", "m")
	r := execute(t, a, "check", id, "1", "evidence: crab's test", "--as", "crab")
	if st := r["task"].(Reply)["stage"]; st != "active" {
		t.Fatalf("the coordinator's own task: stage %v, want active", st)
	}
	if r["next"] != "hug task done "+id {
		t.Fatalf("the coordinator's own task: next %v", r["next"])
	}
	// Someone not doing it ticks the last check: no review.
	other := execute(t, a, "new", "Seal's", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "one holds")["task"].(Reply)["id"].(string)
	execute(t, a, "take", other, "--as", "seal", "--model", "m")
	if st := execute(t, a, "check", other, "1", "evidence: a passer-by", "--as", "otter")["task"].(Reply)["stage"]; st != "active" {
		t.Fatalf("a tick by someone not doing it: stage %v", st)
	}
	if n := len(jc.messages()); n != 0 {
		t.Fatalf("daemon got %d messages: %q", n, jc.messages())
	}
	// No coordinator: no review by itself, and next: says done.
	b := app(t)
	b.JcodeHomes = []string{t.TempDir()}
	execute(t, b, "init")
	bid := execute(t, b, "new", "X", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a holds")["task"].(Reply)["id"].(string)
	execute(t, b, "take", bid, "--as", "wolf", "--model", "m")
	r = execute(t, b, "check", bid, "1", "evidence: test a", "--as", "wolf")
	if st := r["task"].(Reply)["stage"]; st != "active" || r["next"] != "hug task done "+bid {
		t.Fatalf("no coordinator: stage %v next %v", st, r["next"])
	}
}

// TestRequestGoesToDoerFirst: check 8. A request on a task wolf is doing
// goes to wolf's live session, not to crab, and History says delivered to
// wolf; with wolf not live it goes to crab, whose message says wolf did
// not have it live, and History has both lines. An answer goes the same
// way.
func TestRequestGoesToDoerFirst(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	wolf := jc.session(t, "wolf", 2100, true)
	a, id := reviewBoard(t, jc)
	r := execute(t, a, "request", id, "rename the flag", "--as", "jhf")
	if r["receipt"] != "pending, delivered to wolf ("+wolf+")" {
		t.Fatalf("receipt %v", r["receipt"])
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("daemon got %q", msgs)
	}
	sid, msg, _ := strings.Cut(msgs[0], "\x00")
	if sid != wolf || !strings.Contains(msg, "hug: jhf wrote request 1 on "+id+" (Review by itself) on board ") || !strings.Contains(msg, ", to you as the agent doing it:\n  rename the flag\n") {
		t.Fatalf("message to %s:\n%s", sid, msg)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " request 1 delivered to wolf ("+wolf+")") || strings.Contains(f, "request 1 delivered to crab") {
		t.Fatalf("History:\n%s", f)
	}
	// wolf not live: crab gets it, told wolf did not.
	jc.mu.Lock()
	jc.live[wolf] = false
	jc.mu.Unlock()
	r = execute(t, a, "request", id, "and the help text", "--as", "jhf")
	reason := "wolf's jcode session " + wolf + " is not live in a jcode daemon"
	if r["receipt"] != "pending, delivered to crab ("+crab+"); not delivered live to wolf, who is doing it: "+reason {
		t.Fatalf("receipt %v", r["receipt"])
	}
	msgs = jc.messages()
	sid, msg, _ = strings.Cut(msgs[len(msgs)-1], "\x00")
	if sid != crab || !strings.Contains(msg, ", to you as the board coordinator:\n  and the help text\n") || !strings.Contains(msg, "\nNot delivered live to wolf, who is doing it ("+reason+"): hug hands it to wolf's next session when it starts.\n") {
		t.Fatalf("message to %s:\n%s", sid, msg)
	}
	f := taskFile(t, a, id)
	if !strings.Contains(f, " request 2 not delivered to wolf: "+reason) || !strings.Contains(f, " request 2 delivered to crab ("+crab+")") {
		t.Fatalf("History lacks which:\n%s", f)
	}
	// An answer to wolf's question goes to wolf first too.
	jc.mu.Lock()
	jc.live[wolf] = true
	jc.mu.Unlock()
	execute(t, a, "ask", id, "--to", "jhf", "keep the old name as an alias?", "--as", "wolf")
	if r = execute(t, a, "answer", id, "no", "--as", "jhf"); r["receipt"] != "pending, delivered to wolf ("+wolf+")" {
		t.Fatalf("answer receipt %v", r["receipt"])
	}
	// A request on a task nobody holds still goes to the coordinator.
	free := execute(t, a, "new", "Nobody's", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a holds")["task"].(Reply)["id"].(string)
	if r = execute(t, a, "request", free, "x", "--as", "jhf"); r["receipt"] != "pending, delivered to crab ("+crab+")" {
		t.Fatalf("todo task: receipt %v", r["receipt"])
	}
}

// TestDoerGetsItAtNextSession: check 9. wolf has no live session: the
// request waits for wolf's next session, and crab (live) gets it now with
// the line saying it was not delivered live to wolf. wolf's next session
// gets it at its start, once, even though crab already acknowledged it;
// a request resolved before wolf's session starts is not handed over.
func TestDoerGetsItAtNextSession(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	a, id := reviewBoard(t, jc)
	r := execute(t, a, "request", id, "split the TUI part", "--as", "jhf")
	if r["receipt"] != "pending, delivered to crab ("+crab+"); not delivered live to wolf, who is doing it: no jcode session named wolf on this machine" {
		t.Fatalf("receipt %v", r["receipt"])
	}
	msgs := jc.messages()
	if len(msgs) != 1 || !strings.Contains(msgs[0], "Not delivered live to wolf, who is doing it (no jcode session named wolf on this machine)") {
		t.Fatalf("crab's message: %q", msgs)
	}
	if ents, _ := os.ReadDir(filepath.Join(jc.home, "hug-inbox", "pending", "wolf")); len(ents) != 1 {
		t.Fatalf("wolf's pending folder holds %d items, want 1", len(ents))
	}
	// crab acknowledges it: it is still wolf's.
	if got, err := Acknowledge(context.Background(), jc.home, crab); err != nil || len(got) != 1 {
		t.Fatalf("crab's ack: %v %v", got, err)
	}
	none := func(string) string { return "" }
	// crab's next session owes nothing (crab had it).
	if got, _ := DeliverPending(context.Background(), jc.home, jc.session(t, "crab", 2500, true), none); len(got) != 0 {
		t.Fatalf("crab got it again: %v", got)
	}
	next := jc.session(t, "wolf", 3000, true)
	got, err := DeliverPending(context.Background(), jc.home, next, none)
	if err != nil || len(got) != 1 || got[0] != id+" request 1" {
		t.Fatalf("wolf's next session start: %v %v", got, err)
	}
	msgs = jc.messages()
	sid, msg, _ := strings.Cut(msgs[len(msgs)-1], "\x00")
	if sid != next || !strings.Contains(msg, ", to you as the agent doing it:\n  split the TUI part\n") || strings.Contains(msg, "Not delivered live") {
		t.Fatalf("message to %s:\n%s", sid, msg)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " request 1 delivered to wolf ("+next+")") {
		t.Fatalf("History:\n%s", f)
	}
	if got, _ := DeliverPending(context.Background(), jc.home, next, none); len(got) != 0 {
		t.Fatalf("delivered twice: %v", got)
	}
	// Resolved before wolf's session starts: nothing is owed.
	jc.mu.Lock()
	jc.live[next] = false
	jc.mu.Unlock()
	execute(t, a, "request", id, "second", "--as", "jhf")
	if ents, _ := os.ReadDir(filepath.Join(jc.home, "hug-inbox", "pending", "wolf")); len(ents) != 1 {
		t.Fatalf("wolf's pending folder holds %d items, want request 2", len(ents))
	}
	execute(t, a, "resolve", id, "2", "done already", "--as", "crab")
	if got, _ := DeliverPending(context.Background(), jc.home, jc.session(t, "wolf", 4000, true), none); len(got) != 0 {
		t.Fatalf("a resolved request was handed over: %v", got)
	}
	if d, _ := (&task.Store{Root: a.Cwd + "/" + TaskDir}).Get(id); d == nil || len(d.Task.Requests) != 2 {
		t.Fatalf("requests: %+v", d)
	}
}
