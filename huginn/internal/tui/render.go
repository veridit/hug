package tui

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

type theme struct {
	renderer                               *lipgloss.Renderer
	normal, dim, accent, selected, warning lipgloss.Style
	// focus marks where focus is in a task: one heading or the task id.
	// ask is every confirm line: a question the reader cannot miss.
	focus, ask             lipgloss.Style
	status, kind, priority map[string]lipgloss.Style
}

func newTheme() theme {
	r := lipgloss.NewRenderer(os.Stdout)
	if os.Getenv("NO_COLOR") != "" {
		r.SetColorProfile(termenv.Ascii)
	}
	base := r.NewStyle()
	color := func(dark, light string) lipgloss.Style {
		return base.Foreground(lipgloss.AdaptiveColor{Dark: dark, Light: light})
	}
	t := theme{renderer: r, normal: base, dim: color("#8795a8", "#586474"), accent: color("#00bfff", "#006c9c").Bold(true),
		selected: base.Background(lipgloss.AdaptiveColor{Dark: "#16364a", Light: "#d8eef8"}).Bold(true), warning: color("#ffb454", "#995200"),
		focus:  base.Background(lipgloss.AdaptiveColor{Dark: "#00bfff", Light: "#006c9c"}).Foreground(lipgloss.AdaptiveColor{Dark: "#0b1620", Light: "#ffffff"}).Bold(true),
		ask:    base.Background(lipgloss.AdaptiveColor{Dark: "#ffb454", Light: "#995200"}).Foreground(lipgloss.AdaptiveColor{Dark: "#1a1200", Light: "#ffffff"}).Bold(true),
		status: map[string]lipgloss.Style{}, kind: map[string]lipgloss.Style{}, priority: map[string]lipgloss.Style{}}
	for i, s := range []string{"todo", "doing", "done", "blocked", "dropped"} {
		t.status[s] = color([]string{"#75bfff", "#00bfff", "#8ac987", "#ff8585", "#8795a8"}[i], []string{"#215da1", "#006c9c", "#31732c", "#ae2525", "#586474"}[i])
	}
	for i, s := range task.Types {
		t.kind[s] = color([]string{"#bc9dff", "#ff8585", "#a7b4c4", "#edbc6f", "#7bd4c2"}[i], []string{"#7143a8", "#ae2525", "#586474", "#885b0c", "#146f61"}[i])
	}
	for i, s := range []string{"critical", "high", "medium", "low"} {
		t.priority[s] = color([]string{"#ff7373", "#ffb454", "#d8cc85", "#8795a8"}[i], []string{"#ae2525", "#995200", "#746115", "#586474"}[i])
	}
	return t
}
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, strings.ToValidUTF8(s, "�"))
}
func oneLine(s string) string { return strings.Join(strings.Fields(safe(s)), " ") }
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	return s + strings.Repeat(" ", max(0, width-lipgloss.Width(s)))
}
func wrap(s string, width int) string {
	return ansi.Hardwrap(ansi.Wrap(strings.ReplaceAll(s, "\t", "    "), max(1, width), ""), max(1, width), false)
}
func (m model) bodyHeight() int { return max(1, m.height-7) }
func (m model) marker(path string) string {
	if m.linted[path] {
		return m.styles.warning.Render("!")
	}
	return " "
}
func styled(styles map[string]lipgloss.Style, value string, width int) string {
	return styles[value].Render(fit(oneLine(value), width))
}
func (m model) taskRow(t task.Task, width int, selected, compact bool) string {
	return m.taskRowMark(t, width, selected, compact, "")
}

// scrollMark puts ↑ on the first shown row when rows are hidden above and ↓
// on the last when rows are hidden below; the header's "a–b shown" gives the
// count. The arrows use the pointer gutter, so they cost no lines.
func scrollMark(n, start, end, total int) string {
	switch {
	case n == start && start > 0:
		return "↑"
	case n == end-1 && end < total:
		return "↓"
	}
	return ""
}
func (m model) taskRowMark(t task.Task, width int, selected, compact bool, mark string) string {
	// Gutter: pointer or scroll arrow, then the lint mark, so "!" costs no
	// column of its own (2026-11-16, user).
	pointer := " "
	if selected {
		pointer = ">"
		if m.grab.active && t.Path == m.grab.path {
			pointer = "≡" // grabbed: it moves with j/k (3xcesq)
		}
	} else if m.yanked != "" && t.ID == m.yanked {
		pointer = m.styles.warning.Render("y") // yanked: P/p paste it
	} else if mark != "" {
		pointer = m.styles.dim.Render(mark)
	}
	head := pointer + m.marker(t.Path) + m.styles.accent.Render(fit(oneLine(t.ID), 8)) + " " // bare, in its colour (xxvm6t)
	row := head
	if compact {
		// Board columns are narrow: type and priority are one coloured
		// letter each, spelled out in the legend under the board.
		row += initial(m.styles.kind, t.Type) + " " + initial(m.styles.priority, t.Priority) + " " + fit(oneLine(t.Title), max(0, width-15))
	} else {
		row += styled(m.styles.status, m.boardStatus(t), 7) + " " + styled(m.styles.kind, t.Type, 7) + " " + styled(m.styles.priority, t.Priority, 8) + " " + fit(oneLine(t.Title), max(0, width-36))
	}
	row = fit(row, width)
	if t.Status == "done" || t.Status == "dropped" {
		row = head + m.styles.dim.Render(ansi.Strip(ansi.Cut(row, 11, width)))
	}
	if selected {
		// Inner cells end with a full reset, which would cut the tint after
		// the first coloured cell; render the row as plain text in one style
		// so the highlight runs the whole width (2026-11-16, user).
		row = m.styles.selected.Render(ansi.Strip(row))
	}
	return row
}

// initial is the value's first letter, upper case, in the value's colour.
func initial(styles map[string]lipgloss.Style, value string) string {
	v := oneLine(value)
	if v == "" {
		return " "
	}
	return styles[value].Render(strings.ToUpper(v[:1]))
}

// legend spells out the board's one-letter type and priority cells. Its two
// words start with the column labels T and P, emphasised, so the reader sees
// which column each half explains (2026-11-16, user).
func (m model) legend() string {
	var parts []string
	for _, k := range task.Types {
		parts = append(parts, initial(m.styles.kind, k)+" "+m.styles.dim.Render(k))
	}
	kinds := strings.Join(parts, " ")
	parts = nil
	for _, p := range []string{"critical", "high", "medium", "low"} {
		parts = append(parts, initial(m.styles.priority, p)+" "+m.styles.dim.Render(p))
	}
	priorities := strings.Join(parts, " ")
	label := func(word string) string {
		return m.styles.normal.Bold(true).Underline(true).Render(word[:1]) + m.styles.dim.Render(word[1:]+": ")
	}
	full := label("Type") + kinds + m.styles.dim.Render(" · ") + label("Priority") + priorities
	if ansi.StringWidth(full) > m.width {
		// Narrow terminals shorten the words to their letters, never a
		// value's meaning.
		full = label("T") + kinds + m.styles.dim.Render(" · ") + label("P") + priorities
	}
	return fit(full, m.width)
}
func (m model) row(t task.Task) string {
	selected, ok := m.selected()
	return m.taskRow(t, m.width, ok && selected.Path == t.Path, false)
}
func (m model) columns(width int, compact bool) string {
	s := "  " + fit("ID", 8) + " "
	if compact {
		return m.styles.dim.Render(fit(s+"T P "+fit("TITLE", max(0, width-15)), width))
	}
	return m.styles.dim.Render(fit(s+fit("STATUS", 7)+" "+fit("TYPE", 7)+" "+fit("PRIORITY", 8)+" "+fit("TITLE", max(0, width-36)), width))
}

// window is the rows of indices shown in rows lines, starting at start as
// far as it can: a section keeps its scroll (h9zghj) and moves it only as
// far as needed to keep the cursor in view. A cursor of -1 is no cursor.
func window(indices []int, cursor, start, rows int) (int, int) {
	n := len(indices)
	rows = max(0, rows)
	if rows >= n {
		return 0, n
	}
	start = max(0, min(start, n-rows))
	for p, i := range indices {
		if i == cursor {
			if p < start {
				start = p
			} else if p >= start+rows {
				start = p - rows + 1
			}
			break
		}
	}
	return start, start + rows
}

func (m model) listText() string {
	indices := m.indices()
	lines := []string{m.columns(m.width, false)}
	if len(indices) == 0 {
		return strings.Join(append(lines, "No tasks match. :clear resets filters; N creates a task."), "\n")
	}
	start, end := window(indices, m.cursor, 0, max(1, m.bodyHeight()-2))
	for n, i := range indices[start:end] {
		lines = append(lines, m.taskRowMark(m.visible[i], m.width, i == m.cursor, false, scrollMark(start+n, start, end, len(indices))))
	}
	lines = append(lines, m.styles.dim.Render(fmt.Sprintf("Rows %d–%d of %d · ! lint finding · status blocked = unmet dependency", start+1, end, len(indices))))
	return strings.Join(lines, "\n")
}
func (m model) overviewText() string {
	names := overviewSections
	rows := m.overviewRows()
	var lines []string
	for p, name := range names {
		indices := m.paneIndices(p)
		pointer := "  "
		if p == m.pane {
			pointer = "> "
		}
		label := fmt.Sprintf("%s%s  [%d]", pointer, name, len(indices))
		if p == sectionLint {
			label += fmt.Sprintf(" · %d findings", len(m.findings))
			if n := len(m.closedFindings); n > 0 {
				label += fmt.Sprintf(" (%d on closed tasks hidden)", n)
			}
		}
		per := rows[p]
		cursor := m.memoCursor(p)
		if p == m.pane {
			cursor = m.cursor
		}
		start, end := window(indices, cursor, m.scrollOf(p), per)
		if len(indices) == 0 {
			// Nothing to navigate to: None goes on the title line (h9zghj).
			label += " · None"
		} else if end-start < len(indices) {
			label += fmt.Sprintf(" · %d–%d shown", start+1, end)
		}
		label = m.styles.accent.Render(fit(label, m.width))
		if p == m.pane {
			label = m.styles.selected.Render(label)
		}
		lines = append(lines, label)
		for n, i := range indices[start:end] {
			mark := scrollMark(start+n, start, end, len(indices))
			t := m.visible[i]
			sel := p == m.pane && i == m.cursor
			if p == sectionDoing {
				owner := strings.Join(t.Assignee, ", ")
				if owner == "" {
					owner = "unassigned"
				}
				// Owners get a reserved column, never lost behind a long
				// title. On the selected row the highlight runs through it,
				// so the owner reads as part of that row.
				ownerCell := fit(oneLine(owner), 16)
				if sel {
					ownerCell = m.styles.selected.Render("  " + ownerCell)
				} else {
					ownerCell = "  " + m.styles.accent.Render(ownerCell)
				}
				lines = append(lines, m.taskRowMark(t, max(1, m.width-18), sel, false, mark)+ownerCell)
			} else {
				lines = append(lines, m.taskRowMark(t, m.width, sel, false, mark))
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// overviewRows shares the overview's rows (h9zghj): every section the
// cursor is not in shows min(count, recentDone) rows, an empty one none
// (None sits on its title line), and the focused section takes all the
// rows left. When even that does not fit, RECENTLY DONE gives way first,
// then the largest of the others, down to one row each (zaec08).
func (m model) overviewRows() []int {
	n := len(overviewSections)
	want, got := make([]int, n), make([]int, n)
	budget := m.bodyHeight() - n*2 // a label and a blank line per section
	others := 0
	for p := range want {
		want[p] = len(m.paneIndices(p))
		if p != m.pane {
			got[p] = min(want[p], recentDone)
			others += got[p]
		}
	}
	// The focused section never gets less than an unfocused one would.
	need := min(want[m.pane], recentDone)
	for others+need > budget {
		shrink := -1
		if got[sectionRecent] > 1 && sectionRecent != m.pane {
			shrink = sectionRecent
		} else {
			for p := range got {
				if p != m.pane && got[p] > 1 && (shrink < 0 || got[p] > got[shrink]) {
					shrink = p
				}
			}
		}
		if shrink < 0 {
			break
		}
		got[shrink]--
		others--
	}
	if m.pane >= 0 && m.pane < n {
		got[m.pane] = min(want[m.pane], max(need, budget-others))
	}
	return got
}

// scrollOf is the first row section p of the current view showed last.
func (m model) scrollOf(p int) int {
	if v := memoView(m.view); v >= 0 && p >= 0 && p < len(m.memo[v]) {
		return m.memo[v][p].start
	}
	return 0
}

// memoCursor is the row section p had selected when the cursor left it,
// as an index into visible, or -1.
func (m model) memoCursor(p int) int {
	v := memoView(m.view)
	if v < 0 || p < 0 || p >= maxSections {
		return -1
	}
	path := m.memo[v][p].path
	if path == "" {
		return -1
	}
	for i := range m.visible {
		if m.visible[i].Path == path {
			return i
		}
	}
	return -1
}

// sectionRows is how many rows section p of the current view shows now.
func (m model) sectionRows(p int) int {
	if m.view == "overview" {
		return m.overviewRows()[p]
	}
	return max(1, m.bodyHeight()-4) // a board column
}
func (m model) boardText() string {
	count := min(len(boardStatuses), max(1, (m.width+2)/40))
	first := min(max(0, m.pane-count+1), len(boardStatuses)-count)
	width := max(1, (m.width-2*(count-1))/count)
	cols := make([][]string, count)
	rows := max(1, m.bodyHeight()-4)
	for c := 0; c < count; c++ {
		p := first + c
		indices := m.paneIndices(p)
		pointer := "  "
		if p == m.pane {
			pointer = "> "
		}
		label := fmt.Sprintf("%s%s [%d]", pointer, strings.ToUpper(boardStatuses[p]), len(indices))
		if len(indices) == 0 {
			label += " · None" // one line, as in the overview (h9zghj)
		}
		label = m.styles.status[boardStatuses[p]].Bold(true).Render(fit(label, width))
		if p == m.pane {
			label = m.styles.selected.Render(label)
		}
		heading := m.columns(width, true)
		if len(indices) == 0 {
			heading = fit("", width)
		}
		cols[c] = append(cols[c], label, heading)
		cursor := m.memoCursor(p)
		if p == m.pane {
			cursor = m.cursor
		}
		start, end := window(indices, cursor, m.scrollOf(p), rows)
		for n, i := range indices[start:end] {
			cols[c] = append(cols[c], m.taskRowMark(m.visible[i], width, p == m.pane && i == m.cursor, true, scrollMark(start+n, start, end, len(indices))))
		}
		for len(cols[c]) < rows+2 {
			cols[c] = append(cols[c], fit("", width))
		}
	}
	var lines []string
	for n := range cols[0] {
		line := ""
		for c := range cols {
			if c > 0 {
				line += "  "
			}
			line += cols[c][n]
		}
		lines = append(lines, line)
	}
	lines = append(lines, m.legend())
	lines = append(lines, m.styles.dim.Render(fmt.Sprintf("Columns %d–%d of %d · h/l or Tab/Shift-Tab reveals other columns", first+1, first+count, len(boardStatuses))))
	return strings.Join(lines, "\n")
}
func (m model) paletteText() string {
	lines := []string{m.styles.accent.Render("COMMAND PALETTE"), m.input.View(), ""}
	cs := m.completions()
	if len(cs) == 0 {
		return strings.Join(append(lines, "No completions. Esc closes; ? shows help."), "\n")
	}
	// Use the screen: as many rows as fit (2026-11-16, user).
	rows := max(1, m.bodyHeight()-5)
	start := max(0, m.completion-rows+1)
	nameWidth := min(40, max(8, m.width/3))
	for i := start; i < min(len(cs), start+rows); i++ {
		pointer := "  "
		if i == m.completion {
			pointer = "> "
		}
		name, desc := oneLine(cs[i].value), oneLine(cs[i].description)
		if cs[i].branch {
			name = strings.TrimRight(name, " ") + " ›"
		}
		if cs[i].field != "" {
			desc = fmt.Sprintf("%d tasks", m.valueCount(cs[i].field, cs[i].fieldValue))
		}
		line := fit(pointer+fit(name, nameWidth)+"  "+desc, m.width)
		if i == m.completion {
			line = m.styles.selected.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.styles.dim.Render(fmt.Sprintf("%d choices · › opens a level · Tab or Enter completes · Enter runs a complete command", len(cs))))
	return strings.Join(lines, "\n")
}
func (m model) View() string {
	mode := m.view
	if m.detail {
		mode = "detail"
	}
	if m.prompt == ":" {
		mode = "palette"
	}
	if m.prompt == "#" {
		mode = "jump"
	}
	if m.prompt == "filter" {
		mode = "filter"
	}
	if m.help {
		mode = "help"
	}
	header := m.styles.accent.Render("HUG TASKS · "+mode) + m.styles.dim.Render(fmt.Sprintf("   %d/%d tasks · ! %d findings", len(m.visible), len(m.all), len(m.findings)))
	tabs := []string{}
	// The shortcut letter is bold and underlined inside the tab name, which
	// costs no width (2026-11-11, user). lint has no letter of its own; its
	// key "!" shows as a prefix. The full list is on the help screen.
	for _, v := range []struct{ key, name string }{{"o", "overview"}, {"i", "list"}, {"b", "board"}, {"!", "lint"}} {
		base := m.styles.dim
		// In a task no view tab is marked: focus is in the task, and the
		// trail after the tabs says where it came from (2026-11-16, user).
		current := v.name == m.view && !m.detail
		if current {
			base = m.styles.selected
		}
		keyStyle := base.Bold(true).Underline(true)
		label := ""
		if i := strings.Index(v.name, v.key); i >= 0 {
			label = base.Render(v.name[:i]) + keyStyle.Render(v.key) + base.Render(v.name[i+1:])
		} else {
			label = keyStyle.Render(v.key) + base.Render(v.name)
		}
		if current {
			label = base.Render("[") + label + base.Render("]")
		} else {
			label = base.Render(" ") + label + base.Render(" ")
		}
		tabs = append(tabs, label)
	}
	if m.detail {
		tabs = append(tabs, m.detailTrail())
	}
	// Disjoint counts that add up to the tasks shown (zaec08): every task
	// has one place; doing names how many of its tasks wait on a dependency.
	counts := map[string]int{}
	doingBlocked := 0
	for _, t := range m.visible {
		counts[m.place(t)]++
		if t.Status == "doing" && m.blocked[t.Path] {
			doingBlocked++
		}
	}
	summary := []string{}
	for _, s := range boardStatuses {
		text := fmt.Sprintf("%s %d", s, counts[s])
		if s == "doing" && doingBlocked > 0 {
			text += fmt.Sprintf(" (%d blocked)", doingBlocked)
		}
		summary = append(summary, m.styles.status[s].Render(text))
	}
	content := ""
	switch {
	case m.help:
		content = m.helpViewport.View()
	case m.prompt == ":":
		content = m.paletteText()
	case m.prompt == "#":
		content = m.jumpText()
	case m.prompt == "filter":
		content = m.filterText()
	case m.detail:
		if t, ok := m.selected(); ok {
			content = strings.Join(m.detailHeader(t), "\n") + "\n" + m.styles.dim.Render(strings.Repeat("─", m.width)) + "\n" + m.viewport.View()
		}
	case m.view == "overview":
		content = m.overviewText()
	case m.view == "board":
		content = m.boardText()
	default:
		content = m.listText()
	}
	lines := []string{header, strings.Join(tabs, "  "), strings.Join(summary, "  ·  "), m.filterBar()}
	body := strings.Split(content, "\n")
	for n := 0; n < m.bodyHeight(); n++ {
		line := ""
		if n < len(body) {
			line = body[n]
		}
		lines = append(lines, line)
	}
	footer := m.message
	messageLines := strings.Split(m.message, "\n")
	if len(messageLines) > 1 {
		footer = messageLines[0]
	}
	if footer == "" {
		filters := []string{}
		for _, f := range []struct{ k, v string }{{"status", m.filter.Status}, {"type", m.filter.Type}, {"group", m.filter.Group}, {"label", m.filter.Label}, {"find", m.query}} {
			if f.v != "" {
				filters = append(filters, f.k+"="+oneLine(f.v))
			}
		}
		footer = "All tasks · : filters and commands"
		if len(filters) > 0 {
			footer = strings.Join(filters, " · ") + " · Esc clears"
		}
		if m.detail {
			footer = fmt.Sprintf("Detail · %.0f%% · Esc back", m.viewport.ScrollPercent()*100)
			if pos := m.detailPosition(); pos != "" {
				footer = fmt.Sprintf("Task %s · %.0f%% · n next · p previous · Esc back", pos, m.viewport.ScrollPercent()*100)
			}
		}
		if m.grab.active {
			rows := m.indices()
			footer = fmt.Sprintf("Moving %s · row %d of %d in %s (was %d) · nothing written until Enter", m.grab.id, m.rowOf(rows, m.grab.path)+1, len(rows), m.sectionName(), m.grab.start+1)
		}
	}
	lines = append(lines, m.styles.dim.Render(oneLine(footer)))
	inputLine := ""
	if len(messageLines) > 1 {
		for _, line := range messageLines[1:] {
			if strings.HasPrefix(line, "fix: ") {
				inputLine = safe(line)
				break
			}
		}
	}
	if notice, bar, ok := m.editNoticeLines(); ok && m.confirm == "" && m.prompt == "" && !m.help && m.message == "" {
		lines[len(lines)-1] = notice
		inputLine = bar
	}
	if m.confirm != "" {
		inputLine = m.confirmLine()
	} else if m.prompt != "" && m.prompt != ":" && m.prompt != "#" && m.prompt != "filter" {
		inputLine = m.input.View()
	} else if m.pendingG {
		inputLine = "g … press g for top"
	}
	lines = append(lines, inputLine, m.keyBar())
	// Final cell-aware boundary also protects malformed fields, Unicode, and tiny resizes.
	if len(lines) > m.height {
		lines = lines[:m.height]
		lines[len(lines)-1] = m.keyBar()
	}
	for i, line := range lines {
		lines[i] = fit(line, m.width)
	}
	return strings.Join(lines, "\n")
}

// editNoticeLines are the two lines shown after an edit that left the task
// with a finding: what is wrong (task, line, problem) in the warning colour,
// then a bar in the confirm style with the fix and the key that reopens the
// editor, which survives truncation (rx74bp, 2026-11-16: the only sign of a
// broken edit was "editor closed; re-linted").
func (m model) editNoticeLines() (string, string, bool) {
	t, fs, ok := m.noticeTask()
	if !ok {
		return "", "", false
	}
	f := fs[0]
	what := "⚠ " + oneLine(t.ID)
	if f.Line > 0 {
		what += fmt.Sprintf(" line %d", f.Line)
	}
	what += ": " + oneLine(f.Problem)
	if len(fs) > 1 {
		what += fmt.Sprintf(" (+%d more)", len(fs)-1)
	}
	keys := " · e edit again · Esc dismiss"
	fix := "▶ fix: " + oneLine(f.FixLine())
	line := ansi.Truncate(fix, max(1, m.width-ansi.StringWidth(keys)), "…") + keys
	return m.styles.warning.Render(fit(what, m.width)), m.styles.ask.Render(fit(line, m.width)), true
}

// confirmLine is the question every confirm asks, in one style that cannot be
// missed: a full-width bold bar in the warning colour, starting "▶ " so it
// stands out without colour too, and ending in the question mark and the
// answer keys, which survive truncation (2026-11-16, user: a plain prompt
// went unnoticed).
func (m model) confirmLine() string {
	question := "Confirm " + actionNames[m.confirm] + " "
	switch m.confirm {
	case "n":
		question += oneLine(m.pendingValue)
	case "e":
		for _, t := range m.all {
			if t.Path == m.confirmPath {
				question = "Edit " + oneLine(t.ID) + " in " + oneLine(m.pendingValue)
				break
			}
		}
	case " ":
		// Say exactly what Space will do: tick or untick which item, quoted
		// as the ticket shows it, box and number after the colon.
		for _, t := range m.all {
			if t.Path == m.confirmPath && len(t.Acceptance) > 0 {
				n := min(m.acceptance, len(t.Acceptance)-1)
				verb, box := "Tick", "☐"
				if t.Acceptance[n].Checked {
					verb, box = "Untick", "☑"
				}
				question = fmt.Sprintf("%s: %s %d. %s", verb, box, n+1, oneLine(cleanMarkdown(t.Acceptance[n].Text)))
				break
			}
		}
	default:
		if t, ok := m.taskAt(m.confirmPath); ok {
			question += oneLine(t.ID) + " " + oneLine(t.Title)
		}
	}
	answer := "? Enter yes · Esc no"
	question = strings.TrimRight(question, " ")
	line := "▶ " + ansi.Truncate(question, max(1, m.width-2-ansi.StringWidth(answer)), "…") + answer
	return m.styles.ask.Render(fit(line, m.width))
}

// detailTrail is the path to the open task, after the view tabs: the view it
// was opened from, any tasks followed through dependencies, then the task.
// With page focus the task id carries the focus highlight; with a section
// focused, that heading does instead.
func (m model) detailTrail() string {
	t, ok := m.selected()
	if !ok {
		return ""
	}
	parts := []string{m.styles.dim.Render(m.view)}
	for _, path := range m.detailStack {
		for _, v := range m.all {
			if v.Path == path {
				parts = append(parts, m.styles.dim.Render(oneLine(v.ID)))
			}
		}
	}
	// Page focus shows as a ▸ before the id, as a focused heading shows it,
	// so it reads without colour too; the id itself stays bare (xxvm6t).
	id := oneLine(t.ID)
	if m.detailPane == focusPage {
		id = m.styles.focus.Render("▸ " + id)
	} else {
		id = m.styles.accent.Render(id)
	}
	return strings.Join(append(parts, id), m.styles.dim.Render(" › "))
}
