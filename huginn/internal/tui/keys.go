package tui

import (
	"strings"

	bkey "github.com/charmbracelet/bubbles/key"
)

func binding(keys []string, label, description string) bkey.Binding {
	return bkey.NewBinding(bkey.WithKeys(keys...), bkey.WithHelp(label, description))
}

var keys = struct{ down, up, top, bottom, prev, next bkey.Binding }{
	binding([]string{"j", "down"}, "j/k ↑/↓", "move / scroll"),
	binding([]string{"k", "up"}, "k/↑", "up"),
	binding([]string{"<", "home"}, "gg / < / Home", "top"),
	binding([]string{"G", ">", "end"}, "G / > / End", "bottom"),
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
		hints = []bkey.Binding{hint("j/k", "scroll"), hint("gg/G", "ends"), hint("PgUp/PgDn", "page"), hint("Esc", "back"), hint("?", "close")}
		if m.prompt == "" {
			hints = append(hints, hint("q", "quit"))
		}
	case m.confirm != "":
		hints = []bkey.Binding{hint("Enter", "confirm"), hint("Esc", "cancel"), hint("?", "help")}
	case m.prompt == ":":
		hints = []bkey.Binding{hint("Tab/Enter", "complete/run"), hint("↑↓ ^n/^p", "pick"), hint("Esc", "close"), hint("?", "help")}
	case m.prompt == "#":
		hints = []bkey.Binding{hint("Enter", "open"), hint("Tab", "complete"), hint("↑↓ ^n/^p", "pick"), hint("Esc", "back"), hint("?", "help")}
	case m.prompt == "filter":
		hints = []bkey.Binding{hint("j/k ↑↓", "pick"), hint("Enter", "add chip"), hint("type", "narrow"), hint("Esc", "back"), hint("?", "help")}
	case m.prompt == "/":
		hints = []bkey.Binding{hint("Enter", "keep"), hint("Esc", "clear"), hint("?", "help")}
	case m.prompt != "":
		hints = []bkey.Binding{hint("Enter", "submit"), hint("Esc", "cancel"), hint("?", "help")}
	case m.detail:
		hints = []bkey.Binding{hint("j/k", "scroll"), hint("[/]", "select"), hint("Tab", "section"), hint("Enter", "follow"), hint("Esc", "back"), hint(":", "cmd"), hint("?", "help"), hint("q", "quit")}
	default:
		hints = []bkey.Binding{hint("j/k", "move"), hint("Enter", "open"), hint("h/l", "pane"), hint("f/#", "filter/id"), hint("/", "find"), hint(":", "cmd"), hint("?", "help"), hint("q", "quit")}
	}
	h := m.helpModel
	h.ShortSeparator = " · "
	return h.ShortHelpView(hints)
}
func (m model) helpText() string {
	navigation := []bkey.Binding{
		keys.down, keys.top, keys.bottom,
		hint("ctrl-d / ctrl-u", "half page down / up"), hint("PgDn / PgUp", "full page down / up"),
		keys.prev, keys.next, hint("Enter", "open full task detail"),
		hint("Esc", "back one level, never quits"), hint("q / ctrl-c", "quit (q only outside inputs)"),
		hint("/", "live find; Enter keeps, Esc clears"), hint(":", "palette with command + value completion"),
		hint("Tab / Enter", "complete palette choice; Enter runs complete command"),
		hint("ctrl-n/p / ↑↓", "select palette completion"), hint("?", "help overlay, including from inputs"),
		hint("#", "jump by id prefix / TASK-number alias"), hint("f", "filter: s status, t type, p priority, g group, l label, o owner"),
		hint("1 / 2 / 3 / 4", "filter todo / doing / done / blocked"), hint("Backspace / F", "remove last chip / clear all filters"),
		hint("o / v / b / !", "overview / list / board / lint"),
		hint("s / y", "status / type palette shortcut"), hint("c", "clear all filters"), hint("r", "refresh files and lint"),
	}
	actions := []bkey.Binding{
		hint("[ / ]", "previous / next acceptance or dependency in detail"), hint("Space", "toggle acceptance (default first)"),
		hint("Tab + Enter", "focus dependencies and follow selected task"),
		hint("t / d / x", "take / done / drop task"), hint("J / K", "move down / up in global rank order"),
		hint("n", "new title, goal and one verification item"), hint("e", "$EDITOR (default vi), then refresh + lint"),
		hint("Enter / Esc", "confirm / cancel every editing action"),
	}
	return m.styles.accent.Render("NAVIGATION AND FIND") + "\n" + m.helpModel.FullHelpView([][]bkey.Binding{navigation}) + "\n\n" +
		m.styles.accent.Render("EDIT · every action requires confirmation") + "\n" + m.helpModel.FullHelpView([][]bkey.Binding{actions}) + "\n\n" +
		wrap("PALETTE: "+strings.Join([]string{"overview, list, board, todo, doing, done, blocked, dropped, status <s>, type <t>, group <g>, label <l>, priority <p>, owner <o>, goto <id>, lint, clear, help"}, ""), m.width) + "\n" +
		wrap("Blocked is derived from unmet dependencies, not a persisted status. Counts overlap todo/doing. Board pages columns to fit. Slug is in detail only. Filters combine.", m.width)
}
