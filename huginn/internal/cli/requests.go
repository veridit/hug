package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// runRequests is hug task requests [ID] [--all] (hbgybn): the open requests
// across the board, oldest first, each with who asked, when, its first line
// and the command that closes it; with ID, that task's requests, resolved
// ones included with --all.
func (a *App) runRequests(s *session, f flags) (Reply, error) {
	if len(f.args) > 1 {
		return nil, usage("requests takes at most one task", "hug task requests ID")
	}
	ts, e := s.s.Tasks()
	if e != nil {
		return nil, e
	}
	if len(f.args) == 1 {
		t, e := task.Resolve(ts, f.args[0])
		if e != nil {
			return nil, e
		}
		ts = []task.Task{t}
	}
	rows := []RequestRow{}
	for _, t := range ts {
		for _, r := range t.Requests {
			if !r.Open() && !f.has("all") {
				continue
			}
			x := RequestRow{ID: t.ID, Title: t.Title, Owner: t.Assignee, Req: r}
			if r.Open() {
				x.Fix = "hug task resolve " + t.ID + " " + strconv.Itoa(r.N) + ` "what changed"`
			}
			rows = append(rows, x)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Req.At < rows[j].Req.At })
	next := "hug task next"
	if len(rows) > 0 && rows[0].Fix != "" {
		next = "hug task show " + rows[0].ID + ", then: " + rows[0].Fix
	}
	return Reply{"requests": rows, "next": next}, nil
}

// RequestRow is one request hug task requests lists.
type RequestRow struct {
	ID    string       `json:"id"`
	Title string       `json:"title"`
	Owner []string     `json:"assignee"`
	Req   task.Request `json:"request"`
	Fix   string       `json:"fix,omitempty"`
}

// requestsText is hug task requests as text: one block per request, its
// text verbatim, then the command that closes it.
func requestsText(rows []RequestRow) string {
	var b strings.Builder
	if len(rows) == 0 {
		b.WriteString("no open requests\n")
	}
	for _, x := range rows {
		head := fmt.Sprintf("%s request %d by %s, %s", x.ID, x.Req.N, x.Req.By, strings.Replace(x.Req.At, "T", " ", 1))
		if x.Req.Resolved != nil {
			head += ", resolved by " + x.Req.Resolved.By
		}
		b.WriteString(head + ": " + x.Title + "\n")
		b.WriteString(requestBody(x.Req))
		if x.Fix != "" {
			b.WriteString("  fix: " + x.Fix + "\n")
		}
	}
	return b.String()
}

// requestBody is a request's text indented two, and its resolve account
// under "resolved by X:" indented four, every line verbatim.
func requestBody(r task.Request) string {
	var b strings.Builder
	for _, l := range strings.Split(r.Text, "\n") {
		b.WriteString("  " + l + "\n")
	}
	if r.Resolved != nil {
		b.WriteString("  resolved by " + r.Resolved.By + ", " + strings.Replace(r.Resolved.At, "T", " ", 1) + ":\n")
		for _, l := range strings.Split(r.Resolved.Text, "\n") {
			b.WriteString("    " + l + "\n")
		}
	}
	return b.String()
}

// runCoordinator is hug task coordinator [NAME | --clear] (hbgybn, crab
// 00:58): the session that hears what no owner hears, kept in hug.toml
// [board] coordinator. Without a name it says who it is.
func (a *App) runCoordinator(s *session, f flags) (Reply, error) {
	root := s.r.Dir
	if len(f.args) > 1 {
		return nil, usage("coordinator takes one name", "hug task coordinator NAME")
	}
	if len(f.args) == 0 && !f.has("clear") {
		name, e := task.Coordinator(root)
		if e != nil {
			return nil, e
		}
		next := "hug task requests"
		if name == "" {
			next = "hug task coordinator NAME"
		}
		return Reply{"coordinator": name, "next": next}, nil
	}
	name := ""
	if len(f.args) == 1 {
		name = strings.TrimSpace(strings.TrimPrefix(f.args[0], "@"))
		if name == "" || strings.ContainsAny(name, " \t\"[]=") {
			return nil, usage("a coordinator is one name, as the board shows it: "+strconv.Quote(f.args[0]), "hug task coordinator NAME")
		}
	}
	if e := task.SetCoordinator(root, name); e != nil {
		return nil, e
	}
	msg := "hug task coordinator " + name
	if name == "" {
		msg = "hug task coordinator --clear"
	}
	if e := s.commit(msg); e != nil {
		return nil, e
	}
	return Reply{"coordinator": name, "changed": true, "next": "hug task requests"}, nil
}

// actionLine is a write as the command reads, for an answer's outcome
// (41hhgn check 11): hug task note ID "the text", each text cut to 120
// characters on one line, stdin ("-") said as such.
func actionLine(cmd string, args []string) string {
	parts := []string{"hug task " + cmd}
	for _, x := range args {
		switch {
		case x == "-":
			parts = append(parts, "(from stdin)")
		case strings.ContainsAny(x, " \t\n\""):
			x = strings.Join(strings.Fields(x), " ")
			if r := []rune(x); len(r) > 120 {
				x = string(r[:119]) + "…"
			}
			parts = append(parts, strconv.Quote(x))
		default:
			parts = append(parts, x)
		}
	}
	return strings.Join(parts, " ")
}
