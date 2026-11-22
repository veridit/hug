package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// zhvkmj check 1: a done task shows its check count in the done colour,
// "● 5", never 5/5; todo with nothing ticked shows "○ 3" dim; a fraction
// stays on todo, doing and blocked while some are ticked (Jørgen
// 2026-11-17: it marks resumable work); dropped shows none.
func TestProgressCellByStatus(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	m := manyModel(t)
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	w := m.progressWidth() - 1
	for _, c := range []struct {
		status      string
		done, total int
		text        string
		style       string
	}{
		{"done", 5, 5, "● 5", "done"},
		{"todo", 0, 3, "○ 3", "dim"},
		{"todo", 4, 5, "◕ 4/5", "progress"},
		{"doing", 1, 4, "◔ 1/4", "progress"},
		{"doing", 0, 4, "○ 4", "dim"},
		{"todo", 3, 3, "● 3/3", "done"},
		{"dropped", 1, 4, "", ""},
	} {
		tk := withChecks(c.done, c.total)
		tk.State = c.status
		got := m.progress(tk)
		if c.text == "" {
			if strings.TrimSpace(got) != "" {
				t.Errorf("%s %d/%d: %q, want blank", c.status, c.done, c.total, got)
			}
			continue
		}
		style := map[string]func(...string) string{"done": m.styles.status["done"].Render, "dim": m.styles.dim.Render, "progress": m.styles.progress.Render}[c.style]
		if want := style(fit(c.text, w)) + " "; got != want {
			t.Errorf("%s %d/%d: %q, want %q", c.status, c.done, c.total, got, want)
		}
	}
}

// zhvkmj check 2: each overview section title takes its status's colour,
// the same as the count line and the status word.
func TestSectionTitlesShareStatusColours(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	m := longBoard(t)
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.width, m.height = 140, 40
	m.resize()
	m.setView("overview")
	m.pane = sectionLint // no title is drawn selected but LINT
	m.clamp()
	screen := m.View()
	prefix := func(style string) string {
		r := m.styles.status[style].Bold(true).Render("X")
		return r[:strings.Index(r, "X")]
	}
	for title, status := range map[string]string{"IN PROGRESS": "doing", "NEXT UP": "todo", "RECENTLY DONE": "done"} {
		line := ""
		for _, l := range strings.Split(screen, "\n") {
			if strings.Contains(ansi.Strip(l), title) {
				line = l
				break
			}
		}
		// The title on its rule (v68n82): the rule's lead, then the title.
		if _, title, ok := strings.Cut(line, "─── "+"\x1b[0m"); !ok || !strings.HasPrefix(title, prefix(status)) {
			t.Errorf("%s title is not in the %s colour: %q", title, status, line)
		}
		if !strings.Contains(screen, m.styles.status[status].Render(status + " ")[:len(prefix(status))-len("\x1b[1m")]) {
			t.Errorf("no %s count in the %s colour", status, status)
		}
	}
	// The status word on a done row keeps the done colour.
	m.setView("list")
	for _, l := range strings.Split(m.View(), "\n") {
		if strings.Contains(ansi.Strip(l), "done A very long") && !strings.Contains(l, m.styles.status["done"].Render(fit("done", stageCell-1))) {
			t.Errorf("done row status word not in the done colour: %q", l)
		}
	}
}

// zhvkmj check 3: without colour, everything still reads by text: the
// pie glyph and count, the status word, the section title.
func TestNoColorReadsByText(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := longBoard(t)
	m.styles = newTheme()
	m.width, m.height = 140, 40
	m.resize()
	m.retireDefault() // the done task's ● in the list: the default aside (7zg9mf)
	m.setView("list")
	screen := m.View()
	if strings.Contains(screen, "\x1b[3") {
		t.Fatalf("colour with NO_COLOR:\n%q", screen)
	}
	for _, want := range []string{"● 12 ", "○ 12 ", "doing", "todo", "done"} {
		if !strings.Contains(screen, want) {
			t.Errorf("NO_COLOR list lacks %q", want)
		}
	}
	if strings.Contains(screen, "12/12") {
		t.Errorf("a done row shows 12/12")
	}
}
