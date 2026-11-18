package cli

import (
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// One request for several tasks (ec6z6j): written once on its home, the
// first task named, with a link on every other target. These helpers name
// the targets the coordinator resolves and that show and requests list.

// targetRef is one target of a request: the task and the request's number
// on it (the home's own number on the home, the link's on a link).
type targetRef struct {
	ID string
	N  int
}

// requestTargets are the targets of request n on t, home first, each with
// its own request number; nil for a plain request. A link's number comes
// from the link block, found through the board's tasks ts.
func requestTargets(ts []task.Task, t task.Task, n int) []targetRef {
	var req *task.Request
	for i := range t.Requests {
		if t.Requests[i].N == n {
			req = &t.Requests[i]
		}
	}
	if req == nil || req.Targets == nil {
		return nil
	}
	var out []targetRef
	for _, tg := range req.Targets {
		if strings.EqualFold(tg.ID, t.ID) {
			out = append(out, targetRef{t.ID, n})
			continue
		}
		ref := targetRef{tg.ID, 0}
		for _, o := range ts {
			if !strings.EqualFold(o.ID, tg.ID) {
				continue
			}
			for _, r := range o.Requests {
				if r.Home != nil && strings.EqualFold(r.Home.ID, t.ID) && r.Home.N == n {
					ref.N = r.N
				}
			}
		}
		out = append(out, ref)
	}
	return out
}

// forTargets is " for 3 tasks: a, b, c" for a request for several tasks.
func forTargets(targets []targetRef) string {
	if len(targets) < 2 {
		return ""
	}
	var ids []string
	for _, x := range targets {
		ids = append(ids, x.ID)
	}
	return " for " + strconv.Itoa(len(targets)) + " tasks: " + strings.Join(ids, ", ")
}

// requestedIDs is the selection of t's newest request, comma-separated,
// as the request command named it: the commit message says what was
// written.
func requestedIDs(t task.Task) string {
	if it, ok := task.NewestItem(t, "request"); ok {
		for _, r := range t.Requests {
			if r.N == it.N && r.Targets != nil {
				var ids []string
				for _, tg := range r.Targets {
					ids = append(ids, tg.ID)
				}
				return strings.Join(ids, ",")
			}
		}
	}
	return t.ID
}

// targetsLine is "targets: a pending, b resolved, c pending".
func targetsLine(r task.Request) string {
	if r.Targets == nil {
		return ""
	}
	var parts []string
	for _, tg := range r.Targets {
		state := "pending"
		if tg.Resolved {
			state = "resolved"
		}
		parts = append(parts, tg.ID+" "+state)
	}
	return "targets: " + strings.Join(parts, ", ")
}

// linkSuffix is ": see HOME request N" on a link, else "".
func linkSuffix(r task.Request) string {
	if r.Home == nil {
		return ""
	}
	return ": see " + r.Home.ID + " request " + strconv.Itoa(r.Home.N)
}

// withHomes gives each link the home's text and targets (ec6z6j): the
// person and the agent read one request wherever it is linked from.
func withHomes(ts []task.Task, rs []task.Request) []task.Request {
	var out []task.Request
	for _, r := range rs {
		if r.Home != nil {
			for _, o := range ts {
				if !strings.EqualFold(o.ID, r.Home.ID) {
					continue
				}
				for _, h := range o.Requests {
					if h.N == r.Home.N && h.Targets != nil {
						r.Text, r.Targets, r.AllResolved = h.Text, h.Targets, h.AllResolved
					}
				}
			}
		}
		out = append(out, r)
	}
	return out
}
