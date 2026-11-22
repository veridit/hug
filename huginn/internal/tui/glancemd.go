package tui

import (
	"path/filepath"
	"strings"
	"time"

	bkey "github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Markdown files reuse the detail's quiet renderer (tmz060). The source
// is escaped before rendering. Formatting is a command, never UI work:
// even a 900 KiB file yields a frame and spinner before formatting starts.
// One generation identifies the file AND its render width. A result from
// a file we left, or an old width, cannot replace the current page.
type glanceMarkdown struct {
	source, formatted string
	eligible, raw     bool
	loading, started  bool
	width             int
	since             time.Time
}

type glanceFormattedMsg struct {
	gen  uint64
	body string
}

// A test counts calls; the command captures the function before it runs.
var formatGlanceMarkdown = renderMarkdown

func (m model) markdownHint() bkey.Binding {
	if m.glanceMD.raw {
		return hint("M", "Markdown")
	}
	return hint("M", "raw source")
}

func markdownName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

// escapedFileSource preserves line boundaries for the Markdown parser,
// but no terminal control from the file. Tabs are four spaces, as in raw.
func escapedFileSource(b []byte) string {
	text := strings.TrimSuffix(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = escapeControls(strings.ReplaceAll(line, "\t", "    "))
	}
	return strings.Join(lines, "\n")
}

func (m *model) beginGlanceMarkdown(source string) {
	m.glanceMD = glanceMarkdown{source: source, eligible: true, loading: true, width: m.width, since: time.Now()}
}

// formatGlance queues each generation exactly once. Only immutable values
// cross the command boundary, not the model's terminal renderer or state.
func (m *model) formatGlance() tea.Cmd {
	if !m.glance || !m.glanceMD.loading || m.glanceMD.started {
		return nil
	}
	m.glanceMD.started = true
	return m.glanceFormatCommand()
}

func (m model) glanceFormatCommand() tea.Cmd {
	gen, source, width := m.glanceGen, m.glanceMD.source, m.glanceMD.width
	dark, profile := m.styles.renderer.HasDarkBackground(), m.styles.renderer.ColorProfile()
	format := formatGlanceMarkdown
	return func() tea.Msg {
		return glanceFormattedMsg{gen: gen, body: format(source, width, dark, profile)}
	}
}

func (m *model) glanceFormatted(msg glanceFormattedMsg) {
	if !m.detail || !m.glance || msg.gen != m.glanceGen || !m.glanceMD.eligible {
		return
	}
	m.glanceMD.formatted, m.glanceMD.loading = msg.body, false
	m.drawGlanceMarkdown()
	m.seekSection() // the heading a link named, now it is drawn (zbdbtm)
}

// drawGlanceMarkdown also redoes an existing find against the text now
// drawn, never the Markdown source hidden behind it (4s3g5q).
func (m *model) drawGlanceMarkdown() {
	if !m.glanceMD.eligible {
		return
	}
	if (!m.glanceMD.loading || m.glanceMD.raw) && m.message == "formatting Markdown · M shows raw source" {
		m.message = ""
	}
	y := m.viewport.YOffset
	switch {
	case m.glanceMD.raw:
		var lines []string
		for _, line := range strings.Split(m.glanceMD.source, "\n") {
			lines = append(lines, wrap(line, m.width))
		}
		m.glanceBody = strings.Join(lines, "\n")
	case m.glanceMD.loading:
		m.glanceBody = "formatting Markdown · M shows raw source"
	default:
		m.glanceBody = m.glanceMD.formatted
	}
	m.viewport.SetContent(m.glanceBody)
	if m.find.text != "" && (!m.glanceMD.loading || m.glanceMD.raw) {
		m.refind()
		m.message = m.findSays()
	}
	m.viewport.SetYOffset(y)
}

func (m *model) toggleGlanceMarkdown() {
	if !m.glanceMD.eligible {
		return
	}
	m.glanceMD.raw = !m.glanceMD.raw
	m.drawGlanceMarkdown()
}

func (m model) markdownLoading() bool {
	return m.glance && m.glanceMD.loading && !m.glanceMD.raw
}

// resizeGlanceMarkdown invalidates only the formatted cache, not the file
// or raw/formatted choice. Late results for the old width are ignored.
func (m *model) resizeGlanceMarkdown() {
	if !m.glanceMD.eligible || m.glanceMD.width == m.width {
		return
	}
	m.glanceGen++
	m.glanceMD.width = m.width
	m.glanceMD.formatted = ""
	m.glanceMD.loading, m.glanceMD.started = true, false
	m.glanceMD.since = time.Now()
	m.drawGlanceMarkdown()
}
