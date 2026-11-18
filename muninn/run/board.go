package run

import (
	"context"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/board"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/recall"
)

// Board nudges in the recall hook (qawmnb). boardSource reads the task board
// of the repository at cwd and the session's tool calls since the last
// check; activity reads them from the harness transcript.
func boardSource(ctx context.Context, on bool, cwd, name string, activity func(since time.Time) harness.Activity) recall.BoardSource {
	if cwd == "" || !on {
		return nil
	}
	return func(since time.Time, wantLint bool) *recall.BoardInput {
		act := activity(since)
		snap, err := board.Open(ctx, cwd)
		if err != nil {
			logHook("board: %v", err)
			return nil
		}
		if snap == nil {
			return nil
		}
		defer snap.Close()
		in := &recall.BoardInput{Self: recall.BoardSelf{Name: name, Model: act.Model, Branch: snap.Place.Branch, Worktree: snap.Place.Worktree, Linked: snap.Place.Linked}}
		in.Tasks = boardTasks(snap.Tasks)
		in.Coordinator = snap.Coordinator
		in.Actions = boardActions(act.Actions, snap.Place.Worktree)
		if wantLint {
			// Read now: the snapshot closes when this returns.
			fs, err := snap.Findings()
			if err != nil {
				logHook("board lint: %v", err)
			}
			found := boardFindings(fs)
			in.Findings = func() []recall.BoardFinding { return found }
		}
		return in
	}
}

func boardTasks(ts []board.Task) []recall.BoardTask {
	out := make([]recall.BoardTask, 0, len(ts))
	for _, t := range ts {
		out = append(out, BoardTask(t))
	}
	return out
}

// BoardTask is the recall view of one board task, its unread answer
// included (3jtfw9).
func BoardTask(t board.Task) recall.BoardTask {
	bt := recall.BoardTask{ID: t.ID, Title: t.Title, State: t.State, Assignee: t.Assignee, Branch: t.Branch, Worktree: t.Worktree, Checks: t.Checks, Open: t.Open}
	if a := t.Answer; a != nil {
		bt.Answer = &recall.BoardAnswer{Stage: a.Stage, Question: a.Question, Text: a.Text, By: a.By, For: a.For, At: a.At}
	}
	if o := t.Outcome; o != nil {
		bt.Outcome = &recall.BoardOutcome{Stage: o.Stage, By: o.By, For: o.For, At: o.At, Text: o.Text}
	}
	for _, r := range t.Requests {
		bt.Requests = append(bt.Requests, recall.BoardRequest{N: r.N, By: r.By, At: r.At, Text: r.Text})
	}
	return bt
}

func boardFindings(fs []board.Finding) []recall.BoardFinding {
	out := make([]recall.BoardFinding, 0, len(fs))
	for _, f := range fs {
		out = append(out, recall.BoardFinding{TaskID: f.TaskID, TaskState: f.TaskState, Problem: f.Problem, Fix: f.Fix})
	}
	return out
}

func boardActions(as []harness.Action, root string) []recall.BoardAction {
	out := make([]recall.BoardAction, 0, len(as))
	for _, a := range as {
		b := recall.BoardAction{Edits: a.EditsIn(root), Subject: a.Subject}
		for _, h := range a.Hug {
			b.Hug = append(b.Hug, recall.BoardHug{Verb: h.Verb, ID: h.ID, N: h.N})
		}
		out = append(out, b)
	}
	return out
}

// sessionName is the harness's name for a jcode session (session_<name>_...),
// the name hug task take records as assignee.
func sessionName(sid string) string {
	if name, _, ok := strings.Cut(strings.TrimPrefix(sid, "session_"), "_"); ok && strings.HasPrefix(sid, "session_") {
		return name
	}
	return ""
}
