package tui

import (
	"fmt"
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
	binding([]string{"h", "left", "shift+tab"}, "h/← Shift-Tab", "previous pane / column"),
	binding([]string{"l", "right", "tab"}, "l/→ Tab", "next pane / column"),
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
		hints = []bkey.Binding{hint("j/k", "move"), hint("Enter", "open"), hint("h/l", "pane"), hint("f/#", "filter/id"), hint("/", "find"), hint(":", "cmd"), hint("?", "help"), hint("q", "quit")}
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
	case focusFiles:
		hints = append(hints, hint("j/k", "file"), hint("Enter", "edit"))
	default:
		hints = append(hints, hint("j/k", "scroll"))
	}
	if len(m.filesOf(t)) > 0 {
		hints = append(hints, hint("E", "files"))
	}
	hints = append(hints, hint("n/p", "task"))
	if len(m.focusStops(t)) > 1 {
		hints = append(hints, hint("Tab", "section"))
	}
	hints = append(hints, hint("u", "up"), hint("Esc", "back"))
	if !narrow {
		hints = append(hints, hint(":", "cmd"))
	}
	return append(hints, hint("?", "help"), hint("q", "quit"))
}
func (m model) helpText() string {
	navigation := []bkey.Binding{
		keys.down, keys.top, keys.bottom,
		hint("ctrl-d / ctrl-u", "half page down / up"), hint("PgDn / PgUp", "full page down / up"),
		keys.prev, keys.next, hint("Enter", "open full task detail"),
		hint("Esc", "back one level, never quits"), hint("q / ctrl-c", "quit (q only outside inputs)"),
		hint("/", "live find; Enter goes to it, Tab keeps"), hint(":", "palette with command + value completion"),
		hint("Tab / Enter", "complete palette choice; Enter runs complete command"),
		hint("ctrl-n/p / ↑↓", "select palette completion"), hint("?", "help overlay, including from inputs"),
		hint("#", "jump by id prefix / TASK-number alias"), hint("f", "filter picker: s/t/p/g/l/a/e"),
		hint("1 / 2 / 3 / 4", "filter todo / doing / done / blocked"), hint("Backspace / F", "remove last chip / clear all filters"),
		hint("o / i / b / !", "views: overview / list (also v) / board / lint"),
		hint("s / y", "status / type palette (y: inside a task)"), hint("c", "clear all filters"), hint("r", "refresh now (changes on disk show by themselves)"),
		hint(": sort", "order by fields; : sort rank to go back"),
	}
	if m.newer != "" {
		// R exists only while a newer hug is installed (j2vwk0).
		navigation = append(navigation, hint(reloadKey, "reload into hug "+m.newer+", keeping this place"))
	}
	actions := []bkey.Binding{
		hint("[ and ]", "previous / next check or dependency"), hint("n and p", "next / previous task, in a task"), hint("u", "in a task: up to the list (as Esc)"), hint("Space/-/Enter", "tick / strike out check / follow dep >"),
		hint("Tab / Shift-Tab", "in a task: focus page, checks, deps, links, files"),
		hint("E", "in a task with files (⎘2): its folder in the editor"),
		hint("t / d / x", "take / done / drop task"),
	}
	if len(m.sort) == 0 {
		// Moves change rank, so they show only in rank order (mkzcqr).
		actions = append(actions, hint("J / K", "move below / above the next row, no confirm"),
			hint("m / y, P p", "grab: j/k, Enter places it / yank, paste above below"))
	} else {
		actions = append(actions, hint(": sort rank", "rank order again: then J/K, m, y, P/p move"))
	}
	actions = append(actions,
		hint("N", "new title, goal and one verification item"), hint("e", "edit in $VISUAL, else $EDITOR, git core.editor, vi"),
		hint("e again", "after an edit left a finding: edit at once"),
		hint("Enter / Esc", "confirm / cancel every editing action"),
	)
	if m.helpQuery != "" || m.helpTyping {
		return m.helpSearchText(append(navigation, actions...))
	}
	if m.width >= 130 {
		return m.styles.accent.Render("NAVIGATION AND FIND                                     EDIT · every action requires confirmation") + "\n" + m.helpModel.FullHelpView([][]bkey.Binding{navigation, actions}) + "\n\n" + wrap("PALETTE: overview, list, board, todo, doing, done, blocked, dropped, status <s>, type <t>, priority <p>, group <g>, label <l>, assignee <a>, epic <id>, goto <id>, sort, lint, clear, help", m.width) + "\n" + wrap("Filter fields: s status, t type, p priority, g group, l label, a assignee, e epic. Blocked is a todo task with unmet dependencies; a doing task stays doing, drawn blocked. Each task counts once. Filters combine.", m.width)
	}
	return m.styles.accent.Render("NAVIGATION AND FIND") + "\n" + m.helpModel.FullHelpView([][]bkey.Binding{navigation}) + "\n\n" +
		m.styles.accent.Render("EDIT · every action requires confirmation") + "\n" + m.helpModel.FullHelpView([][]bkey.Binding{actions}) + "\n\n" +
		wrap("PALETTE: "+strings.Join([]string{"overview, list, board, todo, doing, done, blocked, dropped, status <s>, type <t>, group <g>, label <l>, priority <p>, assignee <a>, epic <id>, goto <id>, sort, lint, clear, help"}, ""), m.width) + "\n" +
		wrap("Blocked is derived from unmet dependencies, not a persisted status. Counts overlap todo/doing. Board pages columns to fit. Slug is in detail only. Filters combine.", m.width)
}

// helpSearchText lists the help rows whose key or description contains every
// word of the query, case-insensitively (2026-11-11: find one action fast).
func (m model) helpSearchText(all []bkey.Binding) string {
	words := strings.Fields(strings.ToLower(m.helpQuery))
	var hits []bkey.Binding
	for _, b := range all {
		h := b.Help()
		text := strings.ToLower(h.Key + " " + h.Desc)
		ok := true
		for _, w := range words {
			if !strings.Contains(text, w) {
				ok = false
				break
			}
		}
		if ok {
			hits = append(hits, b)
		}
	}
	cursor := ""
	if m.helpTyping {
		cursor = "█"
	}
	head := m.styles.accent.Render("FIND IN HELP / ") + m.helpQuery + cursor + m.styles.dim.Render(fmt.Sprintf("   %d of %d", len(hits), len(all)))
	if len(hits) == 0 {
		return head + "\n\n" + m.styles.dim.Render("No key matches. Backspace edits, Esc clears.")
	}
	return head + "\n\n" + m.helpModel.FullHelpView([][]bkey.Binding{hits})
}
