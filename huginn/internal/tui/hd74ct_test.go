package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// hd74ct, variant C (Jørgen 2026-11-22 21:30Z, request 1, screen-2230.ans:
// 'How about just showing the for the active section?'): only the
// focused section sits between its titled top rule and its end rule, ↑/↓
// on them as in A; every other section's title is a plain line, no
// dashes, no end rule, its count kept. Moving focus moves the rules.
// Built behind sectionFrame="focus"; A stays the default.
func TestFocusRulesVariantC(t *testing.T) {
	if sectionFrame != "end" {
		t.Fatalf("the default frame is %q, want variant A, \"end\"", sectionFrame)
	}
	t.Cleanup(func() { sectionFrame = "end" })
	for _, size := range gridSizes {
		w, h := size[0], size[1]
		sectionFrame = "focus"
		m := gridModel(t, w, h, "overview")
		for _, keys := range [][]string{nil, {"l", "l"}, {"l", "l", "l"}, {"n"}} {
			mm := press(m, keys...)
			what := fmt.Sprintf("overview C at %dx%d, keys %v, focus %s", w, h, keys, overviewSections[mm.pane])
			lines := strings.Split(ansi.Strip(mm.overviewText()), "\n")
			screen := strings.Join(lines, "\n")
			titled, ends, plain := 0, 0, 0
			focusTop, focusEnd := -1, -1
			for n, line := range lines {
				line = strings.TrimRight(line, " ")
				name := ""
				for _, s := range overviewSections {
					if strings.Contains(line, s+"  [") {
						name = s
					}
				}
				switch {
				case name != "" && titledRule.MatchString(line):
					titled++
					if name != overviewSections[mm.pane] {
						t.Fatalf("%s: %s is on a titled rule, not focused: %q\n%s", what, name, line, screen)
					}
					focusTop = n
				case name != "":
					plain++
					if strings.ContainsAny(line, "─↑↓") || !strings.HasPrefix(line, "    "+name) {
						t.Fatalf("%s: %s's title is not a plain line: %q\n%s", what, name, line, screen)
					}
					if !strings.Contains(line, "[") {
						t.Fatalf("%s: %s lost its count: %q", what, name, line)
					}
				case endRule.MatchString(line):
					ends++
					focusEnd = n
				}
			}
			if titled != 1 || plain != len(overviewSections)-1 {
				t.Fatalf("%s: %d titled rules, %d plain titles; want 1 and %d\n%s", what, titled, plain, len(overviewSections)-1, screen)
			}
			rows := len(mm.paneRows(mm.pane))
			if wantEnd := map[bool]int{true: 1, false: 0}[rows > 0]; ends != wantEnd {
				t.Fatalf("%s: %d end rules, want %d\n%s", what, ends, wantEnd, screen)
			}
			if rows > 0 && focusEnd <= focusTop {
				t.Fatalf("%s: the end rule (line %d) is not under the titled rule (line %d)\n%s", what, focusEnd, focusTop, screen)
			}
			// ↑/↓ as in A: on the focused section's rules exactly when rows
			// are hidden.
			cursor := mm.rowCursor(mm.pane)
			idx := mm.paneRows(mm.pane)
			start, end := mm.window(idx, cursor, mm.scrollOf(mm.pane), mm.overviewRows()[mm.pane], mm.selIn(mm.pane))
			if up := strings.HasPrefix(lines[focusTop], "↑") || strings.HasPrefix(lines[focusTop], "↓↑"); up != (rows > 0 && start > 0) {
				t.Fatalf("%s: ↑ on the titled rule %v, rows from %d\n%s", what, up, start, screen)
			}
			if rows > 0 {
				if down := strings.HasPrefix(lines[focusEnd], "↓"); down != (end < len(idx)) {
					t.Fatalf("%s: ↓ on the end rule %v, rows to %d of %d\n%s", what, down, end, len(idx), screen)
				}
			}
			if len(lines) > mm.bodyHeight() {
				t.Fatalf("%s: %d lines in a body of %d", what, len(lines), mm.bodyHeight())
			}
		}
	}
}
