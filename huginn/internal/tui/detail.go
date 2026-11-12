package tui

import (
	"fmt"
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
	owner := strings.Join(t.Assignee, ", ")
	if owner == "" {
		owner = "unassigned"
	}
	badges = append(badges, m.styles.dim.Render("group:"+oneLine(group)), m.styles.accent.Render("owner:"+oneLine(owner)))
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
func (m model) detailText(t task.Task) string {
	sections := m.sections(t)
	var b strings.Builder
	section := func(title, body, empty string) {
		if strings.TrimSpace(body) == "" {
			if empty != "" {
				b.WriteString(m.styles.dim.Render(empty) + "\n\n")
			}
			return
		}
		b.WriteString(m.styles.accent.Render(strings.ToUpper(title)) + "\n" + body + "\n\n")
	}
	text := func(name string) string {
		var out []string
		for _, s := range sections {
			if s.name == name && strings.TrimSpace(s.text) != "" {
				out = append(out, s.text)
			}
		}
		return strings.Join(out, "\n")
	}
	section("Description", m.markdown(text("description")), "no description yet")
	var checks []string
	for i, a := range t.Acceptance {
		box, pointer := "☐", " "
		state := "todo"
		if a.Checked {
			box = "☑"
			state = "done"
		}
		if m.detailPane == 0 && i == m.acceptance {
			pointer = ">"
		}
		label := fmt.Sprintf("%s %s %d. %s", pointer, box, i+1, strings.TrimSpace(m.markdown(a.Text)))
		checks = append(checks, m.styles.status[state].Render(label))
	}
	section("Acceptance · [ / ] select · Space toggle", strings.Join(checks, "\n"), "no acceptance items yet")
	var dependencies []string
	for i, d := range task.DependencyDetails(t, m.all) {
		state := d.Status
		if d.Missing {
			state = "blocked"
		}
		pointer := " "
		if m.detailPane == 1 && i == m.dependency {
			pointer = ">"
		}
		flag := ""
		if d.Status != "done" && d.Status != "dropped" {
			flag = " BLOCKED"
		}
		dependencies = append(dependencies, fmt.Sprintf("%s ↳ %s %s [%s]%s %s", pointer, m.styles.status[state].Render(oneLine(d.ID)), oneLine(d.Title), oneLine(state), flag, oneLine(d.Note)))
	}
	section("Dependencies · Tab focuses · [ / ] select · Enter follows", strings.Join(dependencies, "\n"), "no dependencies")
	section("Notes", m.markdown(text("notes")), "")
	section("History", m.markdown(text("history")), "")
	for _, s := range sections {
		if !contains([]string{"description", "acceptance", "dependencies", "notes", "history"}, s.name) {
			section(s.name, m.markdown(s.text), "")
		}
	}
	return wrap(strings.TrimRight(b.String(), "\n"), m.width)
}
func (m *model) revealAcceptance() {
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
	m.acceptance = 0
	m.dependency = 0
	m.detailPane = 0
	m.syncDetail()
	m.viewport.GotoTop()
}
