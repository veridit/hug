package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/huginn/internal/task"
)

// Acceptance progress on rows (d8yx7y): a pie that fills by quarters and
// N/M before the id, e.g. "◑ 2/4". Some ticked is yellow, done green,
// none dim; the glyph and text stay without colour (NO_COLOR). The
// fraction is for partial work only (zhvkmj): see progress.

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
	if m.stages != nil {
		return m.progressCols // the snapshot's (46ek1f)
	}
	return progressWidthOf(m.all)
}

// progressWidthOf is progressWidth for tasks.
func progressWidthOf(tasks []task.Task) int {
	w := 0
	for _, t := range tasks {
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
// without checks. The fraction shows only while it says something
// (zhvkmj, Jørgen 2026-11-17): a done task has every check ticked, so it
// shows the count, "● 5", in the done colour; a task with nothing ticked
// shows its count dim, "○ 3"; one with some ticked keeps "◑ 2/4", which
// marks resumable work. A dropped task shows none.
func (m model) progress(t task.Task) string {
	width := m.progressWidth()
	done, total := checkCount(t)
	if total == 0 || t.State == "dropped" {
		return fit("", width)
	}
	style := m.styles.dim
	switch {
	case t.State == "done" || done >= total:
		style = m.styles.status["done"]
	case done > 0:
		style = m.styles.progress
	}
	return style.Render(fit(progressText(t), width-1)) + " "
}

// progressCell is progress without the space after it: the ✓ cell of a
// row, width wide.
func (m model) progressCell(t task.Task, width int) string {
	done, total := checkCount(t)
	if total == 0 || t.State == "dropped" {
		return fit("", width)
	}
	style := m.styles.dim
	switch {
	case t.State == "done" || done >= total:
		style = m.styles.status["done"]
	case done > 0:
		style = m.styles.progress
	}
	return style.Render(fit(progressText(t), width))
}

// progressText is the text of t's progress cell: "● 5" done, "○ 3" none
// ticked, "◑ 2/4" partly ticked (zhvkmj).
func progressText(t task.Task) string {
	done, total := checkCount(t)
	switch {
	case total == 0 || t.State == "dropped":
		return ""
	case t.State == "done":
		return fmt.Sprintf("%s %d", pies[4], total)
	case done == 0:
		return fmt.Sprintf("%s %d", pies[0], total)
	}
	return fmt.Sprintf("%s %d/%d", pie(done, total), done, total)
}

// progressStyle is the yellow for a task partly ticked.
func progressStyle(color func(dark, light string) lipgloss.Style) lipgloss.Style {
	return color("#e5c07b", "#8a6d00")
}
