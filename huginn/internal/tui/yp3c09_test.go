package tui

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// yp3c09 (Jørgen 2026-11-19 11:30Z, v0.7.81, p87ngv > before.md 3 of 6):
// the file view left two empty rows between the file and the keys while
// the file went on below. Those rows are the message line and the input
// line, drawn blank under the bottom rule; while nothing is in them they
// hold more of the file, and a message still gets its line.

// fileViewBoard is p87ngv with before.md (as attached to p87ngv on hug's
// board, hug/tasks 4cc90adf), an 80-line text file and a picture, opened
// as jhf at w x h with Files focused, in terminal env.
func fileViewBoard(t *testing.T, w, h int, env map[string]string) model {
	t.Helper()
	m := p87ngvBoard(t, "testdata/p87ngv/five-questions.md", w, h)
	m.opts.Getenv = func(k string) string { return env[k] }
	before, err := os.ReadFile("testdata/yp3c09/before.md")
	if err != nil {
		t.Fatal(err)
	}
	// The file as on hug's board, LF, whatever git checked out (Windows: CRLF).
	before = bytes.ReplaceAll(before, []byte("\r\n"), []byte("\n"))
	long := ""
	for i := 1; i <= 80; i++ {
		long += fmt.Sprintf("line %d of 80\n", i)
	}
	for _, f := range []struct {
		name string
		body []byte
	}{{"before.md", before}, {"long.txt", []byte(long)}, {"board.png", png1}} {
		if _, _, err := m.store.Attach("p87ngv", f.name, f.body, "", false); err != nil {
			t.Fatal(err)
		}
	}
	m.refresh("")
	return toFiles(t, m)
}

// fileFrameGap checks a file view screen of height h: the crumb, a rule,
// the file, a rule, then only lines that say something, then the keys.
// It returns the file's rows.
func fileFrameGap(t *testing.T, what, screen string, w, h int) []string {
	t.Helper()
	lines := strings.Split(screen, "\n")
	if len(lines) != h {
		t.Fatalf("%s: %d lines, want %d\n%s", what, len(lines), h, screen)
	}
	rule := strings.Repeat("─", w)
	bottom := -1
	for i := len(lines) - 1; i > 0; i-- {
		if lines[i] == rule {
			bottom = i
			break
		}
	}
	if bottom < 0 || lines[1] != rule {
		t.Fatalf("%s: not framed (top rule %v, bottom rule at %d):\n%s", what, lines[1] == rule, bottom, screen)
	}
	for i := bottom + 1; i < len(lines)-1; i++ {
		if strings.TrimSpace(lines[i]) == "" {
			t.Fatalf("%s: row %d of %d, between the bottom rule (row %d) and the keys, is blank:\n%s", what, i+1, h, bottom+1, screen)
		}
	}
	return lines[2:bottom]
}

// yp3c09 check 1 and 2: a file longer than the frame fills every row
// between the rules, and the bottom rule sits on the keys, at 100x30 and
// 160x45: Markdown, its raw source (M), a text file, and a picture's rows.
func TestFileViewFillsToTheKeys(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		t.Run(fmt.Sprintf("%dx%d", w, h), func(t *testing.T) {
			m := fileViewBoard(t, w, h, nil)
			var golden []string
			for _, view := range []struct{ file, keys, first string }{
				{"before.md", "", ""}, // the renderer opens with a blank line
				{"before.md", "M", "---"},
				{"long.txt", "", "line 1 of 80"},
			} {
				v := showFile(t, m, view.file)
				if view.keys != "" {
					v = press(v, view.keys)
				}
				what := view.file + " " + view.keys
				if v.viewport.TotalLineCount() <= h {
					t.Fatalf("%s: %d lines is not longer than the screen", what, v.viewport.TotalLineCount())
				}
				screen := ansi.Strip(v.View())
				body := fileFrameGap(t, what, screen, w, h)
				// Crumb, two rules, keys: the rest is the file.
				if len(body) != h-4 || len(body) != v.viewport.Height {
					t.Fatalf("%s: %d file rows (viewport %d), want %d\n%s", what, len(body), v.viewport.Height, h-4, screen)
				}
				// The last row is the file's line at the offset plus the
				// rows, not a filler: the file goes on past it.
				src := strings.Split(ansi.Strip(v.glanceBody), "\n")
				if last := v.viewport.YOffset + len(body) - 1; last >= len(src)-1 || strings.TrimRight(body[len(body)-1], " ") != strings.TrimRight(src[last], " ") {
					t.Fatalf("%s: the last file row %q is not line %d of %d of the file:\n%s", what, body[len(body)-1], last+1, len(src), screen)
				}
				if got := strings.TrimSpace(body[0]); view.first != "" && got != view.first {
					t.Fatalf("%s: first file row %q, want %q", what, got, view.first)
				}
				// The end of the file shows in the same rows.
				end := press(v, "G")
				if body := fileFrameGap(t, what+" G", ansi.Strip(end.View()), w, h); len(body) != h-4 {
					t.Fatalf("%s G: %d file rows, want %d", what, len(body), h-4)
				}
				golden = append(golden, "== "+what, screen)
			}
			goldenFile(t, fmt.Sprintf("testdata/yp3c09/file-view-%dx%d.txt", w, h), strings.Join(golden, "\n")+"\n")
		})
	}
}

// yp3c09: a message in the file view keeps its line under the bottom
// rule (crab 11:44Z: an error must still show), and the file gives that
// one row back; the input line stays gone while it is blank.
func TestFileViewMessageKeepsItsLine(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		m := fileViewBoard(t, w, h, nil)
		v := showFile(t, m, "long.txt")
		v = press(v, "p", "p", "p", "p", "p", "p", "p", "p") // to the first file, then past it
		if !strings.HasPrefix(v.message, "first file") {
			t.Fatalf("%dx%d: p past the first file says %q", w, h, v.message)
		}
		screen := ansi.Strip(v.View())
		body := fileFrameGap(t, "message", screen, w, h)
		lines := strings.Split(screen, "\n")
		if got := strings.TrimSpace(lines[h-2]); !strings.HasPrefix(got, "first file") {
			t.Fatalf("%dx%d: the message is not on the row above the keys: %q\n%s", w, h, got, screen)
		}
		if len(body) != h-5 || len(body) != v.viewport.Height {
			t.Fatalf("%dx%d: %d file rows (viewport %d) with a message, want %d\n%s", w, h, len(body), v.viewport.Height, h-5, screen)
		}
		// The next key clears it, and the row goes back to the file.
		next := press(v, "j")
		if next.message == "" {
			if body := fileFrameGap(t, "message gone", ansi.Strip(next.View()), w, h); len(body) != h-4 || next.viewport.Height != h-4 {
				t.Fatalf("%dx%d: after the message, %d file rows (viewport %d), want %d", w, h, len(body), next.viewport.Height, h-4)
			}
		}
	}
}

// yp3c09 check 2: a picture is drawn over every row between the rules,
// none left blank under it.
func TestPictureFillsToTheKeys(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		m := fileViewBoard(t, w, h, map[string]string{"TERM_PROGRAM": "iTerm.app"})
		v := showFile(t, m, "board.png")
		body := fileFrameGap(t, "board.png", ansi.Strip(v.frameView()), w, h)
		if len(body) != h-4 {
			t.Fatalf("%dx%d: %d picture rows, want %d", w, h, len(body), h-4)
		}
		if want := fmt.Sprintf(";height=%d;", h-4); !strings.Contains(v.View(), want) {
			t.Fatalf("%dx%d: the picture is not sized to the frame's %d rows (%s)", w, h, h-4, want)
		}
	}
}
