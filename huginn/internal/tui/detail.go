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
	out, err := renderer.Render(s)
	if err != nil {
		return wrap(s, m.width)
	}
	return strings.Trim(out, "\n")
}
func (m model) detailHeader(t task.Task) []string {
	title := m.styles.normal.Bold(true).Render(fit(oneLine(t.Title), m.width))
	badges := []string{m.styles.status[m.boardStatus(t)].Bold(true).Render("[" + oneLine(m.boardStatus(t)) + "]"), m.styles.kind[t.Type].Render("[" + oneLine(t.Type) + "]"), m.styles.priority[t.Priority].Render("[" + oneLine(t.Priority) + "]"), m.styles.accent.Render("[" + oneLine(t.ID) + "]")}
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
	problems := []string{}
	for _, f := range m.findings {
		if f.Path == t.Path && !contains(problems, f.Problem) {
			problems = append(problems, f.Problem)
		}
	}
	if len(problems) > 0 {
		warning := "⚠ " + oneLine(problems[0])
		if len(problems) > 1 {
			warning += fmt.Sprintf(" (+%d more findings)", len(problems)-1)
		}
		lines = append(lines, m.styles.warning.Render(fit(warning, m.width)))
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
	m.viewport.GotoTop()
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
