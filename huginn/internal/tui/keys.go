package tui

import (
	"strings"

	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func binding(keys []string, label, description string) bkey.Binding {
	return bkey.NewBinding(bkey.WithKeys(keys...), bkey.WithHelp(label, description))
}

var keys = struct{ down, up, top, bottom, prev, next bkey.Binding }{
	binding([]string{"j", "down"}, "j/k ↑/↓", "move / scroll"),
	binding([]string{"k", "up"}, "k/↑", "up"),
	binding([]string{"<", "home"}, "< / gg / Home", "top"),
	binding([]string{"G", ">", "end"}, "> / G / End", "bottom"),
	binding([]string{"h", "left"}, "h/←", "previous section / pane"),
	binding([]string{"l", "right"}, "l/→", "next section / pane"),
}

func hint(label, description string) bkey.Binding {
	return binding([]string{label}, label, description)
}
func (m model) keyBar() string {
	var hints []bkey.Binding
	switch {
	case m.help:
		if m.helpTyping {
			hints = []bkey.Binding{hint("type", "find"), hint("Enter", "keep"), hint("Esc", "clear")}
			break
		}
		hints = []bkey.Binding{hint("/", "find"), hint("j/k", "scroll"), hint("</>", "ends"), hint("PgUp/PgDn", "page"), hint("Esc", "back"), hint("?", "close")}
		if m.prompt == "" {
			hints = append(hints, hint("q", "quit"))
		}
	case m.confirm != "":
		hints = []bkey.Binding{hint("Enter", "confirm"), hint("Esc", "cancel"), hint("?", "help"), hint("q", "quit")}
	case m.grab.active:
		// Only the keys that work while a task is grabbed (3xcesq).
		hints = []bkey.Binding{hint("j/k", "move"), hint("g/G", "top/bottom"), hint("Enter", "drop"), hint("Esc", "cancel")}
	case m.yanked != "" && m.prompt == "" && !m.detail:
		hints = []bkey.Binding{hint("j/k", "select"), hint("P", "paste above"), hint("p", "paste below"), hint("Esc", "forget "+m.yanked)}
	case m.prompt == ":":
		hints = []bkey.Binding{hint("Tab/Enter", "complete/run"), hint("↑↓ ^n/^p", "pick"), hint("Esc", "close"), hint("?", "help")}
	case m.prompt == "#":
		hints = []bkey.Binding{hint("Enter", "open"), hint("Tab", "complete"), hint("↑↓ ^n/^p", "pick"), hint("Esc", "back"), hint("?", "help")}
	case m.prompt == "filter":
		hints = []bkey.Binding{hint("j/k ↑↓", "pick"), hint("Enter", "add chip"), hint("type", "narrow"), hint("Esc", "back"), hint("?", "help")}
	case m.prompt == "sort":
		hints = []bkey.Binding{hint("letter", "add / flip"), hint("⌫", "remove last"), hint("Enter", "apply"), hint("Esc", "keep order"), hint("?", "help")}
	case m.prompt == "/":
		hints = []bkey.Binding{hint("↑↓", "pick"), hint("Enter", "go to it"), hint("Tab", "keep as filter"), hint("Esc", "clear"), hint("?", "help")}
	case m.prompt != "":
		hints = []bkey.Binding{hint("Enter", "submit"), hint("Esc", "cancel"), hint("?", "help")}
	case m.detail:
		hints = m.detailHints(m.width < 100)
	default:
		hints = []bkey.Binding{hint("j/k", "move"), hint("Enter", "open"), hint("h/l", "pane"), hint("f/#", "filter/id"), hint("/", "find"), hint(":", "cmd"), hint("?", "help"), hint("q", "quit"), hint("Tab/o", "column/order")}
	}
	if m.width < 100 && !m.help && m.prompt == "" && m.confirm == "" && !m.detail && !m.grab.active && m.yanked == "" {
		hints = []bkey.Binding{hint("j/k", "move"), hint("Enter", "open"), hint("f/#", "find"), hint("/", "search"), hint(":", "cmd"), hint("?", "help"), hint("q", "quit")}
	}
	h := m.helpModel
	h.ShortSeparator = " · "
	// Never wider than the terminal: the help model drops trailing hints
	// and shows "…" (2026-11-16: the footer ran to 75 columns at width 70).
	h.Width = m.width
	bar := h.ShortHelpView(hints)
	return m.withVersion(bar)
}

// withVersion ends the key bar with "hug vX.Y.Z", right-aligned and dim,
// when the hints leave room for it and two spaces. On a narrow line the
// version is dropped first: it never pushes a hint off (Jørgen, 2026-11-16).
func (m model) withVersion(bar string) string {
	corner := m.corner()
	if corner == "" {
		return bar
	}
	gap := m.width - lipgloss.Width(bar) - lipgloss.Width(corner)
	if gap < 2 && m.newer != "" {
		// A newer hug is news: it takes the room of the last hints.
		bar = ansi.Truncate(bar, max(0, m.width-lipgloss.Width(corner)-3), "…")
		gap = m.width - lipgloss.Width(bar) - lipgloss.Width(corner)
	}
	if gap < 2 {
		return bar
	}
	style := m.styles.dim
	if m.newer != "" {
		style = m.styles.accent
	}
	return bar + strings.Repeat(" ", gap) + style.Render(corner)
}

// detailHints are the keys that work in the open task with its current
// focus, and only those: j/k say what they move, Space shows only with the
// checks focused, Enter only with dependencies, Tab only when there is
// somewhere to go (2026-11-16, user).
func (m model) detailHints(narrow bool) []bkey.Binding {
	t, _ := m.selected()
	var hints []bkey.Binding
	if m.glance {
		return []bkey.Binding{hint("j/k", "scroll"), hint("n/p", "file"), hint("Enter", "edit"), hint("u/Esc", "files"), hint("?", "help"), hint("q", "quit")}
	}
	switch m.detailPane {
	case focusAcceptance:
		hints = append(hints, hint("j/k", "check"), hint("Space", "tick"))
		// - strikes the check out (sfa8w4); on a narrower bar the
		// Acceptance heading and ? help say so, so no other hint is lost.
		if m.width >= 120 {
			hints = append(hints, hint("-", "strike out"))
		}
	case focusDeps:
		hints = append(hints, hint("j/k", "dependency"), hint("Enter", "follow"))
	case focusLinks:
		hints = append(hints, hint("j/k", "link"), hint("Enter", "follow"))
	case focusIDs:
		hints = append(hints, hint("j/k", "scroll"), hint("Enter", "open"))
	case focusFiles:
		hints = append(hints, hint("j/k", "file"), hint("Space", "show"), hint("Enter", "edit"))
	default:
		hints = append(hints, hint("j/k", "scroll"))
	}
	// The task ids the detail shows, one Tab each (4bwcgp): right after
	// the focus's keys, so a narrow bar keeps it.
	if len(m.idStops(t)) > 0 {
		hints = append(hints, hint("Tab", "next id"))
	}
	// Every key that writes is named here, right after the focus's own
	// keys so a narrow bar cuts navigation first (x1ra0t); each asks
	// first, in a confirm line or a prompt, so nothing changes unseen.
	if m.forYou(t) {
		hints = append(hints, hint("a", "answer")) // the question asks you (3jtfw9)
	}
	hints = append(hints, hint("t/d/x/e", "take/done/drop/edit"))
	if !narrow {
		hints = append(hints, hint("N", "new"))
	}
	hints = append(hints, hint("u", "up"), hint("n/p", "task"))
	if len(m.focusStops(t)) > 1 && (m.width >= 160 || len(m.idStops(t)) == 0) {
		// With Tab next id on the bar, h/l is named in ? help below 160,
		// so S screen still fits at 140 (5z4zt4).
		hints = append(hints, hint("h/l", "section"))
	}
	hints = append(hints, hint("J/K", "½ page"))
	if len(m.filesOf(t)) > 0 {
		hints = append(hints, hint("E", "files"))
	}
	if m.width >= 140 {
		// S copies the screen and asks to attach it (5z4zt4); on a
		// narrower bar ? help names it, as - (sfa8w4).
		hints = append(hints, hint("S", "screen"))
	}
	hints = append(hints, hint("Esc", "back"))
	if !narrow {
		hints = append(hints, hint(":", "cmd"))
	}
	return append(hints, hint("?", "help"), hint("q", "quit"))
}
