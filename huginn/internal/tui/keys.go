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
	binding([]string{"<", "home"}, "< / Home", "top"),
	binding([]string{">", "end"}, "> / End", "bottom"),
	binding([]string{"h", "left"}, "h/←", "previous section / pane"),
	binding([]string{"l", "right"}, "l/→", "next section / pane"),
}

func hint(label, description string) bkey.Binding {
	return binding([]string{label}, label, description)
}
func (m model) keyBar() string {
	// Never wider than the terminal: trailing hints drop and "…" shows
	// (2026-11-16: the footer ran to 75 columns at width 70). A letter key
	// is marked in its word, and neighbours sharing a last word are one
	// item (e3sevr, Jørgen 00:44-00:45): footer.go.
	bar, _ := m.renderHintsShown(m.barHints(), m.width)
	return m.withVersion(bar)
}

// barHints are the hints the key bar offers here, before any are cut.
func (m model) barHints() []bkey.Binding {
	var hints []bkey.Binding
	switch {
	case m.help:
		if m.helpTyping {
			hints = []bkey.Binding{hint("type", "find"), hint("Enter", "keep"), hint("Esc", "clear")}
			break
		}
		hints = []bkey.Binding{hint("/", "find"), hint("j/k", "scroll"), hint("</>", "ends"), hint("PgUp/PgDn", "page"), hint("Esc", "back"), hint(m.helpKey(), "close")}
		if m.prompt == "" {
			hints = append(hints, hint("q", "quit"))
		}
	case m.pendingV:
		// v waits for its view (qh9ep1): only its branches and Esc.
		// The footer above lists the branches (viewPrefix).
		hints = []bkey.Binding{hint("Esc", "cancel"), hint("?", "help")}
	case m.confirm != "":
		hints = []bkey.Binding{hint("Enter", "confirm"), hint("Esc", "cancel"), hint("?", "help"), hint("q", "quit")}
	case m.prompt == capturePrompt:
		// hjbfzd: s and c are on the line above, beside the field.
		hints = []bkey.Binding{hint("Enter", "attach"), hint("Tab", "complete"), hint("Esc", "not attached"), hint(m.helpKey(), "help")}
	case m.prompt == captureTitlePrompt:
		hints = []bkey.Binding{hint("Enter", "create and attach"), hint("Esc", "back"), hint(m.helpKey(), "help")}
	case m.picking && m.prompt == "" && !m.help:
		hints = []bkey.Binding{hint("j/k", "move"), hint("Enter", "attach here"), hint("/", "find"), hint("#", "id"), hint("Esc", "not attached")}
	case m.grab.active:
		// Only the keys that work while a task is grabbed (3xcesq).
		hints = []bkey.Binding{hint("j/k", "move"), hint("< >", "top/bottom"), hint("Enter", "drop"), hint("Esc", "cancel")}
	case m.yanked != "" && m.prompt == "" && !m.detail:
		hints = []bkey.Binding{hint("j/k", "select"), hint("P", "paste above"), hint("p", "paste below"), hint("Esc", "forget "+m.yanked)}
	case m.prompt == ":":
		hints = []bkey.Binding{hint("Tab/Enter", "complete/run"), hint("↑↓ ^n/^p", "pick"), hint("Esc", "up/close"), hint(m.helpKey(), "help")}
	case m.prompt == "#":
		hints = []bkey.Binding{hint("Enter", "open"), hint("Tab", "complete"), hint("↑↓ ^n/^p", "pick"), hint("Esc", "back"), hint(m.helpKey(), "help")}
	case m.prompt == "filter":
		hints = []bkey.Binding{hint("j/k ↑↓", "pick"), hint("Enter", "add chip"), hint("!", "not"), hint("type", "narrow"), hint("Esc", "back"), hint(m.helpKey(), "help")}
	case m.prompt == "sort":
		// The filter picker's movement keys first (17w9r6): the same
		// words in both footers.
		hints = []bkey.Binding{hint("j/k ↑↓", "pick"), hint("Space", "add / flip"), hint("Enter", "apply"), hint("⌫", "remove last"), hint("Esc", "keep order"), hint(m.helpKey(), "help")}
	case m.prompt == findPrompt:
		hints = []bkey.Binding{hint("type", "find"), hint("Enter", "keep"), hint("↑↓", "match"), hint("Esc", "clear"), hint(m.helpKey(), "help")}
	case m.prompt == "/":
		hints = []bkey.Binding{hint("↑↓", "pick"), hint("Enter", "results"), hint("Tab", "keep as filter"), hint("Esc", "cancel"), hint(m.helpKey(), "help")}
	case m.boxPrompt():
		hints = []bkey.Binding{hint("Enter", "send"), hint("Alt-Enter", "new line"), hint("Esc", "cancel"), hint(m.helpKey(), "help")}
	case m.prompt != "":
		hints = []bkey.Binding{hint("Enter", "submit"), hint("Esc", "cancel"), hint(m.helpKey(), "help")}
	case m.detail:
		hints = m.detailHints(m.width < 100)
	case m.resultsShown():
		// Enter's results (5cjy88): the keys that walk and leave them.
		hints = []bkey.Binding{hint("j/k", "match"), hint("Enter", "open"), hint("Esc", "all tasks"), hint("m", "move"), hint("/", "refine"), hint("?", "help"), hint("q", "quit")}
	default:
		// h/l are the panes, as Tab and Shift-Tab are (e3sevr); the bar
		// names h/l only, so 140 columns keep the version (rxr4z8, crab
		// 21:26), and ? lists Tab.
		hints = []bkey.Binding{hint("j/k", "move"), hint("Space", "mark"), hint("Enter", "open"), hint("h/l", "pane"), hint("f/#", "filter/id"), hint("/", "find"), hint(":", "cmd"), hint("?", "help"), hint("q", "quit"), hint("H/L", "column"), hint("o", "order"), hint("r", "request")}
	}
	if m.width < 100 && !m.help && m.prompt == "" && m.confirm == "" && !m.detail && !m.grab.active && m.yanked == "" && !m.pendingV && !m.picking && !m.resultsShown() {
		hints = []bkey.Binding{hint("j/k", "move"), hint("Space", "mark"), hint("Enter", "open"), hint("f", "find"), hint("#", "id"), hint("/", "search"), hint(":", "cmd"), hint("?", "help"), hint("q", "quit")}
	}
	return hints
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
	if m.find.text != "" {
		// A find is on (4s3g5q): its keys come first, as in less.
		hints = []bkey.Binding{hint("n", "next match"), hint("N", "previous match"), hint("/", "find again")}
		if m.glance && m.glanceMD.eligible {
			hints = append(hints, m.markdownHint())
		}
		return append(hints, hint("Esc", "clear"), hint("?", "help"), hint("q", "quit"))
	}
	if m.glance {
		// Esc goes back where the file was shown from: Files, or the
		// question whose link showed it (qcq3tm).
		up := "files"
		if m.glanceReturn != nil {
			up = "question"
			if m.glanceReturn.link {
				up = "link" // a file link opened it (zbdbtm)
			}
		}
		if m.glancePicture() {
			// A picture (33v71x): zoom while the terminal draws it, scroll
			// and fit once zoomed, open it in the viewer always.
			var hints []bkey.Binding
			if m.zoomable() {
				hints = append(hints, hint("+/-", "zoom"))
				if m.picZoom > 0 {
					hints = append(hints, hint("h/j/k/l", "scroll"), hint("0", "fit"))
				}
			}
			return append(hints, hint("o", "open in "+viewerName()), hint("n", "next file"), hint("p", "previous file"), hint("u/Esc", up), hint("?", "help"), hint("q", "quit"))
		}
		// n and p spelled out (scrxak, Jørgen 00:08: nobody guesses n/p).
		hints := []bkey.Binding{hint("n", "next file"), hint("p", "previous file"), hint("Enter", "edit"), hint("/", "find"), hint("u/Esc", up), hint("?", "help"), hint("q", "quit")}
		if m.glanceMD.eligible {
			hints = append([]bkey.Binding{m.markdownHint()}, hints...)
		}
		if m.viewport.TotalLineCount() > m.viewport.Height {
			// j/k only when the file is longer than the screen (qtj177).
			hints = append([]bkey.Binding{hint("j/k", "scroll")}, hints...)
		}
		return hints
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
		if _, ok := m.markedFile(t); ok {
			// A file link (zbdbtm): Space shows it in line, Enter in the
			// file frame; open in line, Esc folds it.
			if m.inline != nil {
				hints = append(hints, hint("Space/Esc", "close"), hint("Enter", "open file"))
			} else {
				hints = append(hints, hint("Space", "show"), hint("Enter", "open file"))
			}
			break
		}
		hints = append(hints, hint("j/k", "scroll"), hint("Enter", "open"))
	case focusQuestion:
		// a answers, named first: it is what the question is for (0gztsy).
		hints = append(hints, hint("a", "answer"), hint("j/k", "scroll"))
	case focusFold:
		hints = append(hints, hint("Enter", "open/fold"), hint("j/k", "scroll"))
	case focusFiles:
		if m.markedPicture() {
			hints = append(hints, hint("j/k", "file"), hint("Space", "show"), hint("Enter", "open in "+viewerName()))
			break
		}
		hints = append(hints, hint("j/k", "file"), hint("Space", "show"), hint("Enter", "edit"))
	default:
		// Focus on the task itself (its id on the title line): the bar
		// names what acts on the whole task (e3sevr, Jørgen 23:59: no
		// stop may offer nothing).
		hints = append(hints, hint("j/k", "scroll"), hint("e", "edit"))
		switch t.State {
		case "todo":
			hints = append(hints, hint("t", "take"))
		case "doing":
			hints = append(hints, hint("d", "done"), hint("w", "wait"))
		}
		if t.State != "dropped" {
			hints = append(hints, hint("x", "drop"))
		}
		if m.moveRefusal() == "" {
			hints = append(hints, hint("y", "yank"))
		}
	}
	// One ring of stops, each task id one (4bwcgp, e3sevr): right after
	// the focus's keys, so a narrow bar keeps it.
	if len(m.focusRing(t)) > 1 {
		hints = append(hints, hint("h/l Tab", "next"))
	}
	// Every key that writes is named here, right after the focus's own
	// keys so a narrow bar cuts navigation first (x1ra0t); each asks
	// first, in a confirm line or a prompt, so nothing changes unseen.
	if m.forYou(t) && m.detailPane != focusQuestion {
		hints = append(hints, hint("a", "answer")) // the question asks you (3jtfw9)
	}
	if m.detailPane != focusPage {
		hints = append(hints, hint("t/d/x/e", "take/done/drop/edit"))
	}
	if !narrow {
		hints = append(hints, hint("N", "new"))
	}
	if m.width >= 140 {
		// S copies the screen and asks to attach it (5z4zt4), a writer
		// like the rest, so ahead of navigation; on a narrower bar
		// ? help names it, as - (sfa8w4).
		hints = append(hints, hint("S", "screen"))
		// V pastes the clipboard into the task (q7n2zd), beside S.
		hints = append(hints, hint(clipKey, "paste"))
	}
	files := len(m.filesOf(t)) > 0
	if files && !narrow {
		hints = append(hints, hint("E", "edit files")) // commits when the editor closes
	}
	// r opens the request box (hbgybn): it writes only on Enter, in
	// the box, so it follows up; at 80 the bar cuts it and ? help and
	// the list bar name it.
	hints = append(hints, hint("u", "up"), hint("r", "request"), hint("/", "find"), hint("n", "next task"), hint("p", "previous task"))
	hints = append(hints, hint("J/K", "½ page"))
	if files && narrow {
		hints = append(hints, hint("E", "edit files")) // u up first (9xfnwr check 3)
	}
	hints = append(hints, hint("Esc", "back"))
	if !narrow {
		hints = append(hints, hint(":", "cmd"))
	}
	return append(hints, hint("?", "help"), hint("q", "quit"))
}
