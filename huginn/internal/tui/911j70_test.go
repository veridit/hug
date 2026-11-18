package tui

import (
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

// selectionTint is the one selection tint of 911j70 (#2b5a78 on dark) as
// the renderer writes a background.
const selectionTint = "48;2;43;89;120"

// 911j70 check 3: the view line marks the active view, the word on the
// selection tint, in every view and while v waits; the other views'
// words are not tinted. The key colours are stable: after v the line
// from the views on is drawn exactly as before (it no longer lights up
// as a whole), only v and the slot after 'view:' change.
func TestViewLineMarksActiveView(t *testing.T) {
	for _, v := range views {
		m := colourModel(t, 140, v.name)
		idle := m.viewLine(m.stateCounts())
		// The word as drawn: !lint carries its key in front.
		want := v.name
		if !strings.Contains(v.name, v.key) {
			want = v.key + v.name
		}
		words := tintedWords(idle)
		if len(words) != 1 || words[0] != want {
			t.Fatalf("in %s the tinted words are %q, want only %q:\n%q", v.name, words, want, idle)
		}
		pending := press(m, "v")
		lit := pending.viewLine(pending.stateCounts())
		if got := tintedWords(lit); len(got) != 2 || got[0] != "v" || got[1] != words[0] {
			t.Fatalf("in %s after v the tinted words are %q, want v (entered) and %q:\n%q", v.name, got, words[0], lit)
		}
		const head = len("view: ")
		for col := head; col < m.width; col++ {
			if a, b := cellAt(idle, col), cellAt(lit, col); a != b {
				t.Fatalf("in %s v recoloured column %d of the line after 'view: ':\nidle    %q\npending %q", v.name, col, a, b)
			}
		}
		if ansi.Strip(idle) != ansi.Strip(lit) {
			t.Fatalf("in %s v changed the text of the line:\n%q\n%q", v.name, ansi.Strip(idle), ansi.Strip(lit))
		}
	}
}

// tintedWords are the runs of visible text drawn on the selection tint,
// in order: a run ends at the first character off the tint, not at a
// code (a key letter in its own colour stays in its word).
func tintedWords(s string) []string {
	var words []string
	var cur strings.Builder
	on := false
	flush := func() {
		if w := strings.TrimSpace(cur.String()); w != "" {
			words = append(words, w)
		}
		cur.Reset()
	}
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := strings.IndexByte(s[i:], 'm')
			if j < 0 {
				break
			}
			code := s[i+2 : i+j]
			switch {
			case strings.Contains(code, selectionTint):
				on = true
			case code == "0" || code == "" || strings.Contains(";"+code+";", ";49;") || strings.Contains(code, "48;"):
				on = false
			}
			i += j + 1
			continue
		}
		if on {
			cur.WriteByte(s[i])
		} else {
			flush()
		}
		i++
	}
	flush()
	return words
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

// 911j70 check 3: after v one underlined cursor (theme.cursor, shared
// with the answer box, qpp30y) stands in the slot after 'view:' where
// the second key goes, and blinks at 530 ms; it goes when the view is
// chosen or v is cancelled, and stops blinking then.
func TestViewCursorBlinksAfterV(t *testing.T) {
	m := colourModel(t, 140, "overview")
	if _, on := cursorSlot(t, m); on {
		t.Fatal("the cursor shows before v")
	}
	m, cmd := key(m, "v")
	if cmd == nil {
		t.Fatal("v returned no command: the cursor's blink never starts")
	}
	if m.vCursor.Mode() != cursor.CursorBlink || m.vCursor.BlinkSpeed != 530*time.Millisecond {
		t.Fatalf("view cursor mode %v speed %v, want blink at 530ms", m.vCursor.Mode(), m.vCursor.BlinkSpeed)
	}
	cell, on := cursorSlot(t, m)
	if !on {
		t.Fatalf("after v the slot after 'view:' is no underlined accent cursor: %q", cell)
	}
	if strings.Contains(cell, "\x1b[7m") || strings.Contains(cell, ";7m") {
		t.Fatalf("the cursor is a reversed block: %q", cell)
	}
	// One tick blinks it off and asks for the next; the next on again.
	m, tick := vBlinkTick(t, m)
	next, again := m.Update(tick)
	off := next.(model)
	if _, on := cursorSlot(t, off); on || again == nil {
		t.Fatalf("one blink tick left the cursor on (%v), or stopped blinking (%v)", on, again == nil)
	}
	off, tick = vBlinkTick(t, off)
	next, again = off.Update(tick)
	if _, on := cursorSlot(t, next.(model)); !on || again == nil {
		t.Fatal("the second tick did not bring the cursor back")
	}
	// Chosen or cancelled, the cursor is gone and its tick asks nothing.
	for _, k := range []string{"o", "esc", "x"} {
		done, tick := vBlinkTick(t, press(m, k))
		if _, on := cursorSlot(t, done); on {
			t.Fatalf("v %s left the cursor on the view line", k)
		}
		if _, c := done.Update(tick); c != nil {
			t.Fatalf("after v %s the cursor keeps blinking", k)
		}
	}
	// The box's blink and the view cursor's do not cross: a tick of the
	// view cursor leaves a closed box alone and the reverse.
	if _, c := m.Update(cursor.BlinkMsg{}); c != nil {
		t.Fatal("a blink no cursor expects was answered")
	}
}

// 911j70 check 3 golden: the view line at 80 and 140 in the overview,
// idle, after v (cursor on, then off one blink later) and after v l, as
// text and with its codes visible.
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
	}
	goldenFile(t, "testdata/header/view-cursor.txt", strings.Join(golden, "\n")+"\n")
}
