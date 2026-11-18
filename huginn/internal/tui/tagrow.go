package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/huginn/internal/task"
)

// Tags on rows (txjkrm checks 3 and 6, Jørgen 19:35Z): right after the
// id, because they say at a glance what a task is about: pie, id, tags,
// type, priority, stage, text. Dim, at most tagMax columns, one cell wide
// for the whole board so the columns after it line up.

// tagText is t's tags as its row shows them: the ones hug.toml lists, in
// the list's order (the board's order of areas, as g groups them), then
// the others in the task's own order; "" with none.
func (m model) tagText(t task.Task) string {
	if len(t.Labels) == 0 {
		return ""
	}
	tags := make([]string, 0, len(t.Labels))
	for _, tg := range t.Labels {
		if tg = oneLine(tg); tg != "" {
			tags = append(tags, tg)
		}
	}
	sort.SliceStable(tags, func(i, j int) bool {
		a, okA := m.tagOrder[tags[i]]
		b, okB := m.tagOrder[tags[j]]
		if okA != okB {
			return okA
		}
		return okA && a < b
	})
	return strings.Join(tags, " ")
}

// tagWidth is the tags cell's width before the row is laid out: the
// widest tag text on the board, at most tagMax; 0 on a board without tags,
// whose rows read as before.
func (m model) tagWidth() int {
	if m.stages != nil {
		return m.tagCols // the snapshot's, like progressCols (46ek1f)
	}
	return m.tagWidthOf(m.all)
}

// tagWidthOf is tagWidth for tasks.
func (m model) tagWidthOf(tasks []task.Task) int {
	w := 0
	for _, t := range tasks {
		if len(t.Labels) > 0 {
			w = max(w, lipgloss.Width(m.tagText(t)), len("TAGS")) // the header names the cell whole
			if w >= tagMax {
				return tagMax
			}
		}
	}
	return w
}
