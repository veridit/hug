package cli

import (
	"context"
	"fmt"
	"io"
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
	}
	return fs, nil
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
	if sum.mode != "all" && len(groups) > lintBound {
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
		fmt.Fprintf(w, "%d of %d tasks shown; every one: hug task lint --all; one kind: hug task lint --code CODE\n", len(shown), len(groups))
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
