package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// sgrs are the SGR parameter lists in s, in order: "1", "48;2;0;191;255".
func sgrs(s string) []string {
	var out []string
	for {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			return out
		}
		s = s[i+2:]
		j := strings.IndexByte(s, 'm')
		if j < 0 {
			return out
		}
		out = append(out, s[:j])
		s = s[j+1:]
	}
}

// loud says which of bold or a background fill the SGR lists turn on.
func loud(codes []string) []string {
	var found []string
	for _, c := range codes {
		parts := strings.Split(c, ";")
		for i := 0; i < len(parts); i++ {
			switch parts[i] {
			case "1":
				found = append(found, "bold")
			case "48":
				found = append(found, "background "+c)
				i = len(parts)
			case "38":
				i += 4 // 38;2;r;g;b: its numbers are no codes
			}
		}
	}
	return found
}

// lineWith is the coloured line of the page whose plain text holds want.
func lineWith(t *testing.T, m model, want string) string {
	t.Helper()
	for _, l := range strings.Split(m.View(), "\n") {
		if strings.Contains(ansi.Strip(l), want) {
			return l
		}
	}
	t.Fatalf("no line holds %q:\n%s", want, ansi.Strip(m.View()))
	return ""
}

// kg07bt check 1 (Jørgen 14:58 on 0gztsy: "too much visual noise"): the
// focused ACCEPTANCE heading has no background fill and the weight of the
// other section headings; the ▸ and the dim key hint say it has focus.
func TestFocusedHeadingIsCalm(t *testing.T) {
	m := askedBoard(t)
	m.width, m.height = 100, 40
	m.resize()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { m.styles.renderer.SetColorProfile(termenv.Ascii) })
	m = openTask(t, m, "aaaaaa")
	if m.detailPane != focusAcceptance {
		t.Fatalf("focus pane %d, want the checks", m.detailPane)
	}
	focused := lineWith(t, m, "ACCEPTANCE")
	other := lineWith(t, m, "DESCRIPTION")
	if !strings.HasPrefix(ansi.Strip(focused), "▸ ACCEPTANCE") {
		t.Fatalf("the focused heading lost its ▸: %q", ansi.Strip(focused))
	}
	for _, c := range sgrs(focused) {
		if strings.Contains(c, "48;") {
			t.Fatalf("the focused heading has a background: %q", focused)
		}
	}
	bold := func(l string) bool {
		for _, f := range loud(sgrs(l)) {
			if f == "bold" {
				return true
			}
		}
		return false
	}
	if bold(focused) != bold(other) {
		t.Fatalf("ACCEPTANCE bold %v, DESCRIPTION bold %v: not the same weight\n%q\n%q", bold(focused), bold(other), focused, other)
	}
}

// kg07bt check 2: the focused asked check reads as a normal row: no
// background fill, no bold, its text in the colour it has unfocused; the
// > before it is the focus sign (q3sh69: no underline; it pulses). The same for
// a question that names no check. Focus and keys are unchanged.
func TestFocusedAskedRowIsCalm(t *testing.T) {
	for _, c := range []struct{ id, want, pane string }{
		{"aaaaaa", "> ? ☐ 2. b", "checks"},
		{"cccccc", "> ? which colour should the ASKED title", "question"},
	} {
		m := askedBoard(t)
		m.width, m.height = 100, 40
		m.resize()
		m.styles.renderer.SetColorProfile(termenv.TrueColor)
		m = openTask(t, m, c.id)
		row := lineWith(t, m, c.want)
		// The > is the focus sign, no underline (q3sh69 replaced
		// kg07bt's theme.cursor: an underline reads as a key); it
		// pulses, and opens drawn on (TestFocusSignPulsesInPage).
		cursor := m.signIn(">", false)
		if !strings.HasPrefix(row, cursor) || sgrAt(row, 0).underline {
			t.Fatalf("%s: the > is not the focus sign %q: %q", c.id, cursor, row)
		}
		rest := strings.TrimPrefix(row, cursor)
		if f := loud(sgrs(rest)); len(f) != 0 {
			t.Fatalf("%s: the focused %s row is %v after the cursor: %q", c.id, c.pane, f, row)
		}
		for _, code := range sgrs(rest) {
			for _, p := range strings.Split(code, ";") {
				if p == "4" {
					t.Fatalf("%s: more than the cursor is underlined: %q", c.id, row)
				}
			}
		}
		m.styles.renderer.SetColorProfile(termenv.Ascii)
		// Keys unchanged: a answers.
		if after := press(m, "a"); after.prompt != answerPrompt {
			t.Fatalf("%s: a on the focused row: prompt %q", c.id, after.prompt)
		}
	}
}

// kg07bt: the focused rows in colour at 80 and 140, each SGR shown as
// <codes>, so a golden diff shows what changed in look: the ACCEPTANCE
// heading, the asked check and the question block.
func TestCalmFocusGolden(t *testing.T) {
	visible := func(s string) string {
		var b strings.Builder
		for {
			i := strings.Index(s, "\x1b[")
			if i < 0 {
				b.WriteString(s)
				return strings.TrimRight(b.String(), " ")
			}
			j := strings.IndexByte(s[i:], 'm')
			b.WriteString(s[:i] + "<" + s[i+2:i+j] + ">")
			s = s[i+j+1:]
		}
	}
	var golden []string
	for _, w := range []int{80, 140} {
		for _, c := range []struct {
			id   string
			rows []string
		}{
			{"aaaaaa", []string{"ACCEPTANCE", "> ? ☐ 2. b", "☑ 1. a"}},
			{"cccccc", []string{"> ? which colour should the ASKED title"}},
		} {
			m := askedBoard(t)
			m.width, m.height = w, 40
			m.resize()
			m.styles.renderer.SetColorProfile(termenv.TrueColor)
			m = openTask(t, m, c.id)
			golden = append(golden, fmt.Sprintf("== %s at %d", c.id, w))
			for _, r := range c.rows {
				golden = append(golden, visible(lineWith(t, m, r)))
			}
		}
	}
	goldenFile(t, "testdata/detail/calm-focus.txt", strings.Join(golden, "\n")+"\n")
}
