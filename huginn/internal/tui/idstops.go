package tui

import (
	"regexp"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// In a task's detail, Tab and l (Shift-Tab and h) step one focus ring in
// reading order, every task id the detail shows a stop of its own (4bwcgp,
// e3sevr: focusRing), and Enter opens the one marked; Esc or u comes back
// (Jørgen 19:43: "there is no way to tab to the dependency" in jr55s8's
// "waiting on after 46ek1f"). The ids are marked where they stand:
// the parent badge and the stage line in the header, the Dependencies,
// Children and Links rows, and an id named in the Notes (crab 19:50). An
// id or alias that resolves to no task is not a stop.

// Where an id stop stands.
const (
	stopParent   = "parent"
	stopStage    = "stage"
	stopDeps     = "deps"
	stopChildren = "children"
	stopLinks    = "links"
	stopNotes    = "notes"
)

// idStop is one task id the detail shows: where it stands, its place in
// that list (deps, children, links), the text shown (an id or an alias)
// and the task it opens.
type idStop struct {
	where, shown string
	index        int
	target       task.Task
}

// idWord is a word that may name a task: an id or an alias such as
// STATBUS-402. Only words that resolve exactly count (no prefix, no slug).
var idWord = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9_-]*[A-Za-z0-9]`)

// idsIn are the tasks text names by id or alias, in order, each once,
// never t itself or one in skip.
func (m model) idsIn(t task.Task, text string, skip map[string]bool) []idStop {
	var out []idStop
	for _, w := range idWord.FindAllString(text, -1) {
		if len(w) < 3 || len(w) > 40 {
			continue
		}
		target, err := task.ResolveDependency(m.all, w)
		if err != nil || strings.EqualFold(target.ID, t.ID) || skip[strings.ToLower(target.ID)] {
			continue
		}
		skip[strings.ToLower(target.ID)] = true
		out = append(out, idStop{shown: w, target: target})
	}
	return out
}

// children are the tasks whose parent is t, in board order.
func (m model) children(t task.Task) []task.Task {
	var out []task.Task
	for _, c := range m.all {
		if c.Parent != "" && strings.EqualFold(task.BareRef(c.Parent), t.ID) {
			out = append(out, c)
		}
	}
	return out
}

// stageWords is the stage line's text where it may name a task: what a
// wait, a question or a deferral says. A depends line lists dependencies,
// which have their own stops.
func (m model) stageWords(t task.Task) string {
	switch m.stageOf(t) {
	case "waiting", "asked", "design", "review", "deferred":
		_, text, _ := m.note(t)
		return text
	}
	return ""
}

// idStops are the task ids t's detail shows, in reading order.
func (m model) idStops(t task.Task) []idStop {
	var out []idStop
	if p := strings.TrimSpace(t.Parent); p != "" {
		if target, err := task.ResolveDependency(m.all, p); err == nil {
			out = append(out, idStop{where: stopParent, shown: target.ID, target: target})
		}
	}
	// The stage's line names ids; asked of you it is not drawn (0gztsy).
	if _, mine := m.askedOf(t); !mine {
		for _, s := range m.idsIn(t, m.stageWords(t), map[string]bool{}) {
			s.where = stopStage
			out = append(out, s)
		}
	}
	for i, d := range task.DependencyDetails(t, m.all) {
		if d.Missing {
			continue
		}
		if target, err := task.Resolve(m.all, d.ID); err == nil {
			out = append(out, idStop{where: stopDeps, index: i, shown: target.ID, target: target})
		}
	}
	for i, c := range m.children(t) {
		out = append(out, idStop{where: stopChildren, index: i, shown: c.ID, target: c})
	}
	linked := map[string]bool{}
	for i, l := range m.links(t) {
		linked[strings.ToLower(l.ID)] = true
		if l.Missing {
			continue
		}
		if target, ok := m.taskAt(l.Path); ok {
			out = append(out, idStop{where: stopLinks, index: i, shown: l.ID, target: target})
		}
	}
	// A task:ID in the Notes is already a link; a bare id is a stop of its
	// own, marked in the text.
	notes := ""
	for _, s := range m.sections(t) {
		if s.name == "notes" {
			notes += s.text + "\n"
		}
	}
	for _, s := range m.idsIn(t, notes, linked) {
		s.where = stopNotes
		out = append(out, s)
	}
	return out
}

// idAtFocus is the stop the detail marks now, or -1: a dependency or a
// link marked > is the stop it shows; with focusIDs, m.idAt.
func (m model) idAtFocus(stops []idStop) int {
	for k, s := range stops {
		switch {
		case m.detailPane == focusIDs && k == m.idAt:
			return k
		case m.detailPane == focusDeps && s.where == stopDeps && s.index == m.dependency:
			return k
		case m.detailPane == focusLinks && s.where == stopLinks && s.index == m.link:
			return k
		}
	}
	return -1
}

// markedStop is the stop marked in the header or the text (parent,
// stage, children, notes), if one is.
func (m model) markedStop(t task.Task) (idStop, bool) {
	if m.detailPane != focusIDs {
		return idStop{}, false
	}
	stops := m.idStops(t)
	if m.idAt < 0 || m.idAt >= len(stops) {
		return idStop{}, false
	}
	return stops[m.idAt], true
}

// followID opens the task id marked in the header or the text; Esc comes
// back, as after following a dependency.
func (m *model) followID() {
	t, ok := m.selected()
	if !ok {
		return
	}
	s, ok := m.markedStop(t)
	if !ok {
		m.message = "no task id marked: Tab or l steps to the next one"
		return
	}
	m.detailStack = append(m.detailStack, t.Path)
	m.detailPath = s.target.Path
	m.startFocus()
	m.syncDetail()
	m.openTop()
}

// markWord draws the first whole word w in s marked: ▸ before it and the
// selection tint, so it shows without colour too. rest styles the text
// around it.
func (m model) markWord(s, w string, rest func(string) string) string {
	at := -1
	for from := 0; from < len(s); {
		i := strings.Index(s[from:], w)
		if i < 0 {
			break
		}
		i += from
		end := i + len(w)
		if (i == 0 || !isWordByte(s[i-1])) && (end == len(s) || !isWordByte(s[end])) {
			at = i
			break
		}
		from = i + 1
	}
	if at < 0 {
		return rest(s)
	}
	return rest(s[:at]) + m.styles.selected.Render("▸"+w) + rest(s[at+len(w):])
}

func isWordByte(b byte) bool {
	return b == '_' || b == '-' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
