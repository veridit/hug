package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// colourModel is manyModel drawn in true colour on a dark background, w
// wide, in view.
func colourModel(t *testing.T, w int, view string) model {
	t.Helper()
	m := manyModel(t)
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.styles.renderer.SetHasDarkBackground(true)
	styleAnswerCursor(&m.answerBox, m.styles.cursor)
	m.width, m.height = w, 24
	m.resize()
	m.setView(view)
	return m
}

// qh9ep1 check 9 (Jørgen 19:54: "too intense and over the top ...
// something more subtle"), 911j70 check 3: one cue per meaning on the
// view line. An underline marks only a key you can press; colour tells the
// keys apart (v the accent, o l b ! violet) and none is bold; the view you
// are in is told by bold alone on its word; nothing has a background; no
// cell is underlined, bold and coloured at once. Since a82j3n there is no
// cursor after 'view:'; after v only v changes (solid, picked) while the
// underlines are on, and the key colours are stable.
func TestViewLineMarksActiveView(t *testing.T) {
	const slot = len("view:") // the column after 'view:'
	for _, w := range []int{80, 140} {
		for _, v := range views {
			m := colourModel(t, w, v.name)
			idle := m.viewLine(m.stateCounts())
			pending := press(m, "v")
			lit := pending.viewLine(pending.stateCounts())
			text := ansi.Strip(idle)
			end := strings.Index(text, "lint") + len("lint")
			for col := 1; col < end; col++ { // v at 0 is solid: TestPickedVIsSolid
				if a, b := cellAt(idle, col), cellAt(lit, col); a != b {
					t.Fatalf("at %d in %s v recoloured column %d:\nidle    %q\npending %q", w, v.name, col, a, b)
				}
			}
			for _, s := range []model{m, pending} {
				if cell, on := cursorSlot(t, s); on {
					t.Fatalf("at %d in %s a cursor stands after 'view:': %q", w, v.name, cell)
				}
			}
			active := strings.Index(text, v.name)
			for col := 0; col < end; col++ {
				if col == slot || text[col] == ' ' {
					continue
				}
				c := sgrAt(idle, col)
				ch := string(text[col])
				if c.bg {
					t.Fatalf("at %d in %s %q at column %d has a background: %q", w, v.name, ch, col, cellAt(idle, col))
				}
				if c.underline && c.bold && c.fg != "" {
					t.Fatalf("at %d in %s %q at column %d is underlined, bold and coloured: %q", w, v.name, ch, col, cellAt(idle, col))
				}
				key := col == 0 || keyAt(text, col)
				if c.underline != key {
					t.Fatalf("at %d in %s %q at column %d: underlined %v, a key %v: %q", w, v.name, ch, col, c.underline, key, cellAt(idle, col))
				}
				if key {
					want := sgrColour(stepTwoColour[0])
					if col == 0 {
						want = sgrColour("#00bfff")
					}
					if c.fg != want || c.bold {
						t.Fatalf("at %d in %s key %q: colour %q bold %v, want %q not bold", w, v.name, ch, c.fg, c.bold, want)
					}
					continue
				}
				inActive := col >= active && col < active+len(v.name)
				if c.bold != inActive {
					t.Fatalf("at %d in %s %q at column %d: bold %v, in the active word %v", w, v.name, ch, col, c.bold, inActive)
				}
				if c.fg != sgrColour("#8795a8") {
					t.Fatalf("at %d in %s %q at column %d is %q, not the dim of the line: bold alone tells the view", w, v.name, ch, col, c.fg)
				}
			}
		}
	}
}

// keyAt says whether column col of the view line's text is a view's key
// letter: the l of list, the ! of !lint.
func keyAt(text string, col int) bool {
	for _, v := range views {
		label := v.name
		at := strings.Index(v.name, v.key)
		if at < 0 {
			label, at = v.key+v.name, 0
		}
		if i := strings.Index(text, " "+label); i >= 0 && col == i+1+at {
			return true
		}
	}
	return false
}

// sgrColour is a #rrggbb foreground as the renderer writes it.
func sgrColour(hex string) string {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

// cell is what a terminal draws a character with.
type cell struct {
	bold, underline, bg bool
	fg                  string
}

// sgrAt is the attributes in effect at column col of a one-line s.
func sgrAt(s string, col int) cell {
	var c cell
	for _, code := range regexp.MustCompile(`\x1b\[([0-9;]*)m`).FindAllStringSubmatch(cellAt(s, col), -1) {
		ps := strings.Split(code[1], ";")
		for i := 0; i < len(ps); i++ {
			switch p := ps[i]; {
			case p == "" || p == "0":
				c = cell{}
			case p == "1":
				c.bold = true
			case p == "22":
				c.bold = false
			case p == "4":
				c.underline = true
			case p == "24":
				c.underline = false
			case p == "39":
				c.fg = ""
			case p == "49":
				c.bg = false
			case (p == "38" || p == "48") && i+4 < len(ps) && ps[i+1] == "2":
				if p == "38" {
					c.fg = strings.Join(ps[i:i+5], ";")
				} else {
					c.bg = true
				}
				i += 4
			case len(p) == 2 && p[0] == '4', len(p) == 3 && p[:2] == "10":
				c.bg = true // 40-47, 100-107
			case len(p) == 2 && (p[0] == '3' || p[0] == '9'):
				c.fg = p
			}
		}
	}
	return c
}

// cursorSlot is the cell after 'view:' with its codes, and whether the
// one cursor (theme.cursor: underlined, the accent) is drawn there.
func cursorSlot(t *testing.T, m model) (string, bool) {
	t.Helper()
	line := m.viewLine(m.stateCounts())
	if !strings.HasPrefix(ansi.Strip(line), "view: ") {
		t.Fatalf("view line reads %q", ansi.Strip(line))
	}
	cell := cellAt(line, 5)
	under := strings.Contains(cell, "\x1b[4") || strings.Contains(cell, ";4m") || strings.Contains(cell, ";4;")
	return cell, under && strings.Contains(cell, "38;2;0;191;255")
}

// cellAt is the visible character at column col of a one-line s, after
// the codes in effect for it (those since the last reset).
func cellAt(s string, col int) string {
	active, at := "", 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := strings.IndexByte(s[i:], 'm')
			if j < 0 {
				break
			}
			if code := s[i : i+j+1]; code == "\x1b[0m" || code == "\x1b[m" {
				active = ""
			} else {
				active += code
			}
			i += j + 1
			continue
		}
		_, n := utf8.DecodeRuneInString(s[i:])
		if at == col {
			return active + s[i:i+n]
		}
		at++
		i += n
	}
	return ""
}

// vBlinkTick is the blink message the view cursor waits for, taken from
// its own blink command without waiting 530 ms, and m expecting it.
func vBlinkTick(t *testing.T, m model) (model, tea.Msg) {
	t.Helper()
	c := m.vCursor
	c.BlinkSpeed = time.Millisecond
	msg := c.BlinkCmd()()
	if _, ok := msg.(cursor.BlinkMsg); !ok {
		t.Fatalf("blink command gave %T", msg)
	}
	c.BlinkSpeed = 530 * time.Millisecond
	m.vCursor = c
	return m, msg
}

// 911j70 check 3, as a82j3n has it: after v the blink (vCursor, bubbles'
// cursor at 530 ms, no timer of its own) drives the underlines of o l b !;
// no cursor stands after 'view:' in either phase. Chosen or cancelled,
// the blink stops (TestViewKeysSteadyAfterSecondKey has the look).
func TestViewCursorBlinksAfterV(t *testing.T) {
	m := colourModel(t, 140, "overview")
	m, cmd := key(m, "v")
	if cmd == nil {
		t.Fatal("v returned no command: the blink never starts")
	}
	if m.vCursor.Mode() != cursor.CursorBlink || m.vCursor.BlinkSpeed != 530*time.Millisecond {
		t.Fatalf("view blink mode %v speed %v, want blink at 530ms", m.vCursor.Mode(), m.vCursor.BlinkSpeed)
	}
	underlined := func(m model) bool {
		line := m.viewLine(m.stateCounts())
		return sgrAt(line, strings.Index(ansi.Strip(line), "list")).underline
	}
	// One tick blinks the underlines off and asks for the next; the next
	// brings them back. No phase draws a cursor after 'view:'.
	if cell, on := cursorSlot(t, m); on || !underlined(m) {
		t.Fatalf("after v: cursor %q, l underlined %v", cell, underlined(m))
	}
	m, tick := vBlinkTick(t, m)
	next, again := m.Update(tick)
	off := next.(model)
	if _, on := cursorSlot(t, off); on || underlined(off) || again == nil {
		t.Fatalf("one tick: cursor %v, l underlined %v, stopped %v", on, underlined(off), again == nil)
	}
	off, tick = vBlinkTick(t, off)
	next, again = off.Update(tick)
	if !underlined(next.(model)) || again == nil {
		t.Fatal("the second tick did not bring the underlines back")
	}
	for _, k := range []string{"o", "esc", "x"} {
		done, tick := vBlinkTick(t, press(m, k))
		if _, c := done.Update(tick); c != nil {
			t.Fatalf("after v %s the underlines keep blinking", k)
		}
	}
	// The box's blink and the view cursor's do not cross: a tick of the
	// view cursor leaves a closed box alone and the reverse.
	if _, c := m.Update(cursor.BlinkMsg{}); c != nil {
		t.Fatal("a blink no cursor expects was answered")
	}
}

// 911j70 check 3 golden: the view line at 80 and 140 in the overview,
// idle, after v (underlines on, then off one blink later; a82j3n) and
// after v l, as text and with its codes visible.
func TestViewCursorGolden(t *testing.T) {
	var golden []string
	for _, w := range []int{80, 140} {
		m := colourModel(t, w, "overview")
		pending := press(m, "v")
		p2, tick := vBlinkTick(t, pending)
		next, _ := p2.Update(tick)
		blinked := next.(model)
		chosen := press(pending, "l")
		golden = append(golden, "== "+strconv.Itoa(w))
		for _, s := range []struct {
			name string
			m    model
		}{{"idle", m}, {"after v", pending}, {"after v, blink off", blinked}, {"after v l", chosen}} {
			line := s.m.viewLine(s.m.stateCounts())
			golden = append(golden, s.name+": "+ansi.Strip(line), "  "+strconv.Quote(line))
		}
		// NO_COLOR: the renderer writes no attribute at all; the line is
		// its text, the same before and after v.
		plain := m
		plain.styles = newTheme()
		plain.styles.renderer.SetColorProfile(termenv.Ascii)
		plainV := press(plain, "v")
		for _, s := range []struct {
			name string
			m    model
		}{{"NO_COLOR idle", plain}, {"NO_COLOR after v", plainV}} {
			golden = append(golden, s.name+": "+strconv.Quote(s.m.viewLine(s.m.stateCounts())))
		}
	}
	goldenFile(t, "testdata/header/view-cursor.txt", strings.Join(golden, "\n")+"\n")
}
