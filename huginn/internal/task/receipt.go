package task

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Receipt (gmzd87, Jørgen 11:21Z: delivery by design, not by doctrine). A
// person's request (hbgybn) or answer (41hhgn) goes to the board's
// coordinator (hug.toml [board] coordinator) the moment it is written: hug
// itself hands it to the coordinator's live session. Saved is not
// received, so each item's receipt is kept in History, one line per step,
// each written by the side that saw the step happen:
//
//	- 2026-11-18T12:30 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
//	- 2026-11-18T12:30 request 1 not delivered to crab: no live jcode session named crab
//	- 2026-11-18T12:31 request 1 acknowledged by crab
//	- 2026-11-18T12:33 request 1 in progress by crab: hug task set 6ephqk --goal "..."
//
// delivered / not delivered: the hug that wrote the item, from the jcode
// daemon's reply. Delivered is still pending: the daemon took it, the
// coordinator has not read it. acknowledged: hug's hook inside the
// coordinator's own session, once the message is in that session's
// transcript (its context). in progress: the acknowledger's first write on
// the task after that. answered: a request's resolve (hbgybn); an answer's
// is the asker's first write after it (41hhgn check 11, its read line).
//
// An item is "request N" (N its number on the task) or "answer N" (the
// Nth answer block in History, counted from 1, never renumbered).

// Receipt states, in order.
const (
	ReceiptPending      = "pending"
	ReceiptAcknowledged = "acknowledged"
	ReceiptInProgress   = "in progress"
	ReceiptAnswered     = "answered"
)

// ReceiptStates are the states an item goes through, in order.
var ReceiptStates = []string{ReceiptPending, ReceiptAcknowledged, ReceiptInProgress, ReceiptAnswered}

// Receipt is how far an item has reached the coordinator.
type Receipt struct {
	// State is one of ReceiptStates.
	State string `json:"state"`
	// To is who it was delivered to (or meant for); Session the jcode
	// session that took it.
	To      string `json:"to,omitempty"`
	Session string `json:"session,omitempty"`
	// Delivered is when the daemon took it; NotDelivered why it could not
	// (empty when it was delivered or no delivery was tried).
	Delivered    string `json:"delivered,omitempty"`
	NotDelivered string `json:"not_delivered,omitempty"`
	// By and At: who acknowledged it, when.
	By string `json:"by,omitempty"`
	At string `json:"at,omitempty"`
	// Progress is the acknowledger's first write after the ack.
	Progress string `json:"progress,omitempty"`
}

// Label is the receipt in a few words, as hug task requests and the TUI
// print it: "pending, not delivered: REASON", "pending, delivered to crab",
// "acknowledged by crab", "in progress by crab", "answered".
func (r Receipt) Label() string {
	switch r.State {
	case ReceiptPending:
		switch {
		case r.NotDelivered != "":
			return "pending, not delivered: " + r.NotDelivered
		case r.Delivered != "":
			return "pending, delivered to " + r.To
		}
		return "pending, not sent"
	case ReceiptAcknowledged, ReceiptInProgress:
		return r.State + " by " + r.By
	}
	return r.State
}

var (
	receiptLine   = regexp.MustCompile(`^- (\S+) ((?:request|answer|outcome|review) \d+) (delivered to|not delivered to|not delivered|acknowledged by|in progress by)(?: ([^\s:()]+))?(?: \((\S+)\))?(?:: (.*))?$`)
	answerHeadAny = regexp.MustCompile(`^- \S+ answered \w+`)
)

// receipts reads every item's receipt lines from History.
func receipts(history []string) map[string]*Receipt {
	out := map[string]*Receipt{}
	for _, l := range history {
		m := receiptLine.FindStringSubmatch(strings.TrimRight(l, "\r\n"))
		if m == nil {
			continue
		}
		r := out[m[2]]
		if r == nil {
			r = &Receipt{State: ReceiptPending}
			out[m[2]] = r
		}
		switch m[3] {
		case "delivered to":
			r.To, r.Session, r.Delivered, r.NotDelivered = m[4], m[5], m[1], ""
		case "not delivered to", "not delivered":
			r.To, r.NotDelivered = m[4], m[6]
		case "acknowledged by":
			if r.State == ReceiptPending {
				r.State, r.By, r.At = ReceiptAcknowledged, m[4], m[1]
			}
		case "in progress by":
			if r.State == ReceiptAcknowledged {
				r.State, r.Progress = ReceiptInProgress, m[6]
			}
		}
	}
	return out
}

// answerCount is how many answer blocks History holds: the newest answer
// is "answer <count>".
func answerCount(history []string) int {
	n := 0
	for _, l := range history {
		if answerHeadAny.MatchString(strings.TrimRight(l, "\r\n")) {
			n++
		}
	}
	return n
}

// answerReads says, per answer N, whether its asker has acted on it: a
// read line by the asker after the block (41hhgn check 11).
func answerReads(history []string) map[int]bool {
	out := map[int]bool{}
	n, asker := 0, ""
	for _, l := range history {
		l = strings.TrimRight(l, "\r\n")
		if m := answeredHead.FindStringSubmatch(l); m != nil {
			n, asker = n+1, m[5]
			if asker == "" || strings.EqualFold(asker, m[4]) {
				out[n] = true
			}
			continue
		}
		if m := answeredLine.FindStringSubmatch(l); m != nil {
			n, asker = n+1, m[6]
			if asker == "" || strings.EqualFold(asker, m[5]) {
				out[n] = true
			}
			continue
		}
		if m := readLine.FindStringSubmatch(l); m != nil && n > 0 && strings.EqualFold(m[2], asker) {
			out[n] = true
		}
	}
	return out
}

// Item is one thing a person wrote that the coordinator must receive: a
// request or an answer, with its receipt.
type Item struct {
	// Kind is "request" or "answer"; N its number on the task.
	Kind string `json:"kind"`
	N    int    `json:"n"`
	// Receipt is how far it has reached the coordinator.
	Receipt Receipt `json:"receipt"`
}

// Name is the item as receipt lines and commands name it: "request 1".
func (i Item) Name() string { return i.Kind + " " + strconv.Itoa(i.N) }

// readReceipts fills every request's receipt, and the answers' (Task.Answers).
func readReceipts(t *Task, history []string) {
	rs := receipts(history)
	get := func(name string) Receipt {
		if r := rs[name]; r != nil {
			return *r
		}
		return Receipt{State: ReceiptPending}
	}
	for i := range t.Requests {
		r := get("request " + strconv.Itoa(t.Requests[i].N))
		// A request for several tasks is answered when every target is
		// (ec6z6j): its home's own resolve is one target of it.
		if t.Requests[i].Targets != nil && t.Requests[i].AllResolved || t.Requests[i].Targets == nil && !t.Requests[i].Open() {
			r.State = ReceiptAnswered
		}
		t.Requests[i].Receipt = r
	}
	t.Answers = nil
	reads := answerReads(history)
	for n := 1; n <= answerCount(history); n++ {
		name := "answer " + strconv.Itoa(n)
		r := get(name)
		if reads[n] {
			r.State = ReceiptAnswered
		}
		// Only answers that went (or tried to go) to a coordinator are
		// items; an answer before gmzd87 has no receipt lines.
		if rs[name] == nil {
			continue
		}
		t.Answers = append(t.Answers, Item{Kind: "answer", N: n, Receipt: r})
	}
	// The review hug handed to its reviewer (5yh7hr), while the task is in
	// it: the board cue does not say it again to a session that has it.
	t.Review = nil
	if n := reviewCount(history); n > 0 && ownHolds(*t, "review") {
		t.Review = &Item{Kind: "review", N: n, Receipt: get("review " + strconv.Itoa(n))}
	}
	// An outcome is told, not acted on: acknowledged is as far as it goes.
	if o := t.Outcome; o != nil && o.N > 0 {
		if r := rs["outcome "+strconv.Itoa(o.N)]; r != nil {
			c := *r
			o.Receipt = &c
		}
	}
}

// NewestItem is the item a write just made: the request with the highest
// N for "request", the newest answer block for "answer"; false if none.
func NewestItem(t Task, kind string) (Item, bool) {
	switch kind {
	case "request":
		best := -1
		for i, r := range t.Requests {
			if best < 0 || r.N > t.Requests[best].N {
				best = i
			}
		}
		if best >= 0 {
			return Item{Kind: "request", N: t.Requests[best].N, Receipt: t.Requests[best].Receipt}, true
		}
	case "answer":
		if t.AnswerCount > 0 {
			return Item{Kind: "answer", N: t.AnswerCount}, true
		}
	case "outcome":
		if o := t.Outcome; o != nil && o.N > 0 {
			return Item{Kind: "outcome", N: o.N}, true
		}
	}
	return Item{}, false
}

// Delivered records what the daemon said about item on ref: delivered to
// session, or why not (reason). It is hug's own write: it marks nothing
// read and starts no progress.
func (s *Store) Delivered(ref string, item Item, to, session, reason string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if !hasItem(d, item) {
		return Result{}, usageErr(d.Task.ID, d.Task.ID+" has no "+item.Name(), "hug task requests "+d.Task.ID)
	}
	line := stamp() + " " + item.Name()
	switch {
	case reason != "" && strings.TrimSpace(to) == "":
		line += " not delivered: " + oneLine(reason)
	case reason != "":
		line += " not delivered to " + receiptName(to) + ": " + oneLine(reason)
	default:
		line += " delivered to " + receiptName(to) + " (" + session + ")"
	}
	d.History(line)
	s.quiet = true
	defer func() { s.quiet = false }()
	return s.finish(d, true)
}

// Acknowledge records that by's session received item on ref: hug's hook
// found it in that session's transcript. Once only; a second ack, or one
// on an item already answered, changes nothing.
func (s *Store) Acknowledge(ref string, item Item, by string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if !hasItem(d, item) {
		return Result{}, usageErr(d.Task.ID, d.Task.ID+" has no "+item.Name(), "hug task requests "+d.Task.ID)
	}
	if r := receipts(d.historyLines())[item.Name()]; r != nil && r.State != ReceiptPending {
		return Result{Task: d.Task, Next: hint(d.Task)}, nil
	}
	d.History(stamp() + " " + item.Name() + " acknowledged by " + receiptName(by))
	s.quiet = true
	defer func() { s.quiet = false }()
	return s.finish(d, true)
}

// markProgress writes "ITEM in progress by ACTOR: WRITE" for every item on
// d that actor acknowledged and has not acted on yet, when actor writes on
// the task (the first write after the ack).
func markProgress(d *Document, actor, action string) {
	if actor = strings.TrimSpace(actor); actor == "" {
		return
	}
	rs := receipts(d.historyLines())
	var names []string
	for name, r := range rs {
		if r.State == ReceiptAcknowledged && strings.EqualFold(r.By, receiptName(actor)) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return
	}
	open := map[string]bool{}
	for _, r := range d.Task.Requests {
		open["request "+strconv.Itoa(r.N)] = r.Open()
	}
	reads := answerReads(d.historyLines())
	for n := 1; n <= answerCount(d.historyLines()); n++ {
		open["answer "+strconv.Itoa(n)] = !reads[n]
	}
	sort.Strings(names)
	write := strings.Join(strings.Fields(action), " ")
	for _, name := range names {
		if !open[name] {
			continue
		}
		line := stamp() + " " + name + " in progress by " + receiptName(actor)
		if write != "" {
			line += ": " + write
		}
		d.History(line)
	}
}

// receiptName is a name as one word, the way receipt lines hold it.
func receiptName(n string) string {
	n = strings.Join(strings.Fields(n), "-")
	if n == "" {
		return "nobody"
	}
	return strings.Map(func(r rune) rune {
		if r == ':' || r == '(' || r == ')' {
			return '-'
		}
		return r
	}, n)
}

// oneLine is text on one line, spaces collapsed.
func oneLine(text string) string { return strings.Join(strings.Fields(text), " ") }

// LastAnswer is the newest answer block in d's History, seen or not; nil
// when there is none.
func (d *Document) LastAnswer() *Answer {
	_, _, last := scanAnswers(d.historyLines())
	return last
}

// hasItem says whether d holds item: request N in Requests, answer N among
// its History answers.
func hasItem(d *Document, item Item) bool {
	switch item.Kind {
	case "request":
		for _, r := range d.Task.Requests {
			if r.N == item.N {
				return true
			}
		}
	case "answer":
		return item.N >= 1 && item.N <= d.Task.AnswerCount
	case "outcome":
		// The outcome of answer N: the asker has acted on it (41hhgn
		// check 11). Only the newest answer's outcome is read.
		o := d.Task.Outcome
		return o != nil && o.N == item.N
	case "review":
		// The Nth move into review (5yh7hr).
		return item.N >= 1 && item.N <= reviewCount(d.historyLines())
	}
	return false
}

// ItemReceipt is item's receipt on d (bbxdd7: whether a pending delivery
// is still owed), false when d has no such item.
func (d *Document) ItemReceipt(item Item) (Receipt, bool) {
	if !hasItem(d, item) {
		return Receipt{}, false
	}
	switch item.Kind {
	case "request":
		for _, r := range d.Task.Requests {
			if r.N == item.N {
				return r.Receipt, true
			}
		}
	case "answer":
		for _, a := range d.Task.Answers {
			if a.N == item.N {
				return a.Receipt, true
			}
		}
		return Receipt{State: ReceiptPending}, true
	case "outcome":
		if r := d.Task.Outcome.Receipt; r != nil {
			return *r, true
		}
		return Receipt{State: ReceiptPending}, true
	case "review":
		if r := receipts(d.historyLines())[item.Name()]; r != nil {
			return *r, true
		}
		return Receipt{State: ReceiptPending}, true
	}
	return Receipt{}, false
}

// AnswerN is the Nth answer block in d's History (from 1), nil if none.
func (d *Document) AnswerN(n int) *Answer {
	var out *Answer
	i := 0
	h := d.historyLines()
	for j := range h {
		l := strings.TrimRight(h[j], "\r\n")
		if answerHeadAny.MatchString(l) {
			i++
			if i == n {
				_, _, out = scanAnswers(h[:j+blockEnd(h, j)])
				return out
			}
		}
	}
	return nil
}

// blockEnd is how many lines the answer at h[j] spans: its head and its
// "  " continuation lines.
func blockEnd(h []string, j int) int {
	k := j + 1
	for k < len(h) && strings.HasPrefix(h[k], "  ") {
		k++
	}
	return k - j
}
