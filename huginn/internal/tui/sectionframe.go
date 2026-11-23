package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// The section frame (v68n82, Jørgen 2026-11-20 12:33Z, screen-1332.ans):
// each section of the overview, the list and each board column sits
// between two rules, the frame of a task's page (4ddcwa). The top rule
// carries the section's title and count, '─── WAITING  [20] · 2–18 shown
// ───'; a rule ends it. ↑ and ↓ (rows hidden above, below) sit at the
// rules' left end, as on the page: '↑── WAITING …', '↓─────'; they left
// the rows' gutter, one cue per meaning. The focused section's rules are
// brighter, the others dim, no fill; they never pulse: the focus sign >
// does (q3sh69).

// sectionFrame is the overview's frame: "end", each section with its own
// end rule (variant A, Jørgen's sketch, the default), or "shared", the
// next section's titled rule doubling as the end of the one above and
// the last section with an end rule (variant B, v68n82 check 2), or
// "focus", only the focused section between its titled rule and its end
// rule and every other section's title on a plain line (variant C,
// hd74ct, Jørgen 2026-11-22 21:30Z: 'just showing the for the active
// section'). A string, so a build can pick B or C with -ldflags -X
// github.com/veridit/hug/huginn/internal/tui.sectionFrame=focus.
var sectionFrame = "end"

// sharedRules says the overview draws variant B.
func sharedRules() bool { return sectionFrame == "shared" }

// focusRules says the overview draws variant C: rules only around the
// focused section.
func focusRules() bool { return sectionFrame == "focus" }

// plainTitle is an unfocused section's title line in variant C: the
// title where a rule would put it, after four blank cells, no dashes,
// no arrows (its count '[19] · 1–3 shown' says rows are hidden).
func plainTitle(title string, width int) string {
	return ansi.Truncate("    "+title, width, "…")
}

// ruleLead is a rule's first four cells: the arrows, then the rule.
// up is ↑ of the section the rule opens, down ↓ of the one it closes.
func ruleLead(up, down bool) string {
	switch {
	case up && down:
		return "↓↑─"
	case up:
		return "↑──"
	case down:
		return "↓──"
	}
	return "───"
}

// ruleInk draws rule glyphs: bright on the focused section's rules,
// dim on the others. Without colour (NO_COLOR) bright is bold and dim
// faint, as the focus sign's phases (q3sh69). Steady: no pulse.
func (m model) ruleInk(s string, bright bool) string {
	if m.plainCues() {
		if bright {
			return "\x1b[1m" + s + "\x1b[0m"
		}
		return "\x1b[2m" + s + "\x1b[0m"
	}
	if bright {
		return m.styles.normal.Bold(true).Render(s)
	}
	return m.styles.dim.Render(s)
}

// sectionRule is one rule of a section's frame, width cells: lead (see
// ruleLead), then title (rendered) when there is one, then the rule to
// the edge. A tight title (a narrow board column) runs on to the edge
// and gives up the rule after it before it gives up its own text; a
// title still too long is cut, never the lead.
func (m model) sectionRule(lead, title string, width int, bright bool) string {
	if width <= 0 {
		return ""
	}
	if width < 8 || strings.TrimSpace(ansi.Strip(title)) == "" {
		return m.ruleInk(ansi.Truncate(lead+strings.Repeat("─", max(0, width-ansi.StringWidth(lead))), width, ""), bright)
	}
	title = ansi.Truncate(title, width-4, "…") // the lead and a space
	rest := width - 4 - ansi.StringWidth(title)
	tail := strings.Repeat(" ", rest)
	if rest >= 2 {
		tail = m.ruleInk(" "+strings.Repeat("─", rest-1), bright)
	}
	return m.ruleInk(lead+" ", bright) + title + tail
}

// overviewChrome is how many lines of the overview are rules: each
// section's top rule, and an end rule under each section with rows
// (variant A) or under the last section when it has rows (B), or under
// the focused section alone when it has rows (C; the others' titles are
// a plain line each). An empty section is its titled rule alone ("·
// None", h9zghj).
func overviewChrome(rows []int, focus int) int {
	lines := len(rows)
	for p, n := range rows {
		if n > 0 && endRuleUnder(p, len(rows), focus) {
			lines++
		}
	}
	return lines
}

// endRuleUnder says section p (of n, focus the focused one) with rows
// gets an end rule: in A each, in B the last, in C the focused one.
func endRuleUnder(p, n, focus int) bool {
	switch {
	case sharedRules():
		return p == n-1
	case focusRules():
		return p == focus
	}
	return true
}
