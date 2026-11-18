package tui

import (
	"fmt"
	"os"
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
	if mockOn() {
		return m.fitViewLine(m.mockViewLeft(), summary)
	}
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
	return m.fitViewLine(left, summary)
}

// mockViewLeft is the PROTOTYPE (mock/grid) view line: the two steps of
// the key trie in two stable colours, v the first (accent cyan, bold), o l
// b ! the second (violet). The line never recolours as a whole. While v
// waits, v shows as entered (the selection tint, as a typed key) and the
// one "you are here" cursor (Jørgen 11:43Z: an underline that blinks
// gently, the only motion on the screen; shared with the answer box,
// qpp30y) stands in the slot after "view:", where the second key goes.
// It goes when the second key is chosen. MOCKGRID_VNEXT=letters is the
// alternative: the four second-step letters underlined, static.
func (m model) mockViewLeft() string {
	dim, cur := m.styles.dim, m.styles.viewCurrent
	vkey := m.styles.viewPrefixKey.Render("v")
	if m.pendingV {
		vkey = m.styles.rowHeader.Render("v")
	}
	letters := os.Getenv("MOCKGRID_VNEXT") == "letters"
	parts := []string{}
	for _, v := range views {
		style := dim
		if v.name == m.view && !m.detail {
			style = cur
		}
		k := m.styles.viewKey
		if m.pendingV && letters {
			k = m.styles.viewNext
		}
		if v.name == m.view && !m.detail {
			// The view you are in: the whole word on the selection tint
			// (the same tint as the selected id and column heading), its
			// key letter keeping its step-2 colour on it.
			k = k.Background(m.styles.viewCurrent.GetBackground())
		}
		label := ""
		if i := strings.Index(v.name, v.key); i >= 0 {
			label = style.Render(v.name[:i]) + k.Render(v.key) + style.Render(v.name[i+1:])
		} else {
			label = k.Render(v.key) + style.Render(v.name)
		}
		parts = append(parts, label)
	}
	slot := " "
	if m.pendingV && !letters {
		slot = m.cursorCell(" ")
	}
	return vkey + dim.Render("iew:") + slot + strings.Join(parts, dim.Render("  "))
}

// cursorCell draws text as the one "you are here" cursor: underlined in
// the cursor colour while the blink is on, plain while it is off.
func (m model) cursorCell(text string) string {
	if m.vBlinkOff {
		return text
	}
	return m.styles.cursor.Render(text)
}

// fitViewLine puts the state counts right of left, dropping them from the
// right while the line is tight.
func (m model) fitViewLine(left string, summary []string) string {
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
