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
// !lint' with v and each view's letter marked, the current view in its
// selection colour, then the state counts right-aligned after a dim │.
// While v waits the whole line lights up. When the line is tight the
// counts drop from the right first (dropped, done …), then the separator.
func (m model) viewLine(summary []string) string {
	base := m.styles.dim
	if m.pendingV {
		base = m.styles.accent // the line its prefix belongs to (qh9ep1 check 3)
	}
	key := m.styles.accent.Underline(true)
	if m.pendingV {
		key = m.styles.forYou.Bold(true).Underline(true)
	}
	parts := []string{}
	for _, v := range views {
		style := base
		if v.name == m.view && !m.detail && !m.pendingV {
			style = m.styles.selected
		}
		label := ""
		if i := strings.Index(v.name, v.key); i >= 0 {
			label = style.Render(v.name[:i]) + key.Render(v.key) + style.Render(v.name[i+1:])
		} else {
			label = key.Render(v.key) + style.Render(v.name)
		}
		parts = append(parts, label)
	}
	left := key.Render("v") + base.Render("iew: ") + strings.Join(parts, base.Render("  "))
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
