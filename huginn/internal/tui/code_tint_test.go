package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// sgr is "38;2;R;G;B" (foreground) or "48;2;R;G;B" (background) for a hex
// colour, as a truecolor terminal receives it.
func sgr(layer int, hex string) string {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("%d;2;%d;%d;%d", layer, r, g, b)
}

// redSGR matches the reds code used to get: glamour's 256-colour 203 and
// any truecolor foreground with a strong red and weak green and blue.
var redSGR = regexp.MustCompile(`38;5;(?:203|196|160|9)m|38;2;(2[0-5][0-9]|1[89][0-9]);([0-9]|[1-9][0-9]|1[01][0-9]);([0-9]|[1-9][0-9]|1[01][0-9])m`)

// 8bsx4y: inline code and code blocks in a quiet tint, no padding, no red;
// light and dark terminals get their own pair; NO_COLOR is plain text.
func TestCodeQuietTint(t *testing.T) {
	m := fixture(t)
	src := "Run `hug task next` first.\n\n```go\nfmt.Println(\"synthetic\")\n```\n"
	for _, c := range []struct {
		dark   bool
		fg, bg string
	}{{true, "#c9d6e3", "#1a2630"}, {false, "#2b3a48", "#e8eef3"}} {
		m.styles.renderer.SetColorProfile(termenv.TrueColor)
		m.styles.renderer.SetHasDarkBackground(c.dark)
		out := m.markdown(src)
		for _, want := range []string{sgr(38, c.fg), sgr(48, c.bg)} {
			if !strings.Contains(out, want) {
				t.Fatalf("dark=%v: code lacks %s\n%q", c.dark, want, out)
			}
		}
		if red := redSGR.FindString(out); red != "" {
			t.Fatalf("dark=%v: red in code (%s)\n%q", c.dark, red, out)
		}
		plain := ansi.Strip(out)
		if !strings.Contains(plain, "Run hug task next first.") {
			t.Fatalf("dark=%v: inline code padded or fenced: %q", c.dark, plain)
		}
		if !strings.Contains(plain, `fmt.Println("synthetic")`) {
			t.Fatalf("dark=%v: code block text lost: %q", c.dark, plain)
		}
	}
	// NO_COLOR: no colour at all (glamour still writes empty and bold SGRs,
	// which carry none; the golden's colour contract uses the same rule).
	m.styles.renderer.SetColorProfile(termenv.Ascii)
	out := m.markdown(src)
	if c := ansiColour.FindString(out); c != "" {
		t.Fatalf("NO_COLOR: colour %q in code:\n%q", c, out)
	}
	if plain := ansi.Strip(out); !strings.Contains(plain, "Run hug task next first.") || !strings.Contains(plain, `fmt.Println("synthetic")`) {
		t.Fatalf("NO_COLOR: code text changed:\n%s", plain)
	}
}
