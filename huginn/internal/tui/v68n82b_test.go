package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// v68n82 check 2, variant B: the next section's titled rule doubles as
// the end of the one above, and carries its ↓ ('↓── NEXT UP …', or
// '↓↑─' when the section it opens is scrolled too); the last section
// with rows gets an end rule. Built behind sectionFrame, so Jørgen's
// pick of B is one line; A stays the default.
func TestSharedRulesVariantB(t *testing.T) {
	if sectionFrame != "end" {
		t.Fatalf("the default frame is %q, want variant A, \"end\"", sectionFrame)
	}
	sectionFrame = "shared"
	t.Cleanup(func() { sectionFrame = "end" })
	for _, size := range gridSizes {
		w, h := size[0], size[1]
		what := fmt.Sprintf("overview B at %dx%d", w, h)
		m := gridModel(t, w, h, "overview")
		lines := strings.Split(ansi.Strip(m.overviewText()), "\n")
		secs := sectionsOf(t, what, lines, overviewSections, true)
		_, downs := checkArrows(t, what, lines, secs, true)
		if downs == 0 {
			t.Fatalf("%s: no ↓\n%s", what, strings.Join(lines, "\n"))
		}
		ends := 0
		for k, s := range secs {
			if s.end >= 0 {
				ends++
				if k != len(secs)-1 {
					t.Fatalf("%s: %s has an end rule of its own; the next title ends it\n%s", what, s.title, strings.Join(lines, "\n"))
				}
			}
		}
		if last := secs[len(secs)-1]; len(last.rows) > 0 && ends != 1 {
			t.Fatalf("%s: the last section has rows and %d end rules", what, ends)
		}
		// B draws one rule less per section with rows than A, and the
		// rows get them: the body is as tall, the rows more.
		a := gridModel(t, w, h, "overview")
		sectionFrame = "end"
		aLines := strings.Split(ansi.Strip(a.overviewText()), "\n")
		aRows := 0
		for _, s := range sectionsOf(t, what+" (A)", aLines, overviewSections, false) {
			aRows += len(s.rows)
		}
		sectionFrame = "shared"
		bRows := 0
		for _, s := range secs {
			bRows += len(s.rows)
		}
		if bRows <= aRows || len(lines) > m.bodyHeight() {
			t.Fatalf("%s: B shows %d row lines, A %d; body %d lines of %d", what, bRows, aRows, len(lines), m.bodyHeight())
		}
		goldenFile(t, fmt.Sprintf("testdata/v68n82/overview-shared-%dx%d.txt", w, h), ansi.Strip(m.View())+"\n")
	}
}
