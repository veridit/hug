package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/harness"
	"github.com/veridit/hug/huginn/internal/task"
)

// gmzd87 check 3: the instructions hug installs say how requests and
// answers reach the coordinator in DeliveryFlow's words, and each claim in
// those words is a behaviour this test drives. A sentence added to
// DeliveryFlow without a claim here fails ("no check for"); a claim whose
// behaviour changed fails at its check. The installed texts are the
// AGENTS.md section hug setup writes and the MCP descriptions
// (huginn/internal/mcp TestDeliveryFlowInMCP).
func TestDeliveryInstructionsMatchCode(t *testing.T) {
	t.Parallel()
	if !strings.Contains(agentSection, "\n"+DeliveryFlow+"\n") {
		t.Fatalf("the AGENTS.md section hug setup writes does not carry DeliveryFlow verbatim:\n%s", agentSection)
	}
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	acked := make(chan []string, 4)
	jc.hook = func(sid string) {
		got, _ := Acknowledge(context.Background(), jc.home, sid)
		acked <- got
	}
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	// Each wait follows a receipt that says delivered, and the hook answers
	// on every path: wait for it, not for a clock (awht8w, as in
	// TestRequestReachesCoordinator).
	waitAck := func() { <-acked }
	// reviewMsgs: what the tick that started a review sent (5yh7hr).
	var reviewMsgs int
	naut7000 := "session_nautilus_7000_00ab"
	// Each claim: a phrase DeliveryFlow must contain, and the check that
	// the code does it. They run in order on one board.
	claims := []struct {
		phrase string
		check  func(t *testing.T)
	}{
		{"Not delivered (no coordinator set", func(t *testing.T) {
			r := execute(t, a, "request", id, "before any coordinator", "--as", "jhf")
			if !strings.HasPrefix(r["receipt"].(string), "pending, not delivered: no coordinator set") {
				t.Fatalf("receipt %v", r["receipt"])
			}
		}},
		{`stays "pending, not delivered: REASON"`, func(t *testing.T) {
			if _, out := runText(a, "requests", id); !strings.Contains(out, "receipt: pending, not delivered: no coordinator set") {
				t.Fatalf("requests:\n%s", out)
			}
		}},
		{"the request still comes first in hug task next", func(t *testing.T) {
			if got := execute(t, a, "next", "--as", "wyvern")["task"].(Reply)["id"]; got != id {
				t.Fatalf("next gave %v", got)
			}
		}},
		{"reach the agent doing the task, else the board coordinator (hug task coordinator NAME), by hug itself", func(t *testing.T) {
			execute(t, a, "coordinator", "crab")
		}},
		{"(or nobody does the task) to the coordinator's", func(t *testing.T) {
			// Nobody does it yet (todo): the coordinator's session.
			n := len(jc.messages())
			r := execute(t, a, "request", id, "make it per board", "--as", "jhf")
			if r["receipt"] != "pending, delivered to crab ("+crab+")" || len(jc.messages()) != n+1 {
				t.Fatalf("receipt %v, daemon messages %d", r["receipt"], len(jc.messages()))
			}
			waitAck()
		}},
		{"History says delivered or not delivered with the reason", func(t *testing.T) {
			f := taskFile(t, a, id)
			if !strings.Contains(f, " request 1 not delivered: no coordinator set") || !strings.Contains(f, " request 2 delivered to crab ("+crab+")") {
				t.Fatalf("History:\n%s", f)
			}
		}},
		{"The receiving session acknowledges it on receipt (hug's hook)", func(t *testing.T) {
			if d, _ := (&task.Store{Root: a.Cwd + "/" + TaskDir}).Get(id); d == nil || d.Task.Requests[1].Receipt.State != task.ReceiptAcknowledged {
				t.Fatalf("request 2 not acknowledged: %+v", d.Task.Requests[1].Receipt)
			}
		}},
		{"its first write on the task marks it in progress", func(t *testing.T) {
			a.Env["HUG_USER"] = "crab"
			execute(t, a, "note", id, "on it")
			a.Env["HUG_USER"] = "tester"
			if _, out := runText(a, "requests", id); !strings.Contains(out, "receipt: in progress by crab") {
				t.Fatalf("requests:\n%s", out)
			}
		}},
		{"hug task resolve ID N answers a request", func(t *testing.T) {
			execute(t, a, "resolve", id, "2", "per board now", "--as", "crab")
			if _, out := runText(a, "requests", id, "--all"); !strings.Contains(out, "receipt: answered") {
				t.Fatalf("requests:\n%s", out)
			}
		}},
		{"hug task requests shows each state: pending, acknowledged, in progress, answered.", func(t *testing.T) {
			if strings.Join(task.ReceiptStates, ", ") != "pending, acknowledged, in progress, answered" {
				t.Fatalf("states: %v", task.ReceiptStates)
			}
		}},
		{"when a person writes hug task request or hug task answer (or r and a in hug tui)", func(t *testing.T) {
			// The CLI's request: above. The answer, and the TUI's r and a:
			// TestAnswerReachesCoordinator and tui TestDeliverFromBoxes.
			execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
			execute(t, a, "ask", id, "--to", "jhf", "ok?", "--as", "nautilus")
			if r := execute(t, a, "answer", id, "yes", "--as", "jhf"); r["receipt"] != "pending, delivered to crab ("+crab+"); not delivered live to nautilus, who is doing it: no jcode session named nautilus on this machine" {
				t.Fatalf("answer receipt %v", r["receipt"])
			}
			waitAck()
		}},
		{"; when that agent has none", func(t *testing.T) {
			// The answer above: nautilus, doing it, has no session, so crab
			// has it.
			if f := taskFile(t, a, id); !strings.Contains(f, " answer 1 not delivered to nautilus: no jcode session named nautilus on this machine") || !strings.Contains(f, " answer 1 delivered to crab ("+crab+")") {
				t.Fatalf("History:\n%s", f)
			}
		}},
		{"saying it was not delivered live to that agent", func(t *testing.T) {
			msgs := jc.messages()
			_, msg, _ := strings.Cut(msgs[len(msgs)-1], "\x00")
			if !strings.Contains(msg, "Not delivered live to nautilus, who is doing it (no jcode session named nautilus on this machine)") {
				t.Fatalf("crab's message:\n%s", msg)
			}
		}},
		{"(an answer: the asker's next write", func(t *testing.T) {
			a.Env["HUG_USER"] = "nautilus"
			r := execute(t, a, "note", id, "acted")
			a.Env["HUG_USER"] = "tester"
			d, _ := (&task.Store{Root: a.Cwd + "/" + TaskDir}).Get(id)
			if len(d.Task.Answers) != 1 || d.Task.Answers[0].Receipt.State != task.ReceiptAnswered {
				t.Fatalf("answers: %+v", d.Task.Answers)
			}
			if r["outcome_receipt"] != "pending, delivered to crab ("+crab+")" {
				t.Fatalf("outcome receipt %v", r["outcome_receipt"])
			}
			waitAck()
		}},
		{"its outcome, which hug hands to the coordinator the same way in one line: \"NAME acted on PERSON's answer on ID: WRITE\")", func(t *testing.T) {
			// The delivery above; its words: TestOutcomeReachesCoordinator.
			msgs := jc.messages()
			_, msg, _ := strings.Cut(msgs[len(msgs)-1], "\x00")
			if !strings.HasPrefix(msg, "hug: nautilus acted on jhf's answer on "+id+" ") || !strings.Contains(msg, `: hug task note `+id+` acted`) || strings.Contains(msg, "\n") {
				t.Fatalf("outcome message: %q", msg)
			}
		}},
		{"(an idle session starts a turn, a busy one gets it at its next step)", func(t *testing.T) {
			// Both are the daemon's (jcode client_actions.rs
			// handle_notify_session: idle -> run_live_system_turn_if_idle,
			// busy -> queue_soft_interrupt_for_session). hug's side: the ack
			// waits until the message is in the transcript, whichever way
			// it came in. Busy: the inbox holds the item, the transcript
			// does not yet: no ack; once it lands: ack.
			busy := jc.session(t, "busybee", 3000, true)
			it := harness.InboxItem{Board: a.Cwd, Task: id, Kind: "request", N: 2, To: "busybee", Token: "hug " + id + " request 2 deadbeef"}
			harness.AddInbox(jc.home, busy, it)
			if got, err := Acknowledge(context.Background(), jc.home, busy); len(got) != 0 || err != nil {
				t.Fatalf("acknowledged before the message reached the transcript: %v %v", got, err)
			}
			jf, _ := os.OpenFile(filepath.Join(jc.home, "sessions", busy+".journal.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			// The session quoting the token in a tool call, or a tool's
			// result holding it, is not receipt.
			jf.WriteString(`{"append_messages":[{"role":"assistant","content":[{"type":"tool_use","input":{"command":"grep '` + it.Token + `'"}}]}]}` + "\n")
			jf.WriteString(`{"append_messages":[{"role":"user","content":[{"type":"tool_result","content":"` + it.Token + `"}]}]}` + "\n")
			if got, err := Acknowledge(context.Background(), jc.home, busy); len(got) != 0 || err != nil {
				t.Fatalf("a tool call or result quoting the token counted as receipt: %v %v", got, err)
			}
			jf.WriteString(`{"append_messages":[{"role":"user","content":[{"type":"text","text":"[` + it.Token + `]"}]}]}` + "\n")
			jf.Close()
			if got, _ := Acknowledge(context.Background(), jc.home, busy); len(got) != 1 {
				t.Fatalf("not acknowledged once in the transcript: %v", got)
			}
		}},
		{"no live session", func(t *testing.T) {
			jc.mu.Lock()
			jc.live[crab] = false
			jc.mu.Unlock()
			r := execute(t, a, "request", id, "while away", "--as", "jhf")
			if !strings.Contains(r["receipt"].(string), "is not live in a jcode daemon") {
				t.Fatalf("receipt %v", r["receipt"])
			}
		}},
		{"a harness hug cannot reach yet such as Claude Code or Codex)", func(t *testing.T) {
			// No jcode home, no session: not delivered, with the reason.
			b := app(t)
			b.JcodeHomes = []string{t.TempDir()}
			execute(t, b, "init")
			execute(t, b, "coordinator", "crab")
			bid := execute(t, b, "new", "X", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
			if r := execute(t, b, "request", bid, "x", "--as", "jhf"); r["receipt"] != "pending, not delivered: no jcode session named crab on this machine" {
				t.Fatalf("receipt %v", r["receipt"])
			}
		}},
		{"What no live session could take goes to that name's next session when it starts, once (hug's hook)", func(t *testing.T) {
			// "while away" (above) found crab not live. crab's next
			// session starts: its hook hands it over, once (bbxdd7).
			n := len(jc.messages())
			next := jc.session(t, "crab", 4000, true)
			none := func(string) string { return "" }
			if got, err := DeliverPending(context.Background(), jc.home, next, none); err != nil || len(got) != 1 || !strings.HasSuffix(got[0], " request 3") {
				t.Fatalf("next session start: %v %v", got, err)
			}
			// The daemon starts the turn; its hook acknowledges and writes
			// the board. Wait for it, or it races the test's cleanup.
			waitAck()
			if got, _ := DeliverPending(context.Background(), jc.home, next, none); len(got) != 0 || len(jc.messages()) != n+1 {
				t.Fatalf("delivered twice: %v, daemon messages %d", got, len(jc.messages()))
			}
		}},
		{"; a coordinator renamed before then gets it under the old name.", func(t *testing.T) {
			// The item waits for the name it was written to: a session of
			// the new coordinator's name gets nothing (accepted, crab 15:49).
			jc.mu.Lock()
			for s := range jc.live {
				jc.live[s] = false
			}
			jc.mu.Unlock()
			execute(t, a, "request", id, "while renamed", "--as", "jhf")
			execute(t, a, "coordinator", "lobster")
			n := len(jc.messages())
			lobster := jc.session(t, "lobster", 5000, true)
			none := func(string) string { return "" }
			if got, _ := DeliverPending(context.Background(), jc.home, lobster, none); len(got) != 0 || len(jc.messages()) != n {
				t.Fatalf("the renamed coordinator got %v", got)
			}
			crab := jc.session(t, "crab", 6000, true)
			if got, _ := DeliverPending(context.Background(), jc.home, crab, none); len(got) != 1 {
				t.Fatalf("the old name's session got %v", got)
			}
			waitAck()
			// Leave the board as the claims after this one expect it:
			// crab coordinates, and no session this claim made is live
			// (the newest live crab session would take what is next).
			execute(t, a, "coordinator", "crab")
			jc.mu.Lock()
			jc.live[lobster], jc.live[crab] = false, false
			jc.mu.Unlock()
		}},
		{", the agent's doing the task too", func(t *testing.T) {
			// nautilus's answer (above) waits for nautilus's next session:
			// TestDoerGetsItAtNextSession follows it there.
			if ents, _ := os.ReadDir(filepath.Join(jc.home, "hug-inbox", "pending", "nautilus")); len(ents) == 0 {
				t.Fatalf("nothing waits for nautilus's next session")
			}
		}},
		{"No agent polls.", func(t *testing.T) {
			// Delivery is in the write: every message above reached the
			// daemon inside the command that wrote the item (counted
			// there), and nothing of hug's runs between commands: the
			// hook acks inside the coordinator's own turn.
			if n := len(jc.messages()); n != 5 {
				t.Fatalf("daemon got %d messages, want 5 (request 2, answer 1, outcome 1, requests 3 and 4 at crab's next starts)", n)
			}
		}},
		{"hug hands it at once to the live session of the agent doing it", func(t *testing.T) {
			// nautilus, doing it, is live now: the request is its, crab
			// hears nothing.
			naut := jc.session(t, "nautilus", 7000, true)
			n := len(jc.messages())
			r := execute(t, a, "request", id, "for the doer", "--as", "jhf")
			if r["receipt"] != "pending, delivered to nautilus ("+naut+")" || len(jc.messages()) != n+1 {
				t.Fatalf("receipt %v, daemon messages %d", r["receipt"], len(jc.messages()))
			}
			waitAck()
			jc.mu.Lock()
			jc.live[naut] = false
			jc.mu.Unlock()
		}},
		{"A worker's last own tick (every open check left is a person's, or none) puts its task in review by the coordinator (doing/review)", func(t *testing.T) {
			// TestLastTickStartsReview; here: otter's last tick on the task
			// it does puts it in review by crab.
			jc.mu.Lock()
			jc.live[crab] = true
			jc.mu.Unlock()
			rid := execute(t, a, "new", "R", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "one", "--accept", "two")["task"].(Reply)["id"].(string)
			execute(t, a, "take", rid, "--as", "otter", "--model", "m")
			n := len(jc.messages())
			if r := execute(t, a, "check", rid, "1", "evidence: test one", "--as", "otter"); r["review_receipt"] != nil || len(jc.messages()) != n {
				t.Fatalf("the first tick told someone: %v", r["review_receipt"])
			}
			r := execute(t, a, "check", rid, "2", "evidence: test two", "--as", "otter")
			if tk := r["task"].(Reply); tk["stage"] != "review" || r["review_receipt"] != "pending, delivered to crab ("+crab+")" {
				t.Fatalf("stage %v, receipt %v", tk["stage"], r["review_receipt"])
			}
			reviewMsgs = len(jc.messages()) - n
			waitAck()
		}},
		{", whose session hug tells in one line the same way; earlier ticks tell nobody.", func(t *testing.T) {
			if reviewMsgs != 1 {
				t.Fatalf("the review told %d messages, want 1", reviewMsgs)
			}
			msgs := jc.messages()
			if _, msg, _ := strings.Cut(msgs[len(msgs)-1], "\x00"); strings.Contains(msg, "\n") || !strings.Contains(msg, "it is in review by you") {
				t.Fatalf("review message:\n%s", msg)
			}
		}},
		{"or ticks the check a question to them is about (that tick is their answer)", func(t *testing.T) {
			// qsqhpe: the person asked ticks the asked check: an answer,
			// delivered. The full message: TestTickAnswerReachesCoordinator.
			jc.mu.Lock()
			jc.live[crab] = true
			jc.mu.Unlock()
			tid := id // nautilus holds it; its question was answered above
			execute(t, a, "ask", tid, "--to", "jhf", "--check", "1", "holds?", "--as", "nautilus")
			a.Env["HUG_USER"] = "jhf"
			r := execute(t, a, "check", tid, "1")
			a.Env["HUG_USER"] = "tester"
			if r["receipt"] != "pending, delivered to crab ("+crab+"); not delivered live to nautilus, who is doing it: nautilus's jcode session "+naut7000+" is not live in a jcode daemon" {
				t.Fatalf("tick receipt %v", r["receipt"])
			}
			waitAck()
		}},
	}
	rest := DeliveryFlow
	for _, c := range claims {
		if !strings.Contains(DeliveryFlow, c.phrase) {
			t.Fatalf("DeliveryFlow no longer says %q: change the claim with the text", c.phrase)
		}
		rest = strings.Replace(rest, c.phrase, "", 1)
		t.Run(c.phrase, c.check)
	}
	// What is left must be glue, not an unchecked claim.
	for _, w := range strings.FieldsFunc(rest, func(r rune) bool { return strings.ContainsRune(" ,.:;()\"", r) }) {
		switch w {
		case "Requests", "and", "answers", "", "it", "the", "a", "is", "to", "else":
			continue
		}
		t.Errorf("DeliveryFlow says %q with no check for it here; rest: %q", w, rest)
	}
	os.RemoveAll(jc.home)
}
