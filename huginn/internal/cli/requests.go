package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/internal/memshare"
)

// runRequests is hug task requests [ID] [--all] (hbgybn): the open requests
// across the board, oldest first, each with who asked, when, its first line
// and the command that closes it; with ID, that task's requests, resolved
// ones included with --all.
func (a *App) runRequests(s *session, f flags) (Reply, error) {
	if len(f.args) > 1 {
		return nil, f.wrongCount("requests takes at most one task")
	}
	ts, e := s.s.Tasks()
	if e != nil {
		return nil, e
	}
	all := ts
	if len(f.args) == 1 {
		t, e := task.Resolve(ts, f.args[0])
		if e != nil {
			return nil, e
		}
		ts = []task.Task{t}
	}
	rows := []RequestRow{}
	for _, t := range ts {
		for _, r := range withHomes(all, t.Requests) {
			// A request for several tasks stays listed on its home until
			// every target is resolved (ec6z6j): a partial fix is partial.
			listed := r.Open() || r.Home == nil && r.Targets != nil && !r.Closed()
			if !listed && !f.has("all") {
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
		b.WriteString(head + ": " + x.Title + linkSuffix(x.Req) + "\n")
		// Its receipt (gmzd87): how far it has reached the coordinator.
		b.WriteString("  receipt: " + x.Req.Receipt.Label() + "\n")
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
	// One request for several tasks (ec6z6j): each target pending or
	// resolved, so a partial fix shows as partial.
	if line := targetsLine(r); line != "" {
		b.WriteString("  " + line + "\n")
	}
	if r.Resolved != nil {
		b.WriteString("  resolved by " + r.Resolved.By + ", " + strings.Replace(r.Resolved.At, "T", " ", 1) + ":\n")
		for _, l := range strings.Split(r.Resolved.Text, "\n") {
			b.WriteString("    " + l + "\n")
		}
	}
	return b.String()
}

// runMemory is hug task memory [share | local] (34r1ft, Jørgen 18:18: the
// setting is committed with the tasks): share writes hug.toml [memory]
// share = true on hug/tasks, so hug mem sync may push this repository's
// project memories to its remote; local removes it (the default). Bare,
// it says which.
func (a *App) runMemory(s *session, f flags) (Reply, error) {
	root := s.r.Dir
	if len(f.args) > 1 {
		return nil, f.wrongCount("memory takes share or local")
	}
	says := func(share bool) string {
		if share {
			return "share"
		}
		return "local"
	}
	if len(f.args) == 0 {
		share, e := task.MemoryShare(root)
		if e != nil {
			return nil, e
		}
		next := memshare.Fix
		msg := "project memories: local (hug mem sync pushes nothing; hug.toml [memory] share is not true)"
		if share {
			next = "hug mem sync"
			msg = "project memories: share (hug.toml [memory] share = true; hug mem sync pushes them to hug/memory)"
		}
		return Reply{"memory": says(share), "message": msg, "next": next}, nil
	}
	var share bool
	switch f.args[0] {
	case "share":
		share = true
	case "local":
	default:
		return nil, usage("memory is share or local, not "+strconv.Quote(f.args[0]), memshare.Fix)
	}
	changed, e := task.SetMemoryShare(root, share)
	if e != nil {
		return nil, e
	}
	next := "hug mem sync"
	msg := "project memories: share, committed on hug/tasks; every clone follows it"
	if !share {
		next = "hug task memory"
		msg = "project memories: local, committed on hug/tasks; hug mem sync pushes nothing"
	}
	if !changed {
		return Reply{"memory": says(share), "changed": false, "message": "project memories: already " + says(share), "next": next}, nil
	}
	if e := s.commit("hug task memory " + says(share)); e != nil {
		return nil, e
	}
	return Reply{"memory": says(share), "changed": true, "message": msg, "next": next}, nil
}

// runCoordinator is hug task coordinator [NAME | --clear] (hbgybn, crab
// 00:58): the session that hears what no owner hears, kept in hug.toml
// [board] coordinator. Without a name it says who it is.
func (a *App) runCoordinator(s *session, f flags) (Reply, error) {
	root := s.r.Dir
	if len(f.args) > 1 {
		return nil, f.wrongCount("coordinator takes one name")
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
