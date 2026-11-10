package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/muninn/internal/config"
)

var htmlComments = regexp.MustCompile(`(?s)<!--.*?(?:-->|$)`)
var htmlTags = regexp.MustCompile(`(?i)</?[a-z][a-z0-9]*(?:\s[^>]*)?/?>`)
var autolinks = regexp.MustCompile(`<(https?://[^>]+)>`)

// Storage markers and inline HTML are not user-facing memory content. Strip
// them before Markdown parsing, never after rendering ANSI produced by Glamour.
func markdownSource(s string) string {
	s = safe(s)
	s = htmlComments.ReplaceAllString(s, "")
	s = autolinks.ReplaceAllString(s, "$1")
	return htmlTags.ReplaceAllString(s, "")
}
func renderMarkdown(s string, width int) string {
	s = markdownSource(s)
	if strings.TrimSpace(s) == "" {
		return dim("No text recorded.")
	}
	style := styles.DarkStyleConfig
	if !lipgloss.HasDarkBackground() {
		style = styles.LightStyleConfig
	}
	colour := "#00bfff"
	if !lipgloss.HasDarkBackground() {
		colour = "#006c9c"
	}
	zero := uint(0)
	style.Document.Margin = &zero
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""
	style.Heading.Color = &colour
	style.Link.Color = &colour
	style.LinkText.Color = &colour
	style.H1.Color = &colour
	style.H1.BackgroundColor = nil
	style.H1.Prefix = ""
	style.H1.Suffix = ""
	style.H2.Prefix = ""
	style.H2.Color = &colour
	style.H3.Prefix = ""
	style.H3.Color = &colour
	style.H4.Prefix = ""
	style.H5.Prefix = ""
	style.H6.Prefix = ""
	style.CodeBlock.Margin = &zero
	style.Task.Ticked = "☑ "
	style.Task.Unticked = "☐ "
	profile := termenv.TrueColor
	if os.Getenv("NO_COLOR") != "" {
		profile = termenv.Ascii
	}
	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(style), glamour.WithWordWrap(max(1, width)), glamour.WithColorProfile(profile))
	if err != nil {
		return wrapText(s, width)
	}
	out, err := renderer.Render(s)
	if err != nil {
		return wrapText(s, width)
	}
	out = strings.TrimSpace(out)
	if os.Getenv("NO_COLOR") != "" {
		out = ansi.Strip(out)
	}
	// Code and unbreakable URLs must also fit. Glamour wraps prose, and this last
	// ANSI-aware pass covers long code tokens without destroying their styling.
	return ansi.Hardwrap(out, max(1, width), false)
}

type detailMetadata struct {
	strength, reinforcements int
	origin                   string
}

func (m model) loadDetailMetadata() detailMetadata {
	u := m.unit
	out := detailMetadata{strength: 1}
	meta, err := nativeMeta(m.a, u)
	_ = m.a.DB.QueryRow(`SELECT COUNT(*) FROM capture_reinforcements WHERE sid=?`, u.SID).Scan(&out.reinforcements)
	var lines []string
	origin := u.Origin
	if origin == "" {
		origin = "owned memory"
	}
	lines = append(lines, "Origin/session: "+safe(origin), "Place: "+safe(u.Scope), "Updated: "+u.Updated.Local().Format("2006-01-02 15:04:05 MST"))
	if err == nil {
		out.strength = max(1, meta.Strength)
		lines = append(lines, "Created: "+meta.Created.Local().Format("2006-01-02 15:04:05 MST"))
		if meta.CaptureSupersedes != "" {
			lines = append(lines, "Supersedes: "+meta.CaptureSupersedes)
		}
	} else {
		out.strength += out.reinforcements
	}
	out.reinforcements = max(out.reinforcements, out.strength-1)
	lines = append(lines, fmt.Sprintf("Strength: %d · reinforcements: %d", out.strength, out.reinforcements))
	if o, err := m.a.Origin(u); err == nil {
		path := o.Path
		if home := config.Home(); home != "" {
			path = strings.Replace(path, home, "$MUNINN_HOME", 1)
		}
		if home, _ := os.UserHomeDir(); home != "" {
			path = strings.Replace(path, home, "~", 1)
		}
		lines = append(lines, "Source: "+o.Source+" · "+safe(path))
	}
	if !u.Active {
		lines = append(lines, "State: retired (retained in history)")
	}
	out.origin = strings.Join(lines, "\n")
	return out
}
func (m model) detailBadges() string {
	u := m.unit
	return colourCategory("["+flat(u.Category)+"]") + " " + colourPlace("["+flat(m.placeName(u.Scope))+"]") + " " + dim("["+u.Updated.Local().Format("2006-01-02")+"] ["+u.SID+"]") + " " + paint(fmt.Sprintf("[strength %d]", m.metadata.strength), typeColours[2], false)
}
