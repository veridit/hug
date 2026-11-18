package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/task"
)

// Every check reports every problem at once (fkpyg3): hug task lint is the
// board overview, one compact line per task; hug task lint ID, show, take,
// new and set give one task's findings in detail with ONE combined fix:
// line. The same block everywhere, so an agent fixes a task in one round.

// boardFindings is every finding lint knows: file content (cached),
// attached files and citations, git branch state and harness sessions
// (jcodeHome is the app's JCODE_HOME, "" for ~/.jcode).
func (s *session) boardFindings(ctx context.Context, cwd, jcodeHome string) ([]task.Finding, error) {
	fs, e := s.c.Lint(ctx)
	if e != nil {
		return nil, e
	}
	forbid, e := task.ForbidCite(s.r.Dir)
	if e != nil {
		return nil, e
	}
	more, e := s.s.LintFiles(forbid)
	if e != nil {
		return nil, e
	}
	fs = append(fs, more...)
	// Git branch state, not file content: recomputed on every lint (x1t1t1).
	if ts, e := s.s.Tasks(); e == nil {
		fs = append(fs, task.BranchFindings(cwd, gitbranch.DefaultBranch(cwd), ts)...)
		// Waits hug cannot check here (16dxsr): git and the machine, not files.
		fs = append(fs, task.WaitFindings(ts)...)
		// Who a question is for must be someone hug can find (89y0b7).
		if people, e := task.People(s.r.Dir); e == nil {
			fs = append(fs, task.PeopleFindings(people, ts)...)
		}
		// Harness session state, not file content (v3k6cp).
		fs = append(fs, task.OwnerFindings(task.JcodeLivenessAt(jcodeHome), ts, time.Now())...)
		// A doing ticket nothing has touched for 2 hours is a question to its
		// owner (at3qep), unless it already has one; then one question per
		// ticket, the most important.
		asked := map[string]bool{}
		for _, f := range fs {
			if task.QuestionCode(task.FindingCode(f.Problem)) {
				if _, id, _, err := task.ParseName(f.Path); err == nil {
					asked[id] = true
				}
			}
		}
		fs = append(fs, task.IdleFindings(ts, touched(ctx, s.r.Dir, cwd, ts), asked, time.Now())...)
		fs = task.TopQuestions(fs)
	}
	return fs, nil
}

// touched maps each ticket to its newest commit: on the task branch (files
// changed) and on main (subjects naming it), over the last day. Nil when
// no doing ticket is active or git cannot say: then nothing is asked.
func touched(ctx context.Context, board, cwd string, ts []task.Task) map[string]time.Time {
	active := false
	for _, t := range ts {
		active = active || t.State == "doing" && t.Stage == "active"
	}
	if !active {
		return nil
	}
	since := "--since=" + time.Now().Add(-24*time.Hour).Format(time.RFC3339)
	log := func(dir string, args ...string) ([]string, bool) {
		c := exec.CommandContext(ctx, "git", append([]string{"-C", dir, "log", since, "--format=%x00%ct%x09%s"}, args...)...)
		c.Env = append(c.Environ(), "GIT_OPTIONAL_LOCKS=0")
		b, err := c.Output()
		if err != nil {
			return nil, false
		}
		var out []string
		for _, rec := range strings.Split(string(b), "\x00") {
			parts := strings.Split(strings.TrimSpace(rec), "\n")
			if parts[0] == "" {
				continue
			}
			line := parts[0]
			for _, f := range parts[1:] {
				if f = strings.TrimSpace(f); f != "" {
					line += "\t" + f
				}
			}
			out = append(out, line)
		}
		return out, true
	}
	mine, ok := log(board, "--name-only", "HEAD")
	if !ok {
		return nil
	}
	if main := gitbranch.DefaultBranch(cwd); main != "" {
		if more, ok := log(cwd, main); ok {
			mine = append(mine, more...)
		}
	}
	return task.TouchedSince(mine)
}

// forTask keeps the findings on t's file or its attached files.
func forTask(fs []task.Finding, t task.Task) []task.Finding {
	dir := task.FolderOf(t) + "/"
	out := []task.Finding{}
	for _, f := range fs {
		if f.Path == t.Path || strings.HasPrefix(f.Path, dir) {
			out = append(out, f)
		}
	}
	return out
}

// readyToHand refuses to hand ref to a person (ask --to, review --by,
// design --for) while its own file has a broken finding (wdkqzx check 5,
// Jørgen 23:51: a person is never handed a broken task). The agent sees
// every broken finding and one fix hug takes. design passes draft: a
// missing Goal or acceptance does not hold a design question back, since
// the answer is often what fills them (crab 00:05). Nothing is written.
func (s *session) readyToHand(cmd, ref, person string, draft bool, check ...int) error {
	d, e := s.s.Get(ref)
	if e != nil {
		return e
	}
	l := s.taskLint(d.Task)
	if l == nil {
		return nil
	}
	var broken []task.Finding
	// Asked about the one open check of a shipped task (wdkqzx check 6,
	// 6ephqk's shape): the ask is the release question, so it is not held
	// back by it, and after the ask lint has none.
	asksOpen := len(check) == 1 && onlyOpen(d.Task, check[0])
	for _, f := range l.Findings {
		code := task.FindingCode(f.Problem)
		if f.Severity() != task.Broken || (draft && (code == "missing-goal" || code == "no-acceptance")) || (asksOpen && code == "shipped-not-done") {
			continue
		}
		broken = append(broken, f)
	}
	if len(broken) == 0 {
		return nil
	}
	parts := make([]string, len(broken))
	for i, f := range broken {
		parts[i] = "[" + task.FindingCode(f.Problem) + "] " + f.Problem
	}
	return &UsageError{
		Problem: "hug task " + cmd + ": " + d.Task.ID + " is not ready to hand to " + strings.TrimSpace(person) + ": " + plural(len(broken), "broken finding", "broken findings") + ": " + strings.Join(parts, "; "),
		Fix:     task.CombinedFix(d.Task.ID, broken),
	}
}

// onlyOpen reports whether check n (1-based) is t's one open check.
func onlyOpen(t task.Task, n int) bool {
	if n < 1 || n > len(t.Acceptance) || t.Acceptance[n-1].Settled() {
		return false
	}
	for i, a := range t.Acceptance {
		if i != n-1 && !a.Settled() {
			return false
		}
	}
	return true
}

// writesTask are the commands an agent runs that write one task and end
// with its lint block (wdkqzx check 4). done and drop close the task and
// get none; request is the person's own words, not the agent's write.
var writesTask = map[string]bool{"new": true, "set": true, "take": true, "check": true, "strike": true, "note": true,
	"detach": true, "dep": true, "wait": true, "ask": true, "resume": true, "review": true, "design": true, "answer": true, "resolve": true}

// TaskLint is one task's findings with the one combined fix, as show,
// take, new, set and MCP return it.
type TaskLint struct {
	Findings []task.Finding `json:"findings"`
	Fix      string         `json:"fix"`
}

// taskLint lints the board and keeps t's findings. A failing lint is no
// reason to fail the command that wrote: it then reports nothing. It lints
// the board this command has read (or just wrote), never a second walk and
// HEAD read (44w1ev: one walk per command; fkpyg3's lint block cost show and
// take one more git process and one more walk of every task file).
func (s *session) taskLint(t task.Task) *TaskLint {
	if _, e := s.s.Tasks(); e != nil {
		return nil
	}
	fs := s.c.LintListed()
	mine := forTask(fs, t)
	if len(mine) == 0 {
		return nil
	}
	return &TaskLint{Findings: mine, Fix: task.CombinedFix(t.ID, mine)}
}

// withIdle adds the idle question (at3qep b) to show's lint block: git, not
// file content, so one bounded log of the ticket's file, and only on show
// (a write has just touched the ticket, so it is never idle there), for a
// doing, active ticket without another question.
func (reply Reply) withIdle(s *session, t task.Task) Reply {
	if t.State != "doing" || t.Stage != "active" {
		return reply
	}
	l, _ := reply["lint"].(*TaskLint)
	if l != nil && hasQuestion(l.Findings) {
		return reply
	}
	last, ok := lastTouched(s.r.Dir, t)
	if !ok {
		return reply
	}
	idle := task.IdleFindings([]task.Task{t}, map[string]time.Time{t.ID: last}, nil, time.Now())
	if len(idle) == 0 {
		return reply
	}
	var fs []task.Finding
	if l != nil {
		fs = l.Findings
	}
	fs = append(fs, idle...)
	reply["lint"] = &TaskLint{Findings: fs, Fix: task.CombinedFix(t.ID, fs)}
	return reply
}

func hasQuestion(fs []task.Finding) bool {
	for _, f := range fs {
		if task.QuestionCode(task.FindingCode(f.Problem)) {
			return true
		}
	}
	return false
}

// lastTouched is the time of the newest commit on the task branch that
// changed t's file.
func lastTouched(board string, t task.Task) (time.Time, bool) {
	c := exec.Command("git", "-C", board, "log", "-1", "--format=%ct", "--", t.Path)
	c.Env = append(c.Environ(), "GIT_OPTIONAL_LOCKS=0")
	b, err := c.Output()
	if err != nil {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(sec, 0), true
}

// withLint puts t's lint block on a reply (show, take, new, set) and
// returns the reply.
func (reply Reply) withLint(s *session, t task.Task) Reply {
	if l := s.taskLint(t); l != nil {
		reply["lint"] = l
	}
	return reply
}

// writeTaskLint prints one task's findings in detail and the one fix.
func writeTaskLint(w io.Writer, l *TaskLint) {
	fmt.Fprintf(w, "lint: %s\n", plural(len(l.Findings), "finding", "findings"))
	for _, f := range l.Findings {
		fmt.Fprintf(w, "  line %d: [%s] %s\n", f.Line, task.FindingCode(f.Problem), f.Problem)
	}
	fmt.Fprintln(w, "fix: "+l.Fix)
}

// writeOverview prints the board overview: totals per code, then one line
// per task with its codes and the command that shows the detail.
func writeOverview(w io.Writer, fs []task.Finding, sum LintSummary) {
	groups := task.ByTask(fs)
	fmt.Fprintf(w, "%s (%d broken) on %s, in %s:\n", plural(sum.Total, "finding", "findings"), sum.Broken, plural(len(groups), "task", "tasks"), plural(len(sum.Codes), "code", "codes"))
	writeCodes(w, sum.Codes)
	shown := groups
	if len(groups) > lintBound {
		shown = groups[:lintShown]
		fmt.Fprintf(w, "first %d tasks:\n", lintShown)
	}
	wid := 0
	for _, g := range shown {
		wid = max(wid, len(label(g)))
	}
	for _, g := range shown {
		fmt.Fprintf(w, "  %-*s  %s  -> %s\n", wid, label(g), strings.Join(g.Codes, ", "), detailCommand(g))
	}
	if len(shown) < len(groups) {
		fmt.Fprintf(w, "%d of %d tasks shown; one kind: hug task lint --code CODE; every one: hug task lint --json\n", len(shown), len(groups))
	}
}

func label(g task.TaskFindings) string {
	if g.ID != "" {
		return g.ID
	}
	return g.Path
}

func detailCommand(g task.TaskFindings) string {
	if g.ID != "" {
		return "hug task lint " + g.ID
	}
	return g.Fix
}
