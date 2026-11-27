package task

import (
	"strings"
)

// Review starts by itself (5yh7hr, Jørgen 2026-11-26: the worker ticks the
// ticket's checks, asks for review, and the coordinator hears that one
// event). The same shape as after-release asking (afterrelease.go
// personChecks), one step earlier: when the agent doing a task ticks its
// last own check (every open check left is a person's, or none), hug
// moves the task to doing/review with review_by the board's coordinator
// (hug.toml [board] coordinator), and the CLI hands the coordinator's live
// session one line (Deliver, kind review). Earlier ticks do nothing. The
// reviewer is never the worker: a task the coordinator is doing does not
// enter review by itself. done stays the reviewer's step: an approval is
// ship it, then done (erv680).

// ReviewWhy is the reason the move into review records in History.
func ReviewWhy(who string) string {
	return "review: " + receiptName(who) + " ticked the last own check"
}

// ReviewDue reports whether who's tick on t hands t to review, and to
// whom (the coordinator): t is doing and active, not approved, with no
// open question; who is its assignee and not the coordinator, and the
// coordinator does not do it too; every open check is a person's (people:
// hug.toml [people]) or none is open.
func ReviewDue(t Task, people []Person, who, coordinator string) (string, bool) {
	coordinator = strings.TrimSpace(coordinator)
	if coordinator == "" || t.State != "doing" || (t.Stage != "" && t.Stage != "active") {
		return "", false
	}
	if len(t.Acceptance) == 0 || t.Approval != nil || HasOpenQuestion(t) {
		return "", false
	}
	if !IsAssignee(t, who) || SameName(who, coordinator) || IsAssignee(t, coordinator) {
		return "", false
	}
	if !onlyPersonsOpen(t, people) {
		return "", false
	}
	return coordinator, true
}

// onlyPersonsOpen: every open check of t addresses a person, or none is
// open.
func onlyPersonsOpen(t Task, people []Person) bool {
	for _, a := range t.Acceptance {
		if a.Settled() {
			continue
		}
		if _, is := AddressesPerson(people, a.Text, len(t.Acceptance)); !is {
			return false
		}
	}
	return true
}

// SameName reports whether a and b name the same agent or person: case,
// spaces and a leading @ aside; two empty names are not the same.
func SameName(a, b string) bool {
	a = strings.TrimPrefix(strings.TrimSpace(a), "@")
	b = strings.TrimPrefix(strings.TrimSpace(b), "@")
	return a != "" && strings.EqualFold(a, b)
}

// IsAssignee reports whether who is one of t's assignees.
func IsAssignee(t Task, who string) bool {
	for _, a := range t.Assignee {
		if SameName(a, who) {
			return true
		}
	}
	return false
}

// StartReview runs after who ticked a check on id (the CLI and MCP check,
// the TUI's Space), after AfterTick: when the tick hands the task to
// review (ReviewDue), it moves it to doing/review with review_by the
// coordinator and returns that name; "" when nothing moved. A shipped
// task is past review: the release step (AfterTick) owns it.
func (s *Store) StartReview(id, who string) (string, error) {
	d, ts, e := s.load(id)
	if e != nil || releaseOf(d) != "" {
		return "", nil
	}
	people, _ := People(s.Root)
	coord, _ := Coordinator(s.Root)
	by, ok := ReviewDue(d.Task, people, who, coord)
	if !ok {
		return "", nil
	}
	set := map[string]string{"review_by": by, "review_since": stamp()}
	if e = applyMove(d, ts, move{state: "doing", stage: "review", set: set, clear: []string{"review_show"}, why: ReviewWhy(who)}); e != nil {
		return "", e
	}
	if _, e = s.finish(d, true); e != nil {
		return "", e
	}
	return by, nil
}

// reviewCount is how many moves into doing/review History holds: review N
// is the Nth, the item hug hands to its reviewer (Deliver, kind review).
func reviewCount(history []string) int {
	n := 0
	for _, l := range history {
		if m := moveLine.FindStringSubmatch(strings.TrimRight(l, "\r\n")); m != nil && m[1] == "doing" && m[2] == "review" {
			n++
		}
	}
	return n
}

// NewestItem is the item a write on d just made (task.NewestItem), and
// the newest review for "review".
func (d *Document) NewestItem(kind string) (Item, bool) {
	if kind == "review" {
		if n := reviewCount(d.historyLines()); n > 0 && ownHolds(d.Task, "review") {
			return Item{Kind: "review", N: n}, true
		}
		return Item{}, false
	}
	return NewestItem(d.Task, kind)
}

// OwedTo reports whether item on d is still owed to name's session (a
// pending delivery waiting for name's next session, bbxdd7): no line says
// it was delivered to name, or that name acknowledged it, and it is still
// open for name. The agent doing the task (5yh7hr) is owed a request
// until it is resolved and an answer until its asker acted on it, even
// when the coordinator took it meanwhile; anyone else (the coordinator)
// only while nobody has acknowledged it. A review is owed to its reviewer
// while the task is still in that review.
func (d *Document) OwedTo(item Item, name string) bool {
	if !hasItem(d, item) {
		return false
	}
	for _, l := range d.historyLines() {
		m := receiptLine.FindStringSubmatch(strings.TrimRight(l, "\r\n"))
		if m == nil || m[2] != item.Name() {
			continue
		}
		if (m[3] == "delivered to" || m[3] == "acknowledged by") && SameName(m[4], receiptName(name)) {
			return false
		}
	}
	r, _ := d.ItemReceipt(item)
	switch item.Kind {
	case "review":
		return item.N == reviewCount(d.historyLines()) && ownHolds(d.Task, "review") && SameName(d.Task.Fields["review_by"], name)
	case "request", "answer":
		if d.Task.State == "doing" && IsAssignee(d.Task, name) {
			return r.State != ReceiptAnswered
		}
	}
	return r.State == ReceiptPending
}
