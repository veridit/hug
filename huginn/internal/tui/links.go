package tui

import (
	"regexp"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// taskRef is a reference in running text: task:ID, the form notes, goals
// and checks use (xxvm6t). The id is six letters or digits.
var taskRef = regexp.MustCompile(`(?i)\btask:([0-9a-z]{6})\b`)

// linkDetail is one task a task's text refers to.
type linkDetail struct {
	ID, Title, State, Path string
	Missing                bool
}

// links are the tasks t's text refers to as task:ID, in the order they
// first appear in the file. The Depends on section is left out
// (Dependencies lists it), so are History and a reference to t itself.
func (m model) links(t task.Task) []linkDetail { return m.facts(t).links }

// linksOf are the links of t's sections (pageMemo keeps them per file).
func (m model) linksOf(t task.Task, sections []detailSection) []linkDetail {
	var text []string
	for _, s := range sections {
		if s.name != "dependencies" && s.name != "history" {
			text = append(text, s.text)
		}
	}
	var out []linkDetail
	seen := map[string]bool{strings.ToLower(t.ID): true}
	for _, match := range taskRef.FindAllStringSubmatch(strings.Join(text, "\n"), -1) {
		id := strings.ToLower(match[1])
		if seen[id] {
			continue
		}
		seen[id] = true
		l := linkDetail{ID: id, Missing: true}
		if target, err := task.ResolveDependency(m.all, id); err == nil {
			l = linkDetail{ID: target.ID, Title: target.Title, State: target.State, Path: target.Path}
		}
		out = append(out, l)
	}
	return out
}

// followLink opens the linked task marked >; Esc comes back, as after
// following a dependency.
func (m *model) followLink() {
	t, ok := m.selected()
	if !ok {
		return
	}
	links := m.links(t)
	if len(links) == 0 {
		m.message = "no task:ID in this task to follow"
		return
	}
	l := links[min(m.link, len(links)-1)]
	if l.Missing {
		m.message = "no task has the id " + l.ID + ": hug task ls"
		return
	}
	m.detailStack = append(m.detailStack, t.Path)
	m.detailPath = l.Path
	m.startFocus()
	m.syncDetail()
	m.openTop()
}
