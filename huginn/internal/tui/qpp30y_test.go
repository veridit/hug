package tui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// designAsked is the stage board with eeeeee asking jhf a design question,
// opened on it at w x 30.
func designAsked(t *testing.T, w int) model {
	t.Helper()
	m := stageBoard(t)
	m.opts.User = "jhf"
	m.width, m.height = w, 30
	m.resize()
	oldNow := task.Now
	task.Now = func() time.Time { return time.Date(2026, 11, 17, 9, 45, 0, 0, time.Local) }
	t.Cleanup(func() { task.Now = oldNow })
	if _, e := m.store.Design("eeeeee", "drop type from the rows, keep it, or show it only when not feature?", "crab", "jhf"); e != nil {
		t.Fatal(e)
	}
	m.refresh("")
	m.setView("list")
	m = press(m, "#", "eeeeee", "enter")
	if !m.detail {
		t.Fatal("detail not open")
	}
	return m
}

// qpp30y check 1: a and r draw the box, label and cursor, in the frame
// the key is pressed, before anything is typed.
func TestBoxShowsAtOnce(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, w := range []int{80, 140} {
		m := designAsked(t, w)
		a := press(m, "a")
		if a.prompt != answerPrompt {
			t.Fatalf("at %d a: prompt %q", w, a.prompt)
		}
		if screen := ansi.Strip(a.View()); !strings.Contains(screen, "  answer: ") {
			t.Fatalf("at %d a drew no box before a key:\n%s", w, screen)
		}
		r := press(press(m, "esc"), "#", "eeeeee", "enter", "r")
		if r.prompt != requestPrompt {
			t.Fatalf("at %d r: prompt %q", w, r.prompt)
		}
		if screen := ansi.Strip(r.View()); !strings.Contains(screen, "request: ") {
			t.Fatalf("at %d r drew no box before a key:\n%s", w, screen)
		}
	}
}

// qpp30y check 2: the box's cursor is theme.cursor, an underlined bold
// accent, and blinks at bubbles' 530 ms; its blink keeps going while the
// box is open and stops when it closes. No other input blinks.
func TestBoxCursorBlinksUnderline(t *testing.T) {
	m := designAsked(t, 100)
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.styles.renderer.SetHasDarkBackground(true)
	m, cmd := key(m, "a")
	if m.answerBox.Cursor.Mode() != cursor.CursorBlink || m.answerBox.Cursor.BlinkSpeed != 530*time.Millisecond {
		t.Fatalf("box cursor mode %v speed %v, want blink at 530ms", m.answerBox.Cursor.Mode(), m.answerBox.Cursor.BlinkSpeed)
	}
	if cmd == nil {
		t.Fatal("a returned no command: the cursor's blink never starts")
	}
	if m.input.Cursor.Mode() == cursor.CursorBlink {
		t.Fatal("the one-line input blinks too: only the box cursor moves")
	}
	on := m.answerLines()[0]
	if !strings.Contains(on, "\x1b[4") && !strings.Contains(on, ";4m") && !strings.Contains(on, ";4;") {
		t.Fatalf("cursor is not underlined: %q", on)
	}
	if !strings.Contains(on, "38;2;0;191;255") {
		t.Fatalf("cursor is not the accent: %q", on)
	}
	if strings.Contains(on, "\x1b[7m") || strings.Contains(on, ";7m") {
		t.Fatalf("cursor is a reversed block: %q", on)
	}
	// The blink: a BlinkMsg the box expects flips it off, and asks for
	// the next.
	m, tick := blinkTick(t, m)
	next, again := m.Update(tick)
	off := next.(model)
	if off.answerLines()[0] == on || again == nil {
		t.Fatalf("one blink tick left the cursor as it was, or stopped blinking:\n%q\n%q", on, off.answerLines()[0])
	}
	if !strings.Contains(ansi.Strip(off.viewport.View()), "answer: ") || off.viewport.View() == m.viewport.View() {
		t.Fatal("the page was not drawn again for the blink")
	}
	// Closed, the box no longer blinks: its tick asks for nothing.
	closed, tick := blinkTick(t, press(off, "esc"))
	if _, c := closed.Update(tick); c != nil {
		t.Fatal("a closed box keeps blinking")
	}
}

// blinkTick is the blink message the box's cursor waits for, taken from
// its own blink command without waiting, and m expecting it (blink speed
// 0: no timer, no clock; vBlinkTick, 6zhyq2).
func blinkTick(t *testing.T, m model) (model, tea.Msg) {
	t.Helper()
	c := m.answerBox.Cursor
	c.BlinkSpeed = 0
	msg := c.BlinkCmd()()
	if _, ok := msg.(cursor.BlinkMsg); !ok {
		t.Fatalf("blink command gave %T", msg)
	}
	c.BlinkSpeed = 530 * time.Millisecond
	m.answerBox.Cursor = c
	return m, msg
}

// qpp30y check 2 golden: the open answer box at 80 and 140, as text,
// then its first line with the cursor's codes made visible (on, and off
// one blink later).
func TestBoxGolden(t *testing.T) {
	var golden []string
	for _, w := range []int{80, 140} {
		m := designAsked(t, w)
		m.styles.renderer.SetColorProfile(termenv.TrueColor)
		m.styles.renderer.SetHasDarkBackground(true)
		m = press(m, "a")
		lines := strings.Split(ansi.Strip(m.View()), "\n")
		var top []string
		for _, l := range lines[:12] {
			top = append(top, strings.TrimRight(l, " "))
		}
		golden = append(golden, "== answer box at "+strconv.Itoa(w), strings.Join(top, "\n"))
		m, tick := blinkTick(t, m)
		next, _ := m.Update(tick)
		golden = append(golden, "cursor on:  "+strconv.Quote(m.answerLines()[0]), "cursor off: "+strconv.Quote(next.(model).answerLines()[0]))
	}
	goldenFile(t, "testdata/detail/answer-box.txt", strings.Join(golden, "\n")+"\n")
}
