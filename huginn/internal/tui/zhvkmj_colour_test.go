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
		meanings := map[string]string{"waits on you": forYouColour[side], "a problem": problemColour[side]}
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
}
