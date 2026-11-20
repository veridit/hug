package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// 911j70 (Jørgen 2026-11-20 08:26Z, answer 1): the lists read as a calm
// sectioned spreadsheet, order B: ID TITLE STAGE ✓ TYPE PRIORITY OWNER.
// Tags are a dim prefix of the TITLE cell, not a column. The current cell
// carries the > cursor, with no underline and no background: an underline
// marks the active view on the view line, a passive state. No row bar, no
// column band; the one tint is on the cursor row's id (the row header) and
// the marked column's heading.

// gridSizes are the sizes the goldens are taken at.
var gridSizes = [][2]int{{100, 30}, {160, 45}}

// gridModel is manyModel at w x h in true colour on a dark background, in
// view, with the cursor a few rows into its section and PRIORITY marked.
func gridModel(t *testing.T, w, h int, view string) model {
	t.Helper()
	m := manyModel(t)
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.styles.renderer.SetHasDarkBackground(true)
	m.width, m.height = w, h
	m.resize()
	m.setView(view)
	switch view {
	case "board":
		m = press(m, "l", "j", "j", "j") // DOING, its fourth row
	default:
		m = press(m, "j", "j", "j")
	}
	return markField(t, m, "priority")
}

// bodyLines is the view's own lines, without the header bands above and,
// on the board, the legend under it (it spells the T and P keys).
func bodyLines(m model) []string {
	var text string
	switch m.view {
	case "overview":
		text = m.overviewText()
	case "board":
		text = m.boardText()
	default:
		text = m.listText()
	}
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if s := ansi.Strip(l); strings.HasPrefix(s, "Type:") || strings.HasPrefix(s, "T:") {
			continue
		}
		out = append(out, l)
	}
	return out
}

// 911j70 check 2: overview, list and board render order B at 100x30 and
// 160x45: the header reads ID TITLE STAGE ✓ TYPE PRIORITY (OWNER), the >
// stands before the cursor row's PRIORITY cell, and there is no row bar
// or column band: the only backgrounds are the cursor row's id and the
// PRIORITY heading. Goldens testdata/grid/<view>-<w>x<h>.txt.
func TestGridOrderBGoldens(t *testing.T) {
	head := regexp.MustCompile(`^ {3}ID {7}TITLE +(STAGE +)?(✓ +)?(TYPE +)?(PRIORITY|P)( +OWNER)?`)
	board := regexp.MustCompile(`^ {3}ID {7}TITLE +✓ +T P`)
	for _, size := range gridSizes {
		w, h := size[0], size[1]
		for _, view := range []string{"overview", "list", "board"} {
			where := fmt.Sprintf("%s at %dx%d", view, w, h)
			m := gridModel(t, w, h, view)
			sel, ok := m.selected()
			if !ok {
				t.Fatalf("%s: no task under the cursor", where)
			}
			lines := bodyLines(m)
			tinted := 0
			heads := 0
			for _, l := range lines {
				text := ansi.Strip(l)
				if head.MatchString(text) || board.MatchString(text) {
					heads++
				}
				cells := []rune(text) // one cell each on this board: no wide runes
				for col := range cells {
					if !sgrAt(l, col).bg {
						continue
					}
					tinted++
					// A tinted cell is the cursor row's id or the marked
					// heading's PRIORITY (or P): nothing else.
					from := col
					for from > 0 && cells[from-1] != ' ' {
						from--
					}
					word := strings.TrimLeft(strings.Fields(string(cells[from:]))[0], "·!") // the lint mark's cell
					if word != sel.ID && word != "PRIORITY" && word != "P" {
						t.Fatalf("%s: %q at column %d has a background: a bar or a band:\n%q", where, word, col, l)
					}
				}
			}
			if heads == 0 {
				t.Fatalf("%s: no ID TITLE STAGE ✓ TYPE PRIORITY header:\n%s", where, ansi.Strip(strings.Join(lines, "\n")))
			}
			if want := len(sel.ID) + len("PRIORITY"); view != "board" && tinted != want || view == "board" && tinted != len(sel.ID)+1 {
				t.Fatalf("%s: %d tinted cells, want the id and the marked heading only", where, tinted)
			}
			row := headerLine(t, ansi.Strip(strings.Join(lines, "\n")), sel.ID)
			if strings.Count(row, ">") != 1 || !regexp.MustCompile(`>(`+regexp.QuoteMeta(sel.Priority)+`|`+strings.ToUpper(sel.Priority[:1])+`)`).MatchString(row) {
				t.Fatalf("%s: the cursor row has no lone > before its priority: %q", where, row)
			}
			golden(t, filepath.Join("testdata", "grid", fmt.Sprintf("%s-%dx%d.txt", view, w, h)), "TestGridOrderBGoldens", ansi.Strip(m.View())+"\n")
		}
	}
}

// 911j70 check 5: the tags are a dim prefix of the TITLE cell, "parser
// compiler · Both areas"; there is no TAGS column, and the title text is
// cut after the prefix. Goldens at 100x30 and 160x45 in list, overview
// and board.
func TestTagsPrefixTitleGolden(t *testing.T) {
	for _, size := range gridSizes {
		w, h := size[0], size[1]
		var shots []string
		for _, view := range []string{"list", "overview", "board"} {
			m, ids := tagRowBoard(t, w)
			m.height = h
			m.resize()
			m.setView(view)
			s := screen(m)
			if strings.Contains(s, "TAGS") {
				t.Fatalf("%s at %dx%d has a TAGS column:\n%s", view, w, h, s)
			}
			// A narrow board column cuts the tags to leave the title
			// room; where the title would keep less than titleMinCompact
			// (the board at 160, four columns) they give way whole, never
			// the title.
			row := rowOf(t, s, ids["Both areas"])
			count := min(len(boardStates), (w+2)/40) // as boardText splits the width
			gone := view == "board" && m.rowLayout((w-2*(count-1))/count, true, 0, 0).title-1 < titleMinCompact+3+4
			prefix := regexp.MustCompile(regexp.QuoteMeta(ids["Both areas"]) + `   parser (compiler|com…) · Both areas`)
			if gone && !strings.Contains(row, ids["Both areas"]+"   Both areas") || !gone && !prefix.MatchString(row) {
				t.Fatalf("%s at %dx%d: the tags are not the title's prefix (gave way: %v):\n%q", view, w, h, gone, row)
			}
			// Cut: a long title is cut after the prefix, never the prefix.
			if long := rowOf(t, s, ids["Long tags"]); !gone && !regexp.MustCompile(`   compiler (codegen[^·]*|c…) · `).MatchString(long) {
				t.Fatalf("%s at %dx%d: the long row lost its prefix:\n%q", view, w, h, long)
			}
			for title, letter := range map[string]string{"Lex the input": "a", "Both areas": "b", "Long tags": "c", "Odd one": "d", "Bare": "e"} {
				s = strings.ReplaceAll(s, ids[title], strings.Repeat(letter, len(ids[title])))
			}
			shots = append(shots, "== "+view, s)
		}
		// The prefix is dim, the title after it in the plain colour.
		m, ids := tagRowBoard(t, w)
		m.styles = newTheme()
		m.styles.renderer.SetColorProfile(termenv.TrueColor)
		m.styles.renderer.SetHasDarkBackground(true)
		row := rowOf(t, m.View(), ids["Both areas"])
		at := strings.Index(ansi.Strip(row), "parser")
		if c := sgrAt(row, at); c.fg != sgrColour("#8795a8") {
			t.Fatalf("at %d the tags are not dim: %q", w, cellAt(row, at))
		}
		if c := sgrAt(row, strings.Index(ansi.Strip(row), "Both")); c.fg == sgrColour("#8795a8") || c.bg || c.underline {
			t.Fatalf("at %d the title after the tags is dim, filled or underlined: %q", w, row)
		}
		golden(t, filepath.Join("testdata", "grid", fmt.Sprintf("tags-%dx%d.txt", w, h)), "TestTagsPrefixTitleGolden", strings.Join(shots, "\n")+"\n")
	}
}

// 911j70 check 6: the current cell is marked by the > alone: the list
// body (every view's rows and headers) has no SGR 4 underline, and the
// cell after the > has no background. In every view, with and without a
// marked column, at both sizes; the view line keeps its underline (r80svs).
func TestCurrentCellIsTheCursorSignOnly(t *testing.T) {
	underline := regexp.MustCompile(`\x1b\[(?:[0-9;]*;)?4(?:;[0-9;]*)?m`)
	for _, size := range gridSizes {
		for _, view := range []string{"overview", "list", "board"} {
			for _, marked := range []bool{false, true} {
				m := gridModel(t, size[0], size[1], view)
				if !marked {
					m.column = -1
				}
				where := fmt.Sprintf("%s at %dx%d, column marked %v", view, size[0], size[1], marked)
				sel, _ := m.selected()
				for _, l := range bodyLines(m) {
					// SGR 4 alone; 38;2;r;g;b holds a 4 only as a colour.
					for _, code := range underline.FindAllString(l, -1) {
						if sgrAt(code+"x", 0).underline {
							t.Fatalf("%s: an underline in the list body: %q in %q", where, code, l)
						}
					}
					text := ansi.Strip(l)
					i := strings.Index(text, sel.ID)
					if i < 0 || !strings.Contains(text, ">") {
						continue
					}
					// The cursor row: one >, then a cell with no background.
					from := 0
					if view == "board" {
						from = max(0, strings.LastIndex(text[:i], "  ")-1)
					}
					cur := strings.Index(text[from:], ">") + from
					next := lipgloss.Width(text[:cur]) + 1
					if c := sgrAt(l, next); marked && (c.bg || c.underline) {
						t.Fatalf("%s: the current cell after > has a background or underline: %q", where, cellAt(l, next))
					}
					if c := sgrAt(l, lipgloss.Width(text[:cur])); c.bg || c.underline {
						t.Fatalf("%s: the > is filled or underlined: %q", where, cellAt(l, lipgloss.Width(text[:cur])))
					}
				}
				// The view line still marks the active view with its
				// underline: underline is its, not the list's.
				if line := m.viewLine(m.stateCounts()); !underline.MatchString(line) {
					t.Fatalf("%s: the view line lost its underline: %q", where, line)
				}
			}
		}
	}
}
