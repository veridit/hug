package tui

import (
	"sort"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// Tags on rows (txjkrm checks 3 and 6; 911j70, Jørgen 2026-11-20 08:26Z):
// a dim prefix of the TITLE cell, "design · Design: imported data pass",
// so they say at a glance what a task is about and take no column width
// of their own. At most tagMax columns (titleCell).

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
