package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

var htmlComments = regexp.MustCompile(`(?s)<!--.*?(?:-->|$)`)

func cleanMarkdown(s string) string { return htmlComments.ReplaceAllString(safe(s), "") }

// literalAngles escapes every "<" that markdown would read as an HTML tag
// or autolink, so a note like "git worktree add --detach <dir> <rev>" shows
// as written instead of losing <dir> and <rev> as raw HTML (heype5). Code
// spans and fenced blocks are left alone: there "<" is already literal and
// a backslash would show.
func literalAngles(s string) string {
	var b strings.Builder
	fence := ""
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			b.WriteString(line)
			continue
		}
		if len(line)-len(trimmed) < 4 && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			fence = trimmed[:3]
			b.WriteString(line)
			continue
		}
		for j := 0; j < len(line); j++ {
			c := line[j]
			switch {
			case c == '\\' && j+1 < len(line):
				b.WriteString(line[j : j+2]) // already escaped
				j++
			case c == '`':
				// A code span runs to the next run of as many backticks.
				run := j
				for run < len(line) && line[run] == '`' {
					run++
				}
				ticks := line[j:run]
				if end := strings.Index(line[run:], ticks); end >= 0 {
					stop := run + end + len(ticks)
					b.WriteString(line[j:stop])
					j = stop - 1
				} else {
					b.WriteString(ticks)
					j = run - 1
				}
			case c == '<' && j+1 < len(line) && tagStart(line[j+1]):
				b.WriteString(`\<`)
			default:
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

func tagStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '/' || c == '!' || c == '?'
}
func (m model) markdown(s string) string {
	s = cleanMarkdown(s)
	if strings.TrimSpace(s) == "" {
		return ""
	}
	style := styles.DarkStyleConfig
	accent := "#00bfff"
	if !m.styles.renderer.HasDarkBackground() {
		style = styles.LightStyleConfig
		accent = "#006c9c"
	}
	zero := uint(0)
	style.Document.Margin = &zero
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""
	style.Heading.Color = &accent
	// Heading styles are copied, never mutate package defaults.
	style.H1.Prefix = ""
	style.H2.Prefix = ""
	style.H3.Prefix = ""
	style.H4.Prefix = ""
	style.H5.Prefix = ""
	style.H6.Prefix = ""
	style.H1.Color = &accent
	style.H2.Color = &accent
	style.H3.Color = &accent
	style.CodeBlock.Margin = &zero
	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(style), glamour.WithWordWrap(max(1, m.width-2)), glamour.WithColorProfile(m.styles.renderer.ColorProfile()))
	if err != nil {
		return wrap(s, m.width)
	}
	out, err := renderer.Render(literalAngles(s))
	if err != nil {
		return wrap(s, m.width)
	}
	return strings.Trim(out, "\n")
}
func (m model) detailHeader(t task.Task) []string {
	title := m.styles.normal.Bold(true).Render(fit(oneLine(t.Title), m.width))
	badges := []string{m.styles.status[m.boardStatus(t)].Bold(true).Render("[" + oneLine(m.boardStatus(t)) + "]")}
	// An unset type or priority shows no badge, not "[]" (2026-11-16, user).
	if v := oneLine(t.Type); v != "" {
		badges = append(badges, m.styles.kind[t.Type].Render("["+v+"]"))
	}
	if v := oneLine(t.Priority); v != "" {
		badges = append(badges, m.styles.priority[t.Priority].Render("["+v+"]"))
	}
	badges = append(badges, m.styles.accent.Render("["+oneLine(t.ID)+"]"))
	group := t.Group
	if group == "" {
		group = "root"
	}
	owner := "assigned:" + strings.Join(t.Assignee, ", ")
	if len(t.Assignee) == 0 {
		owner = "unassigned"
	}
	badges = append(badges, m.styles.dim.Render("group:"+oneLine(group)), m.styles.accent.Render(oneLine(owner)))
	lines := []string{title}
	lines = append(lines, strings.Split(wrap(strings.Join(badges, "  "), m.width), "\n")...)
	lines = append(lines, m.styles.dim.Render("created "+oneLine(t.Created)+" · updated "+oneLine(t.Updated)+" · "+oneLine(t.Slug)))
	var problems []task.Finding
	seen := map[string]bool{}
	// What the last edit broke comes first, then the rest in line order.
	var fresh []task.Finding
	if nt, fs, ok := m.noticeTask(); ok && nt.Path == t.Path {
		fresh = fs
	}
	// One task opened on purpose shows all its findings, closed or not.
	for _, f := range append(append(fresh, m.findings...), m.closedFindings...) {
		if f.Path == t.Path && !seen[f.Problem] {
			seen[f.Problem] = true
			problems = append(problems, f)
		}
	}
	if len(problems) > 0 {
		// What is wrong, on which line, then the fix on its own line so a
		// long problem never cuts it off (rx74bp).
		f := problems[0]
		warning := "⚠ "
		if f.Line > 0 {
			warning += fmt.Sprintf("line %d: ", f.Line)
		}
		warning += oneLine(f.Problem)
		if len(problems) > 1 {
			warning += fmt.Sprintf(" (+%d more findings)", len(problems)-1)
		}
		lines = append(lines, m.styles.warning.Render(fit(warning, m.width)))
		lines = append(lines, m.styles.warning.Render(fit("  fix: "+oneLine(f.FixLine()), m.width)))
	}
	return lines
}

type detailSection struct{ name, text string }

func (m model) sections(t task.Task) []detailSection {
	d := m.docs[t.Path]
	if d == nil {
		return []detailSection{{"description", cleanMarkdown(t.Goal)}}
	}
	raw := strings.Split(strings.ReplaceAll(cleanMarkdown(string(d.Bytes())), "\r\n", "\n"), "\n")
	start := 0
	if len(raw) > 0 && raw[0] == "---" {
		for i := 1; i < len(raw); i++ {
			if raw[i] == "---" {
				start = i + 1
				break
			}
		}
	}
	out := []detailSection{{name: "description"}}
	fence := ""
	for _, line := range raw[start:] {
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) <= 3 && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			if fence == "" {
				n := 0
				for n < len(trimmed) && trimmed[n] == trimmed[0] {
					n++
				}
				fence = trimmed[:n]
			} else if strings.HasPrefix(trimmed, fence) && strings.TrimSpace(strings.TrimPrefix(trimmed, fence)) == "" {
				fence = ""
			}
		}
		if fence == "" && strings.HasPrefix(line, "## ") {
			name := strings.ToLower(strings.TrimSpace(line[3:]))
			switch name {
			case "goal":
				name = "description"
			case "depends on":
				name = "dependencies"
			case "acceptance criteria":
				name = "acceptance"
			}
			out = append(out, detailSection{name: name})
			continue
		}
		out[len(out)-1].text += line + "\n"
	}
	return out
}

// Detail focus: the page scrolls, or one list (acceptance checks or
// dependencies) has a selection that j/k step through. Exactly one place
// shows focus: the task id in the tab bar for the page, else the focused
// section's heading (2026-11-16, user).
const (
	focusPage = iota
	focusAcceptance
	focusDeps
)

// focusStops lists the focus states the task offers, in Tab order. Empty
// sections are skipped.
func focusStops(t task.Task) []int {
	stops := []int{focusPage}
	if len(t.Acceptance) > 0 {
		stops = append(stops, focusAcceptance)
	}
	if len(t.Dependencies) > 0 {
		stops = append(stops, focusDeps)
	}
	return stops
}

// startFocus is the focus a freshly opened task gets: its checks when it has
// any, otherwise the page.
func (m *model) startFocus() {
	m.acceptance, m.dependency = 0, 0
	m.detailPane = focusPage
	if t, ok := m.selected(); ok && len(t.Acceptance) > 0 {
		m.detailPane = focusAcceptance
	}
}

// cycleFocus moves focus to the next (step 1) or previous (step -1) stop.
func (m *model) cycleFocus(step int) {
	t, ok := m.selected()
	if !ok {
		return
	}
	stops := focusStops(t)
	if len(stops) == 1 {
		m.message = "nothing to focus: this task has no checks or dependencies"
		return
	}
	pos := 0
	for i, s := range stops {
		if s == m.detailPane {
			pos = i
		}
	}
	m.detailPane = stops[(pos+step+len(stops))%len(stops)]
}

// sectionHeading is a section's name in capitals. The focused section gets
// the highlight, a ▸ that shows without colour, and the keys that work in it;
// keys are never upper-cased, since J and K are other keys.
func (m model) sectionHeading(title, hint string, focused bool) string {
	head := strings.ToUpper(title)
	if !focused {
		return m.styles.accent.Render(head)
	}
	if hint != "" {
		head += " · " + hint
	}
	return m.styles.focus.Render(fit("▸ "+head, m.width))
}
func (m model) detailText(t task.Task) string {
	sections := m.sections(t)
	var b strings.Builder
	section := func(heading, body, empty string) {
		if strings.TrimSpace(body) == "" {
			if empty != "" {
				b.WriteString(m.styles.dim.Render(empty) + "\n\n")
			}
			return
		}
		b.WriteString(heading + "\n" + body + "\n\n")
	}
	plain := func(title string) string { return m.sectionHeading(title, "", false) }
	text := func(name string) string {
		var out []string
		for _, s := range sections {
			if s.name == name && strings.TrimSpace(s.text) != "" {
				out = append(out, s.text)
			}
		}
		return strings.Join(out, "\n")
	}
	section(plain("Description"), m.markdown(text("description")), "no description yet")
	var checks []string
	for i, a := range t.Acceptance {
		box, pointer := "☐", " "
		if a.Checked {
			box = "☑"
		}
		if m.detailPane == focusAcceptance && i == m.acceptance {
			pointer = ">"
		}
		// Wrap to the room left after the prefix and hang continuation lines
		// under the text, so each item reads as one block (2026-11-15).
		prefix := fmt.Sprintf("%s %s %d. ", pointer, box, i+1)
		indent := strings.Repeat(" ", lipgloss.Width(prefix))
		body := wrap(oneLine(cleanMarkdown(a.Text)), max(10, m.width-len(indent)))
		lines := strings.Split(body, "\n")
		for j := range lines {
			if j == 0 {
				lines[j] = prefix + lines[j]
			} else {
				lines[j] = indent + lines[j]
			}
		}
		label := strings.Join(lines, "\n")
		if m.detailPane == focusAcceptance && i == m.acceptance {
			checks = append(checks, m.styles.selected.Render(label))
		} else {
			checks = append(checks, m.checkStyle(a.Checked).Render(label))
		}
	}
	section(m.sectionHeading("Acceptance", "j/k or [ ] step · Space ticks", m.detailPane == focusAcceptance), strings.Join(checks, "\n"), "no acceptance items yet")
	var dependencies []string
	for i, d := range task.DependencyDetails(t, m.all) {
		state := d.Status
		if d.Missing {
			state = "blocked"
		}
		pointer := " "
		if m.detailPane == focusDeps && i == m.dependency {
			pointer = ">"
		}
		flag := ""
		if d.Status != "done" && d.Status != "dropped" {
			flag = " BLOCKED"
		}
		line := fmt.Sprintf("%s ↳ %s %s [%s]%s %s", pointer, m.styles.status[state].Render(oneLine(d.ID)), oneLine(d.Title), oneLine(state), flag, oneLine(d.Note))
		if pointer == ">" {
			line = m.styles.selected.Render(ansi.Strip(line))
		}
		dependencies = append(dependencies, line)
	}
	section(m.sectionHeading("Dependencies", "j/k or [ ] step · Enter follows", m.detailPane == focusDeps), strings.Join(dependencies, "\n"), "no dependencies")
	section(plain("Notes"), m.markdown(text("notes")), "")
	section(plain("History"), m.markdown(text("history")), "")
	for _, s := range sections {
		if !contains([]string{"description", "acceptance", "dependencies", "notes", "history"}, s.name) {
			section(plain(s.name), m.markdown(s.text), "")
		}
	}
	return wrap(strings.TrimRight(b.String(), "\n"), m.width)
}

// openTop shows a just-opened task from the top, scrolled only as far as
// needed to keep the focused check in view: the header can grow (a finding
// and its fix take two lines, rx74bp) and must not hide the focus.
func (m *model) openTop() {
	m.viewport.GotoTop()
	if m.detailPane == focusPage {
		return
	}
	if t, ok := m.selected(); ok {
		for i, line := range strings.Split(ansi.Strip(m.detailText(t)), "\n") {
			if strings.HasPrefix(line, "> ") {
				if i >= m.viewport.Height {
					m.viewport.SetYOffset(i - m.viewport.Height + 1)
				}
				return
			}
		}
	}
}
func (m *model) revealAcceptance() {
	if m.detailPane == focusPage {
		return // nothing is marked; the page keeps its scroll
	}
	if t, ok := m.selected(); ok {
		for i, line := range strings.Split(ansi.Strip(m.detailText(t)), "\n") {
			if strings.HasPrefix(line, "> ") {
				if i < m.viewport.YOffset || i >= m.viewport.YOffset+m.viewport.Height {
					m.viewport.SetYOffset(max(0, i-m.viewport.Height/2))
				}
				break
			}
		}
	}
}
func (m *model) followDependency() {
	t, ok := m.selected()
	if !ok {
		return
	}
	deps := task.DependencyDetails(t, m.all)
	if len(deps) == 0 {
		m.message = "no dependency to follow"
		return
	}
	d := deps[min(m.dependency, len(deps)-1)]
	if d.Missing {
		m.message = "dependency not found: " + d.ID
		return
	}
	target, err := task.Resolve(m.all, d.ID)
	if err != nil {
		m.message = err.Error()
		return
	}
	m.detailStack = append(m.detailStack, t.Path)
	m.detailPath = target.Path
	m.startFocus()
	m.syncDetail()
	m.openTop()
}

// checkStyle colours an acceptance check: ticked ones green, open ones in the
// plain text colour, so they do not read as headings, which are blue
// (2026-11-16, user).
func (m model) checkStyle(checked bool) lipgloss.Style {
	if checked {
		return m.styles.status["done"]
	}
	return m.styles.normal
}
