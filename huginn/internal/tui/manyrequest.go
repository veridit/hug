package tui

import (
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// A request for several tasks in a task's page (ec6z6j check 4,
// board-request-mock.md section 4). Its home, the first task named, holds
// the text; every other target holds a link with no text. Both pages draw
// the text and a targets line, each target pending or resolved, so a
// partial fix reads as partial; a link's head also says where the request
// lives, "request N on HOME". The CLI says the same (hug task show,
// hug task requests: targetsLine, withHomes).

// shownRequest is r as a page draws it: a link takes its home's text and
// targets, and, while it is open, the home's receipt: the request was
// delivered once, through its home.
func (m model) shownRequest(r task.Request) task.Request {
	if r.Home == nil {
		return r
	}
	for _, o := range m.all {
		if !strings.EqualFold(o.ID, r.Home.ID) {
			continue
		}
		for _, h := range o.Requests {
			if h.N == r.Home.N {
				r.Text, r.Targets, r.AllResolved = h.Text, h.Targets, h.AllResolved
				if r.Resolved == nil {
					r.Receipt = h.Receipt
				}
			}
		}
	}
	return r
}

// requestTargets is "targets: a pending, b resolved, c pending", or ""
// for a request for one task.
func requestTargets(r task.Request) string {
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

// requestHome is "request N on HOME" on a link, else "".
func requestHome(r task.Request) string {
	if r.Home == nil {
		return ""
	}
	return "request " + strconv.Itoa(r.Home.N) + " on " + r.Home.ID
}

// requestStopText is the text of request r whose ids are stops, in the
// order its lines draw them: the home on a link's head, the text, the
// targets.
func requestStopText(r task.Request) string {
	parts := []string{r.Text}
	if h := requestHome(r); h != "" {
		parts = append([]string{h}, parts...)
	}
	if tg := requestTargets(r); tg != "" {
		parts = append(parts, tg)
	}
	return strings.Join(parts, "\n")
}
