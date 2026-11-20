package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// d8yx7y check 2: the pie fills by quarters.
func TestPieSteps(t *testing.T) {
	for _, c := range []struct {
		done, total int
		want        string
	}{{0, 4, "○"}, {1, 4, "◔"}, {2, 4, "◑"}, {3, 4, "◕"}, {4, 4, "●"}, {1, 6, "◔"}, {1, 2, "◑"}, {5, 6, "◕"}, {0, 1, "○"}, {1, 1, "●"}} {
		if got := pie(c.done, c.total); got != c.want {
			t.Errorf("%d/%d: %s, want %s", c.done, c.total, got, c.want)
		}
	}
}

func withChecks(done, total int) task.Task {
	t := task.Task{ID: "aaaaaa", Path: "aaaaaa.a.md", Title: "A", State: "todo"}
	for i := 0; i < total; i++ {
		t.Acceptance = append(t.Acceptance, task.Acceptance{Text: "c", Checked: i < done})
	}
	return t
}

// d8yx7y check 3: all ticked green, partly yellow, none dim; NO_COLOR keeps
// glyph and text.
func TestProgressColours(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	m := manyModel(t)
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	cases := []struct {
		done, total int
		style       lipgloss.Style
		text        string
	}{{4, 4, m.styles.status["done"], "● 4/4"}, {2, 4, m.styles.progress, "◑ 2/4"}, {0, 4, m.styles.dim, "○ 4"}}
	seen := map[string]bool{}
	for _, c := range cases {
		got := m.progress(withChecks(c.done, c.total))
		want := c.style.Render(fit(c.text, m.progressWidth()-1)) + " "
		if got != want || !strings.Contains(got, "\x1b[38;2;") {
			t.Errorf("%d/%d: %q, want %q in truecolor", c.done, c.total, got, want)
		}
		seen[got[:strings.Index(got, "m")+1]] = true
	}
	if len(seen) != 3 {
		t.Fatalf("all, some and none ticked share a colour: %v", seen)
	}
	if got := m.progress(withChecks(0, 0)); strings.TrimSpace(got) != "" || lipgloss.Width(got) != m.progressWidth() {
		t.Fatalf("no checks: %q", got)
	}
	t.Setenv("NO_COLOR", "1")
	m.styles = newTheme()
	got := m.progress(withChecks(2, 4))
	if ansiColour.MatchString(got) || ansi.Strip(got) != fit("◑ 2/4", m.progressWidth()-1)+" " {
		t.Fatalf("NO_COLOR: %q", got)
	}
}

// d8yx7y check 1 at model level (the goldens show it on screen): each row
// of a task with checks has its glyph and N/M in its ✓ cell, after the
// title and the stage (911j70, order B), in overview, list and board.
func TestRowsShowProgress(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	for _, view := range []string{"overview", "list", "board"} {
		m.setView(view)
		text := ansi.Strip(m.View())
		seen := 0
		for _, tk := range m.all {
			if progressText(tk) == "" || !strings.Contains(text, tk.ID) {
				continue
			}
			// The lint mark's cell stands right before the id (e3sevr);
			// the pie later on the same row (911j70).
			row := ""
			for _, line := range strings.Split(text, "\n") {
				if strings.Contains(line, m.lintMark(tk.Path)+tk.ID) {
					row = line[strings.Index(line, m.lintMark(tk.Path)+tk.ID):]
					break
				}
			}
			if cell := progressText(tk); !strings.Contains(row, " "+cell+" ") {
				t.Fatalf("%s: %q not on %s's row:\n%s", view, cell, tk.ID, text)
			}
			seen++
		}
		if seen == 0 {
			t.Fatalf("%s: no rows with progress", view)
		}
	}
}
