package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// v68n82 (Jørgen 2026-11-20 12:33Z, request 1, screen-1332.ans): each
// section of the overview, the list and each board column sits between
// two rules, the frame of a task's page (4ddcwa). The top rule carries
// the section's title and count: '─── WAITING  [20] · 2–18 shown ───'; a
// rule ends it. ↑ and ↓ (rows hidden above, below) sit at the rules' left
// end, as on the page, never in the rows' gutter. The focused section's
// rules are brighter, the others dim; they never pulse: only the focus
// sign > does (q3sh69).

// titledRule matches a section's top rule: the lead (arrows, then the
// rule), a space, the title, a space and the rule to the edge; a title
// that fills a narrow column ends there.
var titledRule = regexp.MustCompile(`^(↓↑─|↑──|↓──|───) (.*?\S)(?: ─+)?$`)

// endRule matches a plain rule: a section's end, ↓ at its left end when
// rows are hidden below.
var endRule = regexp.MustCompile(`^(↓|─)─+$`)

// shownRange reads a title's count and its "a–b shown": total, a and b,
// or a=1, b=total when every row shows.
var shownRange = regexp.MustCompile(`\[(\d+)\](?:.*· (\d+)–(\d+) shown)?`)

// frameSection is one section as the screen draws it.
type frameSection struct {
	top, end    int // line numbers of the rules; end -1 when there is none
	lead        string
	title       string
	rows        []string
	total, a, b int
}

// sectionsOf cuts lines (stripped, one view's body, one column wide) into
// framed sections. names are the titles the sections start with. It
// fails when a title is not on a titled rule, or a section with rows has
// no end rule (shared: the next titled rule may end it, variant B).
func sectionsOf(t *testing.T, what string, lines []string, names []string, shared bool) []frameSection {
	t.Helper()
	var out []frameSection
	screen := strings.Join(lines, "\n")
	for n, line := range lines {
		line = strings.TrimRight(line, " ")
		for _, name := range names {
			if !strings.Contains(line, name+"  [") && !strings.Contains(line, name+" [") {
				continue
			}
			m := titledRule.FindStringSubmatch(line)
			if m == nil {
				t.Fatalf("%s: the title of %s is not on a titled rule: %q\n%s", what, name, line, screen)
			}
			s := frameSection{top: n, end: -1, lead: m[1], title: m[2]}
			c := shownRange.FindStringSubmatch(s.title)
			if c == nil {
				t.Fatalf("%s: no count on %q", what, line)
			}
			s.total, _ = strconv.Atoi(c[1])
			s.a, s.b = 1, s.total
			if c[2] != "" {
				s.a, _ = strconv.Atoi(c[2])
				s.b, _ = strconv.Atoi(c[3])
			}
			out = append(out, s)
		}
	}
	if len(out) != len(names) {
		t.Fatalf("%s: %d titled rules, want %d\n%s", what, len(out), len(names), screen)
	}
	for k := range out {
		s := &out[k]
		stop := len(lines)
		if k+1 < len(out) {
			stop = out[k+1].top
		}
		for n := s.top + 1; n < stop; n++ {
			line := strings.TrimRight(lines[n], " ")
			if endRule.MatchString(line) {
				s.end = n
				break
			}
			if line != "" {
				s.rows = append(s.rows, line)
			}
		}
		if len(s.rows) > 0 && s.end < 0 && (!shared || k == len(out)-1) {
			t.Fatalf("%s: %s has rows and no end rule\n%s", what, s.title, screen)
		}
	}
	return out
}

// checkArrows: ↑ on the titled rule exactly when rows are hidden above,
// ↓ on the end rule (B: on the next titled rule) exactly when hidden
// below; never an arrow in a row's gutter.
func checkArrows(t *testing.T, what string, lines []string, secs []frameSection, shared bool) (ups, downs int) {
	t.Helper()
	for k, s := range secs {
		up := strings.Contains(s.lead, "↑")
		if up != (s.a > 1) {
			t.Fatalf("%s: %s: ↑ on its rule %v, rows %d–%d of %d", what, s.title, up, s.a, s.b, s.total)
		}
		closing := ""
		switch {
		case s.end >= 0:
			closing = strings.TrimRight(lines[s.end], " ")
		case shared && k+1 < len(secs):
			closing = secs[k+1].lead
		}
		down := strings.HasPrefix(closing, "↓")
		if len(s.rows) > 0 && down != (s.b < s.total) {
			t.Fatalf("%s: %s: ↓ on its end %v (%q), rows %d–%d of %d", what, s.title, down, closing, s.a, s.b, s.total)
		}
		for _, r := range s.rows {
			if strings.HasPrefix(r, "↑") || strings.HasPrefix(r, "↓") {
				t.Fatalf("%s: %s: an arrow in a row's gutter: %q", what, s.title, r)
			}
		}
		if up {
			ups++
		}
		if down {
			downs++
		}
	}
	return ups, downs
}

// viewSections is the frame of m's view: its body lines, as one column
// for the board (the column under the cursor and the one left of it),
// and the section names.
func viewSections(t *testing.T, m model) (cols [][]string, names [][]string) {
	t.Helper()
	switch m.view {
	case "overview":
		return [][]string{strings.Split(ansi.Strip(m.overviewText()), "\n")}, [][]string{overviewSections}
	case "board":
		count := min(len(boardStates), max(1, (m.width+2)/40))
		first := min(max(0, m.pane-count+1), len(boardStates)-count)
		width := max(1, (m.width-2*(count-1))/count)
		lines := strings.Split(ansi.Strip(m.boardText()), "\n")
		for c := 0; c < count; c++ {
			var col []string
			for _, l := range lines[:len(lines)-2] { // the legend and the column count
				r := []rune(l)
				from := min(len(r), c*(width+2))
				col = append(col, string(r[from:min(len(r), from+width)]))
			}
			cols = append(cols, col)
			names = append(names, []string{strings.ToUpper(boardStates[first+c])})
		}
		return cols, names
	}
	return [][]string{strings.Split(ansi.Strip(m.listText()), "\n")}, [][]string{{"TASKS"}}
}

// v68n82 check 3: overview, list and board at 100x30 and 160x45: every
// section between its titled top rule and its end rule; ↑/↓ on the rules
// exactly when rows are hidden, never in a row's gutter. The grid board
// scrolls IN PROGRESS (overview), the list and DOING (board), so each
// view shows its arrows. Goldens testdata/v68n82/<view>-<w>x<h>.txt.
func TestSectionFrameRulesAndArrows(t *testing.T) {
	for _, size := range gridSizes {
		w, h := size[0], size[1]
		for _, view := range []string{"overview", "list", "board"} {
			what := fmt.Sprintf("%s at %dx%d", view, w, h)
			m := gridModel(t, w, h, view)
			cols, names := viewSections(t, m)
			ups, downs := 0, 0
			for c := range cols {
				secs := sectionsOf(t, what, cols[c], names[c], false)
				u, d := checkArrows(t, what, cols[c], secs, false)
				ups, downs = ups+u, downs+d
			}
			if downs == 0 && view != "board" {
				// The board's columns show every row at both sizes; its
				// arrows: TestBoardFrameArrows.
				t.Fatalf("%s: no section shows ↓: the fixture scrolls none\n%s", what, ansi.Strip(m.View()))
			}
			if view == "overview" && h < 45 && ups == 0 {
				// At 160x45 IN PROGRESS shows all 14; ↑ shows at 100x30.
				t.Fatalf("%s: no section shows ↑\n%s", what, ansi.Strip(m.View()))
			}
			goldenFile(t, filepath.Join("testdata", "v68n82", fmt.Sprintf("%s-%dx%d.txt", view, w, h)), ansi.Strip(m.View())+"\n")
		}
	}
}

// v68n82 check 3: scrolled to the end of the list, ↑ on the top rule and
// no ↓ on the end rule; at the top neither ↑ nor a ↓-less end.
func TestListFrameArrowsAtTheEnds(t *testing.T) {
	m := gridModel(t, 100, 30, "list")
	for _, at := range []struct {
		keys     []string
		up, down bool
	}{{[]string{"<"}, false, true}, {[]string{">"}, true, false}} {
		v := press(m, at.keys...)
		lines := strings.Split(ansi.Strip(v.listText()), "\n")
		secs := sectionsOf(t, "list", lines, []string{"TASKS"}, false)
		checkArrows(t, "list", lines, secs, false)
		s := secs[0]
		if up, down := strings.Contains(s.lead, "↑"), strings.HasPrefix(lines[s.end], "↓"); up != at.up || down != at.down {
			t.Fatalf("list after %v: ↑ %v ↓ %v, want %v %v\n%s", at.keys, up, down, at.up, at.down, strings.Join(lines, "\n"))
		}
	}
}

// v68n82 check 3 on the board: a section is a column, one status. At
// 100x20 DOING (15 tasks) scrolls: at its top ↓ on its end rule only;
// at its end ↑ on its titled rule only; TODO, all shown, has neither.
func TestBoardFrameArrows(t *testing.T) {
	m := gridModel(t, 100, 20, "board")
	for _, at := range []struct {
		keys     []string
		up, down bool
	}{{[]string{"<"}, false, true}, {[]string{">"}, true, false}} {
		v := press(m, at.keys...)
		cols, names := viewSections(t, v)
		for c := range cols {
			secs := sectionsOf(t, "board", cols[c], names[c], false)
			checkArrows(t, "board", cols[c], secs, false)
			s := secs[0]
			up, down := strings.Contains(s.lead, "↑"), strings.HasPrefix(cols[c][s.end], "↓")
			want := names[c][0] == "DOING"
			if up != (want && at.up) || down != (want && at.down) {
				t.Fatalf("board after %v: %s ↑ %v ↓ %v\n%s", at.keys, names[c][0], up, down, strings.Join(cols[c], "\n"))
			}
		}
	}
}

// ruleCell is the attributes of the first rule glyph ─ on line, after the
// lead's arrows.
func ruleCell(t *testing.T, line string) cell {
	t.Helper()
	for col, r := range []rune(ansi.Strip(line)) {
		if r == '─' {
			return sgrAt(line, col)
		}
	}
	t.Fatalf("no rule on %q", ansi.Strip(line))
	return cell{}
}

// v68n82 check 3: the focused section's rules are brighter than the
// others': bold, not the dim grey; every other section's rules dim. No
// fill, no underline. They never pulse: the pulse's off phase leaves
// them as they were (q3sh69: only the focus sign > pulses).
func TestFocusedSectionRulesBrighterNotPulsing(t *testing.T) {
	for _, view := range []string{"overview", "board"} {
		m := gridModel(t, 160, 45, view)
		m.pulse.live = true
		m = press(m, "j", "k")
		text := func(m model) []string {
			if m.view == "board" {
				return strings.Split(m.boardText(), "\n")
			}
			return strings.Split(m.overviewText(), "\n")
		}
		lines := text(m)
		names := overviewSections
		focus := ""
		if view == "board" {
			names = []string{strings.ToUpper(boardStates[m.pane])}
			focus = names[0]
		} else {
			focus = overviewSections[m.pane]
		}
		bright, dim := 0, 0
		offLines := text(pulseOff(t, m))
		for n, l := range lines {
			plain := strings.TrimRight(ansi.Strip(l), " ")
			if !strings.HasPrefix(plain, "↑──") && !strings.HasPrefix(plain, "↓──") && !strings.HasPrefix(plain, "───") && !strings.HasPrefix(plain, "↓↑─") {
				continue
			}
			c := ruleCell(t, l)
			if view == "board" {
				// Several columns share a line: the focused column's rule
				// is the ─ right after its title.
				at := strings.Index(plain, focus+" [")
				if at < 0 {
					continue
				}
				col := len([]rune(plain[:at]))
				r := []rune(plain)
				for col < len(r) && r[col] != '─' {
					col++
				}
				c = sgrAt(l, col)
			}
			if c.bg || c.underline {
				t.Fatalf("%s: a rule is filled or underlined: %q", view, plain)
			}
			isFocus := view == "board" || ownerOf(lines, n, names) == focus
			if isFocus {
				if !c.bold || c.fg == dimFg {
					t.Fatalf("%s: the focused section's rule is not bright: bold=%v fg=%q: %q", view, c.bold, c.fg, plain)
				}
				bright++
			} else {
				if c.bold || c.fg != dimFg {
					t.Fatalf("%s: another section's rule is not dim: bold=%v fg=%q: %q", view, c.bold, c.fg, plain)
				}
				dim++
			}
			if offLines[n] != l {
				t.Fatalf("%s: the rule changed with the pulse:\non  %q\noff %q", view, l, offLines[n])
			}
		}
		if view == "overview" && (bright != 2 || dim == 0) {
			t.Fatalf("%s: %d bright rules (want the focused section's 2), %d dim", view, bright, dim)
		}
		if view == "board" && bright != 1 {
			t.Fatalf("board: %d bright titled rules over %s, want 1", bright, focus)
		}
	}
}

// ownerOf is the section a rule at line n belongs to: the last title at
// or above it.
func ownerOf(lines []string, n int, names []string) string {
	for k := n; k >= 0; k-- {
		plain := ansi.Strip(lines[k])
		for _, name := range names {
			if strings.Contains(plain, name+"  [") {
				return name
			}
		}
	}
	return ""
}
