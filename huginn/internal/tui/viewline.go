package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// views are the four list views under the v prefix (qh9ep1, header-spec.md
// line 2): v and the view's letter pick it, vo vl vb v!. Each is one
// branch of the key trie; no single key switches views any more.
var views = []struct{ key, name string }{{"o", "overview"}, {"l", "list"}, {"b", "board"}, {"!", "lint"}}

// viewPrefix is the footer while v waits for its second key.
func viewPrefix() string {
	parts := []string{}
	for _, v := range views {
		parts = append(parts, v.key+" "+v.name)
	}
	return "v: " + strings.Join(parts, " · ") + " · Esc cancel"
}

// pickViewName is name when it is a view (the palette's view branch).
func pickViewName(name string) (string, bool) {
	for _, v := range views {
		if v.name == name {
			return name, true
		}
	}
	return "", false
}

// pickView is the view the key after v names, if any.
func pickView(k string) (string, bool) {
	for _, v := range views {
		if v.key == k {
			return v.name, true
		}
	}
	return "", false
}

// viewLine is the global band's first line: 'view: overview  list  board
// !lint', then the state counts right-aligned after a dim │. When the
// line is tight the counts drop from the right first (dropped, done …),
// then the separator.
//
// One cue per meaning (r80svs, Jørgen 2026-11-19 11:09Z, 11:11Z; qh9ep1
// check 9). Bold is a key you can press: v o l b !. Colour is the step of
// the key sequence, v in the accent and o l b ! violet, in every state:
// v opens the others. Underline is state: a steady underline under a
// whole word is the view you are in (its key letter too); a steady
// underline under one letter is the key you picked, v while it waits; a
// blinking underline is the keys you choose between now, o l b ! after
// v, on vCursor's ticks (bubbles' 530 ms, a82j3n), the active word's
// other letters keeping theirs. The rest is dim; nothing is filled, and
// there is no cursor cell after 'view:'. With NO_COLOR the bold and
// underlines are written all the same (cue), so only the step is lost.
func (m model) viewLine(summary []string) string {
	dim := m.styles.dim
	prefix, key, current := m.styles.viewPrefixKey, m.styles.viewKey, m.styles.viewCurrent
	choosing := key // the keys you pick between: o l b !
	activeKey := key.Underline(true)
	if m.pendingV {
		prefix = m.styles.viewPicked
		choosing = key.Underline(true)
		if m.vCursor.Blink { // bubbles: Blink is the hidden phase
			choosing = key
		}
		activeKey = choosing
	}
	parts := []string{}
	for _, view := range views {
		style, k := dim, choosing
		if view.name == m.view && !m.detail {
			style, k = current, activeKey
		}
		label := ""
		if i := strings.Index(view.name, view.key); i >= 0 {
			label = m.cue(style, view.name[:i]) + m.cue(k, view.key) + m.cue(style, view.name[i+1:])
		} else {
			label = m.cue(k, view.key) + m.cue(style, view.name)
		}
		parts = append(parts, label)
	}
	left := m.cue(prefix, "v") + dim.Render("iew:") + " " + strings.Join(parts, dim.Render("  "))
	sep := m.styles.dim.Render("│ ")
	for n := len(summary); n > 0; n-- {
		right := strings.Join(summary[:n], m.styles.dim.Render(" · "))
		gap := m.width - lipgloss.Width(left) - lipgloss.Width(sep) - lipgloss.Width(right)
		if gap >= 2 {
			return left + strings.Repeat(" ", gap) + sep + right
		}
	}
	return left
}

// cue is s in style on the view line. Under NO_COLOR the renderer writes
// no attribute at all; the view line's bold and underlines are its cues,
// so they are written all the same (r80svs check 3), with no colour.
func (m model) cue(style lipgloss.Style, s string) string {
	if s == "" || !m.styles.noColor || m.styles.renderer.ColorProfile() != termenv.Ascii {
		return style.Render(s)
	}
	codes := []string{}
	if style.GetBold() {
		codes = append(codes, "1")
	}
	if style.GetUnderline() {
		codes = append(codes, "4")
	}
	if len(codes) == 0 {
		return s
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + s + "\x1b[0m"
}

// stateCounts are the disjoint counts that add up to the tasks shown
// (zaec08): every task has one place; doing names how many of its tasks
// depend on another.
func (m model) stateCounts() []string {
	counts := map[string]int{}
	doingBlocked := 0
	for _, t := range m.visible {
		counts[m.place(t)]++
		if t.State == "doing" && m.blocked[t.Path] {
			doingBlocked++
		}
	}
	summary := []string{}
	for _, s := range boardStates {
		text := fmt.Sprintf("%s %d", s, counts[s])
		if s == "doing" && doingBlocked > 0 {
			text += fmt.Sprintf(" (%d depends)", doingBlocked)
		}
		summary = append(summary, m.styles.status[s].Render(text))
	}
	return summary
}
