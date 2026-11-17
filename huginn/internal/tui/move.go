package tui

import (
	"fmt"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// sectionName names the rows a grab moves within, as the screen titles them:
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
