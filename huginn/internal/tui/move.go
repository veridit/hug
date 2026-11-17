package tui

import (
	"fmt"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// moveTarget is the row J (below) or K (above) moves t past: the next or
// previous row shown in the current section, after filters. A task that is
// not shown there (a filtered-out task, an epic missing from NEXT UP, a task
// of another status column) is never the target, so every move changes what
// the reader sees (ksv323). why says what stopped the move, if anything.
func (m model) moveTarget(t task.Task, k string) (task.Task, string) {
	rows := m.indices() // in a detail, the list it was opened from
	at := -1
	for p, i := range rows {
		if m.visible[i].Path == t.Path {
			at = p
			break
		}
	}
	section := m.sectionName()
	switch {
	case at < 0:
		return task.Task{}, fmt.Sprintf("%s is not shown in %s, so there is no row to move it past", t.ID, section)
	case k == "J" && at+1 >= len(rows):
		return task.Task{}, fmt.Sprintf("%s is already the last row in %s; nothing moved", t.ID, section)
	case k == "K" && at == 0:
		return task.Task{}, fmt.Sprintf("%s is already the first row in %s; nothing moved", t.ID, section)
	case k == "J":
		return m.visible[rows[at+1]], ""
	default:
		return m.visible[rows[at-1]], ""
	}
}

// sectionName names the rows J and K move within, as the screen titles them:
// "NEXT UP", "the TODO column", "the list".
func (m model) sectionName() string {
	name := "the list"
	switch m.view {
	case "overview":
		title, _, _ := strings.Cut(overviewSections[min(max(m.pane, 0), len(overviewSections)-1)], " · ")
		name = title
	case "board":
		name = "the " + strings.ToUpper(boardStates[min(max(m.pane, 0), len(boardStates)-1)]) + " column"
	case "lint":
		name = "the lint list"
	}
	if m.query != "" || m.filter != (task.Filter{}) {
		name += " (filtered)"
	}
	return name
}

// rowPlace is where the task at path now sits among the rows shown:
// ", row 3 of 7 in NEXT UP". Empty when it is not shown.
func (m model) rowPlace(path string) string {
	rows := m.indices()
	for p, i := range rows {
		if m.visible[i].Path == path {
			return fmt.Sprintf(", row %d of %d in %s", p+1, len(rows), m.sectionName())
		}
	}
	return ""
}

// taskAt is the task at path, if it still exists.
func (m model) taskAt(path string) (task.Task, bool) {
	for _, t := range m.all {
		if t.Path == path {
			return t, true
		}
	}
	return task.Task{}, false
}
