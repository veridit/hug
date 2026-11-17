package task

import (
	"regexp"
	"strings"

	"github.com/veridit/hug/huginn/internal/id"
)

// DependencyDetail is the id-based display/MCP form of a source dependency.
// Ref in Document.Task is the original source token, which may be an alias.
type DependencyDetail struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Note    string `json:"note,omitempty"`
	Missing bool   `json:"missing,omitempty"`
	// Blocking: the dependency still holds the task back. For an epic that
	// is while any child is open (16dxsr), whatever the epic's own status;
	// OpenChildren counts them.
	Blocking     bool `json:"blocking,omitempty"`
	OpenChildren int  `json:"open_children,omitempty"`
}

func DependencyDetails(t Task, tasks []Task) []DependencyDetail {
	out := make([]DependencyDetail, 0, len(t.Dependencies))
	for _, d := range t.Dependencies {
		target, e := ResolveDependency(tasks, d.Ref)
		detail := DependencyDetail{ID: target.ID, Title: target.Title, Status: target.Status, Path: target.Path, Line: d.Line, Note: d.Note}
		if e != nil {
			detail.ID = d.Ref
			detail.Missing = true
			detail.Blocking = true
		} else {
			detail.Blocking = !Closed(target, tasks)
			for _, i := range childPositions(tasks, target) {
				if s := tasks[i].Status; s != "done" && s != "dropped" {
					detail.OpenChildren++
				}
			}
		}
		out = append(out, detail)
	}
	return out
}

// dependencyAlias is an imported id such as TASK-123, GH-12 or TASK-7@drafts:
// letters, a hyphen, then a part with at least one digit.
var dependencyAlias = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*-[A-Za-z0-9._]*[0-9][A-Za-z0-9._]*(@[A-Za-z0-9._/-]+)?$`)

// parseDependencyLine reads one non-empty line of the Depends on section.
// A dependency line is an optional "- " or "* " bullet, one reference (a
// task id or an alias, optionally written task:ID), then an optional note.
// Anything else is prose, and ok is false: prose is reported by lint and
// never becomes a dependency, so it cannot block the task (rx74bp).
func parseDependencyLine(line string) (ref, note string, ok bool) {
	t := strings.TrimSpace(line)
	for _, bullet := range []string{"- ", "* "} {
		if strings.HasPrefix(t, bullet) {
			t = strings.TrimSpace(t[len(bullet):])
			break
		}
	}
	fields := strings.Fields(t)
	if len(fields) == 0 {
		return "", "", false
	}
	ref = fields[0]
	note = strings.TrimSpace(strings.TrimPrefix(t, ref))
	if len(ref) > 5 && strings.EqualFold(ref[:5], "task:") {
		ref = ref[5:]
	}
	if id.Valid(strings.ToLower(ref)) || dependencyAlias.MatchString(ref) {
		return ref, note, true
	}
	return "", "", false
}

// quoteLine quotes a source line for a message, cut to max runes.
func quoteLine(s string, max int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) > max {
		return "\"" + string(r[:max-1]) + "…\""
	}
	return "\"" + string(r) + "\""
}
