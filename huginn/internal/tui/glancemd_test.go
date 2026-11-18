package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

const markdownSample = "# A quiet heading\n\nA paragraph with `inline code` and **bold words**, long enough to wrap at a narrow pane without losing the words at its edge.\n\n- First item\n  - Nested item\n- Second item\n\n```go\nfmt.Println(\"quiet code\")\n```\n\nFind the needle here, and another needle here.\n"

// Replace only the fixture file. The viewer still uses its real read,
// classification, queued render command and completion message.
func markdownViewer(t *testing.T, width int, body []byte) (model, *[]string, string) {
	t.Helper()
	m, reads := glanceModel(t, width, 32)
	m = openTask(t, m, "000001")
	sel, _ := m.selected()
	path, err := m.insideFolder(sel, "bench/README.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	m = toFiles(t, m)
	return m, reads, path
}

func finishMarkdown(m model) model {
	next, _ := m.Update(m.glanceFormatCommand()())
	return next.(model)
}

// tmz060 check 1: one renderer/palette, 80/140-column fixtures, quiet
// headings and code, indented lists and no raw Markdown fences.
func TestMarkdownGlanceFormatted(t *testing.T) {
	var screens []string
	for _, width := range []int{80, 140} {
		m, reads, _ := markdownViewer(t, width, []byte(markdownSample))
		m.styles.renderer.SetColorProfile(termenv.TrueColor)
		m.styles.renderer.SetHasDarkBackground(true)
		m = press(m, " ")
		if !m.glanceMD.eligible || m.glanceMD.loading || len(*reads) != 1 {
			t.Fatalf("width %d: eligible=%v loading=%v reads=%v", width, m.glanceMD.eligible, m.glanceMD.loading, *reads)
		}
		body := ansi.Strip(m.glanceBody)
		for _, unwanted := range []string{"# A quiet heading", "`inline code`", "```"} {
			if strings.Contains(body, unwanted) {
				t.Fatalf("raw Markdown %q at width %d:\n%s", unwanted, width, body)
			}
		}
		for _, want := range []string{"A quiet heading", "inline code", "First item", "Nested item", "Second item", `fmt.Println("quiet code")`} {
			if !strings.Contains(body, want) {
				t.Fatalf("missing %q at width %d:\n%s", want, width, body)
			}
		}
		for _, want := range []string{sgr(38, "#00bfff"), sgr(38, "#c9d6e3"), sgr(48, "#1a2630")} {
			if !strings.Contains(m.glanceBody, want) {
				t.Fatalf("shared quiet palette lacks %q: %q", want, m.glanceBody)
			}
		}
		if !strings.Contains(body, "  • Nested item") {
			t.Fatalf("nested list lost its indentation: %q", body)
		}
		if red := redSGR.FindString(m.glanceBody); red != "" {
			t.Fatalf("red in Markdown: %s", red)
		}
		if m.glanceBody != m.markdown(escapedFileSource([]byte(markdownSample))) {
			t.Fatal("file used a different renderer than task details")
		}
		screens = append(screens, "== formatted at "+strconv.Itoa(width), ansi.Strip(m.View()))
		m = press(m, "M")
		screens = append(screens, "== raw at "+strconv.Itoa(width), ansi.Strip(m.View()))
		m.styles.renderer.SetColorProfile(termenv.Ascii)
	}
	goldenFile(t, "testdata/detail/markdown-file.txt", strings.Join(screens, "\n")+"\n")
}

// tmz060 check 2: controls are escaped before the renderer sees them,
// including when Markdown punctuation surrounds the ESC/BEL bytes.
func TestMarkdownGlanceEscapesBeforeRendering(t *testing.T) {
	src := []byte("# Safe\n\n`\x1b[2J` and \x07 bell\rnot a return\n\n```\n\x1b]52;c;BAD\a\n```\n")
	m, _, _ := markdownViewer(t, 100, src)
	original := formatGlanceMarkdown
	var got string
	formatGlanceMarkdown = func(source string, width int, dark bool, profile termenv.Profile) string {
		got = source
		return original(source, width, dark, profile)
	}
	t.Cleanup(func() { formatGlanceMarkdown = original })
	m = press(m, " ")
	if strings.ContainsAny(got, "\x1b\a\r") || !strings.Contains(got, `\x1b[2J`) || !strings.Contains(got, `\a bell`) {
		t.Fatalf("renderer received unsafe/unescaped input: %q", got)
	}
	for _, raw := range []bool{false, true} {
		if raw {
			m = press(m, "M")
		}
		body := ansi.Strip(m.glanceBody)
		if strings.ContainsAny(body, "\x1b\a\r") || !strings.Contains(body, `\x1b[2J`) || !strings.Contains(body, `\a bell`) {
			t.Fatalf("raw=%v: control escaping lost: %q", raw, body)
		}
	}
}

// tmz060 check 3: M switches one cached file without a read or write;
// non-Markdown stays exactly the original plain text.
func TestMarkdownGlanceToggleAndPlainFiles(t *testing.T) {
	m, reads, path := markdownViewer(t, 100, []byte(markdownSample))
	m = press(m, " ")
	formatted := m.glanceBody
	for i := 0; i < 2; i++ {
		m = press(m, "M")
		if !m.glanceMD.raw || !strings.Contains(m.glanceBody, "# A quiet heading") || !strings.Contains(m.glanceBody, "```") {
			t.Fatal("M did not show source")
		}
		if !strings.Contains(ansi.Strip(m.keyBar()), "Markdown") {
			t.Fatalf("raw footer does not name M Markdown: %s", ansi.Strip(m.keyBar()))
		}
		m = press(m, "M")
		if m.glanceMD.raw || m.glanceBody != formatted {
			t.Fatal("M did not restore cached formatting")
		}
		if !strings.Contains(ansi.Strip(m.keyBar()), "M raw source") {
			t.Fatalf("formatted footer does not name M raw source: %s", ansi.Strip(m.keyBar()))
		}
	}
	if len(*reads) != 1 {
		t.Fatalf("toggle reread the file: %v", *reads)
	}
	if disk, err := os.ReadFile(path); err != nil || string(disk) != markdownSample {
		t.Fatalf("viewer changed source: %q %v", disk, err)
	}
	// A later file starts formatted, not in the previous file's raw mode.
	m = press(m, "M", "u", " ")
	if m.glanceMD.raw {
		t.Fatal("raw choice leaked to another opening")
	}
	for _, name := range []string{"plain.txt", "plain.go"} {
		file := filepath.Join(filepath.Dir(path), name)
		if err := os.WriteFile(file, []byte("# plain `source`\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		m = press(m, "u")
		m.refresh("")
		sel, _ := m.selected()
		for i, f := range m.filesOf(sel) {
			if strings.HasSuffix(f, name) {
				m.file = i
			}
		}
		m = press(m, " ")
		before := m.glanceBody
		m = press(m, "M")
		if m.glanceMD.eligible || m.glanceBody != before || !strings.Contains(before, "# plain `source`") {
			t.Fatalf("M changed %s: %q -> %q", name, before, m.glanceBody)
		}
	}
}

// tmz060 check 4: a real 900 KiB Markdown file queues work, draws a spinner
// BEFORE any formatter call, schedules only once, and accepts its result.
// No elapsed-time assertion or sleeping: the test controls delivery.
func TestMarkdownGlanceLargeIsAsync(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 900<<10)
	m, reads, _ := markdownViewer(t, 100, body)
	original := formatGlanceMarkdown
	calls, size := 0, 0
	formatGlanceMarkdown = func(source string, width int, dark bool, profile termenv.Profile) string {
		calls++
		size = len(source)
		return "large Markdown rendered"
	}
	t.Cleanup(func() { formatGlanceMarkdown = original })
	m, cmd := key(m, " ")
	queued := cmd
	if calls != 0 || cmd == nil || !m.glanceMD.loading || !m.glanceMD.started || len(*reads) != 1 {
		t.Fatalf("Space rendered synchronously: calls %d cmd nil=%v state %+v reads %d", calls, cmd == nil, m.glanceMD, len(*reads))
	}
	if !strings.Contains(m.spinner(), "formatting Markdown") || !strings.Contains(ansi.Strip(m.View()), "formatting Markdown") {
		t.Fatal("no visible spinner before formatting")
	}
	if m.formatGlance() != nil {
		t.Fatal("queued the same render twice")
	}
	m, _ = key(m, "/")
	if m.prompt != "" || calls != 0 {
		t.Fatal("find searched the loading placeholder or forced a render")
	}
	// Execute the actual command returned by Space, not a reconstructed
	// formatter call. The first batch child is rendering, the other is
	// the existing spinner's timer. Withholding this child is the wait.
	msg := queued()
	for {
		batch, ok := msg.(tea.BatchMsg)
		if !ok {
			break
		}
		msg = batch[0]()
	}
	if _, ok := msg.(glanceFormattedMsg); !ok {
		t.Fatalf("Space did not return a renderer command: %T", msg)
	}
	next, _ := m.Update(msg)
	m = next.(model)
	if calls != 1 || size != 900<<10 || m.glanceMD.loading || m.glanceBody != "large Markdown rendered" || m.spinner() != "" {
		t.Fatalf("completion calls=%d bytes=%d loading=%v body=%q spinner=%q", calls, size, m.glanceMD.loading, m.glanceBody, m.spinner())
	}
	if strings.Contains(m.message, "formatting Markdown") {
		t.Fatalf("completed render left a loading footer: %q", m.message)
	}
}

func TestMarkdownGlanceRawWhileLoading(t *testing.T) {
	m, _, _ := markdownViewer(t, 80, []byte(markdownSample))
	m, _ = key(m, " ")
	m, _ = key(m, "/") // asks to wait for formatted text
	m, _ = key(m, "M") // raw source is available without the formatter
	if !m.glanceMD.loading || !m.glanceMD.raw || m.spinner() != "" || strings.Contains(m.message, "formatting Markdown") {
		t.Fatalf("raw while loading: loading=%v raw=%v spinner=%q message=%q", m.glanceMD.loading, m.glanceMD.raw, m.spinner(), m.message)
	}
	if !strings.Contains(m.glanceBody, "# A quiet heading") {
		t.Fatal("M did not show source before render completion")
	}
	m = finishMarkdown(m)
	if !m.glanceMD.raw || !strings.Contains(m.glanceBody, "# A quiet heading") {
		t.Fatal("late completion replaced raw source")
	}
}

func TestMarkdownGlanceLateResultAndResize(t *testing.T) {
	m, _, _ := markdownViewer(t, 80, []byte(markdownSample))
	m, _ = key(m, " ")
	old := m.glanceFormatCommand()()
	m, _ = key(m, "n") // results.tsv: no Markdown
	next, _ := m.Update(old)
	m = next.(model)
	if m.glanceMD.eligible || strings.Contains(m.glanceBody, "A quiet heading") {
		t.Fatal("old Markdown overwrote the next file")
	}
	m = press(m, "p")
	m, _ = key(m, "M") // raw source
	next, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 32})
	m = next.(model)
	newGen := m.glanceGen
	next, _ = m.Update(old)
	m = next.(model)
	if !m.glanceMD.loading || m.glanceGen != newGen || !m.glanceMD.raw {
		t.Fatal("resize accepted an old generation or lost raw choice")
	}
	m = finishMarkdown(m)
	if !m.glanceMD.raw || !strings.Contains(m.glanceBody, "# A quiet heading") {
		t.Fatal("completion overrode raw mode")
	}
	m = press(m, "M")
	if strings.Contains(m.glanceBody, "# A quiet heading") || m.glanceMD.width != 140 {
		t.Fatal("resize did not provide formatted text at the new width")
	}
}

func TestMarkdownGlanceLimits(t *testing.T) {
	for _, c := range []struct {
		body []byte
		want string
	}{{bytes.Repeat([]byte("x"), glanceLimit+1), "too large"}, {[]byte{'#', ' ', 0, 'x'}, "binary"}, {[]byte{0xff, 0xfe}, "binary"}} {
		m, _, _ := markdownViewer(t, 100, c.body)
		m, _ = key(m, " ")
		if m.glanceMD.eligible || m.glanceMD.loading || !strings.Contains(m.glanceBody, c.want) {
			t.Fatalf("limits bypassed: eligible=%v loading=%v body=%q", m.glanceMD.eligible, m.glanceMD.loading, m.glanceBody)
		}
	}
	if !markdownName("a.MD") || !markdownName("a.markdown") || markdownName("a.md.txt") {
		t.Fatal("Markdown file extensions misclassified")
	}
}

// Rendered-text find uses the exact page displayed. M recomputes hits,
// and neither form changes the list query or writes the file.
func TestMarkdownGlanceFindRenderedText(t *testing.T) {
	m, _, _ := markdownViewer(t, 100, []byte(markdownSample))
	m = press(m, " ")
	m = press(typeFind(m, "needle"), "enter")
	if len(m.find.hits) != 2 {
		t.Fatalf("rendered matches: %+v", m.find.hits)
	}
	m = press(m, "n", "N")
	if !strings.Contains(m.message, "match 1 of 2") {
		t.Fatalf("rendered-text n/N: %q", m.message)
	}
	m = press(m, "M")
	if len(m.find.hits) != 2 || !m.glanceMD.raw {
		t.Fatal("toggle lost find")
	}
	m = press(m, "M", "esc", "u", "u")
	if m.query != "" || m.detail {
		t.Fatalf("find became a list filter: %q, detail %v", m.query, m.detail)
	}
	m, _, _ = markdownViewer(t, 100, []byte(markdownSample))
	m = typeFind(press(m, " "), "# A quiet")
	if len(m.find.hits) != 0 {
		t.Fatal("formatted find searched hidden source")
	}
}
