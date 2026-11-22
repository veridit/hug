package tui

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// rgb is a #rrggbb colour as numbers.
func rgb(t *testing.T, hex string) [3]float64 {
	t.Helper()
	var out [3]float64
	for i := range 3 {
		v, err := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		if err != nil {
			t.Fatalf("colour %q: %v", hex, err)
		}
		out[i] = float64(v)
	}
	return out
}

// apart says whether two colours are told apart at a glance: their RGB
// distance is at least 60 (of 441).
func apart(t *testing.T, a, b string) bool {
	x, y := rgb(t, a), rgb(t, b)
	return math.Sqrt((x[0]-y[0])*(x[0]-y[0])+(x[1]-y[1])*(x[1]-y[1])+(x[2]-y[2])*(x[2]-y[2])) >= 60
}

// zhvkmj (crab 02:52 and 03:01, Jørgen 02:32: "it bugs me that waiting
// and lint are two different issues" in one colour): one colour per
// meaning. Amber is only what waits on you, red only what is wrong; no
// type or priority takes either, and the five types stay apart from each
// other, in the dark and the light palette.
func TestOneColourPerMeaning(t *testing.T) {
	for side, name := range []string{"dark", "light"} {
		meanings := map[string]string{"waits on you": forYouColour[side], "a problem": problemColour[side], "depends": dependsColour[side]}
		// zhvkmj check 6 (Jørgen 19:52: "the color for dependency is red
		// ... that seems like dependencies is a problem, it isn't"): depends
		// is its own calm colour, apart from the red, the amber and dim.
		for what, c := range map[string]string{"a problem": problemColour[side], "waits on you": forYouColour[side], "dim": [2]string{"#8795a8", "#586474"}[side]} {
			if !apart(t, dependsColour[side], c) {
				t.Errorf("%s: depends %s is close to %s %s", name, dependsColour[side], what, c)
			}
		}
		for i, ty := range task.Types {
			c := typeColours[i][side]
			for what, mc := range meanings {
				if !apart(t, c, mc) {
					t.Errorf("%s: type %s %s is close to %s %s", name, ty, c, what, mc)
				}
			}
			for j := i + 1; j < len(task.Types); j++ {
				if !apart(t, c, typeColours[j][side]) {
					t.Errorf("%s: types %s %s and %s %s look alike", name, ty, c, task.Types[j], typeColours[j][side])
				}
			}
		}
		for i, p := range []string{"critical", "high", "medium", "low"} {
			c := priorityColours[i][side]
			for what, mc := range meanings {
				if !apart(t, c, mc) {
					t.Errorf("%s: priority %s %s is close to %s %s", name, p, c, what, mc)
				}
			}
		}
	}

	// On screen: the ASKED title and '? N for you' amber; the WAITING
	// title, a wait on CI and advice dim; broken lint's ! and the LINT
	// title red; amber nowhere but on what waits on you.
	m := stageBoard(t)
	m.opts.User = "jhf"
	m.refresh("")
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { m.styles.renderer.SetColorProfile(termenv.Ascii) })
	m.width, m.height = 140, 40
	m.resize()
	m.setView("overview")
	m.clamp()
	// Each meaning's colour as the renderer writes it.
	sgr := func(st lipgloss.Style) string {
		out := st.Render("x")
		i := strings.Index(out, "38;2;")
		if i < 0 {
			t.Fatalf("no colour in %q", out)
		}
		return out[i : strings.IndexByte(out[i:], 'm')+i]
	}
	amber, red, dim := sgr(m.styles.forYou), sgr(m.styles.problem), sgr(m.styles.dim)
	calm := sgr(m.styles.status["depends"])
	if calm == red || calm == amber || calm == dim {
		t.Errorf("depends is drawn %s, the colour of another meaning (red %s, amber %s, dim %s)", calm, red, amber, dim)
	}
	lines := strings.Split(m.View(), "\n")
	line := func(has string) string {
		for _, l := range lines {
			if strings.Contains(ansi.Strip(l), has) {
				return l
			}
		}
		t.Fatalf("no line holds %q", has)
		return ""
	}
	for has, want := range map[string]string{"ASKED · for you": amber, "? 1 for you": amber, "WAITING": dim, "LINT · tasks": red} {
		if l := line(has); !strings.Contains(l, want) {
			t.Errorf("%q is not in its colour %s: %q", has, want, l)
		}
	}
	// cccccc waits on CI: its stage word is dim, not amber.
	if l := line("cccccc"); strings.Contains(l, amber) || !strings.Contains(l, dim) {
		t.Errorf("the wait on CI is not calm: %q", l)
	}
	// Amber only on the rows and lines about what waits on jhf (eeeeee).
	for _, l := range lines {
		plain := ansi.Strip(l)
		if strings.Contains(l, amber) && !strings.Contains(plain, "for you") && !strings.Contains(plain, "eeeeee") {
			t.Errorf("amber on a line that waits on no one: %q", plain)
		}
	}
	// Check 6: depends in its calm colour, never red, in the count line,
	// the row's stage word, its opened note and the board's DEPENDS
	// heading; red stays on broken lint alone.
	if l := line("(1 depends)"); !strings.Contains(l, calm) || strings.Contains(l, red) {
		t.Errorf("the count line's depends is not calm %s: %q", calm, l)
	}
	if l := line("dddddd"); !strings.Contains(l, calm) || strings.Contains(l, red) {
		t.Errorf("the depends row is not calm: %q", l)
	}
	for _, l := range lines {
		plain := ansi.Strip(l)
		if strings.Contains(l, red) && !strings.Contains(plain, "LINT") && !strings.Contains(plain, "broken") && !strings.Contains(plain, "!") {
			t.Errorf("red on a line with nothing wrong: %q", plain)
		}
	}
	for _, id := range []string{"dddddd"} {
		for p := range overviewSections {
			for _, i := range m.paneIndices(p) {
				if m.visible[i].ID == id {
					m.pane, m.cursor = p, i
				}
			}
		}
		for _, l := range strings.Split(m.View(), "\n") {
			// Focused, the note is in the selected row's style: never red.
			if strings.Contains(ansi.Strip(l), "└─ depends on") && strings.Contains(l, red) {
				t.Errorf("the depends note is not calm: %q", l)
			}
		}
	}
	m.setView("board")
	m.clamp()
	board := strings.Split(m.View(), "\n")
	found := false
	for _, l := range board {
		plain := ansi.Strip(l)
		if at := strings.Index(plain, "DEPENDS ["); at >= 0 {
			found = true
			if !strings.Contains(l, calm+"mDEPENDS [") { // on its titled rule (v68n82)
				t.Errorf("the DEPENDS heading is not calm %s: %q", calm, l)
			}
		}
	}
	if !found {
		t.Errorf("no DEPENDS heading on the board:\n%s", ansi.Strip(m.View()))
	}
	// The golden at 140 in colour (zhvkmj check 6): the overview and the
	// board with their colour codes, readable (ESC as \e), reviewed.
	m.setView("overview")
	m.clamp()
	overview := m.View()
	m.setView("board")
	m.clamp()
	shot := "== overview 140x40 truecolor\n" + overview + "\n== board 140x40 truecolor\n" + m.View() + "\n"
	golden(t, "testdata/colour/depends-140.txt", "TestOneColourPerMeaning", strings.ReplaceAll(shot, "\x1b", `\e`))
}
