package tui

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // pictures are PNG, JPEG or GIF (task.IsImageName)
	_ "image/jpeg"
	"image/png"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/veridit/hug/huginn/internal/task"
)

// A picture in the file frame can be looked at properly (33v71x, Jørgen
// on 911j70: puppy's 160x45 comparison sheets cannot be read at terminal
// size): + and - zoom in steps from fit, h/j/k/l scroll the zoomed
// picture, 0 fits it again, and o (and Enter) hand it to the system
// viewer, Preview on macOS, never the editor.

// zoomSteps are the zoom levels as num/den of fit: fit, 150%, 200% …
var zoomSteps = [][2]int{{1, 1}, {3, 2}, {2, 1}, {3, 1}, {4, 1}, {6, 1}, {8, 1}}

// picView is what the frame shows of a picture: the crop x, y, w, h in
// the picture's pixels and the cells cols x rows it is drawn in.
type picView struct{ x, y, w, h, cols, rows int }

// picCrop is the crop of an imgW x imgH picture shown in a frame of cols
// x rows cells at zoom step zoom, its corner at x, y as near as the
// picture allows. A cell is taken as twice as tall as wide. Fit (zoom 0)
// shows the whole picture; each step shows less of it, larger. Integer
// arithmetic only, so the same frame gives the same crop everywhere.
func picCrop(imgW, imgH, cols, rows, zoom, x, y int) picView {
	if imgW <= 0 || imgH <= 0 || cols <= 0 || rows <= 0 {
		return picView{}
	}
	zoom = max(0, min(zoom, len(zoomSteps)-1))
	num, den := zoomSteps[zoom][0], zoomSteps[zoom][1]
	// The scale, cell widths per pixel, is sN/sD: fit by width or height.
	sN, sD := 2*rows*num, imgH*den
	if cols*imgH <= 2*rows*imgW {
		sN, sD = cols*num, imgW*den
	}
	v := picView{w: min(imgW, cols*sD/sN), h: min(imgH, 2*rows*sD/sN)}
	v.w, v.h = max(1, v.w), max(1, v.h)
	v.cols = max(1, min(cols, (2*v.w*sN+sD)/(2*sD)))
	v.rows = max(1, min(rows, (v.h*sN+sD)/(2*sD)))
	v.x = max(0, min(x, imgW-v.w))
	v.y = max(0, min(y, imgH-v.h))
	return v
}

// glancePicture says whether the file shown is a picture.
func (m model) glancePicture() bool {
	if !m.detail || !m.glance {
		return false
	}
	t, ok := m.selected()
	if !ok {
		return false
	}
	name, _, _ := m.glanceFile(t)
	return task.IsImageName(name)
}

// markedPicture says whether the file marked > in a task's Files is a
// picture: Enter opens it in the viewer, not the editor (check 5).
func (m model) markedPicture() bool {
	t, ok := m.selected()
	if !ok {
		return false
	}
	files := m.filesOf(t)
	return len(files) > 0 && task.IsImageName(files[min(m.file, len(files)-1)])
}

// zoomable says whether the shown picture can be zoomed: the terminal
// draws it and hug read its size.
func (m model) zoomable() bool {
	return m.glanceImage != nil && m.glanceImage.w > 0 && m.glanceImage.h > 0
}

// picView is what the frame shows of the picture now.
func (m model) picView() picView {
	if !m.zoomable() {
		return picView{}
	}
	return picCrop(m.glanceImage.w, m.glanceImage.h, m.width, m.viewport.Height, m.picZoom, m.picX, m.picY)
}

// zoomTo goes to zoom step z, keeping the centre of what shows.
func (m *model) zoomTo(z int) {
	img, was := m.glanceImage, m.picView()
	cx, cy := was.x+was.w/2, was.y+was.h/2
	m.picZoom = max(0, min(z, len(zoomSteps)-1))
	// The zoom's message takes its row first (yp3c09), so the crop is
	// centred in the rows it is drawn in.
	m.message = m.zoomSays()
	m.fitGlance()
	next := picCrop(img.w, img.h, m.width, m.viewport.Height, m.picZoom, 0, 0)
	v := picCrop(img.w, img.h, m.width, m.viewport.Height, m.picZoom, cx-next.w/2, cy-next.h/2)
	m.picX, m.picY = v.x, v.y
}

// zoomSays names the zoom, and the keys from it.
func (m model) zoomSays() string {
	if m.picZoom == 0 {
		return "the picture fits the frame: + zooms in"
	}
	return fmt.Sprintf("zoom %s: h/j/k/l scroll, 0 fits", zoomPercent(m.picZoom))
}

func zoomPercent(z int) string {
	return fmt.Sprintf("%d%%", 100*zoomSteps[z][0]/zoomSteps[z][1])
}

// pictureKey answers a key on a shown picture; false leaves it to the
// file frame (n, p, u, E …).
func (m *model) pictureKey(k string) bool {
	switch k {
	case "o":
		m.openPicture()
		return true
	case "+", "=", "-", "0", "h", "l", "left", "right":
	case "j", "k", "up", "down":
		if m.picZoom == 0 {
			return false
		}
	default:
		return false
	}
	if !m.zoomable() {
		m.message = "this terminal draws no pictures (iTerm2 and kitty do): o opens it in " + viewerName()
		return true
	}
	switch k {
	case "+", "=":
		if m.picZoom == len(zoomSteps)-1 {
			m.message = "largest zoom, " + zoomPercent(m.picZoom) + ": - zooms out, 0 fits"
			break
		}
		m.zoomTo(m.picZoom + 1)
	case "-":
		if m.picZoom == 0 {
			m.message = m.zoomSays()
			break
		}
		m.zoomTo(m.picZoom - 1)
	case "0":
		if m.picZoom == 0 {
			m.message = "the picture fits the frame already: + zooms in"
			break
		}
		m.picZoom, m.picX, m.picY = 0, 0, 0
		m.message = m.zoomSays()
	default:
		v := m.picView()
		step := map[string][2]int{"h": {-1, 0}, "left": {-1, 0}, "l": {1, 0}, "right": {1, 0}, "j": {0, 1}, "down": {0, 1}, "k": {0, -1}, "up": {0, -1}}[k]
		next := picCrop(m.glanceImage.w, m.glanceImage.h, m.width, m.viewport.Height, m.picZoom, v.x+step[0]*max(1, v.w/8), v.y+step[1]*max(1, v.h/8))
		if next == v {
			m.message = "the edge of the picture"
			if m.picZoom == 0 {
				m.message = "the whole picture shows: + zooms in"
			}
		}
		m.picX, m.picY = next.x, next.y
	}
	return true
}

// picDrawn is the picture as the terminal is sent it: the file as it is
// at fit, else the crop, encoded once per crop.
func (m model) picDrawn() (data []byte, cols, rows int) {
	img := m.glanceImage
	if m.picZoom == 0 || !m.zoomable() {
		return img.data, m.width, m.viewport.Height
	}
	v := m.picView()
	if img.cropAt == v && img.crop != nil {
		return img.crop, v.cols, v.rows
	}
	if img.decoded == nil {
		d, _, err := image.Decode(bytes.NewReader(img.data))
		if err != nil {
			return img.data, m.width, m.viewport.Height
		}
		img.decoded = d
	}
	r := image.Rect(v.x, v.y, v.x+v.w, v.y+v.h).Add(img.decoded.Bounds().Min)
	out := image.NewRGBA(image.Rect(0, 0, v.w, v.h))
	draw.Draw(out, out.Bounds(), img.decoded, r.Min, draw.Src)
	var b bytes.Buffer
	if err := png.Encode(&b, out); err != nil {
		return img.data, m.width, m.viewport.Height
	}
	img.crop, img.cropAt = b.Bytes(), v
	return img.crop, v.cols, v.rows
}

// pictureSize reads a picture's size in pixels from its header; 0, 0
// when it cannot (then it is not zoomed).
func pictureSize(b []byte) (w, h int) {
	c, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return 0, 0
	}
	return c.Width, c.Height
}

// openerOS is the OS whose viewer opens a picture; tests pin it.
var openerOS = runtime.GOOS

// startOpener starts argv and does not wait for it: the viewer is the
// OS's, hug stays where it is. Tests replace it, so none opens Preview.
var startOpener = func(argv []string) error {
	c := exec.Command(argv[0], argv[1:]...)
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }()
	return nil
}

// openerArgv is the command that shows path in the system viewer.
func openerArgv(path string) []string {
	switch openerOS {
	case "darwin":
		return []string{"open", path}
	case "windows":
		return []string{"explorer", path}
	}
	return []string{"xdg-open", path}
}

// viewerName is what the system viewer is called here.
func viewerName() string {
	if openerOS == "darwin" {
		return "Preview"
	}
	return "the viewer"
}

// openPicture hands the shown or marked picture to the system viewer.
func (m *model) openPicture() {
	t, ok := m.selected()
	if !ok {
		return
	}
	files := m.filesOf(t)
	if len(files) == 0 {
		m.message = "this task has no files: hug task attach " + t.ID + " FILE"
		return
	}
	m.openInViewer(t, files[min(m.file, len(files)-1)])
}

// openInViewer starts the system viewer on t's file name (33v71x checks
// 1 and 5): a picture is looked at, never edited.
func (m *model) openInViewer(t task.Task, name string) {
	path, err := m.insideFolder(t, name)
	if err != nil {
		m.message = err.Error()
		return
	}
	argv := openerArgv(path)
	if err := startOpener(argv); err != nil {
		m.message = fmt.Sprintf("could not open %s: %v\nfix: %s %s", oneLine(name), err, argv[0], strconv.Quote(path))
		return
	}
	m.message = "opened " + oneLine(name) + " in " + viewerName()
}
