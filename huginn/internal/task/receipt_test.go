package task

import (
	"strings"
	"testing"
)

// gmzd87 check 2 (task half): a request's receipt goes pending (not sent,
// then delivered or not delivered with the reason), acknowledged only by
// Acknowledge (the coordinator's hook), in progress at the acknowledger's
// first write after it, answered at resolve. Saved is not received: no
// other write, by anyone, acknowledges it.
func TestRequestReceipt(t *testing.T) {
	s, id := answerStore(t)
	if _, e := s.Request(id, "make it per board", "jhf"); e != nil {
		t.Fatal(e)
	}
	state := func() Receipt {
		t.Helper()
		tk, _ := readTask(t, s, id)
		return tk.Requests[0].Receipt
	}
	if r := state(); r.State != ReceiptPending || r.Label() != "pending, not sent" {
		t.Fatalf("fresh request: %+v %q", r, r.Label())
	}
	item := Item{Kind: "request", N: 1}
	if _, e := s.Delivered(id, item, "crab", "", "no live jcode session named crab"); e != nil {
		t.Fatal(e)
	}
	if r := state(); r.State != ReceiptPending || r.Label() != "pending, not delivered: no live jcode session named crab" {
		t.Fatalf("not delivered: %q", r.Label())
	}
	if _, e := s.Delivered(id, item, "crab", "session_crab_1_ab", ""); e != nil {
		t.Fatal(e)
	}
	if r := state(); r.State != ReceiptPending || r.Label() != "pending, delivered to crab" || r.Session != "session_crab_1_ab" {
		t.Fatalf("delivered: %+v %q", r, r.Label())
	}
	// Writes by the coordinator and others before the ack: still pending.
	s.Actor, s.Action = "crab", "hug task note "+id+` "x"`
	if _, e := s.Note(id, "looking at the board"); e != nil {
		t.Fatal(e)
	}
	if r := state(); r.State != ReceiptPending {
		t.Fatalf("a write before the ack moved the receipt: %+v", r)
	}
	s.Actor, s.Action = "", ""
	if _, e := s.Acknowledge(id, item, "crab"); e != nil {
		t.Fatal(e)
	}
	if r := state(); r.State != ReceiptAcknowledged || r.Label() != "acknowledged by crab" {
		t.Fatalf("acknowledged: %q", r.Label())
	}
	// A second ack changes nothing.
	if res, e := s.Acknowledge(id, item, "crab"); e != nil || res.Changed {
		t.Fatalf("second ack: %v %v", res.Changed, e)
	}
	// Someone else writing is not the coordinator acting.
	s.Actor, s.Action = "wyvern", "hug task note "+id+` "y"`
	s.Note(id, "y")
	if r := state(); r.State != ReceiptAcknowledged {
		t.Fatalf("another agent's write: %+v", r)
	}
	s.Actor, s.Action = "crab", "hug task set "+id+` --goal "per board"`
	s.Note(id, "z")
	if r := state(); r.State != ReceiptInProgress || r.Label() != "in progress by crab" || r.Progress != "hug task set "+id+` --goal "per board"` {
		t.Fatalf("in progress: %+v", r)
	}
	if _, e := s.Resolve(id, 1, "goal per board", "crab"); e != nil {
		t.Fatal(e)
	}
	tk, raw := readTask(t, s, id)
	if r := tk.Requests[0].Receipt; r.State != ReceiptAnswered || r.Label() != "answered" {
		t.Fatalf("answered: %+v", r)
	}
	for _, want := range []string{
		"- 2026-11-17T13:30 request 1 not delivered to crab: no live jcode session named crab\n",
		"- 2026-11-17T13:30 request 1 delivered to crab (session_crab_1_ab)\n",
		"- 2026-11-17T13:30 request 1 acknowledged by crab\n",
		"- 2026-11-17T13:30 request 1 in progress by crab: hug task set " + id + ` --goal "per board"` + "\n",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("History lacks %q:\n%s", want, raw)
		}
	}
	if strings.Count(raw, "in progress by") != 1 {
		t.Fatalf("in progress written more than once:\n%s", raw)
	}
}

// gmzd87 (Jørgen 11:59Z): an answer goes to the coordinator the same way;
// it is "answer N", the Nth answer block in History, and is answered when
// its asker acts on it (41hhgn check 11).
func TestAnswerReceipt(t *testing.T) {
	s, id := answerStore(t)
	if _, e := s.Design(id, "type stays?", "nautilus", "jhf"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Answer(id, "drop it", "jhf"); e != nil {
		t.Fatal(e)
	}
	tk, _ := readTask(t, s, id)
	item, ok := NewestItem(tk, "answer")
	if !ok || item.Name() != "answer 1" || len(tk.Answers) != 0 {
		t.Fatalf("newest answer: %+v %v answers %+v", item, ok, tk.Answers)
	}
	s.Delivered(id, item, "crab", "session_crab_1_ab", "")
	s.Acknowledge(id, item, "crab")
	tk, _ = readTask(t, s, id)
	if len(tk.Answers) != 1 || tk.Answers[0].Receipt.State != ReceiptAcknowledged {
		t.Fatalf("answers: %+v", tk.Answers)
	}
	// The asker acts on it: answered.
	s.Actor, s.Action = "nautilus", "hug task note "+id+` "dropped"`
	s.Note(id, "dropped")
	tk, _ = readTask(t, s, id)
	if tk.Answers[0].Receipt.State != ReceiptAnswered {
		t.Fatalf("after the asker acted: %+v", tk.Answers[0].Receipt)
	}
}
