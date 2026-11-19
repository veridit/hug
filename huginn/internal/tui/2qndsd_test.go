package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// imageViewer is task 000002 (rows.ans, shot.png, no notes) plus
// board.png attached with a note, in terminal env, Files focused.
func imageViewer(t *testing.T, w, h int, env map[string]string) model {
	t.Helper()
	m, _ := pictureModel(t, w, h, env, "")
	if _, _, err := m.store.Attach("000002", "board.png", png1, "screenshot of the board", false); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	return toFiles(t, openTask(t, m, "000002"))
}

// showFile marks name in Files and presses Space.
func showFile(t *testing.T, m model, name string) model {
	t.Helper()
	files := m.filesOf(mustSelected(t, m))
	for i, f := range files {
		if f == name {
			m.file = i
			return press(m, " ")
		}
	}
	t.Fatalf("no %s in %v", name, files)
	return m
}

// frameOf checks the file frame (z82qsy check 4) on a screen and returns
// its body rows: crumb, the note when there is one, a rule, the body, a
// rule, then the message line and the input line when they say something
// (yp3c09), and the keys.
func frameOf(t *testing.T, m model, screen, crumb, note string) []string {
	t.Helper()
	lines := strings.Split(screen, "\n")
	trim := func(i int) string { return strings.TrimRight(lines[i], " ") }
	rule := strings.Repeat("─", m.width)
	if !strings.HasPrefix(trim(0), crumb) {
		t.Fatalf("crumb %q, want %q…\n%s", trim(0), crumb, screen)
	}
	top := 1
	if note != "" {
		if trim(1) != note {
			t.Fatalf("note %q, want %q\n%s", trim(1), note, screen)
		}
		top = 2
	}
	foot, _ := m.footRows()
	bottom := len(lines) - 2 - len(foot)
	if lines[top] != rule || lines[bottom] != rule || len(lines) != m.height {
		t.Fatalf("not framed (rules at %d and %d, %d lines):\n%s", top, bottom, len(lines), screen)
	}
	return lines[top+1 : bottom]
}

// 2qndsd check 1: a PNG opens in the file frame: crumb, note, rule, the
// image in the frame's rows, rule, keys. Where the terminal draws no
// images, its one line sits in the frame (golden).
func TestImageOpensInTheFrame(t *testing.T) {
	var golden []string
	for _, w := range []int{80, 140} {
		m := imageViewer(t, w, 24, nil)
		v := showFile(t, m, "board.png")
		if !v.glance {
			t.Fatalf("at %d Space on an image did not open it in the frame", w)
		}
		screen := ansi.Strip(v.View())
		body := frameOf(t, v, screen, "HUG · list › 000002 › ", "screenshot of the board")
		if got := strings.TrimRight(body[0], " "); got != "image, 67 B: this terminal draws no images (iTerm2 and kitty do): Enter opens it" {
			t.Fatalf("at %d the fallback reads %q", w, got)
		}
		golden = append(golden, fmt.Sprintf("== board.png, no image protocol, at %d", w), screen)
	}
	goldenFile(t, "testdata/detail/image-frame.txt", strings.Join(golden, "\n")+"\n")
}

// 2qndsd check 1: in iTerm2 and kitty (and through tmux) the image is
// drawn in the frame's rows: the cursor saved, moved to the first row
// under the top rule, the image sized to the frame's rows and width, the
// cursor put back. It is drawn after the frame, so the frame's blank rows
// do not paint over it, and the frame itself is the fallback's, empty.
func TestImageDrawnInTheFrameRows(t *testing.T) {
	for _, c := range []struct {
		env        map[string]string
		start, end string
	}{
		{map[string]string{"TERM_PROGRAM": "iTerm.app"}, "\x1b]1337;File=name=Ym9hcmQucG5n;size=67;inline=1;width=100;height=19;preserveAspectRatio=1:", "\a"},
		{map[string]string{"TERM": "xterm-kitty"}, "\x1b_Ga=T,f=100,C=1,r=19,m=0;", "\x1b\\"},
		{map[string]string{"TERM_PROGRAM": "iTerm.app", "TMUX": "/tmp/tmux-1/default,1,0"}, "\x1bPtmux;\x1b\x1b]1337;File=", "\x1b\\"},
	} {
		m := imageViewer(t, 100, 24, c.env)
		v := showFile(t, m, "board.png")
		view := v.View()
		lines := strings.Split(view, "\n")
		last := lines[len(lines)-1]
		at := strings.Index(last, "\x1b7\x1b[4;1H")
		if at < 0 || !strings.HasSuffix(last, "\x1b8") {
			t.Fatalf("%v: the image is not drawn at the frame's first row (4) after the frame: %q", c.env, last[max(0, len(last)-200):])
		}
		seq := last[at+len("\x1b7\x1b[4;1H") : len(last)-len("\x1b8")]
		if !strings.HasPrefix(seq, c.start) || !strings.HasSuffix(seq, c.end) {
			t.Fatalf("%v: sequence %q…%q, want %q…%q", c.env, seq[:min(80, len(seq))], seq[max(0, len(seq)-10):], c.start, c.end)
		}
		body := frameOf(t, v, ansi.Strip(v.frameView()), "HUG · list › 000002 › compiler/000002/board.png · 67 B", "screenshot of the board")
		for i, l := range body {
			if strings.TrimSpace(l) != "" {
				t.Fatalf("%v: frame row %d is not left for the image: %q", c.env, i, l)
			}
		}
		if strings.Contains(v.frameView(), "1337") || strings.Contains(v.frameView(), "\x1b_G") {
			t.Fatalf("%v: the image is in the frame text (a screen capture would keep it)", c.env)
		}
	}
}

// 2qndsd check 2: n and p step between images and text files in the same
// frame; the image is drawn only while an image shows, and in kitty it
// is deleted when the file shown is not one.
func TestStepImagesAndTextInOneFrame(t *testing.T) {
	m := imageViewer(t, 100, 24, map[string]string{"TERM": "xterm-kitty"})
	files := m.filesOf(mustSelected(t, m))
	v := showFile(t, m, files[0])
	for i := range files {
		name := files[i]
		screen := ansi.Strip(v.frameView())
		note := ""
		if name == "board.png" {
			note = "screenshot of the board"
		}
		frameOf(t, v, screen, fmt.Sprintf("HUG · list › 000002 › compiler/000002/%s · ", name), note)
		if !strings.Contains(strings.Split(screen, "\n")[0], fmt.Sprintf("%d of %d", i+1, len(files))) {
			t.Fatalf("%s: crumb lacks %d of %d", name, i+1, len(files))
		}
		view := v.View()
		image := strings.HasSuffix(name, ".png")
		if drawn := strings.Contains(view, "\x1b_Ga=T"); drawn != image {
			t.Fatalf("%s: image drawn %v, want %v", name, drawn, image)
		}
		if !image && !strings.Contains(strings.Split(view, "\n")[0], "\x1b_Ga=d,d=A\x1b\\") {
			t.Fatalf("%s: kitty's image is not deleted under a text file", name)
		}
		if i+1 < len(files) {
			v = press(v, "n")
		}
	}
	back := press(v, "p")
	if name, _, _ := back.glanceFile(mustSelected(t, back)); name != files[len(files)-2] || !back.glance {
		t.Fatalf("p went to %q (glance %v)", name, back.glance)
	}
}

// 2qndsd: the renderer rewrites only lines that changed, and the blank
// rows under an iTerm2 picture do not, so leaving an image (n to a text
// file, u back to Files) repaints the whole screen once; a frame that
// draws the same image again does not.
func TestLeavingAnImageRepaints(t *testing.T) {
	m := imageViewer(t, 100, 24, map[string]string{"TERM_PROGRAM": "iTerm.app"})
	v := showFile(t, m, "board.png")
	if v.imageWasDrawn == nil || v.imageWasDrawn != v.glanceImage {
		t.Fatal("the image shown is not recorded as drawn")
	}
	if again := v; again.eraseImage() != nil {
		t.Fatal("drawing the same image again repainted the screen")
	}
	for _, k := range []string{"n", "u"} {
		before := v
		next := press(v, k)
		if next.imageWasDrawn != nil {
			t.Fatalf("%s: the image is still recorded as drawn", k)
		}
		// The same step through eraseImage, as afterUpdate takes it.
		before.glance = next.glance
		before.glanceImage = next.glanceImage
		if cmd := before.eraseImage(); cmd == nil || fmt.Sprintf("%T", cmd()) != fmt.Sprintf("%T", tea.ClearScreen()) {
			t.Fatalf("%s: leaving the image does not repaint the screen", k)
		}
	}
}
