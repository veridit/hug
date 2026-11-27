package cli

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/harness"
	"github.com/veridit/hug/huginn/internal/task"
)

// Delivery to the agent doing the task, else the board's coordinator
// (gmzd87, Jørgen 11:21Z: by design, not by doctrine; 11:59Z: answers too;
// 5yh7hr, 2026-11-26 22:01: the agent doing it first). Every write of a
// person's request (hug task request, task_update request, the TUI's r
// box) or answer (hug task answer, task_update answer, the TUI's a box)
// calls Deliver once, after its commit, as does a tick that puts a task in
// review (5yh7hr): hug hands the item to the live jcode session itself,
// and records what the daemon said on the task.
//
// DeliveryFlow is that flow in words. The instructions hug installs (the
// AGENTS.md section hug setup writes, the MCP tool descriptions) carry
// exactly these words, and TestDeliveryInstructionsMatchCode drives the
// code through each of its claims.
const DeliveryFlow = `Requests and answers reach the agent doing the task, else the board coordinator (hug task coordinator NAME), by hug itself: when a person writes hug task request or hug task answer (or r and a in hug tui), or ticks the check a question to them is about (that tick is their answer), hug hands it at once to the live session of the agent doing it (an idle session starts a turn, a busy one gets it at its next step); when that agent has none (or nobody does the task) to the coordinator's, saying it was not delivered live to that agent, and History says delivered or not delivered with the reason. The receiving session acknowledges it on receipt (hug's hook), its first write on the task marks it in progress, and hug task resolve ID N answers a request (an answer: the asker's next write, its outcome, which hug hands to the coordinator the same way in one line: "NAME acted on PERSON's answer on ID: WRITE"). hug task requests shows each state: pending, acknowledged, in progress, answered. Not delivered (no coordinator set, no live session, a harness hug cannot reach yet such as Claude Code or Codex) stays "pending, not delivered: REASON" and the request still comes first in hug task next. What no live session could take goes to that name's next session when it starts, once (hug's hook), the agent's doing the task too; a coordinator renamed before then gets it under the old name. A worker's last own tick (every open check left is a person's, or none) puts its task in review by the coordinator (doing/review), whose session hug tells in one line the same way; earlier ticks tell nobody. No agent polls.`

// deliverMessage is what the receiving session reads: who, the task, the
// text verbatim, the asker and the command that closes it. to is who
// reads it: the agent doing the task (5yh7hr: it hears first) or the
// board coordinator; missed, when the agent doing it could not take it
// live, says so to the coordinator.
func deliverMessage(board string, t task.Task, item task.Item, by, text string, answer *task.Answer, token string, targets []targetRef, to, missed string) string {
	var b strings.Builder
	// Which board: a coordinator of several boards runs the command there.
	where := " on board " + filepath.Base(board) + " (" + board + ")"
	role := ", to you as the board coordinator:\n"
	if t.State == "doing" && task.IsAssignee(t, to) {
		role = ", to you as the agent doing it:\n"
	}
	switch item.Kind {
	case "request":
		b.WriteString("hug: " + by + " wrote request " + strconv.Itoa(item.N) + " on " + t.ID + " (" + t.Title + ")" + forTargets(targets) + where + role)
	case "answer":
		head := "hug: " + by + " answered on " + t.ID + " (" + t.Title + ")"
		if answer != nil && answer.Check > 0 {
			head += ", check " + strconv.Itoa(answer.Check)
		}
		if answer != nil && answer.For != "" {
			head += ", the question " + answer.For + " asked"
		}
		b.WriteString(head + where + role)
		if answer != nil && answer.Question != "" {
			for _, l := range strings.Split(answer.Question, "\n") {
				b.WriteString("> " + l + "\n")
			}
		}
	}
	for _, l := range strings.Split(text, "\n") {
		b.WriteString("  " + l + "\n")
	}
	switch item.Kind {
	case "request":
		if ts := targets; len(ts) > 1 {
			// One request for several tasks (ec6z6j): each target has its
			// own resolve; the request closes when the last one resolves.
			b.WriteString("Turn it into changes to each ticket, and resolve each one (in " + board + "):\n")
			for _, tg := range ts {
				b.WriteString("  hug task resolve " + tg.ID + " " + strconv.Itoa(tg.N) + ` "what changed"` + "\n")
			}
		} else {
			b.WriteString("Turn it into changes to the ticket, then (in " + board + "): hug task resolve " + t.ID + " " + strconv.Itoa(item.N) + ` "what changed"` + "\n")
		}
		b.WriteString("If it is unclear: hug task ask " + t.ID + " --to " + by + ` "question"` + "\n")
	case "answer":
		if answer != nil && answer.For != "" && !strings.EqualFold(answer.For, by) && !task.SameName(answer.For, to) {
			b.WriteString("Route it to " + answer.For + ", who asked, or act on it: hug task show " + t.ID + "\n")
		} else {
			b.WriteString("Act on it: hug task show " + t.ID + "\n")
		}
	}
	if missed != "" {
		b.WriteString(missed + "\n")
	}
	b.WriteString("[" + token + "]")
	return b.String()
}

// missedLine tells the coordinator the agent doing the task could not take
// the item live (5yh7hr check 9): it waits for that agent's next session.
func missedLine(agent, reason string) string {
	return "Not delivered live to " + agent + ", who is doing it (" + oneLineText(reason) + "): hug hands it to " + agent + "'s next session when it starts."
}

// reviewMessage is the one line the reviewer's session reads when a task
// enters review by itself (5yh7hr): who ticked the last check, the task,
// and the answer that approves it.
func reviewMessage(board string, t task.Task, token string) string {
	owner := strings.Join(t.Assignee, ", ")
	if owner == "" {
		owner = "its owner"
	}
	return "hug: " + owner + " ticked the last own check of " + t.ID + " (" + oneLineText(t.Title) + ") on board " + filepath.Base(board) + " (" + board + "): it is in review by you. Read it (hug task show " + t.ID + "), then answer: hug task answer " + t.ID + ` "looks right" (or what to change) [` + token + "]"
}

// itemMessage is the message for item on t to session name: the review
// line, the outcome line, or a request's or answer's block.
func itemMessage(board string, t task.Task, item task.Item, by, text string, answer *task.Answer, token string, targets []targetRef, to, missed string) string {
	switch item.Kind {
	case "outcome":
		return outcomeMessage(board, t, token)
	case "review":
		return reviewMessage(board, t, token)
	}
	return deliverMessage(board, t, item, by, text, answer, token, targets, to, missed)
}

// outcomeMessage is the one line the coordinator's session reads when an
// asker acts on an answer (41hhgn check 11, Jørgen 00:26: the outcome comes
// back through the coordinator): who acted on whose answer, the answer cut
// short (an answer still waiting for the coordinator's next session is not
// owed once its asker acted, so this line is all it gets), the task, and
// the write as the command reads.
func outcomeMessage(board string, t task.Task, token string) string {
	o := t.Outcome
	line := "hug: " + o.For + " acted on " + o.By + "'s answer on " + t.ID + " (" + t.Title + ")"
	if o.Check > 0 {
		line += ", check " + strconv.Itoa(o.Check)
	}
	line += ", on board " + filepath.Base(board) + " (" + board + "): " + strings.Join(strings.Fields(o.Text), " ")
	if a := strings.Join(strings.Fields(o.Answer), " "); a != "" {
		if r := []rune(a); len(r) > 120 {
			a = string(r[:119]) + "…"
		}
		line += " (the answer: " + strconv.Quote(a) + ")"
	}
	return line + ". hug task show " + t.ID + " [" + token + "]"
}

// jcodeHomes are the jcode homes whose sessions Deliver looks in: App.
// JcodeHomes when set, else every instance on this machine. A test binary
// gets none unless it sets them: a test never reaches the machine's real
// coordinator.
func (a *App) jcodeHomes() []string {
	if a.JcodeHomes != nil {
		return a.JcodeHomes
	}
	if testing.Testing() {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var extra []string
	if h := a.getenv("JCODE_HOME"); h != "" {
		extra = append(extra, h)
	}
	var out []string
	for _, i := range harness.DiscoverJcode(home, extra) {
		out = append(out, i.Home)
	}
	return out
}

// Deliver hands the item kind ("request", "answer", "outcome" or
// "review") just written on id by by to the session that should hear it,
// and records the daemon's answer on the task with its own commit. It
// returns the receipt label for the reply. A failed delivery is no error:
// it is the receipt, kept on the task.
//
// Who hears it (5yh7hr, Jørgen 2026-11-26 22:01): a request or answer on a
// task an agent is doing goes to that agent's live session first, and to
// the board coordinator only when that agent has none; History says
// which. What the agent's session could not take waits for its next
// session, and the coordinator's message says it was not delivered live.
// A review goes to its reviewer (review_by), an outcome to the
// coordinator.
func (a *App) Deliver(store *task.Store, root, board, id, kind, by string, commit func(string) error) (string, error) {
	d, err := store.Get(id)
	if err != nil {
		return "", err
	}
	item, ok := d.NewestItem(kind)
	if !ok {
		return "", nil
	}
	answer := d.LastAnswer()
	text := ""
	if kind == "answer" && answer != nil {
		text = answer.Text
	}
	for _, r := range d.Task.Requests {
		if kind == "request" && r.N == item.N {
			text = r.Text
		}
	}
	coord, err := task.Coordinator(root)
	if err != nil {
		return "", err
	}
	if kind == "review" {
		coord = strings.TrimSpace(d.Task.Fields["review_by"])
	}
	homes := a.jcodeHomes()
	var targets []targetRef
	if kind == "request" {
		if ts, e := store.Tasks(); e == nil {
			targets = requestTargets(ts, d.Task, item.N)
		}
	}
	// One attempt: to name, with missed for the coordinator's message.
	type attempt struct {
		to      string
		session harness.Session
		reason  string
	}
	try := func(to, missed string) attempt {
		token := newToken(d.Task.ID, item)
		it := harness.InboxItem{Board: board, Task: d.Task.ID, Kind: item.Kind, N: item.N, To: to, Token: token, Missed: missed}
		msg := itemMessage(board, d.Task, item, by, text, answer, token, targets, to, missed)
		s, reason := harness.Deliver(homes, a.getenv, to, msg, it)
		if reason != "" {
			// Not delivered: it waits for to's next session, in every
			// jcode home hug knows (bbxdd7). That session's hook hands it
			// over at its start; History decides it is still owed.
			for _, h := range homes {
				if err := harness.AddPending(h, to, it); err != nil {
					reason += "; cannot keep it for " + to + "'s next session: " + err.Error()
					break
				}
			}
		}
		return attempt{to, s, reason}
	}
	var tries []attempt
	missed := ""
	if agent := a.doer(root, d.Task, kind, by, coord); agent != "" {
		at := try(agent, "")
		tries = append(tries, at)
		if at.reason != "" {
			missed = missedLine(agent, at.reason)
		}
	}
	// The agent doing it has it live: nobody else is told.
	if len(tries) == 0 || tries[0].reason != "" {
		switch {
		case coord != "" && strings.EqualFold(coord, strings.TrimSpace(by)):
			// The coordinator wrote it itself: nothing to deliver to it.
			if len(tries) == 0 {
				return "", nil
			}
		case coord != "":
			tries = append(tries, try(coord, missed))
		default:
			tries = append(tries, attempt{reason: "no coordinator set (hug task coordinator NAME)"})
		}
	}
	actor, action := store.Actor, store.Action
	store.Actor, store.Action = "", ""
	for _, at := range tries {
		if _, err = store.Delivered(d.Task.ID, item, at.to, at.session.ID, at.reason); err != nil {
			break
		}
	}
	store.Actor, store.Action = actor, action
	if err != nil {
		return "", err
	}
	last := tries[len(tries)-1]
	msg := "hug task " + item.Name() + " on " + d.Task.ID + " delivered to " + last.to
	if last.reason != "" {
		msg = "hug task " + item.Name() + " on " + d.Task.ID + " not delivered"
	}
	if err := commit(msg); err != nil {
		return "", err
	}
	label := "pending, delivered to " + last.to + " (" + last.session.ID + ")"
	if last.reason != "" {
		label = "pending, not delivered: " + last.reason
	}
	if len(tries) > 1 && tries[0].reason != "" {
		label += "; not delivered live to " + tries[0].to + ", who is doing it: " + tries[0].reason
	}
	return label, nil
}

// doer is the agent doing t who hears a request or answer first (5yh7hr):
// its first assignee that is not a person on the board (hug.toml
// [people]), not who wrote it and not the coordinator; "" for other kinds
// or when there is none.
func (a *App) doer(root string, t task.Task, kind, by, coord string) string {
	if (kind != "request" && kind != "answer") || t.State != "doing" {
		return ""
	}
	people, _ := task.People(root)
	for _, n := range t.Assignee {
		n = strings.TrimPrefix(strings.TrimSpace(n), "@")
		if n == "" || task.SameName(n, by) || task.SameName(n, coord) {
			continue
		}
		if _, person := task.PersonByName(people, n); person {
			continue
		}
		return n
	}
	return ""
}

// newToken marks one delivery's message, so the coordinator's hook finds
// exactly that message in its transcript.
func newToken(id string, item task.Item) string {
	b := make([]byte, 4)
	rand.Read(b)
	return "hug " + id + " " + item.Kind + " " + strconv.Itoa(item.N) + " " + hex.EncodeToString(b)
}
