package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	waitAck := func() {
		t.Helper()
		select {
		case <-acked:
		case <-time.After(20 * time.Second):
			t.Fatal("no acknowledgement")
		}
	}
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
		{"reach the board coordinator (hug task coordinator NAME) by hug itself", func(t *testing.T) {
			execute(t, a, "coordinator", "crab")
		}},
		{"hug hands it at once to the coordinator's live session", func(t *testing.T) {
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
		{"The coordinator's session acknowledges it on receipt (hug's hook)", func(t *testing.T) {
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
			if r := execute(t, a, "answer", id, "yes", "--as", "jhf"); r["receipt"] != "pending, delivered to crab ("+crab+")" {
				t.Fatalf("answer receipt %v", r["receipt"])
			}
			waitAck()
		}},
		{"(an answer: the asker's next write)", func(t *testing.T) {
			a.Env["HUG_USER"] = "nautilus"
			execute(t, a, "note", id, "acted")
			a.Env["HUG_USER"] = "tester"
			d, _ := (&task.Store{Root: a.Cwd + "/" + TaskDir}).Get(id)
			if len(d.Task.Answers) != 1 || d.Task.Answers[0].Receipt.State != task.ReceiptAnswered {
				t.Fatalf("answers: %+v", d.Task.Answers)
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
		{"No agent polls.", func(t *testing.T) {
			// Delivery is in the write: every message above reached the
			// daemon inside the command that wrote the item (counted
			// there), and nothing of hug's runs between commands: the
			// hook acks inside the coordinator's own turn.
			if n := len(jc.messages()); n != 2 {
				t.Fatalf("daemon got %d messages, want 2 (request 2, answer 1)", n)
			}
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
		case "Requests", "and", "answers", "", "it", "the", "a", "is":
			continue
		}
		t.Errorf("DeliveryFlow says %q with no check for it here; rest: %q", w, rest)
	}
	os.RemoveAll(jc.home)
}
