package task

import (
	"sort"
	"strings"
)

// CodeInfo is one kind of lint finding: its stable code and what it means
// in plain words, for lint summaries and the code list (6neax1).
type CodeInfo struct {
	Code    string `json:"code"`
	Meaning string `json:"meaning"`
	// Severity is Broken or Advice (pbh9xq), see Severity.
	Severity string `json:"severity"`
}

// Severities of a finding (pbh9xq, Jørgen 2026-11-16). Broken: hug cannot
// read or trust the task (a link to nothing, a cycle, a done task with an
// open check): always shown, closed task or not. Advice: the task works but
// is below the house style (title length, sections, acceptance): shown on
// open tasks only, where someone can still act on it.
const (
	Broken = "broken"
	Advice = "advice"
)

// advice are the codes Codes marks Advice.
var advice = map[string]bool{}

// Severity is the severity of a finding code, as Codes says; a code not
// there is Broken, so it is shown until someone decides it is only style.
func Severity(code string) string {
	if advice[code] {
		return Advice
	}
	return Broken
}

// Codes lists every finding code FindingCode can return, in a stable order,
// with its severity: this table is the split (pbh9xq).
var Codes = []CodeInfo{
	{"frontmatter", "the --- header is missing or malformed", Broken},
	{"duplicate-section", "a section heading appears twice", Broken},
	{"duplicate-field", "a header field appears twice", Broken},
	{"missing-field", "id, title or state is missing", Broken},
	{"old-layout", "the file says status: where state: goes, or waiting_for: where a question goes, or has both ## Details and ## Relevant details (a board from before the state: rename or the asked stage, or an older hug's set --details; hug upgrade moves or merges it)", Broken},
	{"invalid-id", "the id is not 6 characters of 0-9 a-z", Broken},
	{"id-mismatch", "the file name id differs from the header id", Broken},
	{"unknown-state", "state is not todo, doing, done or dropped", Broken},
	{"unknown-stage", "the stage is not a stage of the task's state (hug task states)", Broken},
	{"stray-node-field", "a node field (doing_since, waiting_on ...) outside its node", Broken},
	{"missing-node-field", "the state or stage needs a field the task lacks (waiting_on ...)", Broken},
	{"stage-disagrees", "a stage hug sets (draft, depends, ready) does not match the task", Broken},
	{"unknown-type", "type is not feature, bug, chore, spike or docs", Broken},
	{"invalid-field", "a header field has an invalid value", Broken},
	{"missing-goal", "no Goal section, or it is empty", Broken},
	{"missing-section", "an open task lacks Purpose or Principle", Advice},
	{"section-order", "sections are not in Goal, Purpose, Principle, ... order", Advice},
	{"no-acceptance", "no acceptance item (- [ ] ...)", Advice},
	{"filename-collision", "two file names differ only by case", Broken},
	{"duplicate-id", "two tasks have the same id", Broken},
	{"dangling-dependency", "depends on an id no task has", Broken},
	{"not-a-dependency", "a Depends on line that is not a dependency", Broken},
	{"dependency-cycle", "tasks depend on each other in a loop", Broken},
	{"missing-parent", "the epic (parent) does not exist", Broken},
	{"self-parent", "a task is its own parent", Broken},
	{"parent-too-deep", "the parent has a parent itself", Broken},
	{"long-title", "the title is longer than 80 characters", Advice},
	{"goal-too-long", "the Goal is more than one sentence or longer than 140 characters", Advice},
	{"forbidden-citation", "cites a folder that gets deleted", Broken},
	{"file-symlink", "a task file is a symlink", Broken},
	{"duplicate-origin", "two tasks were imported from one source", Broken},
	{"unknown-condition", "a Depends on condition hug cannot check (unreadable, or not checkable here): the task waits", Broken},
	{"branch-gone", "a doing task's branch is merged into main or deleted: the work landed or was dropped", Broken},
	{"unlisted-tag", "a task has a tag hug.toml tags does not list", Advice},
	{"unknown-person", "asked_for, design_for or review_by names nobody in hug.toml [people] and no agent on the board", Broken},
	{"wait-on-person", "a doing task waits on a question to a person ('owner: ...', a [people] name): nobody sees it as theirs; hug task ask", Broken},
	{"stale-owner", "a doing task's assignee has no live harness session: closed over an hour ago, silent over a day, or never had one", Broken},
	{"landed-unchecked", "question: a commit naming a doing task reached main, but no check is ticked", Broken},
	{"shipped-not-done", "question: a release contains the task, but it is not done", Broken},
	{"checks-all-hold", "question: a waiting task has every check and child done, but it is not done", Broken},
	{"todo-has-commit", "question: a todo task has a commit on main", Broken},
	{"doing-idle", "question: a doing task has had no note, check or commit for 2 hours", Broken},
	{"backlog-leftover", "Backlog.md markers or unused imported fields; hug upgrade removes them", Advice},
	{"missing-task-ref", "a task: link names no task", Broken},
	{"missing-file", "a file: link names no file", Broken},
	{"file-content", "an attached file is binary or too large", Broken},
	{"other", "anything else", Broken},
}

// CodeCount is a code with its plain meaning and how many findings have it.
type CodeCount struct {
	Code     string `json:"code"`
	Count    int    `json:"count"`
	Meaning  string `json:"meaning"`
	Severity string `json:"severity"`
	// List is the command that lists the findings of this code.
	List string `json:"list"`
}

// CodeNames are the codes in Codes order.
func init() {
	for _, c := range Codes {
		if c.Severity == Advice {
			advice[c.Code] = true
		}
	}
}

func CodeNames() []string {
	out := make([]string, len(Codes))
	for i, c := range Codes {
		out[i] = c.Code
	}
	return out
}

// Summarize counts findings by code: broken codes first (pbh9xq), each
// severity most frequent first (then by code). With all set, codes without
// findings are listed too, after the others.
func Summarize(fs []Finding, all bool) []CodeCount {
	n := map[string]int{}
	for _, f := range fs {
		n[FindingCode(f.Problem)]++
	}
	var out []CodeCount
	for _, c := range Codes {
		if n[c.Code] > 0 || all {
			out = append(out, CodeCount{Code: c.Code, Count: n[c.Code], Meaning: c.Meaning, Severity: c.Severity, List: "hug task lint --code " + c.Code})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if zi, zj := out[i].Count == 0, out[j].Count == 0; zi != zj {
			return zj
		}
		if bi, bj := out[i].Severity == Broken, out[j].Severity == Broken; bi != bj && out[i].Count > 0 {
			return bi
		}
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].Count == 0 {
			return false // zero rows keep Codes order
		}
		return out[i].Code < out[j].Code
	})
	return out
}

// OpenFindings splits findings into those on open tasks (todo, doing) and
// those on closed ones (done, dropped). A finding belongs to the task whose
// file it is, or whose folder (group/ID/...) holds the file; a finding that
// belongs to no known task (a file hug cannot read as one) is open, since
// nothing says it is finished.
func OpenFindings(fs []Finding, tasks []Task) (open, closed []Finding) {
	closedAt := closedPaths(tasks)
	for _, f := range fs {
		if closedAt(f.Path) {
			closed = append(closed, f)
		} else {
			open = append(open, f)
		}
	}
	return open, closed
}

// Judged is what lint, MCP and the TUI report: the findings on open tasks,
// broken first (pbh9xq). Done and dropped tasks are history: lint judges
// open tasks only and counts nothing else (t01kqm, Jørgen 2026-11-18: "I
// don't think linting should show the done tasks"). A task is checked when
// it closes: done refuses an open check.
func Judged(fs []Finding, tasks []Task) []Finding {
	open, _ := OpenFindings(fs, tasks)
	var broken, advice []Finding
	for _, f := range open {
		if f.Severity() == Broken {
			broken = append(broken, f)
		} else {
			advice = append(advice, f)
		}
	}
	return append(broken, advice...)
}

// closedPaths reports whether a finding's path belongs to a done or dropped
// task: its file, or a file in its folder (group/ID/...). A path no task
// owns (a broken file) is open, since nothing says it is finished.
func closedPaths(tasks []Task) func(path string) bool {
	status := map[string]string{}
	for _, t := range tasks {
		status[t.Path] = t.State
		status[FolderOf(t)+"/"] = t.State
	}
	return func(path string) bool {
		s, ok := status[path]
		if !ok {
			for dir := path; ; {
				i := strings.LastIndex(dir, "/")
				if i < 0 {
					break
				}
				dir = dir[:i]
				if v, hit := status[dir+"/"]; hit {
					s, ok = v, true
					break
				}
			}
		}
		return ok && (s == "done" || s == "dropped")
	}
}
