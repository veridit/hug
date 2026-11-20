package tui

import (
	"encoding/base64"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// Pictures in a task (5z4zt4, Jørgen 14:51: "I'm supposed to look at
// something but I don't know what"). A question about a look carries the
// picture. Text pictures first: an .ans file is a screen capture with its
// colours, text that diffs in git, works over ssh and an agent can read.
// S in hug tui takes one of the screen as you see it. PNG, JPEG and GIF stay
// for things outside the TUI: drawn where the terminal can (iTerm2, kitty),
// one line elsewhere. Nothing is read until it is shown.

// ansName says whether name is an ANSI capture, shown with its colours.
func ansName(name string) bool { return strings.EqualFold(filepath.Ext(name), ".ans") }

// colourOnly keeps the colour (SGR, ESC[...m) sequences of an ANSI capture
// and escapes every other control: a capture can colour text but never move
// the cursor, clear the screen or set a title. Each line is cut to width,
// colours closed at its end, so a line can not paint the next.
func colourOnly(b []byte, width int, colour bool) string {
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	var out []string
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		var keep strings.Builder
		for i := 0; i < len(line); {
			c := line[i]
			if c == 0x1b && i+1 < len(line) && line[i+1] == '[' {
				j := i + 2
				for j < len(line) && (line[j] >= '0' && line[j] <= '9' || line[j] == ';' || line[j] == ':') {
					j++
				}
				if j < len(line) && line[j] == 'm' {
					if colour {
						keep.WriteString(line[i : j+1])
					}
					i = j + 1
					continue
				}
			}
			n := 1
			if c >= 0x80 {
				_, n = utf8.DecodeRuneInString(line[i:])
			}
			keep.WriteString(escapeControls(line[i : i+n]))
			i += n
		}
		// A frame's rows are padded to its width; the padding is not
		// picture, and would cut every row of a full-width capture.
		l := ansi.Truncate(trimPadding(strings.ReplaceAll(keep.String(), "\t", "    ")), max(1, width), "…")
		if colour {
			l += "\x1b[0m"
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

// trimPadding drops the spaces that end a line, also behind colour codes
// that end it ("text\x1b[0m    " and "text    \x1b[0m").
func trimPadding(s string) string {
	for {
		t := strings.TrimRight(s, " ")
		i := strings.LastIndex(t, "\x1b[")
		if i < 0 || !strings.HasSuffix(t, "m") || strings.ContainsAny(t[i+2:len(t)-1], "\x1b ") || strings.TrimRight(t[:i], " ") == t[:i] {
			return t
		}
		s = strings.TrimRight(t[:i], " ") + t[i:]
	}
}

// imageTerminal is the inline image protocol this terminal speaks:
// "iterm" (OSC 1337 File=, iTerm2 and WezTerm), "kitty" (the kitty graphics
// protocol), or "" (none: one line instead).
func imageTerminal(getenv func(string) string) string {
	switch {
	case getenv("TERM_PROGRAM") == "iTerm.app", getenv("TERM_PROGRAM") == "WezTerm", getenv("LC_TERMINAL") == "iTerm2":
		return "iterm"
	case getenv("KITTY_WINDOW_ID") != "", getenv("TERM") == "xterm-kitty", getenv("TERM_PROGRAM") == "ghostty":
		return "kitty"
	}
	return ""
}

// imageSequenceSized is what draws b in the terminal: one OSC 1337 for
// iTerm2, or kitty's graphics protocol in chunks of 4096 base64 bytes,
// each wrapped as a passthrough inside tmux. It fits b into cols x rows
// cells (0: its own size), keeping its aspect, the cursor left where it
// was drawn (kitty C=1), so the frame around it stays put (2qndsd).
func imageSequenceSized(proto, name string, b []byte, tmux bool, cols, rows int) string {
	data := base64.StdEncoding.EncodeToString(b)
	var seq string
	switch proto {
	case "iterm":
		size := "preserveAspectRatio=1;width=auto"
		if cols > 0 && rows > 0 {
			size = fmt.Sprintf("width=%d;height=%d;preserveAspectRatio=1", cols, rows)
		}
		seq = fmt.Sprintf("\x1b]1337;File=name=%s;size=%d;inline=1;%s:%s\a", base64.StdEncoding.EncodeToString([]byte(name)), len(b), size, data)
	case "kitty":
		size := ""
		if rows > 0 {
			// r= fits the height; kitty keeps the aspect, and C=1 does not
			// move the cursor.
			size = fmt.Sprintf(",C=1,r=%d", rows)
		}
		var s strings.Builder
		for i := 0; i < len(data); i += 4096 {
			end := min(i+4096, len(data))
			more := 0
			if end < len(data) {
				more = 1
			}
			if i == 0 {
				fmt.Fprintf(&s, "\x1b_Ga=T,f=100%s,m=%d;%s\x1b\\", size, more, data[i:end])
			} else {
				fmt.Fprintf(&s, "\x1b_Gm=%d;%s\x1b\\", more, data[i:end])
			}
		}
		seq = s.String()
	}
	return wrapTmux(seq, tmux)
}

// wrapTmux passes seq through tmux to the terminal outside it.
func wrapTmux(seq string, tmux bool) string {
	if tmux && seq != "" {
		seq = "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	return seq
}

// glanceImage is the image the file frame shows (2qndsd, Jørgen: a
// screenshot is seen inside the ticket): its bytes, read when shown, and
// the protocol that draws it. nil when the file shown is no image, or the
// terminal draws none.
type glanceImage struct {
	name  string
	proto string
	data  []byte
	// w, h are its size in pixels, from its header (0: unknown, not
	// zoomed); decoded, crop and cropAt keep the last zoomed crop, so a
	// frame drawn again sends the same bytes (33v71x).
	w, h    int
	decoded image.Image
	crop    []byte
	cropAt  picView
}

// glanceReadImage is an image's text in the frame: empty rows the terminal
// draws the image over, or, where it draws none (or the file cannot be
// read), one line that says so. Its bytes are read only here, when shown.
func (m *model) glanceReadImage(path string, size int64) string {
	proto := imageTerminal(m.getenv)
	if proto == "" {
		return fmt.Sprintf("image, %s: this terminal draws no images (iTerm2 and kitty do): Enter opens it", humanBytes(size))
	}
	if size > glanceLimit*16 {
		return fmt.Sprintf("image, %s: too large to draw here, Enter opens it", humanBytes(size))
	}
	b, err := readFile(path)
	if err != nil {
		return err.Error()
	}
	w, h := pictureSize(b)
	m.glanceImage = &glanceImage{name: filepath.Base(path), proto: proto, data: b, w: w, h: h}
	return ""
}

// imageDrawn says whether the frame draws an image now: one shows, and
// nothing covers the frame.
func (m model) imageDrawn() bool {
	return m.detail && m.glance && m.glanceImage != nil && !m.help && m.prompt == "" && m.confirm == ""
}

// eraseImage repaints the whole screen once the image drawn last is gone
// or replaced: the renderer rewrites only lines that changed, and the
// frame's blank rows under an iTerm2 picture do not, so they would keep
// pieces of it. A new zoom or scroll of the same picture is a new one
// (33v71x).
func (m *model) eraseImage() tea.Cmd {
	var now *glanceImage
	view := m.picView()
	if m.imageDrawn() {
		now = m.glanceImage
	}
	was, wasView := m.imageWasDrawn, m.viewWasDrawn
	m.imageWasDrawn, m.viewWasDrawn = now, view
	if was != nil && (was != now || wasView != view) {
		return tea.ClearScreen
	}
	return nil
}

// imageOverlay is what draws the shown image over the frame's rows, after
// the frame (View): the cursor saved, moved to the first row under the
// top rule, the image scaled into the frame's rows and width, the cursor
// put back. The frame itself keeps those rows blank, so a screen capture
// (S) holds no image bytes. In kitty a frame without an image deletes the
// last one; an iTerm2 picture sits in its cells until eraseImage repaints.
func (m model) imageOverlay() (before, after string) {
	img := m.glanceImage
	if !m.imageDrawn() {
		if imageTerminal(m.getenv) == "kitty" {
			return wrapTmux("\x1b_Ga=d,d=A\x1b\\", m.getenv("TMUX") != ""), ""
		}
		return "", ""
	}
	row := 2 // the crumb's row is 1: the top rule, then the image
	if m.glanceNote() != "" {
		row++
	}
	before = ""
	if img.proto == "kitty" {
		before = wrapTmux("\x1b_Ga=d,d=A\x1b\\", m.getenv("TMUX") != "")
	}
	data, cols, rows := m.picDrawn()
	seq := imageSequenceSized(img.proto, img.name, data, m.getenv("TMUX") != "", cols, rows)
	return before, fmt.Sprintf("\x1b7\x1b[%d;1H%s\x1b8", row+1, seq)
}

// getenv reads the environment through Options, so tests choose a terminal.
func (m model) getenv(k string) string {
	if m.opts.Getenv != nil {
		return m.opts.Getenv(k)
	}
	return os.Getenv(k)
}

// questionPicture is the task file a question asks you to look at: the one
// --show named (waiting_show, design_show, review_show), else the first task file the
// question names. "" when it names none.
func (m model) questionPicture(t task.Task, stage, question string) string {
	return m.questionPictureOf(t, t.Fields[map[string]string{"waiting": "waiting_show", "asked": "asked_show", "design": "design_show", "review": "review_show"}[stage]], question)
}

// questionPictureOf is the task file show names, if the task has it, else
// one question names, else "".
func (m model) questionPictureOf(t task.Task, show, question string) string {
	files := m.filesOf(t)
	has := func(name string) bool {
		for _, f := range files {
			if f == name {
				return true
			}
		}
		return false
	}
	if v := strings.TrimSpace(show); v != "" && has(v) {
		return v
	}
	for _, w := range strings.Fields(question) {
		w = strings.TrimPrefix(strings.TrimRight(w, ".,;:)"), "file:")
		w = strings.TrimPrefix(w, "(")
		for _, f := range files {
			if w == f || strings.HasSuffix(w, "/"+f) {
				return f
			}
		}
	}
	return ""
}

// The screen capture (S, Jørgen 14:51): the frame as you see it goes to the
// clipboard as plain text, so it pastes cleanly into a chat with an agent,
// and S asks where to attach it as a coloured .ans file: the task in view,
// another by id, one picked from the rows, or a new one (hjbfzd,
// capture.go).

// captureKey takes the screen.
const captureKey = "S"

// clipboard copies text: OSC 52 through the terminal (iTerm2, kitty, over
// ssh; tmux passes it on), and pbcopy too on a local macOS. Tests replace it.
var clipboard = func(text string, tmux bool) error {
	seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\a"
	if tmux {
		seq = "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	if tty, e := os.OpenFile("/dev/tty", os.O_WRONLY, 0); e == nil {
		_, _ = tty.WriteString(seq)
		tty.Close()
	}
	if runtime.GOOS == "darwin" && os.Getenv("SSH_CONNECTION") == "" {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return nil
}

// captureNow is the clock the capture's name is taken from; tests fix it.
var captureNow = time.Now

// capture takes the frame as drawn now, before the bar asks anything:
// plain text to the clipboard, the coloured frame kept for attaching, and
// the prompt asks where it goes, offering the task in view.
func (m *model) capture() tea.Cmd {
	frame := m.frameView() // text only: a shown image's codes stay out (2qndsd)
	plain := ansi.Strip(frame)
	var lines []string
	for _, l := range strings.Split(plain, "\n") {
		lines = append(lines, strings.TrimRight(l, " "))
	}
	plain = strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	copied := "screen copied"
	if err := clipboard(plain, m.getenv("TMUX") != ""); err != nil {
		copied = "screen not copied (" + err.Error() + ")"
	}
	n := strings.Count(plain, "\n")
	var rows []string
	for _, l := range strings.Split(strings.TrimRight(frame, "\n"), "\n") {
		rows = append(rows, trimPadding(l))
	}
	m.captured = strings.Join(rows, "\n") + "\n"
	m.capturedNote = fmt.Sprintf("%s (%d lines)", copied, n)
	m.captureStamp = captureNow().Format("1504")
	offer := ""
	if t, ok := m.selected(); ok {
		offer = t.ID
	}
	return m.openCapture(offer)
}

// hasFile says whether t already has a file called name.
func (m model) hasFile(t task.Task, name string) bool {
	for _, f := range m.filesOf(t) {
		if f == name {
			return true
		}
	}
	return false
}

// attachCapture attaches the captured frame to t as name (hjbfzd: the
// task typed, offered or picked).
func (m *model) attachCapture(t task.Task, name string) {
	body := []byte(m.captured)
	note := m.capturedNote
	m.captured, m.capturedNote, m.captureTyped = "", "", false
	// The attach and its commit hold the board turn (95ahtk).
	defer m.hold()()
	r, _, err := m.store.Attach(t.ID, name, body, "screen capture from hug tui", false)
	if err != nil {
		m.message = note + " · not attached: " + err.Error()
		return
	}
	if m.opts.Commit != nil {
		if e := m.opts.Commit("hug task attach " + t.ID); e != nil {
			m.message = note + " · " + e.Error()
			return
		}
	}
	m.refresh(r.Task.Path)
	m.message = note + " · attached as " + name + " to " + t.ID
}
