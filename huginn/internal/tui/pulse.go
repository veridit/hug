package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// The focus sign pulses (q3sh69, Jørgen 2026-11-20 11:49Z and 12:10Z):
// one blinking thing on screen. Where you are positioned, the > before
// the current cell (⋗ on a marked row, the > of a focused row in a
// task's page) pulses like a cursor at the box cursor's 530 ms, with no
// underline: an underline reads as a key you can press. Wherever you can
// type (a prompt, the answer box, the / search) the text cursor blinks
// and the sign is on and still. Never both.
//
// The pulse keeps the glyph and dims it (crab 2026-11-21 11:06Z): on is
// the accent, bold; off the dim grey; under NO_COLOR bold, then faint. So
// the plain frame never changes and you never lose your place.
//
// Its cost: one tea.Tick chain, with a generation so a stale chain dies;
// no tick at all while a prompt, box or v holds the cursor, nor where no
// sign is drawn (help, a file, the editor, a grab, a page with no focused
// row). A key shows the sign on through the next tick, as typing does a
// text cursor, without starting a timer. A tick draws the frame again
// but never re-renders a page: its sign is patched in at View time.

// pulseMsg is one tick of the pulse; gen is the chain it belongs to.
type pulseMsg struct{ gen int }

// pulseEvery is the pulse's half period, the box cursor's blink.
const pulseEvery = 530 * time.Millisecond

// pulse is the state of the focus sign's pulse.
type pulse struct {
	live    bool // the program runs it (startModel); a model in a test, only when it says so
	running bool // a tick is on its way
	gen     int  // the running chain's generation
	off     bool // the sign is in its dim phase
	hold    bool // a key: the next tick keeps the sign on
}

// tick is the next tick of chain gen.
func (p pulse) tick() tea.Cmd {
	gen := p.gen
	return tea.Tick(pulseEvery, func(time.Time) tea.Msg { return pulseMsg{gen} })
}

// wantsPulse says whether the sign pulses now: it is drawn, and no text
// cursor holds the screen.
func (m model) wantsPulse() bool {
	return m.pulse.live && m.pulsing()
}

// pulsing says whether the sign moves here: no prompt or box takes text
// (their cursor blinks), v waits for no second key (its underlines
// blink), and a sign is on screen.
func (m model) pulsing() bool {
	switch {
	case m.prompt != "" || m.pendingV || m.help || m.editing || m.grab.active:
		return false
	case m.detail:
		if m.glance || m.detailPane == focusPage {
			return false
		}
		if m.detailPane == focusIDs {
			// An id is marked ▸ where it stands; only a child's row has
			// a > to pulse.
			t, ok := m.selected()
			if !ok {
				return false
			}
			s, ok := m.markedStop(t)
			return ok && s.where == stopChildren
		}
		return true
	}
	return true
}

// onPulse is one tick: the sign turns, or stays on after a key, and the
// next tick is asked for; where the sign no longer pulses the chain ends
// and the sign is on.
func (m model) onPulse(msg pulseMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.pulse.gen || !m.pulse.running {
		return m, nil // a chain that ended
	}
	if !m.wantsPulse() {
		m.pulse.running, m.pulse.off, m.pulse.hold = false, false, false
		return m, nil
	}
	if m.pulse.hold {
		m.pulse.hold = false
	} else {
		m.pulse.off = !m.pulse.off
	}
	return m, m.pulse.tick()
}

// afterPulse runs after every other message: a key shows the sign on; a
// text cursor taking over leaves it on and still; a sign that should
// pulse and has no chain starts one.
func (m *model) afterPulse(msg tea.Msg) tea.Cmd {
	if !m.wantsPulse() {
		m.pulse.off, m.pulse.hold = false, false
		return nil
	}
	if _, key := msg.(tea.KeyMsg); key {
		m.pulse.off, m.pulse.hold = false, true
	}
	if m.pulse.running {
		return nil
	}
	m.pulse.running, m.pulse.gen = true, m.pulse.gen+1
	return m.pulse.tick()
}

// plainCues says the renderer writes no attribute (NO_COLOR): the sign's
// bold and faint are written by hand, as the view line's cues are.
func (m model) plainCues() bool {
	return m.styles.noColor && m.styles.renderer.ColorProfile() == termenv.Ascii
}

// signIn is glyph as the focus sign draws it, on or off.
func (m model) signIn(glyph string, off bool) string {
	if m.plainCues() {
		if off {
			return "\x1b[2m" + glyph + "\x1b[0m"
		}
		return "\x1b[1m" + glyph + "\x1b[0m"
	}
	if off {
		return m.styles.dim.Render(glyph)
	}
	return m.styles.rowCursor.Render(glyph)
}

// sign is glyph as the focus sign, in the pulse's phase now.
func (m model) sign(glyph string) string { return m.signIn(glyph, m.pulse.off) }

// pulsePage dims the sign of a task's page in its off phase. The page is
// rendered with the sign on (cursorMark); the one line that starts with
// it is patched here, so a tick re-renders no markdown.
func (m model) pulsePage(page string) string {
	if !m.pulse.off {
		return page
	}
	on, off := m.signIn(">", false), m.signIn(">", true)
	if on == off {
		return page
	}
	lines := strings.Split(page, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, on) {
			lines[i] = off + l[len(on):]
			break
		}
	}
	return strings.Join(lines, "\n")
}

// styleInputCursor draws the one-line input's cursor as the box's (q3sh69,
// Jørgen's rule: wherever you can type, a blinking underscore): the
// underlined accent at 530 ms, never a reversed block (styleAnswerCursor).
func styleInputCursor(in *textinput.Model, cur lipgloss.Style) {
	in.Cursor.Style = lipgloss.NewStyle().Transform(func(cell string) string {
		return "\x1b[0m" + cur.Render(ansi.Strip(cell))
	})
}

// focusRow is a focused row of a task's page drawn in the selection tint,
// its leading > the focus sign outside the tint, so it pulses as the
// list's does.
func (m model) focusRow(label string) string {
	lines := strings.Split(label, "\n")
	for i, l := range lines {
		if i == 0 && strings.HasPrefix(l, ">") {
			lines[i] = m.signIn(">", false) + m.styles.selected.Render(l[1:])
		} else {
			lines[i] = m.styles.selected.Render(l)
		}
	}
	return strings.Join(lines, "\n")
}
