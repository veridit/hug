// Package holds parses and checks the condition a memory holds under
// (ez50rb): "until 2026-12-01", "while task:ID open", "while branch NAME
// unmerged". The storer states it at write time; hug checks it later
// without a model call. No condition means the memory holds until judged
// otherwise.
package holds

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Kind of condition.
const (
	Until  = "until"
	Task   = "task"
	Branch = "branch"
)

// Cond is one parsed condition. Text is its canonical form, as stored.
type Cond struct {
	Kind string
	Date time.Time // Until: the first day it no longer holds
	Ref  string    // Task: the task id; Branch: the branch name
	Text string
}

var (
	untilRe  = regexp.MustCompile(`^until (\d{4}-\d{2}-\d{2})$`)
	taskRe   = regexp.MustCompile(`^while task:([a-z0-9][a-z0-9.\-]*) (?:is )?open$`)
	branchRe = regexp.MustCompile(`^while (?:branch )?([A-Za-z0-9][A-Za-z0-9._/\-]*) (?:is )?unmerged$`)
)

// Fix is the line an error ends with (MCP: the same words go in holds).
const Fix = `fix: hug mem remember --holds "until 2026-12-01" (or "while task:ID open", "while branch NAME unmerged") "TEXT"`

// Parse reads a condition. Empty input is no condition (ok, zero Cond).
func Parse(s string) (Cond, error) {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if s == "" {
		return Cond{}, nil
	}
	low := strings.ToLower(s)
	if m := untilRe.FindStringSubmatch(low); m != nil {
		d, err := time.ParseInLocation("2006-01-02", m[1], time.Local)
		if err != nil {
			return Cond{}, fmt.Errorf("holds %q: %s is not a date\n%s", s, m[1], Fix)
		}
		return Cond{Kind: Until, Date: d, Text: "until " + m[1]}, nil
	}
	if m := taskRe.FindStringSubmatch(low); m != nil {
		return Cond{Kind: Task, Ref: m[1], Text: "while task:" + m[1] + " open"}, nil
	}
	if m := branchRe.FindStringSubmatch(s); m != nil && !strings.HasPrefix(low, "while task:") {
		return Cond{Kind: Branch, Ref: m[1], Text: "while branch " + m[1] + " unmerged"}, nil
	}
	return Cond{}, fmt.Errorf("holds %q: not a condition hug can check\n%s", s, Fix)
}

// State is what a checker knows about the world.
type State struct {
	Now time.Time
	// TaskOpen reports a task's state: known false when the task is done
	// or dropped; a task not on the board is unknown.
	TaskOpen func(id string) (open, known bool)
	// BranchLive reports a branch: known false when it is merged into the
	// default branch or deleted.
	BranchLive func(name string) (live, known bool)
}

// Ended reports whether the condition no longer holds, and why. An
// unknown answer (no board, not in git) is never an end.
func (c Cond) Ended(st State) (bool, string) {
	switch c.Kind {
	case Until:
		if !st.Now.Before(c.Date) {
			return true, "the date " + c.Date.Format("2006-01-02") + " has passed"
		}
	case Task:
		if st.TaskOpen != nil {
			if open, known := st.TaskOpen(c.Ref); known && !open {
				return true, "task " + c.Ref + " is closed"
			}
		}
	case Branch:
		if st.BranchLive != nil {
			if live, known := st.BranchLive(c.Ref); known && !live {
				return true, "branch " + c.Ref + " is merged or deleted"
			}
		}
	}
	return false, ""
}
