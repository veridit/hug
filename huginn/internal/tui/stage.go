package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// Rows show the stage, not the state (6ephqk check 5, Jørgen 10:28 and
// 10:56): the section header is the state, so a row reads pie, id, type,
// priority, stage, then the text. A stage with extra data shows
// "stage: reason" in place of the title (waiting: "CI run N · 12 min");
// a bare stage (active, ready) shows the title. depends keeps the title
// behind a small bar of dependencies done (Jørgen 11:44; the look is
// still open, so it is one function: dependsCell).

// stageCell is the stage column's width with the space after it: the
// longest stage word and its colon, "deferred:".
const stageCell = 10

// now is the clock "· 12 min" counts from; tests set it.
var now = time.Now

// rowStage is t's stage as its row shows it: the word for the stage
// column ("waiting:" when the text is the reason, "active" when it is the
// title) and the text, or "" for the title. Derived from the fields and
// the board (task.Stage), so a board hug upgrade has not migrated yet
// shows the same stages.
func (m model) rowStage(t task.Task) (word, text string) {
	g := task.Stage(t, m.all)
	if g == "" {
		return "", ""
	}
	if (g == "ready" || g == "active" || g == "draft") && m.blocked[t.Path] {
		// A condition that does not hold yet ("when load below 120",
		// 16dxsr) holds the task back as a dependency does: next skips it,
		// so the row says depends, not ready. The written stage follows
		// dependencies only, so a load or a date does not churn files.
		g = "depends"
	}
	f := func(k string) string { return oneLine(t.Fields[k]) }
	reason := ""
	switch g {
	case "waiting":
		reason = f("waiting_on") + since(t.Fields["waiting_since"])
	case "design":
		reason = f("design_question")
	case "deferred":
		reason = f("deferred_reason")
		if u := f("deferred_until"); u != "" {
			reason += " · until " + u
		}
	case "review":
		if who := f("review_by"); who != "" {
			if t.Status == "doing" {
				reason = who + " reviewing" + since(t.Fields["review_since"])
			} else {
				reason = "review by " + who
			}
		}
	case "depends":
		return g, m.dependsCell(t)
	}
	if reason == "" {
		return g, ""
	}
	return g + ":", reason
}

// dependsCell is a depends row's text: a bar of how many dependencies are
// done, then the title ("▰▰▱ 2/3  A third-party operator installs…").
// Jørgen 11:44: the exact look is still under discussion; this is the one
// place to change it.
func (m model) dependsCell(t task.Task) string {
	done, total := 0, len(t.Dependencies)
	for _, d := range task.DependencyDetails(t, m.all) {
		if !d.Blocking {
			done++
		}
	}
	// Conditions count as dependencies: one cell each, filled when it holds.
	for _, w := range task.Waits(t, m.all) {
		total++
		if w.Met {
			done++
		}
	}
	return dependsBar(done, total) + "  " + oneLine(t.Title)
}

// dependsBar draws done of total as filled and empty cells, at most 5
// cells, then the count.
func dependsBar(done, total int) string {
	if total <= 0 {
		return ""
	}
	cells := min(total, 5)
	filled := done * cells / total
	return strings.Repeat("▰", filled) + strings.Repeat("▱", cells-filled) + fmt.Sprintf(" %d/%d", done, total)
}

// since is " · 12 min" from a stamp ("2006-01-02T15:04"), or "" when it
// does not read.
func since(stamp string) string {
	at, e := time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(stamp), time.Local)
	if e != nil {
		return ""
	}
	d := now().Sub(at)
	switch {
	case d < 0:
		return ""
	case d < time.Hour:
		return fmt.Sprintf(" · %d min", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf(" · %d h", int(d.Hours()))
	}
	return fmt.Sprintf(" · %d d", int(d.Hours()/24))
}

// stageStyle colours the stage word: depends in the colour BLOCKED had,
// waiting and design in the warning colour (someone waits), the rest dim.
func (m model) stageWord(word string, width int) string {
	style := m.styles.dim
	switch strings.TrimSuffix(word, ":") {
	case "depends":
		style = m.styles.status["depends"]
	case "waiting", "design", "review":
		style = m.styles.warning
	case "active":
		style = m.styles.status["doing"]
	case "ready":
		style = m.styles.status["todo"]
	}
	return style.Render(fit(word, width))
}
