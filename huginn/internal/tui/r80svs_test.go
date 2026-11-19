package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// r80svs (Jørgen 2026-11-19 11:09Z, 11:11Z): each cue on the view line has
// one meaning. Bold is a key you can press (v o l b !). Colour is the step
// of the key sequence: v in the accent, o l b ! violet. A steady
// underline under a whole word is the active view; a steady underline
// under one letter is the key you picked (v while it waits); a blinking
// underline is the keys you choose between now (o l b ! after v). The
// rest is dim, and nothing is filled.

// viewEnd is the column after '!lint' on the view line's text: the cells
// the cues are on; the counts after it are not.
func viewEnd(t *testing.T, text string) int {
	t.Helper()
	end := strings.Index(text, "!lint")
	if !strings.HasPrefix(text, "view: ") || end < 0 {
		t.Fatalf("view line reads %q", text)
	}
	return end + len("!lint")
}

// activeWord is the columns of the active view's word on the view line's
// text, its key letter included ('!lint' for lint).
func activeWord(text, view string) (int, int) {
	label := " " + view
	if view == "lint" {
		label = " !lint"
	}
	i := strings.Index(text, label)
	if i < 0 {
		return -1, -1
	}
	return i + 1, i + len(label)
}

// wantCues is what each cell of the view line should be drawn with in a
// state: pending is after v, shown the blink's visible phase.
type cueState struct {
	view           string
	pending, shown bool
}

// checkCues fails unless every cell of line up to '!lint' carries the
// r80svs cues for s; colour says whether the colours are checked too
// (false: NO_COLOR, no colour at all).
func checkCues(t *testing.T, where, line string, s cueState, colour bool) {
	t.Helper()
	text := ansi.Strip(line)
	end := viewEnd(t, text)
	from, to := activeWord(text, s.view)
	for col := 0; col < end; col++ {
		if text[col] == ' ' {
			continue
		}
		c := sgrAt(line, col)
		ch := string(text[col])
		if c.bg {
			t.Fatalf("%s: %q at column %d is filled: %q", where, ch, col, cellAt(line, col))
		}
		isKey := col == 0 || keyAt(text, col)
		inActive := col >= from && col < to
		if c.bold != isKey {
			t.Fatalf("%s: %q at column %d bold %v, a key %v: bold is a key you can press: %q", where, ch, col, c.bold, isKey, cellAt(line, col))
		}
		var under bool
		switch {
		case col == 0: // v: steady while it waits, the key you picked
			under = s.pending
		case isKey && s.pending: // the keys you choose between: they blink
			under = s.shown
		case isKey: // at rest only the active word's key, as part of its word
			under = inActive
		default: // the active word's other letters: steady
			under = inActive
		}
		if c.underline != under {
			t.Fatalf("%s: %q at column %d underlined %v, want %v: %q", where, ch, col, c.underline, under, cellAt(line, col))
		}
		want := ""
		if colour {
			switch {
			case col == 0:
				want = sgrColour("#00bfff")
			case isKey:
				want = sgrColour(stepTwoColour[0])
			default:
				want = sgrColour("#8795a8")
			}
		}
		if c.fg != want {
			t.Fatalf("%s: %q at column %d colour %q, want %q: %q", where, ch, col, c.fg, want, cellAt(line, col))
		}
	}
}

// r80svs check 1: at rest the active view's whole word is underlined, its
// key letter included, and no other word is; the keys are bold with no
// underline (but the active word's own); the rest is dim, nothing filled.
func TestViewLineRestUnderlinesActiveWord(t *testing.T) {
	for _, w := range []int{100, 160} {
		for _, v := range views {
			m := colourModel(t, w, v.name)
			checkCues(t, fmt.Sprintf("%d %s at rest", w, v.name), m.viewLine(m.stateCounts()), cueState{view: v.name}, true)
			// In a task no view is active: no word is underlined.
			in := press(m, "enter")
			if in.detail {
				line := in.viewLine(in.stateCounts())
				text := ansi.Strip(line)
				for col := 0; col < viewEnd(t, text); col++ {
					if sgrAt(line, col).underline {
						t.Fatalf("%d in a task %q at column %d is underlined: %q", w, string(text[col]), col, cellAt(line, col))
					}
				}
			}
		}
	}
}

// r80svs check 2: after v, v keeps a steady underline and no fill; the
// underlines of o l b ! blink together on vCursor's ticks (two frames one
// real 530 ms tick apart); the active word's other letters keep a steady
// underline. The second key or Esc brings the rest look back, the new view
// underlined.
func TestViewLineAfterVBlinksTheChoices(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m, cmd := pendingFrames(t, size[0], size[1])
			on := m.viewLine(m.stateCounts())
			checkCues(t, "after v, blink on", on, cueState{view: "overview", pending: true, shown: true}, true)
			msg, took := realTick(t, cmd)
			if took < 500*time.Millisecond || took > 900*time.Millisecond {
				t.Fatalf("the blink tick came after %v, want about 530ms", took)
			}
			next, _ := m.Update(msg)
			off := next.(model)
			checkCues(t, "after v, blink off", off.viewLine(off.stateCounts()), cueState{view: "overview", pending: true}, true)
			for k, view := range map[string]string{"l": "list", "!": "lint", "esc": "overview"} {
				done := press(off, k)
				checkCues(t, "after v "+k, done.viewLine(done.stateCounts()), cueState{view: view}, true)
			}
		})
	}
}

// r80svs check 3: the colours are the key's step in every state, v the
// accent and o l b ! violet (checkCues above); with NO_COLOR the line
// carries the same bold and underlines, cell for cell, and no colour.
func TestViewLineNoColorKeepsCues(t *testing.T) {
	m := colourModel(t, 140, "list")
	pending := press(m, "v")
	vm, tick := vBlinkTick(t, pending)
	next, _ := vm.Update(tick)
	blinked := next.(model)
	for _, s := range []struct {
		name  string
		m     model
		state cueState
	}{
		{"at rest", m, cueState{view: "list"}},
		{"after v, blink on", pending, cueState{view: "list", pending: true, shown: true}},
		{"after v, blink off", blinked, cueState{view: "list", pending: true}},
	} {
		t.Setenv("NO_COLOR", "1")
		plain := s.m
		plain.styles = newTheme()
		if plain.styles.renderer.ColorProfile() != termenv.Ascii {
			t.Fatal("NO_COLOR left colour on")
		}
		line := plain.viewLine(plain.stateCounts())
		if ansiColour.MatchString(line) {
			t.Fatalf("NO_COLOR %s: the view line has colour: %q", s.name, line)
		}
		checkCues(t, "NO_COLOR "+s.name, line, s.state, false)
		lit := s.m.viewLine(s.m.stateCounts())
		text := ansi.Strip(lit)
		for col := 0; col < viewEnd(t, text); col++ {
			a, b := sgrAt(lit, col), sgrAt(line, col)
			if a.bold != b.bold || a.underline != b.underline {
				t.Fatalf("NO_COLOR %s: %q at column %d bold/underline %v/%v, in colour %v/%v", s.name, string(text[col]), col, b.bold, b.underline, a.bold, a.underline)
			}
		}
	}
}

// cueRow is a row under the view line's text naming each cell's cues, for
// the goldens and the report: B bold, _ underlined, U both, a blank
// neither; and one naming its colour: a the accent, v violet, . dim, a
// blank none.
func cueRow(line string) (cues, colours string) {
	text := ansi.Strip(line)
	var c, k strings.Builder
	for col := range []rune(text) {
		s := sgrAt(line, col)
		switch {
		case s.bold && s.underline:
			c.WriteByte('U')
		case s.bold:
			c.WriteByte('B')
		case s.underline:
			c.WriteByte('_')
		default:
			c.WriteByte(' ')
		}
		switch s.fg {
		case sgrColour("#00bfff"):
			k.WriteByte('a')
		case sgrColour(stepTwoColour[0]):
			k.WriteByte('v')
		case sgrColour("#8795a8"):
			k.WriteByte('.')
		case "":
			k.WriteByte(' ')
		default:
			k.WriteByte('*')
		}
	}
	return strings.TrimRight(c.String(), " "), strings.TrimRight(k.String(), " ")
}
