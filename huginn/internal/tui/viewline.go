package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
// One cue per meaning (qh9ep1 check 9, Jørgen 19:54: "the underscore
// tells you it can be picked ... the colour tells you what you can pick";
// "something more subtle"): v and o l b ! are underlined, the one mark of
// a key you can press, and coloured by step of the key trie, v the accent
// and o l b ! violet, none bold. The view you are in is told by bold alone
// on its word; the line is dim and nothing has a background. The colours
// are stable (911j70): v does not recolour the line. While v waits, the
// one cursor (theme.cursor, shared with the answer box, qpp30y) blinks in
// the slot after 'view:', where the second key goes; nothing else
// changes.
func (m model) viewLine(summary []string) string {
	dim := m.styles.dim
	slot := " "
	if m.pendingV && !m.vCursor.Blink { // bubbles: Blink is the hidden phase
		slot = m.styles.cursor.Render(" ")
	}
	parts := []string{}
	for _, view := range views {
		style := dim
		if view.name == m.view && !m.detail {
			style = m.styles.viewCurrent
		}
		key := m.styles.viewKey
		label := ""
		if i := strings.Index(view.name, view.key); i >= 0 {
			label = style.Render(view.name[:i]) + key.Render(view.key) + style.Render(view.name[i+1:])
		} else {
			label = key.Render(view.key) + style.Render(view.name)
		}
		parts = append(parts, label)
	}
	left := m.styles.viewPrefixKey.Render("v") + dim.Render("iew:") + slot + strings.Join(parts, dim.Render("  "))
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
