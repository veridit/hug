package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/huginn/internal/tui"
)

// AskedRow is one line of hug task ls --asked (fthher): an open question
// asked of someone, as ASKED in hug tui counts it.
type AskedRow struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
	// N is the question's number on the task (hug task answer ID N), 0
	// for a stage question without one (design, review, an old ask).
	N int `json:"n"`
	// Check is the acceptance check the question is about, 0 for none.
	Check    int    `json:"check"`
	Question string `json:"question"`
	By       string `json:"by"`
	To       string `json:"to"`
	At       string `json:"at"`
	// Since is the TUI's words for At: "since 14:42" today, else "since 15 Nov".
	Since string `json:"since"`
}

// askedFor is who ls --asked lists questions for: --to NAME, else the
// name --mine uses (--as, HUG_USER, the agent session, else the model);
// with none of those, a person, found by git email in hug.toml [people]
// as hug tui finds them. email is set only for that last case.
func (a *App) askedFor(ctx context.Context, f flags, dir string, people []task.Person) (name, email string) {
	if to := strings.TrimSpace(f.get("to")); to != "" {
		return to, ""
	}
	if who := a.whoTakes(f); who.Name != "" {
		return who.Name, ""
	}
	if email = gitEmail(ctx, a.Cwd); email != "" {
		return tui.Me(people, "", email), email
	}
	return a.takerModel(f), ""
}

// gitEmail is git config user.email in dir, "" when unset.
func gitEmail(ctx context.Context, dir string) string {
	out, e := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.email").Output()
	if e != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// lsAsked is hug task ls --asked: the open questions asked of the caller
// (or --to NAME) on the tasks the other ls filters keep, oldest first.
func (a *App) lsAsked(ctx context.Context, s *session, f flags, filter task.Filter, ts []task.Task) (Reply, error) {
	people, _ := task.People(s.r.Dir)
	name, email := a.askedFor(ctx, f, s.r.Dir, people)
	if name == "" {
		return nil, usage("ls --asked needs to know who you are: no --to, --as, HUG_USER, agent session, git email or model", "hug task ls --asked --to NAME")
	}
	var kept []task.Task
	for _, t := range ts {
		if matchesFilter(t, filter, ts) {
			kept = append(kept, t)
		}
	}
	user := name
	if email != "" {
		user = "" // the TUI's person: found by email, named by people
	}
	rows := []AskedRow{}
	for _, q := range tui.Asked(kept, people, user, email) {
		rows = append(rows, AskedRow{ID: q.ID, Title: q.Title, State: q.Status, N: q.N, Check: q.Check, Question: q.Text, By: q.By, To: name, At: q.At, Since: tui.SinceText(q.At)})
	}
	reply := Reply{"asked": rows, "asked_for": name}
	if len(rows) == 0 {
		reply["message"] = "no open questions for " + name
		reply["fix"] = "hug task ls --asked --to NAME"
		reply["next"] = "hug task next"
		// An exit code puts the fix after next, so the list ends with it;
		// none waiting is no failure: 0.
		reply["exit_code"] = 0
		return reply, nil
	}
	reply["next"] = answerHint(rows[0])
	// The TUI's '? N for you' counts tasks; a task may ask two things.
	tasks := map[string]bool{}
	for _, r := range rows {
		tasks[r.ID] = true
	}
	reply["message"] = fmt.Sprintf("%s for %s, on %s", plural(len(rows), "open question", "open questions"), name, plural(len(tasks), "task", "tasks"))
	return reply, nil
}

// answerHint is the command that answers r's question.
func answerHint(r AskedRow) string {
	if r.N > 0 {
		return fmt.Sprintf("hug task answer %s %d \"the answer\"", r.ID, r.N)
	}
	return fmt.Sprintf("hug task answer %s \"the answer\"", r.ID)
}

// writeAsked prints ls --asked: id, check ("-" for none), the question's
// first line cut to the width, who asked, since when.
func writeAsked(w io.Writer, rows []AskedRow, width int) {
	by := 0
	for _, r := range rows {
		by = min(20, max(by, utf8.RuneCountInString(r.By)))
	}
	for _, r := range rows {
		check := "-"
		if r.Check > 0 {
			check = fmt.Sprint(r.Check)
		}
		first, _, _ := strings.Cut(strings.TrimSpace(r.Question), "\n")
		first = oneLineText(first)
		since := r.Since
		if since == "" {
			since = "-"
		}
		// id(6) 2 check(3) 2 question 2 by 2 since: the question takes
		// what the rest leaves, at least 20.
		room := max(20, width-6-2-3-2-2-by-2-utf8.RuneCountInString(since))
		fmt.Fprintf(w, "%-6s  %-3s  %-*s  %-*s  %s\n", r.ID, check, room, clip(first, room), by, clip(oneLineText(r.By), by), since)
	}
}
