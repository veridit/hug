package tui

import (
	"fmt"
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

// q3sh69, Jørgen's rule (2026-11-20 11:49Z on 911j70, 12:10Z on bxcsmk):
// one blinking thing on screen. Where you are positioned (the > before
// the current cell, ⋗ on a marked row, the > in a task's page) the sign
// pulses at the box cursor's 530 ms, with no underline: an underline
// reads as a key you can press. Wherever you can type, the text cursor
// blinks and the sign is still. Never both.

// accentFg and dimFg are the sign's two phases in true colour on a dark
// background: on, the accent, bold; off, the dim grey.
const accentFg, dimFg = "38;2;0;191;255", "38;2;135;149;168"

// signLine is the line of m's frame holding the focus sign in its first
// column, before the selected task's id, or the page line holding want.
func signLine(t *testing.T, m model, want string) (string, int) {
	t.Helper()
	for _, l := range strings.Split(m.View(), "\n") {
		plain := ansi.Strip(l)
		if strings.Contains(plain, want) {
			col := strings.IndexAny(plain, ">⋗")
			if col < 0 {
				t.Fatalf("no sign on %q", plain)
			}
			return l, utf8.RuneCountInString(plain[:col])
		}
	}
	t.Fatalf("no line holds %q:\n%s", want, ansi.Strip(m.View()))
	return "", 0
}

// pulseOnce delivers the tick the pulse waits for.
func pulseOnce(m model) (model, tea.Cmd) {
	next, cmd := m.Update(pulseMsg{gen: m.pulse.gen})
	return next.(model), cmd
}

// pulseOff is m after the pulse's first two ticks: a key holds the sign
// on through the next tick, the one after turns it off.
func pulseOff(t *testing.T, m model) model {
	t.Helper()
	held, again := pulseOnce(m)
	if again == nil || held.pulse.off {
		t.Fatalf("the tick after a key: off %v, next asked %v; want on, and the next tick asked", held.pulse.off, again != nil)
	}
	off, again := pulseOnce(held)
	if again == nil {
		t.Fatal("the pulse stopped after two ticks")
	}
	return off
}

// phase checks the sign at col of line: on is bold in the accent, off is
// the dim grey without bold; never underlined, never filled.
func phase(t *testing.T, where, line string, col int, on bool) {
	t.Helper()
	c := sgrAt(line, col)
	if c.underline || c.bg {
		t.Fatalf("%s: the sign is underlined or filled: %q", where, cellAt(line, col))
	}
	if on && (!c.bold || c.fg != accentFg) {
		t.Fatalf("%s: the sign should be on (bold accent), is bold=%v fg=%q: %q", where, c.bold, c.fg, cellAt(line, col))
	}
	if !on && (c.bold || c.fg != dimFg) {
		t.Fatalf("%s: the sign should be off (dim), is bold=%v fg=%q: %q", where, c.bold, c.fg, cellAt(line, col))
	}
}

// q3sh69 check 4: in overview, list and board the > before the current
// cell (or ⋗ on a marked row) pulses at 530 ms with no underline; each
// tick asks for the next; while a prompt takes text it is still and the
// pulse asks for no tick.
func TestFocusSignPulsesInLists(t *testing.T) {
	if pulseEvery != 530*time.Millisecond {
		t.Fatalf("pulse %v, want the box cursor's 530ms", pulseEvery)
	}
	for _, view := range []string{"overview", "list", "board"} {
		for _, marked := range []bool{false, true} {
			for _, column := range []bool{false, true} {
				m := gridModel(t, 100, 30, view)
				if !column {
					m.column = -1
				}
				m.pulse.live = true
				sel, _ := m.selected()
				if marked {
					m.marks = []string{sel.ID}
				}
				m = press(m, "j", "k") // a key: the pulse is armed, the sign on
				sel, _ = m.selected()
				where := fmt.Sprintf("%s marked=%v column=%v", view, marked, column)
				line, col := signLine(t, m, sel.ID)
				glyph := []rune(ansi.Strip(line))[col]
				if marked && !column && glyph != '⋗' || (!marked || column) && glyph != '>' {
					t.Fatalf("%s: sign %q", where, string(glyph))
				}
				phase(t, where+" on", line, col, true)
				off := pulseOff(t, m)
				line, col = signLine(t, off, sel.ID)
				phase(t, where+" off", line, col, false)
				if []rune(ansi.Strip(line))[col] != glyph {
					t.Fatalf("%s: the off phase changed the glyph: %q", where, ansi.Strip(line))
				}
				back, again := pulseOnce(off)
				line, col = signLine(t, back, sel.ID)
				phase(t, where+" back on", line, col, true)
				if again == nil {
					t.Fatalf("%s: the pulse stopped after two ticks", where)
				}
				// A prompt takes text: the sign is on and still, and the
				// pulse's next tick asks for none.
				typing := press(off, "/")
				line, col = signLine(t, typing, sel.ID)
				phase(t, where+" typing", line, col, true)
				still, again := pulseOnce(typing)
				if again != nil {
					t.Fatalf("%s: the sign keeps ticking while / takes text", where)
				}
				line, col = signLine(t, still, sel.ID)
				phase(t, where+" typing, a tick later", line, col, true)
				// Esc: the pulse is armed again.
				if _, again := key(still, "esc"); again == nil {
					t.Fatalf("%s: Esc left the pulse unarmed", where)
				}
			}
		}
	}
}

// q3sh69 check 1: in a task's page the > of the focused row (an asked
// check, the question, a plain check, a dependency) pulses with no
// underline; with the answer box open it is still and only the box
// cursor blinks.
func TestFocusSignPulsesInPage(t *testing.T) {
	for _, c := range []struct{ id, want string }{
		{"aaaaaa", "> ? ☐ 2. b"},
		{"cccccc", "> ? which colour should the ASKED title"},
		{"ffffff", "> ☑ 1. a"},
	} {
		m := askedBoard(t)
		m.width, m.height = 100, 40
		m.resize()
		m.styles.renderer.SetColorProfile(termenv.TrueColor)
		m.styles.renderer.SetHasDarkBackground(true)
		styleAnswerCursor(&m.answerBox, m.styles.cursor)
		m.pulse.live = true
		m = openTask(t, m, c.id)
		line := lineWith(t, m, c.want)
		phase(t, c.id+" on", line, 0, true)
		off := pulseOff(t, m)
		phase(t, c.id+" off", lineWith(t, off, c.want), 0, false)
		if c.id != "aaaaaa" && c.id != "cccccc" {
			continue // a box answers a question only
		}
		// The box open: its cursor blinks; the > is on and still.
		boxed, cmd := key(off, "a")
		if boxed.prompt != answerPrompt || cmd == nil {
			t.Fatalf("%s: a opened %q, blink %v", c.id, boxed.prompt, cmd != nil)
		}
		phase(t, c.id+" box open", lineWith(t, boxed, c.want), 0, true)
		still, again := pulseOnce(boxed)
		if again != nil {
			t.Fatalf("%s: the > keeps ticking with the box open", c.id)
		}
		phase(t, c.id+" box open, a tick later", lineWith(t, still, c.want), 0, true)
		b, tick := blinkTick(t, still)
		blinked, again := b.Update(tick)
		if again == nil || blinked.(model).answerLines()[0] == b.answerLines()[0] {
			t.Fatalf("%s: the box cursor does not blink", c.id)
		}
		phase(t, c.id+" box blinked", lineWith(t, blinked.(model), c.want), 0, true)
	}
}

// inputTick is the blink the one-line input's cursor waits for, taken
// from its own blink command without waiting (as blinkTick).
func inputTick(t *testing.T, m model) (model, tea.Msg) {
	t.Helper()
	c := m.input.Cursor
	c.BlinkSpeed = 0
	msg := c.BlinkCmd()()
	if _, ok := msg.(cursor.BlinkMsg); !ok {
		t.Fatalf("blink command gave %T", msg)
	}
	c.BlinkSpeed = 530 * time.Millisecond
	m.input.Cursor = c
	return m, msg
}

// q3sh69 (crab 11:06Z, Jørgen's rule): every one-line prompt (/ : # and
// the filter's field) has the blinking underscore, theme.cursor at 530 ms,
// never a reversed block; closed, it blinks no more.
func TestPromptCursorBlinks(t *testing.T) {
	for _, keys := range [][]string{{"/"}, {":"}, {"#"}, {"f", "s"}} {
		m := colourModel(t, 100, "list")
		styleInputCursor(&m.input, m.styles.cursor)
		m.pulse.live = true
		opened, cmd := key(m, keys[0])
		for _, k := range keys[1:] {
			opened, cmd = key(opened, k)
		}
		where := strings.Join(keys, " ")
		if !opened.textInput() {
			t.Fatalf("%s: no text prompt open (%q)", where, opened.prompt)
		}
		if opened.input.Cursor.Mode() != cursor.CursorBlink || opened.input.Cursor.BlinkSpeed != 530*time.Millisecond {
			t.Fatalf("%s: input cursor %v at %v, want blink at 530ms", where, opened.input.Cursor.Mode(), opened.input.Cursor.BlinkSpeed)
		}
		if cmd == nil {
			t.Fatalf("%s: opening asked for no blink", where)
		}
		on := opened.input.View()
		if !strings.Contains(on, "\x1b[4") && !strings.Contains(on, ";4m") && !strings.Contains(on, ";4;") || strings.Contains(on, "7m") {
			t.Fatalf("%s: the cursor is not the underline: %q", where, on)
		}
		o, tick := inputTick(t, opened)
		next, again := o.Update(tick)
		if again == nil || next.(model).input.View() == on {
			t.Fatalf("%s: one tick left the cursor as it was, or stopped it:\n%q\n%q", where, on, next.(model).input.View())
		}
		closed, tick := inputTick(t, press(next.(model), "esc"))
		if keys[0] == "f" {
			closed, tick = inputTick(t, press(closed, "esc"))
		}
		if closed.prompt != "" {
			t.Fatalf("%s: esc left %q open", where, closed.prompt)
		}
		after, c := closed.Update(tick)
		if c != nil && after.(model).pulse.gen == closed.pulse.gen {
			t.Fatalf("%s: a closed prompt keeps blinking", where)
		}
	}
}

// q3sh69, crab 11:06Z: exactly one thing blinks in each state. The sign
// ticks only when no prompt or box holds the cursor; a text cursor blinks
// only while one does.
func TestOneThingBlinks(t *testing.T) {
	type state struct {
		name string
		m    model
	}
	list := colourModel(t, 100, "list")
	styleInputCursor(&list.input, list.styles.cursor)
	list.pulse.live = true
	list = press(list, "j")
	page := askedBoard(t)
	page.width, page.height = 100, 40
	page.resize()
	page.pulse.live = true
	page = openTask(t, page, "aaaaaa")
	results := press(list, "/", "task", "enter")
	if !results.resultsShown() {
		t.Fatalf("/ task Enter shows no results: %q", results.message)
	}
	states := []state{
		{"list", list},
		{"page", page},
		{"search results", results},
		{"/ typing", press(list, "/", "t")},
		{": palette", press(list, ":")},
		{"# jump", press(list, "#")},
		{"filter field", press(list, "f", "s")},
		{"answer box", press(page, "a")},
		{"request box", press(page, "r")},
		{"find in page", press(page, "/")},
		{"v prefix", press(list, "v")},
	}
	for _, s := range states {
		_, again := pulseOnce(s.m)
		sign := again != nil
		text := s.m.answerBox.Focused() && s.m.boxPrompt() ||
			s.m.textInput() && !s.m.boxPrompt() && s.m.input.Focused() && s.m.input.Cursor.Mode() == cursor.CursorBlink ||
			s.m.pendingV
		if sign == text {
			t.Errorf("%s: the sign ticks %v, a text cursor blinks %v: want exactly one", s.name, sign, text)
		}
		if !sign && !s.m.pendingV {
			// The sign is drawn on: nothing about it moves.
			if s.m.pulsing() || s.m.pulse.off && s.m.pulsing() {
				t.Errorf("%s: the sign pulses while a cursor holds text", s.name)
			}
		}
	}
}

// q3sh69 golden: the focus sign's two phases as codes, in the list, the
// board and a task's page, and the sign with the answer box open, at 100.
func TestFocusPulseGolden(t *testing.T) {
	var golden []string
	add := func(name string, m model, want string) {
		line, col := signLine(t, m, want)
		golden = append(golden, name+": "+strconv.Quote(cellAt(line, col)))
	}
	for _, view := range []string{"overview", "list", "board"} {
		m := gridModel(t, 100, 30, view)
		m.column = -1
		m.pulse.live = true
		m = press(m, "j", "k")
		sel, _ := m.selected()
		add(view+" on", m, sel.ID)
		add(view+" off", pulseOff(t, m), sel.ID)
		m.marks = []string{sel.ID}
		add(view+" marked on", m, sel.ID)
		add(view+" marked off", pulseOff(t, m), sel.ID)
	}
	m := askedBoard(t)
	m.width, m.height = 100, 40
	m.resize()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.styles.renderer.SetHasDarkBackground(true)
	styleAnswerCursor(&m.answerBox, m.styles.cursor)
	m.pulse.live = true
	m = openTask(t, m, "aaaaaa")
	add("page on", m, "? ☐ 2. b")
	off := pulseOff(t, m)
	add("page off", off, "? ☐ 2. b")
	boxed := press(off, "a")
	add("page, box open", boxed, "? ☐ 2. b")
	golden = append(golden, "box cursor: "+strconv.Quote(boxed.answerLines()[0]))
	// NO_COLOR: no colour; on is bold, off faint (crab 11:06Z).
	t.Setenv("NO_COLOR", "1")
	plain := gridModel(t, 100, 30, "list")
	plain.styles = newTheme()
	plain.column = -1
	plain.pulse.live = true
	plain = press(plain, "j", "k")
	sel, _ := plain.selected()
	add("NO_COLOR list on", plain, sel.ID)
	add("NO_COLOR list off", pulseOff(t, plain), sel.ID)
	goldenFile(t, "testdata/detail/focus-pulse.txt", strings.Join(golden, "\n")+"\n")
}
