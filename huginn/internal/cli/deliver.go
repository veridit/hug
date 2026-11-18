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

// Delivery to the board's coordinator (gmzd87, Jørgen 11:21Z: by design,
// not by doctrine; 11:59Z: answers too). Every write of a person's request
// (hug task request, task_update request, the TUI's r box) or answer (hug
// task answer, task_update answer, the TUI's a box) calls Deliver once,
// after its commit: hug hands the item to the coordinator's live jcode
// session itself, and records what the daemon said on the task.
//
// DeliveryFlow is that flow in words. The instructions hug installs (the
// AGENTS.md section hug setup writes, the MCP tool descriptions) carry
// exactly these words, and TestDeliveryInstructionsMatchCode drives the
// code through each of its claims.
const DeliveryFlow = `Requests and answers reach the board coordinator (hug task coordinator NAME) by hug itself: when a person writes hug task request or hug task answer (or r and a in hug tui), hug hands it at once to the coordinator's live session (an idle session starts a turn, a busy one gets it at its next step) and History says delivered or not delivered with the reason. The coordinator's session acknowledges it on receipt (hug's hook), its first write on the task marks it in progress, and hug task resolve ID N answers a request (an answer: the asker's next write). hug task requests shows each state: pending, acknowledged, in progress, answered. Not delivered (no coordinator set, no live session, a harness hug cannot reach yet such as Claude Code or Codex) stays "pending, not delivered: REASON" and the request still comes first in hug task next. What no live session could take goes to the coordinator's next session when it starts, once (hug's hook); a coordinator renamed before then gets it under the old name. No agent polls.`

// deliverMessage is what the coordinator's session reads: who, the task,
// the text verbatim, the asker and the command that closes it.
func deliverMessage(board string, t task.Task, item task.Item, by, text string, answer *task.Answer, token string) string {
	var b strings.Builder
	// Which board: a coordinator of several boards runs the command there.
	where := " on board " + filepath.Base(board) + " (" + board + ")"
	switch item.Kind {
	case "request":
		b.WriteString("hug: " + by + " wrote request " + strconv.Itoa(item.N) + " on " + t.ID + " (" + t.Title + ")" + where + ", to you as the board coordinator:\n")
	case "answer":
		head := "hug: " + by + " answered on " + t.ID + " (" + t.Title + ")"
		if answer != nil && answer.Check > 0 {
			head += ", check " + strconv.Itoa(answer.Check)
		}
		if answer != nil && answer.For != "" {
			head += ", the question " + answer.For + " asked"
		}
		b.WriteString(head + where + ", to you as the board coordinator:\n")
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
		b.WriteString("Turn it into changes to the ticket, then (in " + board + "): hug task resolve " + t.ID + " " + strconv.Itoa(item.N) + ` "what changed"` + "\n")
		b.WriteString("If it is unclear: hug task ask " + t.ID + " --to " + by + ` "question"` + "\n")
	case "answer":
		if answer != nil && answer.For != "" && !strings.EqualFold(answer.For, by) {
			b.WriteString("Route it to " + answer.For + ", who asked, or act on it: hug task show " + t.ID + "\n")
		} else {
			b.WriteString("Act on it: hug task show " + t.ID + "\n")
		}
	}
	b.WriteString("[" + token + "]")
	return b.String()
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

// Deliver hands the item kind ("request" or "answer") just written on id
// by by to the board's coordinator, and records the daemon's answer on the
// task with its own commit. It returns the receipt label for the reply. A
// failed delivery is no error: it is the receipt, kept on the task.
func (a *App) Deliver(store *task.Store, root, board, id, kind, by string, commit func(string) error) (string, error) {
	d, err := store.Get(id)
	if err != nil {
		return "", err
	}
	item, ok := task.NewestItem(d.Task, kind)
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
	if coord != "" && strings.EqualFold(coord, strings.TrimSpace(by)) {
		// The coordinator wrote it itself: nothing to deliver.
		return "", nil
	}
	var s harness.Session
	reason := "no coordinator set (hug task coordinator NAME)"
	if coord != "" {
		token := newToken(d.Task.ID, item)
		it := harness.InboxItem{Board: board, Task: d.Task.ID, Kind: item.Kind, N: item.N, To: coord, Token: token}
		homes := a.jcodeHomes()
		s, reason = harness.Deliver(homes, a.getenv, coord, deliverMessage(board, d.Task, item, by, text, answer, token), it)
		if reason != "" {
			// Not delivered: it waits for coord's next session, in every
			// jcode home hug knows (bbxdd7). That session's hook hands it
			// over at its start; History decides it is still owed.
			for _, h := range homes {
				if err := harness.AddPending(h, coord, it); err != nil {
					reason += "; cannot keep it for " + coord + "'s next session: " + err.Error()
					break
				}
			}
		}
	}
	actor, action := store.Actor, store.Action
	store.Actor, store.Action = "", ""
	_, err = store.Delivered(d.Task.ID, item, coord, s.ID, reason)
	store.Actor, store.Action = actor, action
	if err != nil {
		return "", err
	}
	msg := "hug task " + item.Name() + " on " + d.Task.ID + " delivered to " + coord
	if reason != "" {
		msg = "hug task " + item.Name() + " on " + d.Task.ID + " not delivered"
	}
	if err := commit(msg); err != nil {
		return "", err
	}
	if reason != "" {
		return "pending, not delivered: " + reason, nil
	}
	return "pending, delivered to " + coord + " (" + s.ID + ")", nil
}

// newToken marks one delivery's message, so the coordinator's hook finds
// exactly that message in its transcript.
func newToken(id string, item task.Item) string {
	b := make([]byte, 4)
	rand.Read(b)
	return "hug " + id + " " + item.Kind + " " + strconv.Itoa(item.N) + " " + hex.EncodeToString(b)
}
