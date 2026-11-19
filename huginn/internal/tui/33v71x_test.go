package tui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// sheetPNG is a w x h picture, every pixel its own colour, like puppy's
// comparison sheets only smaller.
func sheetPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), uint8(x ^ y), 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// sheetViewer is task 000002 with sheet.png (400 x 200) attached and
// shown in the file frame, in terminal env, w x h.
func sheetViewer(t *testing.T, w, h int, env map[string]string) model {
	t.Helper()
	m, _ := pictureModel(t, w, h, env, "")
	if _, _, err := m.store.Attach("000002", "sheet.png", sheetPNG(t, 400, 200), "comparison sheet", false); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	return showFile(t, toFiles(t, openTask(t, m, "000002")), "sheet.png")
}

var iterm = map[string]string{"TERM_PROGRAM": "iTerm.app"}

// openerCalls replaces the OS opener for one test: every argv it was
// asked to start, and the OS it opens for pinned.
func openerCalls(t *testing.T, goos string) *[][]string {
	t.Helper()
	var calls [][]string
	oldStart, oldOS := startOpener, openerOS
	startOpener = func(argv []string) error { calls = append(calls, argv); return nil }
	openerOS = goos
	t.Cleanup(func() { startOpener, openerOS = oldStart, oldOS })
	return &calls
}

// 33v71x check 1: o on a picture hands it to the OS viewer (open on
// macOS, xdg-open on Linux, explorer on Windows) and hug stays on the
// same screen; o on a text file opens nothing.
func TestPictureOpensInTheViewer(t *testing.T) {
	for goos, cmd := range map[string]string{"darwin": "open", "linux": "xdg-open", "windows": "explorer"} {
		calls := openerCalls(t, goos)
		m := sheetViewer(t, 100, 24, iterm)
		path, err := m.insideFolder(mustSelected(t, m), "sheet.png")
		if err != nil {
			t.Fatal(err)
		}
		after, c := key(m, "o")
		if len(*calls) != 1 || strings.Join((*calls)[0], "\x00") != cmd+"\x00"+path {
			t.Fatalf("%s: o started %q, want [%s %s]", goos, *calls, cmd, path)
		}
		if c != nil {
			t.Fatalf("%s: o returned a command; the viewer is started, not waited for", goos)
		}
		if !after.detail || !after.glance || after.glanceHead != m.glanceHead || after.editing {
			t.Fatalf("%s: o left the picture (detail %v glance %v editing %v)", goos, after.detail, after.glance, after.editing)
		}
		want := "opened sheet.png in the viewer"
		if goos == "darwin" {
			want = "opened sheet.png in Preview"
		}
		if !strings.Contains(after.message, want) {
			t.Fatalf("%s: o says %q, want %q", goos, after.message, want)
		}
	}
	calls := openerCalls(t, "darwin")
	g, _ := glanceModel(t, 100, 24)
	text := press(toFiles(t, openTask(t, g, "000001")), " ")
	if !text.glance || task0IsPicture(text) {
		t.Fatal("no text file shown")
	}
	if press(text, "o"); len(*calls) != 0 {
		t.Fatalf("o on a text file opened %q", *calls)
	}
}

func task0IsPicture(m model) bool { return m.glancePicture() }

// 33v71x check 5 (Jørgen on 911j70, request 2): Enter on a picture opens
// the system viewer, never the editor, both in the task's Files and in
// the picture's frame; Enter on a text file still opens the editor. One
// opener with o: the same argv.
func TestEnterOpensPictureInTheViewer(t *testing.T) {
	calls := openerCalls(t, "darwin")
	shown := sheetViewer(t, 100, 24, iterm)
	path, err := shown.insideFolder(mustSelected(t, shown), "sheet.png")
	if err != nil {
		t.Fatal(err)
	}
	files := press(shown, "u")
	if files.glance || files.detailPane != focusFiles {
		t.Fatal("u did not go back to Files")
	}
	for name, m := range map[string]model{"Files": files, "frame": shown} {
		*calls = nil
		after, c := key(m, "enter")
		if after.editing || after.lastExec != m.lastExec {
			t.Fatalf("%s: Enter on a picture started the editor: %v", name, after.lastExec)
		}
		if len(*calls) != 1 || strings.Join((*calls)[0], " ") != "open "+path {
			t.Fatalf("%s: Enter started %q, want [open %s]", name, *calls, path)
		}
		if c != nil || !strings.Contains(after.message, "opened sheet.png in Preview") {
			t.Fatalf("%s: Enter returned %v, says %q", name, c, after.message)
		}
		*calls = nil
		if key(m, "o"); len(*calls) != 1 || strings.Join((*calls)[0], " ") != "open "+path {
			t.Fatalf("%s: o started %q, want the same opener as Enter", name, *calls)
		}
	}
	*calls = nil
	g, _ := glanceModel(t, 100, 24)
	text := toFiles(t, openTask(t, g, "000001"))
	if _, c := key(text, "enter"); c == nil || len(*calls) != 0 {
		t.Fatalf("Enter on a text file: command %v, opener %q; want the editor", c, *calls)
	}
}

// 33v71x check 2: the crop and scale. A cell is twice as tall as wide;
// fit shows the whole picture; each zoom step shows less of it, larger,
// centred where the view was; the crop never leaves the picture.
func TestPictureCrop(t *testing.T) {
	for _, c := range []struct {
		zoom, x, y int
		want       picView
	}{
		{0, 0, 0, picView{x: 0, y: 0, w: 400, h: 200, cols: 80, rows: 20}},
		{1, 33, 33, picView{x: 33, y: 33, w: 333, h: 133, cols: 100, rows: 20}},
		{2, 75, 50, picView{x: 75, y: 50, w: 250, h: 100, cols: 100, rows: 20}},
		{2, -9, 999, picView{x: 0, y: 100, w: 250, h: 100, cols: 100, rows: 20}},
		{len(zoomSteps) - 1, 400, 0, picView{x: 338, y: 0, w: 62, h: 25, cols: 99, rows: 20}}, // 62 px x 1.6 cells/px
	} {
		got := picCrop(400, 200, 100, 20, c.zoom, c.x, c.y)
		if got != c.want {
			t.Fatalf("zoom %d at %d,%d: %+v, want %+v", c.zoom, c.x, c.y, got, c.want)
		}
	}
}

// 33v71x check 2: + and - zoom in steps with fit as the floor, centred;
// j/k and h/l scroll a zoomed picture by an eighth of what shows, and
// stop at its edges; 0 fits it again. The crumb says the zoom.
func TestPictureZoomAndScrollKeys(t *testing.T) {
	m := sheetViewer(t, 100, 24, iterm)
	rows := m.viewport.Height
	if v := m.picView(); v.w != 400 || v.h != 200 {
		t.Fatalf("a picture opens fitted: %+v", v)
	}
	if got := press(m, "-"); got.picZoom != 0 || !strings.Contains(got.message, "fits the frame") {
		t.Fatalf("- at fit: zoom %d, %q", got.picZoom, got.message)
	}
	z := press(m, "+", "+")
	// The crop is centred in the rows it is drawn in: the zoom's message
	// takes one of the frame's rows (yp3c09).
	rows = z.viewport.Height
	want := picCrop(400, 200, 100, rows, 2, 200-picCrop(400, 200, 100, rows, 2, 0, 0).w/2, 0)
	if v := z.picView(); z.picZoom != 2 || v.w != want.w || v.x+v.w/2 != 200 || v.y+v.h/2 != 100 {
		t.Fatalf("+ + : zoom %d view %+v, want centred %dx%d", z.picZoom, v, want.w, want.h)
	}
	if crumb := ansi.Strip(strings.Split(z.View(), "\n")[0]); !strings.Contains(crumb, "zoom 200%") {
		t.Fatalf("the crumb does not say the zoom: %q", crumb)
	}
	v := z.picView()
	if d := press(z, "j").picView(); d.y != v.y+v.h/8 || d.x != v.x {
		t.Fatalf("j: %+v from %+v", d, v)
	}
	if d := press(z, "k").picView(); d.y != v.y-v.h/8 {
		t.Fatalf("k: %+v from %+v", d, v)
	}
	if d := press(z, "l").picView(); d.x != v.x+v.w/8 || d.y != v.y {
		t.Fatalf("l: %+v from %+v", d, v)
	}
	if d := press(z, "h").picView(); d.x != v.x-v.w/8 {
		t.Fatalf("h: %+v from %+v", d, v)
	}
	edge := z
	for i := 0; i < 20; i++ {
		edge = press(edge, "j", "l")
	}
	if d := edge.picView(); d.x+d.w != 400 || d.y+d.h != 200 {
		t.Fatalf("scrolling stops at the picture's edge: %+v", d)
	}
	max := m
	for i := 0; i < len(zoomSteps)+2; i++ {
		max = press(max, "+")
	}
	if max.picZoom != len(zoomSteps)-1 || !strings.Contains(max.message, "largest") {
		t.Fatalf("+ past the last step: zoom %d %q", max.picZoom, max.message)
	}
	if back := press(z, "-"); back.picZoom != 1 {
		t.Fatalf("- from 200%%: zoom %d", back.picZoom)
	}
	// 0 fits the whole picture again, in the rows left beside its message
	// (yp3c09: the message takes a row).
	if fit := press(edge, "0"); fit.picZoom != 0 || fit.picView() != picCrop(400, 200, 100, fit.viewport.Height, 0, 0, 0) || fit.picView().w != 400 || fit.picView().h != 200 {
		t.Fatalf("0: zoom %d view %+v", fit.picZoom, fit.picView())
	}
	if eq := press(m, "="); eq.picZoom != 1 {
		t.Fatal("= (+ without shift) does not zoom in")
	}
	// The next file starts fitted.
	if next := press(z, "p"); next.picZoom != 0 || next.picX != 0 || next.picY != 0 {
		t.Fatalf("p kept the zoom: %d %d,%d", next.picZoom, next.picX, next.picY)
	}
}

// 33v71x check 2: zoomed, the terminal is sent the crop, sized to the
// cells it fills, and a changed crop repaints the screen once (an iTerm2
// picture sits in its cells until then).
func TestZoomedPictureDrawsTheCrop(t *testing.T) {
	z := press(sheetViewer(t, 100, 24, iterm), "+", "+", "j")
	v := z.picView()
	_, after := z.imageOverlay()
	head := regexp.MustCompile(`1337;File=name=[^;]*;size=(\d+);inline=1;width=(\d+);height=(\d+);preserveAspectRatio=1:([^\a]*)\a`).FindStringSubmatch(after)
	if head == nil {
		t.Fatalf("no OSC 1337 picture: %q", after[:min(200, len(after))])
	}
	if head[2] != fmt.Sprint(v.cols) || head[3] != fmt.Sprint(v.rows) {
		t.Fatalf("drawn %sx%s cells, want %dx%d", head[2], head[3], v.cols, v.rows)
	}
	b, err := base64.StdEncoding.DecodeString(head[4])
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if r := img.Bounds(); r.Dx() != v.w || r.Dy() != v.h {
		t.Fatalf("sent a %dx%d picture, want the %dx%d crop", r.Dx(), r.Dy(), v.w, v.h)
	}
	if got, want := img.At(0, 0), color.RGBAModel.Convert(color.RGBA{uint8(v.x), uint8(v.y), uint8(v.x ^ v.y), 255}); color.RGBAModel.Convert(got) != want {
		t.Fatalf("the crop's first pixel is %v, want the sheet's at %d,%d: %v", got, v.x, v.y, want)
	}
	before := press(sheetViewer(t, 100, 24, iterm), "+", "+")
	if _, cmd := key(before, "j"); cmd == nil || fmt.Sprintf("%T", cmd()) != fmt.Sprintf("%T", tea.ClearScreen()) {
		t.Fatal("a scrolled crop does not repaint the screen")
	}
	if _, cmd := key(before, "o"); cmd != nil {
		t.Fatal("the same crop drawn again repaints the screen")
	}
}

// 33v71x check 3: the footer and ? help list + - 0 o for pictures only:
// zoom while the terminal draws the picture, 0 and the scroll keys once
// zoomed, o always for a picture; none of them for a text file. Golden at
// 80, 100 and 140.
func TestPictureKeysListed(t *testing.T) {
	openerCalls(t, "darwin")
	var golden []string
	for _, w := range []int{80, 100, 140} {
		m := sheetViewer(t, w, 24, iterm)
		z := press(m, "+", "+")
		g, _ := glanceModel(t, w, 24)
		text := press(toFiles(t, openTask(t, g, "000001")), " ")
		bare := sheetViewer(t, w, 24, nil) // no image protocol
		for _, c := range []struct {
			name      string
			m         model
			has, lack []string
		}{
			{"picture, fitted", m, []string{"+/- zoom", "open in Preview"}, []string{"0 fit", "scroll"}},
			{"picture, zoomed", z, []string{"+/- zoom", "0 fit"}, nil},
			{"picture, no image protocol", bare, []string{"open in Preview"}, []string{"zoom", "0 fit"}},
			{"text file", text, nil, []string{"zoom", "0 fit", "open in Preview"}},
		} {
			footer := strings.TrimRight(ansi.Strip(c.m.keyBar()), " ")
			plain := plainMarks(footer)
			for _, s := range c.has {
				if w >= 100 && !strings.Contains(plain, s) {
					t.Fatalf("at %d %s: footer lacks %q: %q", w, c.name, s, plain)
				}
			}
			for _, s := range c.lack {
				if strings.Contains(plain, s) {
					t.Fatalf("at %d %s: footer lists %q: %q", w, c.name, s, plain)
				}
			}
			help := ansi.Strip(c.m.sectionText(c.m.contextHelp(), w))
			for _, s := range []string{"zoom in / out", "fit the picture", "open it in Preview"} {
				want := c.name != "text file" && (s == "open it in Preview" || c.name != "picture, no image protocol")
				if strings.Contains(help, s) != want {
					t.Fatalf("at %d %s: help has %q: %v, want %v\n%s", w, c.name, s, !want, want, help)
				}
			}
			golden = append(golden, fmt.Sprintf("== %s at %d", c.name, w), footer, help)
		}
		screen := ansi.Strip(z.frameView())
		golden = append(golden, fmt.Sprintf("== zoomed screen at %dx24", w), screen)
	}
	goldenFile(t, "testdata/detail/picture-zoom.txt", strings.Join(golden, "\n")+"\n")
}

// 33v71x check 3: every key the picture's help lists does something
// there.
func TestPictureHelpKeysAreBound(t *testing.T) {
	openerCalls(t, "darwin")
	m := sheetViewer(t, 100, 24, iterm)
	z := press(m, "+", "+")
	for _, ctx := range []model{m, z} {
		for _, s := range ctx.helpSections()[:1] {
			for _, r := range s.rows {
				for _, k := range r.probe {
					if _, ok := pressed(ctx, k); !ok {
						t.Errorf("zoom %d: %q (%s) does nothing", ctx.picZoom, k, r.key)
					}
				}
			}
		}
	}
	if !strings.HasPrefix(ansi.Strip(press(m, "?").helpText()), "IN THIS PICTURE") {
		t.Fatal("a picture's help is not headed IN THIS PICTURE")
	}
}
