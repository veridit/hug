package tui

import (
	"fmt"
	"strings"

	bkey "github.com/charmbracelet/bubbles/key"
)

// The ? screen leads with the keys for where you are, then the keys that
// work everywhere (jmx10s, Jørgen 2026-11-16: keys move, so the list must
// lead with what applies here). Each context is one short heading and one
// key per row, the key column aligned. A row lists only keys that work in
// that context: TestHelpKeysAreBound presses each row's probe keys there.

// helpRow is one row of the ? screen: what it shows, and the keys a test
// presses to prove it is bound where it is listed ("" when the row names
// typed text, not a key).
type helpRow struct {
	key, desc string
	probe     []string
}

// helpSection is one heading and its rows.
type helpSection struct {
	title string
	rows  []helpRow
}

func row(key, desc string, probe ...string) helpRow {
	return helpRow{key: key, desc: desc, probe: probe}
}

// helpContext names where ? was opened, as the heading says it.
func (m model) helpContext() string {
	switch {
	case m.confirm != "":
		return "confirm"
	case m.prompt == ":":
		return "palette"
	case m.prompt == "filter":
		return "filter"
	case m.prompt == "sort":
		return "sort"
	case m.prompt == "/":
		return "find"
	case m.prompt == findPrompt:
		return "findpage"
	case m.prompt == "#":
		return "jump"
	case m.prompt != "":
		return "prompt"
	case m.detail && m.glance && m.glancePicture():
		return "picture"
	case m.detail && m.glance:
		return "file"
	case m.detail:
		return "task"
	case len(m.marks) > 0:
		return "marked" // the actions on the marked first (rxr4z8 check 8)
	case m.resultsShown():
		return "results" // Enter's results (5cjy88)
	}
	return m.view
}

// contextHelp is the section for where ? was opened.
func (m model) contextHelp() helpSection {
	switch c := m.helpContext(); c {
	case "confirm":
		return helpSection{"IN THIS CONFIRM", []helpRow{row("Enter", "do it", "enter"), row("Esc", "cancel: nothing changes", "esc")}}
	case "palette":
		return helpSection{"IN THE : PALETTE", []helpRow{
			row("Tab", "complete the choice", "tab"), row("Enter", "complete, or run a complete command", "enter"),
			row("↑↓ ctrl-n/p", "pick a completion"), row("Esc", "up a level; on the top level, close", "esc"),
			row("↑ empty", "recall earlier commands, ↑ again older, ↓ newer; Enter runs it"),
			row("TASK id", "the task you are in or on: take, done, drop, edit, note, set, move, mv, dep ..."),
			row("CHECK n", "with a check focused: tick or untick, strike, promote"),
			row("GLOBAL", "view, filter (state, type, priority, group, tag, assignee, epic), goto, sort, grouping, clear, new, rebalance, help"),
		}}
	case "filter":
		return helpSection{"IN THE FILTER PICKER", []helpRow{
			row("j/k ↑↓", "pick", "j", "k"), row("Enter", "add the chip", "enter"), row("!", "not: every task but the value, before or after picking; again flips back", "!"),
			row("type", "narrow the choices"), row("Esc", "back", "esc"),
			row("fields", "s state, t type, p priority, g group, l tag, a assignee, e epic"),
		}}
	case "sort":
		return helpSection{"IN THE SORT PICKER", []helpRow{
			row("j/k ↑↓", "pick", "j", "k"), row("Space", "add the picked field, again flips it", " "),
			row("letter", "add a field (then by), again flips it"), row("r", "rank: the default board order, at once; clears every field", "r"),
			row("c", "the marked column, as o: again flips, then off", "c"),
			row("fields", "p priority, t type, s state, u updated, d created, g group, a assignee, n title"),
			row("Backspace", "remove the last field", "backspace"),
			row("Enter", "add the picked field after j/k, else apply the order", "enter"), row("Esc", "keep the order as it was", "esc"),
		}}
	case "find":
		return helpSection{"IN / FIND", []helpRow{
			row("type", "find as you type"), row("field:value", "keep tasks with it: state:done type:bug group: tag: priority: assignee: epic:"),
			row("↑↓ ctrl-n/p", "pick a match, in every section", "ctrl+n"),
			row("Enter", "show the results: j/k walk them, Esc then all tasks", "enter"),
			row("↑ empty", "recall earlier searches, ↑ again older; Enter shows them"),
			row("Tab", "keep it as a filter until F", "tab"), row("Esc", "cancel: back where / was pressed", "esc"),
		}}
	case "results":
		return helpSection{"IN THE SEARCH RESULTS", []helpRow{
			row("j/k ↑/↓", "next / previous match, section after section", "j", "k"), row("< >", "first / last match", "<", ">"),
			row("Enter", "open the task; Esc comes back here", "enter"),
			row("Esc", "all tasks, keeping the task selected", "esc"),
			row("m", "move the task among all tasks", "m"),
			row("/", "refine the search", "/"),
		}}
	case "findpage":
		return helpSection{"IN / FIND IN THIS " + strings.ToUpper(m.pageNoun()), []helpRow{
			row("type", "find in the text; all lower case ignores case"), row("Enter", "keep the matches; with none, search all tasks", "enter"),
			row("↑↓", "previous / next match"), row("Esc", "clear the matches", "esc"),
			row("↑ empty", "recall earlier searches, ↑ again older, ↓ newer"),
		}}
	case "jump":
		return helpSection{"IN # JUMP", []helpRow{
			row("type", "an id prefix or a TASK-number alias"), row("Tab", "complete", "tab"), row("↑↓ ctrl-n/p", "pick"),
			row("Enter", "open it", "enter"), row("Esc", "back", "esc"),
		}}
	case "prompt":
		rows := []helpRow{row("Enter", "submit", "enter"), row("Esc", "cancel", "esc")}
		if m.boxPrompt() {
			rows = []helpRow{row("Enter", "send the answer", "enter"), row("Alt-Enter ctrl-j", "a new line; a paste keeps its lines", "ctrl+j"), row("Esc", "cancel: nothing is written", "esc")}
		}
		return helpSection{"IN THIS PROMPT", rows}
	case "picture":
		// A picture (33v71x): its zoom keys only while the terminal
		// draws it; o and Enter open it in the system viewer always.
		var rows []helpRow
		if m.zoomable() {
			rows = append(rows, row("+ / -", "zoom in / out, in steps from fit", "+", "-"), row("h/j/k/l", "scroll the zoomed picture", "h", "l"), row("0", "fit the picture to the frame", "0"))
		}
		rows = append(rows, row("o Enter", "open it in "+viewerName()+", never the editor", "o", "enter"), row("n / p", "next / previous file", "n", "p"),
			row("E", "the task's files folder in the editor", "E"), row("u / Esc", "back to the task", "u"))
		return helpSection{"IN THIS PICTURE", rows}
	case "file":
		rows := []helpRow{
			row("j/k PgDn/PgUp", "scroll a long file"), row("n / p", "next / previous file", "n", "p"), row("Enter", "edit it", "enter"),
			row("/", "find in the file; n / N step the matches", "/"),
		}
		if m.glanceMD.eligible {
			rows = append(rows, row("M", "Markdown: raw source / formatted text", "M"))
		}
		return helpSection{"IN THIS FILE", append(rows, row("E", "the task's files folder in the editor", "E"), row("u / Esc", "back to the task", "u"))}
	case "task":
		return m.taskHelp()
	case "marked":
		return m.markedHelp()
	}
	return m.rowsHelp()
}

// taskHelp is the open task's keys, those its focus and its content
// offer first.
func (m model) taskHelp() helpSection {
	t, _ := m.selected()
	var rows []helpRow
	switch m.detailPane {
	case focusAcceptance:
		rows = append(rows, row("j/k", "step through the checks", "j", "k"), row("[ ]", "previous / next check", "[", "]"),
			row("Space", "tick the check marked >", " "), row("-", "strike it out as not relevant, with a reason", "-"))
	case focusDeps:
		rows = append(rows, row("j/k", "step through the dependencies", "j", "k"), row("Enter", "open the dependency marked >", "enter"))
	case focusLinks:
		rows = append(rows, row("j/k", "step through the links", "j", "k"), row("Enter", "open the link marked >", "enter"))
	case focusFiles:
		enter := row("Enter", "edit the file", "enter")
		if m.markedPicture() {
			enter = row("Enter", "open the picture in "+viewerName(), "enter")
		}
		rows = append(rows, row("j/k", "step through the files", "j", "k"), row("Space", "show the file", " "), enter)
	case focusIDs:
		rows = append(rows, row("Enter", "open the task id marked ▸", "enter"), row("j/k", "scroll the page", "j", "k"))
	case focusQuestion:
		rows = append(rows, row("j/k", "scroll the page", "j", "k"))
	case focusFold:
		rows = append(rows, row("Enter", "open the section, or fold it again", "enter"), row("j/k", "scroll the page", "j", "k"))
	default:
		rows = append(rows, row("j/k", "scroll the page", "j", "k"))
	}
	if q, ok := m.focusedQuestion(t); ok && q.file != "" {
		// The question's link line (qcq3tm).
		rows = append(rows, row("Enter", "show "+oneLine(q.file)+", the file the question names; Esc comes back", "enter"))
	}
	if len(m.focusRing(t)) > 1 {
		rows = append(rows, row("h / l  Tab", "previous / next stop: each task id, the page, checks, links, files", "h", "l", "tab", "shift+tab"))
	}
	if m.forYou(t) {
		rows = append(rows, row("a", "answer the question it asks you: Enter sends, Alt-Enter a new line", "a"))
	}
	rows = append(rows,
		row("J / K", "half a page down / up", "J", "K"),
		row("n / p", "next / previous task of the list", "n", "p"),
		row("u / Esc", "up: back from a followed id, else to the view", "u"),
		row("/", "find in this task; n / N step the matches", "/"),
		row("t / d / x", "take / done / drop it", "t", "d", "x"),
		row("e", "edit in $VISUAL, else $EDITOR, git core.editor, vi", "e"),
		row("w", "wait: park the doing task on what it waits on", "w"),
		row("y", "yank: P/p in the list paste it", "y"),
		row("T", "type palette", "T"),
	)
	if len(m.filesOf(t)) > 0 {
		rows = append(rows, row("E", "its files folder in the editor", "E"))
	}
	return helpSection{"IN THIS TASK · " + t.ID, rows}
}

// rowsHelp is the keys of the overview, the list, the board and lint.
func (m model) rowsHelp() helpSection {
	title := map[string]string{"overview": "IN THE OVERVIEW", "list": "IN THE LIST", "board": "ON THE BOARD", "lint": "IN LINT"}[m.view]
	rows := []helpRow{
		row("j/k ↑/↓", "move", "j", "k"), row("< > Home End", "top / bottom", "<", ">", "home", "end"),
		row("ctrl-d/u", "half page down / up", "ctrl+d", "ctrl+u"), row("PgDn / PgUp", "page down / up", "pgdown", "pgup"),
		row("Enter", "open the task", "enter"),
		row("Space", "mark the task, then the next row; the footer acts on the marked", " "),
	}
	rows = append(rows, row("J / K", "page down / up", "J", "K"))
	if m.paneCount() > 1 {
		what := "section"
		if m.view == "board" {
			what = "column"
		}
		rows = append(rows, row("h / l  Tab", "previous / next "+what, "h", "l", "tab", "shift+tab"))
	}
	rows = append(rows,
		row("H / L", "mark the previous / next column, none past either end", "H", "L"), // hktdgs
		row("o", "order by the marked column: ▲ ▼ off", "o"),
		row("t / d / x", "take / done / drop the task", "t", "d", "x"),
		row("e", "edit in $VISUAL, else $EDITOR, git core.editor, vi", "e"),
	)
	if len(m.sort) == 0 {
		// Moves change rank, so they show only in rank order (mkzcqr).
		rows = append(rows, row("m", "grab to move: j/k move it, Enter places it", "m"), row("y, P p", "yank, then paste above / below", "y"))
	} else {
		rows = append(rows, row(": sort rank", "rank order again: then m, y, P/p move"))
	}
	// s and c share a row, so g costs no line and EVERYWHERE stays on the
	// first screen at 80x24.
	rows = append(rows, row("c", "clear all filters", "c"),
		row("g", "group rows: flat, group, epic, tag (tag: first listed tag)", "g"))
	if m.grouping != "" {
		// Only grouped rows have headings to fold (kx2ndm).
		rows = append(rows, row("Space / Enter", "on a heading: fold or unfold it", "enter"))
	}
	return helpSection{title, rows}
}

// globalHelp is the keys that work everywhere outside a prompt.
func (m model) globalHelp() helpSection {
	rows := []helpRow{
		// Prefix keys are two-key rows (qh9ep1): v, then the view.
		row("v o", "view: overview", "v"),
		row("v l", "view: list", "v"),
		row("v b", "view: board", "v"),
		row("v !", "view: lint", "v"),
		row("/", "find: Enter shows results, Tab keeps it as a filter", "/"),
		row(":", "palette: commands with completion", ":"),
		row(": commands", "overview, list, board, todo, doing, done, depends, dropped, state <s>, type <t>, priority <p>, group <g>, tag <t>, assignee <a>, epic <id>, goto <id>, sort, grouping, lint, clear, help"),
		row("#", "jump by id prefix or TASK-number alias", "#"),
		row("f", "filter picker: s t p g l a e (state type priority …)", "f"),
		row("s", "sort picker: c column, p t s u d g a n; again flips", "s"),
		row("Backspace / F", "remove the last chip / clear all filters", "backspace", "F"),
		row(": sort", "order by fields; s r or : sort rank goes back"),
		row("N", "new task", "N"),
		row("S", "copy the screen, attach: id, s select, c create", "S"),
		row(clipKey, "paste the clipboard (image, text); in a list: id, s, c", clipKey),
		row("r", "request: ask the agents for a change; Enter sends", "r"),
		row("Enter / Esc", "confirm / cancel every editing action"),
		row("e again", "after an edit left a finding: edit at once"),
		row("? / F1", "this help (F1 in a text field); / finds in it", "?", "f1"),
		row("Esc", "back one level, never quits", "esc"),
		row("q / ctrl-c", "quit (q only outside inputs)"),
	}
	if m.newer != "" {
		// R exists only while a newer hug is installed (j2vwk0).
		rows = append(rows, row(reloadKey, "reload into hug "+m.newer+", keeping this place"))
	}
	return helpSection{"EVERYWHERE", rows}
}

// helpSections are the ? screen's sections, the context first. Inside a
// prompt the global keys are typed text, so the prompt's keys are all.
func (m model) helpSections() []helpSection {
	ctx := m.contextHelp()
	switch m.helpContext() {
	case "confirm", "palette", "filter", "sort", "find", "findpage", "jump", "prompt", "file", "picture":
		// In a prompt the letters are typed text; in a file they are the
		// file's keys (scrxak): the rest waits for Esc.
		// In text ? is typed (t64v7d): there help is F1.
		help := row("?", "this help, from anywhere; / finds in it", "?")
		if m.textInput() {
			help = row("F1", "this help; ? is typed text here; / finds in it", "f1")
		}
		return []helpSection{ctx, {"EVERYWHERE", []helpRow{help, row("ctrl-c", "quit")}}}
	}
	return []helpSection{ctx, m.globalHelp()}
}

// bindings turns rows into the help model's bindings.
func bindings(rows []helpRow) []bkey.Binding {
	out := make([]bkey.Binding, len(rows))
	for i, r := range rows {
		out[i] = hint(r.key, r.desc)
	}
	return out
}

// helpText is the ? screen: each section's heading over its rows, one key
// per row with the key column aligned. With a search, the rows that match.
func (m model) helpText() string {
	sections := m.helpSections()
	if m.helpQuery != "" || m.helpTyping {
		var all []bkey.Binding
		for _, s := range sections {
			all = append(all, bindings(s.rows)...)
		}
		return m.helpSearchText(all)
	}
	// Wide enough: the context on the left, EVERYWHERE on the right, so
	// both fit a 140x40 screen (as the help did before at 130); else one
	// under the other, scrolled.
	if m.width >= 120 && len(sections) == 2 {
		half := (m.width - 2) / 2
		left := strings.Split(m.sectionText(sections[0], half), "\n")
		right := strings.Split(m.sectionText(sections[1], m.width-half-2), "\n")
		var lines []string
		for i := 0; i < max(len(left), len(right)); i++ {
			l, r := "", ""
			if i < len(left) {
				l = left[i]
			}
			if i < len(right) {
				r = right[i]
			}
			lines = append(lines, fit(l, half)+"  "+r)
		}
		return strings.Join(lines, "\n")
	}
	var parts []string
	for _, s := range sections {
		parts = append(parts, m.sectionText(s, m.width))
	}
	return strings.Join(parts, "\n\n")
}

// sectionText is one section width wide: its heading, then one key per
// row, the key column aligned and a long description wrapped under itself.
func (m model) sectionText(s helpSection, width int) string {
	keyWidth := 0
	for _, r := range s.rows {
		keyWidth = max(keyWidth, len([]rune(r.key)))
	}
	lines := []string{m.styles.accent.Render(s.title)}
	pad := strings.Repeat(" ", keyWidth+2)
	for _, r := range s.rows {
		key := r.key + strings.Repeat(" ", keyWidth-len([]rune(r.key)))
		for i, d := range strings.Split(wrap(r.desc, max(10, width-keyWidth-2)), "\n") {
			k := pad
			if i == 0 {
				k = m.styles.normal.Bold(true).Render(key) + "  "
			}
			lines = append(lines, fit(k+m.styles.dim.Render(d), width))
		}
	}
	return strings.Join(lines, "\n")
}

// helpSearchText lists the help rows whose key or description contains every
// word of the query, case-insensitively (2026-11-11: find one action fast).
func (m model) helpSearchText(all []bkey.Binding) string {
	words := strings.Fields(strings.ToLower(m.helpQuery))
	var hits []bkey.Binding
	for _, b := range all {
		h := b.Help()
		// A two-key row is found by its keys typed together too: vo.
		text := strings.ToLower(h.Key + " " + strings.ReplaceAll(h.Key, " ", "") + " " + h.Desc)
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
