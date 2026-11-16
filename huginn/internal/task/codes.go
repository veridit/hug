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
}

// Codes lists every finding code FindingCode can return, in a stable order.
var Codes = []CodeInfo{
	{"frontmatter", "the --- header is missing or malformed"},
	{"duplicate-section", "a section heading appears twice"},
	{"duplicate-field", "a header field appears twice"},
	{"missing-field", "id, title or status is missing"},
	{"invalid-id", "the id is not 6 characters of 0-9 a-z"},
	{"id-mismatch", "the file name id differs from the header id"},
	{"unknown-status", "status is not todo, doing, done or dropped"},
	{"unknown-type", "type is not feature, bug, chore, spike or docs"},
	{"invalid-field", "a header field has an invalid value"},
	{"missing-goal", "no Goal section, or it is empty"},
	{"missing-section", "a new task lacks Purpose or Principle"},
	{"no-acceptance", "no acceptance item (- [ ] ...)"},
	{"done-unchecked", "done, but an acceptance item is not ticked"},
	{"filename-collision", "two file names differ only by case"},
	{"duplicate-id", "two tasks have the same id"},
	{"dangling-dependency", "depends on an id no task has"},
	{"not-a-dependency", "a Depends on line that is not a dependency"},
	{"dependency-cycle", "tasks depend on each other in a loop"},
	{"missing-parent", "the epic (parent) does not exist"},
	{"self-parent", "a task is its own parent"},
	{"parent-too-deep", "the parent has a parent itself"},
	{"long-title", "the title is longer than 80 characters"},
	{"forbidden-citation", "cites a folder that gets deleted"},
	{"file-symlink", "a task file is a symlink"},
	{"duplicate-origin", "two tasks were imported from one source"},
	{"branch-gone", "a doing task's branch is merged into main or deleted: the work landed or was dropped"},
	{"missing-task-ref", "a task: link names no task"},
	{"missing-file", "a file: link names no file"},
	{"file-content", "an attached file is binary or too large"},
	{"other", "anything else"},
}

// CodeCount is a code with its plain meaning and how many findings have it.
type CodeCount struct {
	Code    string `json:"code"`
	Count   int    `json:"count"`
	Meaning string `json:"meaning"`
	// List is the command that lists the findings of this code.
	List string `json:"list"`
}

// CodeNames are the codes in Codes order.
func CodeNames() []string {
	out := make([]string, len(Codes))
	for i, c := range Codes {
		out[i] = c.Code
	}
	return out
}

// Summarize counts findings by code, most frequent first (then by code).
// With all set, codes without findings are listed too, after the others.
func Summarize(fs []Finding, all bool) []CodeCount {
	n := map[string]int{}
	for _, f := range fs {
		n[FindingCode(f.Problem)]++
	}
	var out []CodeCount
	for _, c := range Codes {
		if n[c.Code] > 0 || all {
			out = append(out, CodeCount{Code: c.Code, Count: n[c.Code], Meaning: c.Meaning, List: "hug task lint --code " + c.Code})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
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

// OpenFindings splits findings into those on open tasks (todo, doing,
// blocked) and those on closed ones (done, dropped). A finding belongs to
// the task whose file it is, or whose folder (group/ID/...) holds the file;
// a finding that belongs to no known task (a broken file) is open, since
// nothing says it is finished. Lint, MCP and the TUI hide closed findings
// by default (6neax1, Jørgen: 986 of 2,477 seid findings were on closed
// tasks).
func OpenFindings(fs []Finding, tasks []Task) (open, closed []Finding) {
	status := map[string]string{}
	for _, t := range tasks {
		status[t.Path] = t.Status
		status[FolderOf(t)+"/"] = t.Status
	}
	for _, f := range fs {
		s, ok := status[f.Path]
		if !ok {
			for dir := f.Path; ; {
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
		if ok && (s == "done" || s == "dropped") {
			closed = append(closed, f)
		} else {
			open = append(open, f)
		}
	}
	return open, closed
}
