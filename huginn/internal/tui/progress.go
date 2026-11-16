package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/huginn/internal/task"
)

// Acceptance progress on rows (d8yx7y): a pie that fills by quarters and
// N/M before the id, e.g. "◑ 2/4". All ticked is green, some yellow, none
// dim; the glyph and text stay without colour (NO_COLOR).

// pies are the glyphs for 0, 1, 2, 3 and 4 quarters.
var pies = []string{"○", "◔", "◑", "◕", "●"}

// pie is the glyph for done of total: the quarters done covers, rounded
// down, except that any tick shows at least one quarter and only all ticked
// shows a full circle.
func pie(done, total int) string {
	if total <= 0 || done <= 0 {
		return pies[0]
	}
	if done >= total {
		return pies[4]
	}
	q := done * 4 / total
	return pies[max(1, min(3, q))]
}

// progressWidth is the progress column: glyph, space, the widest N/M among
// all tasks, space; 0 when no task has checks. Typical boards ("2/4") spend
// 6 columns, so narrow board columns keep their titles.
func (m model) progressWidth() int {
	w := 0
	for _, t := range m.all {
		if done, total := checkCount(t); total > 0 {
			w = max(w, len(fmt.Sprintf("%d/%d", done, total)))
		}
	}
	if w == 0 {
		return 0
	}
	return w + 3
}

// checkCount is t's ticked and total acceptance checks.
func checkCount(t task.Task) (done, total int) {
	for _, a := range t.Acceptance {
		if a.Checked {
			done++
		}
	}
	return done, len(t.Acceptance)
}

// progress is t's progress cell, progressWidth wide: blank for a task
// without checks.
func (m model) progress(t task.Task) string {
	width := m.progressWidth()
	done, total := checkCount(t)
	if total == 0 {
		return fit("", width)
	}
	style := m.styles.dim
	switch {
	case done >= total:
		style = m.styles.status["done"]
	case done > 0:
		style = m.styles.progress
	}
	cell := fmt.Sprintf("%s %d/%d", pie(done, total), done, total)
	return style.Render(fit(cell, width-1)) + " "
}

// progressStyle is the yellow for a task partly ticked.
func progressStyle(color func(dark, light string) lipgloss.Style) lipgloss.Style {
	return color("#e5c07b", "#8a6d00")
}
