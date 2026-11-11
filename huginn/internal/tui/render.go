package tui

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/huginn/internal/task"
)

type theme struct {
	renderer                               *lipgloss.Renderer
	normal, dim, accent, selected, warning lipgloss.Style
	status, kind, priority                 map[string]lipgloss.Style
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
	pointer := "  "
	if selected {
		pointer = "> "
	}
	row := pointer + m.styles.accent.Render(fit("["+oneLine(t.ID)+"]", 8)) + " " + m.marker(t.Path) + " "
	titleWidth := width - 38
	if !compact {
		row += styled(m.styles.status, m.boardStatus(t), 7) + " "
	} else {
		titleWidth += 8
	}
	row += styled(m.styles.kind, t.Type, 7) + " " + styled(m.styles.priority, t.Priority, 8) + " " + fit(oneLine(t.Title), max(0, titleWidth))
	row = fit(row, width)
	if t.Status == "done" || t.Status == "dropped" {
		row = pointer + m.styles.accent.Render(fit("["+oneLine(t.ID)+"]", 8)) + " " + m.marker(t.Path) + " " + m.styles.dim.Render(ansi.Strip(ansi.Cut(row, 13, width)))
	}
	if selected {
		row = m.styles.selected.Render(row)
	}
	return row
}
func (m model) row(t task.Task) string {
	selected, ok := m.selected()
	return m.taskRow(t, m.width, ok && selected.Path == t.Path, false)
}
func (m model) columns(width int, compact bool) string {
	s := "  " + fit("ID", 8) + " ! "
	rest := width - 38
	if !compact {
		s += fit("STATUS", 7) + " "
	} else {
		rest += 8
	}
	return m.styles.dim.Render(fit(s+fit("TYPE", 7)+" "+fit("PRIORITY", 8)+" "+fit("TITLE", max(0, rest)), width))
}
func window(indices []int, cursor, rows int) (int, int) {
	pos := 0
	for p, i := range indices {
		if i == cursor {
			pos = p
			break
		}
	}
	start := max(0, pos-rows+1)
	return start, min(len(indices), start+rows)
}
func (m model) listText() string {
	indices := m.indices()
	lines := []string{m.columns(m.width, false)}
	if len(indices) == 0 {
		return strings.Join(append(lines, "No tasks match. :clear resets filters; n creates a task."), "\n")
	}
	start, end := window(indices, m.cursor, max(1, m.bodyHeight()-2))
	for _, i := range indices[start:end] {
		lines = append(lines, m.taskRow(m.visible[i], m.width, i == m.cursor, false))
	}
	lines = append(lines, m.styles.dim.Render(fmt.Sprintf("Rows %d–%d of %d · ! lint finding · status blocked = unmet dependency", start+1, end, len(indices))))
	return strings.Join(lines, "\n")
}
func (m model) overviewText() string {
	names := []string{"IN PROGRESS · who is working", "NEXT UP · rank order", "BLOCKED · unmet dependencies", "LINT · tasks needing attention"}
	per := max(1, m.bodyHeight()/4-2)
	var lines []string
	for p, name := range names {
		indices := m.paneIndices(p)
		pointer := "  "
		if p == m.pane {
			pointer = "> "
		}
		label := fmt.Sprintf("%s%s  [%d]", pointer, name, len(indices))
		if p == 3 {
			label += fmt.Sprintf(" · %d findings", len(m.findings))
		}
		start, end := window(indices, m.cursor, per)
		if len(indices) > per {
			label += fmt.Sprintf(" · %d–%d shown", start+1, end)
		}
		label = m.styles.accent.Render(fit(label, m.width))
		if p == m.pane {
			label = m.styles.selected.Render(label)
		}
		lines = append(lines, label)
		if len(indices) == 0 {
			lines = append(lines, m.styles.dim.Render("  None"))
		}
		for _, i := range indices[start:end] {
			t := m.visible[i]
			if p == 0 {
				owner := strings.Join(t.Assignee, ", ")
				if owner == "" {
					owner = "unassigned"
				}
				// Owners get a reserved column, never lost behind a long title.
				line := m.taskRow(t, max(1, m.width-18), p == m.pane && i == m.cursor, false) + "  " + m.styles.accent.Render(fit(oneLine(owner), 16))
				lines = append(lines, line)
			} else {
				lines = append(lines, m.taskRow(t, m.width, p == m.pane && i == m.cursor, false))
			}
		}
		for n := max(1, end-start); n < per; n++ {
			lines = append(lines, "")
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
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
		label := fit(fmt.Sprintf("%s%s [%d]", pointer, strings.ToUpper(boardStatuses[p]), len(indices)), width)
		label = m.styles.status[boardStatuses[p]].Bold(true).Render(label)
		if p == m.pane {
			label = m.styles.selected.Render(label)
		}
		cols[c] = append(cols[c], label, m.columns(width, true))
		cursor := -1
		if p == m.pane {
			cursor = m.cursor
		}
		start, end := window(indices, cursor, rows)
		for _, i := range indices[start:end] {
			cols[c] = append(cols[c], m.taskRow(m.visible[i], width, p == m.pane && i == m.cursor, true))
		}
		for len(cols[c]) < rows+2 {
			cols[c] = append(cols[c], fit("", width))
		}
		cols[c] = append(cols[c], m.styles.dim.Render(fit(fmt.Sprintf("%d tasks", len(indices)), width)))
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
	lines = append(lines, m.styles.dim.Render(fmt.Sprintf("Columns %d–%d of %d · h/l or Tab/Shift-Tab reveals other columns", first+1, first+count, len(boardStatuses))))
	return strings.Join(lines, "\n")
}
func (m model) paletteText() string {
	lines := []string{m.styles.accent.Render("COMMAND PALETTE"), m.input.View(), ""}
	cs := m.completions()
	if len(cs) == 0 {
		return strings.Join(append(lines, "No completions. Esc closes; ? shows help."), "\n")
	}
	rows := min(8, max(1, m.bodyHeight()-5))
	start := max(0, m.completion-rows+1)
	for i := start; i < min(len(cs), start+rows); i++ {
		pointer := "  "
		if i == m.completion {
			pointer = "> "
		}
		line := pointer + fit(oneLine(cs[i].value), min(28, max(8, m.width/3))) + "  " + oneLine(cs[i].description)
		line = fit(line, m.width)
		if i == m.completion {
			line = m.styles.selected.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.styles.dim.Render(fmt.Sprintf("%d completions · Tab or Enter completes · Enter runs a complete command", len(cs))))
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
	header := m.styles.accent.Render("HUGINN · "+mode) + m.styles.dim.Render(fmt.Sprintf("   %d/%d tasks · ! %d findings", len(m.visible), len(m.all), len(m.findings)))
	tabs := []string{}
	// Each tab shows the key that opens it (2026-11-11: the row read like
	// labels and nothing said how to reach list, board or lint).
	for _, v := range []struct{ key, name string }{{"o", "overview"}, {"v", "list"}, {"b", "board"}, {"!", "lint"}} {
		s := v.key + " " + v.name
		if v.name == m.view {
			s = m.styles.selected.Render("[" + s + "]")
		} else {
			s = m.styles.accent.Render(" "+v.key) + m.styles.dim.Render(" "+v.name+" ")
		}
		tabs = append(tabs, s)
	}
	counts := map[string]int{}
	blocked := 0
	for _, t := range m.visible {
		counts[t.Status]++
		if m.blocked[t.Path] {
			blocked++
		}
	}
	summary := []string{}
	for _, s := range boardStatuses {
		n := counts[s]
		if s == "blocked" {
			n = blocked
		}
		summary = append(summary, m.styles.status[s].Render(fmt.Sprintf("%s %d", s, n)))
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
	if m.confirm != "" {
		inputLine = "Confirm " + actionNames[m.confirm] + " "
		if m.confirm == "n" {
			inputLine += m.pendingValue
		} else {
			for _, t := range m.all {
				if t.Path == m.confirmPath {
					inputLine += t.ID + " " + t.Title
					break
				}
			}
		}
		inputLine += "?"
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
