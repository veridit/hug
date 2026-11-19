package tui

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// a82j3n (Jørgen 2026-11-19 10:26Z, 10:27Z): while v waits for its second
// key, the underlines of the keys that can come next, o l b !, blink
// together at the cursor's 530 ms; there is no cursor cell after 'view:'
// any more. v itself, picked, was drawn solid (a filled accent cell);
// r80svs replaced that with a steady underline under v, no fill.

// keyColumns are the columns of o l b ! on the view line's text.
func keyColumns(t *testing.T, text string) map[int]bool {
	t.Helper()
	cols := map[int]bool{}
	for col := range text {
		if col < len(text) && keyAt(text, col) {
			cols[col] = true
		}
	}
	if len(cols) != len(views) {
		t.Fatalf("found %d key columns on %q, want %d", len(cols), text, len(views))
	}
	return cols
}

// pendingFrames is a colour model w x h after v, its blink command, and
// the frame it draws.
func pendingFrames(t *testing.T, w, h int) (model, tea.Cmd) {
	t.Helper()
	m := colourModel(t, w, "overview")
	m.height = h
	m.resize()
	m, cmd := key(m, "v")
	if cmd == nil {
		t.Fatal("v returned no command: the blink never starts")
	}
	return m, cmd
}

// realTick runs a blink command as the program would: it waits its
// 530 ms and gives the message, and how long that took.
func realTick(t *testing.T, cmd tea.Cmd) (tea.Msg, time.Duration) {
	t.Helper()
	start := time.Now()
	msg := cmd()
	took := time.Since(start)
	if _, ok := msg.(cursor.BlinkMsg); !ok {
		t.Fatalf("blink command gave %T", msg)
	}
	return msg, took
}

// diffOnlyKeyUnderlines fails unless frames a and b are the same but for
// the underline of o l b ! on the view line, underlined in a as wantA says.
func diffOnlyKeyUnderlines(t *testing.T, a, b string, wantA bool) {
	t.Helper()
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	if len(la) != len(lb) {
		t.Fatalf("frames have %d and %d lines", len(la), len(lb))
	}
	changed := 0
	for i := range la {
		if la[i] == lb[i] {
			continue
		}
		text := ansi.Strip(la[i])
		if !strings.HasPrefix(text, "view: ") || text != ansi.Strip(lb[i]) {
			t.Fatalf("line %d changed and is not the view line's underlines:\nA %q\nB %q", i, la[i], lb[i])
		}
		keys := keyColumns(t, text)
		for col := 0; col < len([]rune(text)); col++ {
			ca, cb := sgrAt(la[i], col), sgrAt(lb[i], col)
			if !keys[col] {
				if ca != cb {
					t.Fatalf("view line column %d (%q) changed: %+v -> %+v", col, string([]rune(text)[col]), ca, cb)
				}
				continue
			}
			if ca.fg != cb.fg || ca.bold != cb.bold || ca.bg != cb.bg {
				t.Fatalf("key %q changed more than its underline: %+v -> %+v", string([]rune(text)[col]), ca, cb)
			}
			if ca.underline != wantA || cb.underline == wantA {
				t.Fatalf("key %q underline %v -> %v, want %v -> %v", string([]rune(text)[col]), ca.underline, cb.underline, wantA, !wantA)
			}
			changed++
		}
	}
	if changed != len(views) {
		t.Fatalf("%d key underlines changed between the frames, want %d (o l b !)", changed, len(views))
	}
}

// a82j3n check 1: two v-pending frames one real blink (about 0.53 s)
// apart differ only in the underline of o, l, b and !; the next tick
// brings them back.
func TestViewKeysBlinkAfterV(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m, cmd := pendingFrames(t, size[0], size[1])
			on := m.View()
			msg, took := realTick(t, cmd)
			if took < 500*time.Millisecond || took > 900*time.Millisecond {
				t.Fatalf("the blink tick came after %v, want about 530ms", took)
			}
			next, cmd := m.Update(msg)
			m = next.(model)
			if cmd == nil {
				t.Fatal("one blink tick asked for no next one")
			}
			off := m.View()
			diffOnlyKeyUnderlines(t, on, off, true)
			msg, _ = realTick(t, cmd)
			next, cmd = m.Update(msg)
			m = next.(model)
			if cmd == nil {
				t.Fatal("the second tick asked for no next one")
			}
			diffOnlyKeyUnderlines(t, off, m.View(), false)
		})
	}
}

// a82j3n check 2: no cursor cell after 'view:' in either phase: the cell
// at column 5 is a bare space; the line's text and width are the same
// idle and after v.
func TestNoCursorCellAfterView(t *testing.T) {
	for _, w := range []int{100, 160} {
		m := colourModel(t, w, "overview")
		idle := m.viewLine(m.stateCounts())
		pending, cmd := pendingFrames(t, w, 30)
		blinked := pending
		vm, tick := vBlinkTick(t, pending)
		next, _ := vm.Update(tick)
		blinked = next.(model)
		_ = cmd
		for _, s := range []struct {
			name string
			m    model
		}{{"blink on", pending}, {"blink off", blinked}} {
			line := s.m.viewLine(s.m.stateCounts())
			if c := sgrAt(line, len("view:")); c != (cell{}) || ansi.Strip(cellAt(line, len("view:"))) != " " {
				t.Fatalf("at %d after v (%s) the cell after 'view:' is drawn: %q", w, s.name, cellAt(line, len("view:")))
			}
			if ansi.Strip(line) != ansi.Strip(idle) || ansi.StringWidth(line) != ansi.StringWidth(idle) {
				t.Fatalf("at %d v changed the line's text or width (%s):\nidle    %q\npending %q", w, s.name, ansi.Strip(idle), ansi.Strip(line))
			}
		}
	}
}

// pickedV says whether v (column 0) is drawn picked: bold accent with a
// steady underline and no fill (r80svs); steady is its look at rest, bold
// accent, no underline, no fill.
func pickedV(line string) (picked, steady bool) {
	c := sgrAt(line, 0)
	accent := c.fg == sgrColour("#00bfff") && c.bold && !c.bg
	return accent && c.underline, accent && !c.underline
}

// a82j3n check 3, as r80svs has it: after the second key or Esc the line
// is at rest again, the view you are in underlined as a word, v back to
// its look, and the last tick asks for no further one.
func TestViewKeysSteadyAfterSecondKey(t *testing.T) {
	m, _ := pendingFrames(t, 140, 40)
	// Leave it in the off phase so a stale phase would show.
	vm, tick := vBlinkTick(t, m)
	next, _ := vm.Update(tick)
	off := next.(model)
	for k, view := range map[string]string{"o": "overview", "l": "list", "b": "board", "!": "lint", "esc": "overview", "x": "overview"} {
		done, tick := vBlinkTick(t, press(off, k))
		for range 3 { // the tick in flight, and any after it
			next, c := done.Update(tick)
			done = next.(model)
			if c != nil {
				t.Fatalf("after v %s the blink keeps ticking", k)
			}
		}
		line := done.viewLine(done.stateCounts())
		checkCues(t, "after v "+k, line, cueState{view: view}, true)
		if _, steady := pickedV(line); !steady {
			t.Fatalf("after v %s v is not back to bold accent: %q", k, cellAt(line, 0))
		}
	}
}

// r80svs check 2 (replaces a82j3n's solid v): while v waits v keeps a
// steady underline, no fill, in both phases; after the second key or Esc
// it is bold accent alone again; at rest it never was underlined.
func TestPickedVIsUnderlined(t *testing.T) {
	m := colourModel(t, 140, "overview")
	if picked, steady := pickedV(m.viewLine(m.stateCounts())); picked || !steady {
		t.Fatalf("idle v is picked %v, steady %v: %q", picked, steady, cellAt(m.viewLine(m.stateCounts()), 0))
	}
	pending, _ := pendingFrames(t, 140, 40)
	vm, tick := vBlinkTick(t, pending)
	next, _ := vm.Update(tick)
	blinked := next.(model)
	for _, s := range []struct {
		name string
		m    model
	}{{"blink on", pending}, {"blink off", blinked}} {
		line := s.m.viewLine(s.m.stateCounts())
		if picked, _ := pickedV(line); !picked {
			t.Fatalf("after v (%s) v is not bold accent with a steady underline: %q", s.name, cellAt(line, 0))
		}
	}
	for _, k := range []string{"l", "esc"} {
		done := press(blinked, k)
		if _, steady := pickedV(done.viewLine(done.stateCounts())); !steady {
			t.Fatalf("after v %s v is not bold accent alone: %q", k, cellAt(done.viewLine(done.stateCounts()), 0))
		}
	}
}

// a82j3n goldens at 100x30 and 160x45, with r80svs's cues: the whole
// screen idle, after v in both blink phases, after v l and v Esc, as
// text, and each frame's view line with its codes visible and a row of
// its cues (cueRow: B bold, _ underlined, U both; a accent, v violet,
// . dim); then the line under NO_COLOR.
func TestViewBlinkGolden(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		name := fmt.Sprintf("%dx%d", size[0], size[1])
		t.Run(name, func(t *testing.T) {
			m := colourModel(t, size[0], "overview")
			m.height = size[1]
			m.resize()
			pending := press(m, "v")
			vm, tick := vBlinkTick(t, pending)
			next, _ := vm.Update(tick)
			blinked := next.(model)
			chosen := press(blinked, "l")
			escaped := press(blinked, "esc")
			var golden []string
			for _, s := range []struct {
				name string
				m    model
			}{{"idle", m}, {"after v, blink on", pending}, {"after v, blink off", blinked}, {"after v l", chosen}, {"after v Esc", escaped}} {
				line := s.m.viewLine(s.m.stateCounts())
				cues, colours := cueRow(line)
				golden = append(golden, "== "+s.name, "view line: "+strconv.Quote(line), "  text:    "+ansi.Strip(line), "  cues:    "+cues, "  colours: "+colours, ansi.Strip(s.m.View()))
			}
			// NO_COLOR (r80svs check 3): the same bold and underlines, no
			// colour, at rest and in both blink phases.
			t.Setenv("NO_COLOR", "1")
			plain := m
			plain.styles = newTheme()
			plainV := press(plain, "v")
			pv, tick := vBlinkTick(t, plainV)
			next, _ = pv.Update(tick)
			plainOff := next.(model)
			for _, s := range []struct {
				name string
				m    model
			}{{"NO_COLOR idle", plain}, {"NO_COLOR after v, blink on", plainV}, {"NO_COLOR after v, blink off", plainOff}} {
				line := s.m.viewLine(s.m.stateCounts())
				cues, colours := cueRow(line)
				golden = append(golden, "== "+s.name, "view line: "+strconv.Quote(line), "  text:    "+ansi.Strip(line), "  cues:    "+cues, "  colours: "+colours)
			}
			goldenFile(t, "testdata/header/view-blink-"+name+".txt", strings.Join(golden, "\n")+"\n")
		})
	}
}
