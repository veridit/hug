package tui

import (
	"encoding/base64"
	"fmt"
	"io"
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
	"golang.org/x/term"
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
		l := ansi.Truncate(strings.ReplaceAll(keep.String(), "\t", "    "), max(1, width), "…")
		if colour {
			l += "\x1b[0m"
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
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

// imageSequence is what draws b in the terminal: one OSC 1337 for iTerm2,
// or kitty's graphics protocol in chunks of 4096 base64 bytes. Inside tmux
// each is wrapped as a passthrough.
func imageSequence(proto, name string, b []byte, tmux bool) string {
	data := base64.StdEncoding.EncodeToString(b)
	var seq string
	switch proto {
	case "iterm":
		seq = fmt.Sprintf("\x1b]1337;File=name=%s;size=%d;inline=1;preserveAspectRatio=1;width=auto:%s\a", base64.StdEncoding.EncodeToString([]byte(name)), len(b), data)
	case "kitty":
		var s strings.Builder
		for i := 0; i < len(data); i += 4096 {
			end := min(i+4096, len(data))
			more := 0
			if end < len(data) {
				more = 1
			}
			if i == 0 {
				fmt.Fprintf(&s, "\x1b_Ga=T,f=100,m=%d;%s\x1b\\", more, data[i:end])
			} else {
				fmt.Fprintf(&s, "\x1b_Gm=%d;%s\x1b\\", more, data[i:end])
			}
		}
		seq = s.String()
	}
	if tmux && seq != "" {
		seq = "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	return seq
}

// imageView draws an image full screen outside the frame and waits for a
// key; the TUI steps aside for it as it does for the editor.
type imageView struct {
	seq, title string
	in         io.Reader
	out        io.Writer
}

func (v *imageView) Run() error {
	out := v.out
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprint(out, "\x1b[2J\x1b[H"+v.title+"\r\n\r\n"+v.seq+"\r\n\r\nany key goes back")
	in := v.in
	if in == nil {
		tty, e := os.Open("/dev/tty")
		if e != nil {
			tty = os.Stdin
		} else {
			defer tty.Close()
		}
		in = tty
	}
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		if old, e := term.MakeRaw(int(f.Fd())); e == nil {
			defer term.Restore(int(f.Fd()), old)
		}
	}
	var one [1]byte
	_, _ = in.Read(one[:])
	return nil
}
func (v *imageView) SetStdin(r io.Reader)  { v.in = r }
func (v *imageView) SetStdout(w io.Writer) { v.out = w }
func (v *imageView) SetStderr(io.Writer)   {}

// imageShown is what an image view reports when it is closed.
type imageShown struct{ err error }

// showImage draws the marked image where the terminal can, else says in
// one line what it is and that Enter opens it. Its bytes are read only here.
func (m *model) showImage(t task.Task, name string) tea.Cmd {
	path, err := m.insideFolder(t, name)
	if err != nil {
		m.message = err.Error()
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		m.message = err.Error()
		return nil
	}
	proto := imageTerminal(m.getenv)
	if proto == "" {
		m.message = fmt.Sprintf("%s · image, %s: this terminal draws no images (iTerm2 and kitty do): Enter opens it", name, humanBytes(info.Size()))
		return nil
	}
	b, err := readFile(path)
	if err != nil {
		m.message = err.Error()
		return nil
	}
	v := &imageView{seq: imageSequence(proto, name, b, m.getenv("TMUX") != ""), title: task.FolderOf(t) + "/" + name + " · " + humanBytes(info.Size())}
	return tea.Exec(v, func(e error) tea.Msg { return imageShown{e} })
}

// getenv reads the environment through Options, so tests choose a terminal.
func (m model) getenv(k string) string {
	if m.opts.Getenv != nil {
		return m.opts.Getenv(k)
	}
	return os.Getenv(k)
}

// questionPicture is the task file a question asks you to look at: the one
// --show named (waiting_show, design_show), else the first task file the
// question names. "" when it names none.
func (m model) questionPicture(t task.Task, stage, question string) string {
	files := m.filesOf(t)
	has := func(name string) bool {
		for _, f := range files {
			if f == name {
				return true
			}
		}
		return false
	}
	if v := strings.TrimSpace(t.Fields[map[string]string{"waiting": "waiting_show", "design": "design_show"}[stage]]); v != "" && has(v) {
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

// askedPicture is the picture of the question t asks you, or "".
func (m model) askedPicture(t task.Task) string {
	if !m.forYou(t) {
		return ""
	}
	switch g := task.Stage(t, m.all); g {
	case "design":
		return m.questionPicture(t, g, t.Fields["design_question"])
	case "waiting":
		_, what := m.waitingFor(t)
		return m.questionPicture(t, g, what)
	}
	return ""
}

// pictureMemo is a question picture as drawn from one read of its file.
type pictureMemo struct {
	size  int64
	mtime time.Time
	lines []string
}

// pictureLimit is how many lines of a capture show under a question.
const pictureLimit = 30

// pictureLines show the question's picture right under it (5z4zt4 check
// 2): an .ans with its colours, at most pictureLimit lines; any other text
// file escaped; an image one line saying where to see it. Only this file is
// read, and only while the question shows.
func (m model) pictureLines(t task.Task, name string) []string {
	head := "  ▸ " + name
	path, err := m.insideFolder(t, name)
	if err != nil {
		return []string{m.styles.warning.Render(fit(head+": "+err.Error(), m.width))}
	}
	info, err := os.Stat(path)
	if err != nil {
		return []string{m.styles.warning.Render(fit(head+": "+err.Error(), m.width))}
	}
	if task.IsImageName(name) {
		where := "Space in FILES shows it"
		if imageTerminal(m.getenv) == "" {
			where = "this terminal draws no images: Enter in FILES opens it"
		}
		return []string{m.styles.accent.Render(fit(head+" · image "+humanBytes(info.Size())+" · "+where, m.width))}
	}
	if info.Size() > glanceLimit {
		return []string{m.styles.accent.Render(fit(head+" · "+humanBytes(info.Size())+": too large to show here, Enter in FILES opens it", m.width))}
	}
	memoKey := fmt.Sprintf("%s\x00%d", path, m.width)
	if p, ok := m.pictures[memoKey]; ok && p.size == info.Size() && p.mtime.Equal(info.ModTime()) {
		return p.lines
	}
	b, err := readFile(path)
	if err != nil {
		return []string{m.styles.warning.Render(fit(head+": "+err.Error(), m.width))}
	}
	out := []string{m.styles.accent.Render(fit(head+" · "+humanBytes(info.Size()), m.width))}
	defer func() {
		if m.pictures != nil {
			m.pictures[memoKey] = pictureMemo{size: info.Size(), mtime: info.ModTime(), lines: out}
		}
	}()
	var body string
	if ansName(name) {
		body = colourOnly(b, m.width-2, os.Getenv("NO_COLOR") == "")
	} else {
		body = glanceText(b, info.Size(), m.width-2)
	}
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if i == pictureLimit {
			out = append(out, m.styles.dim.Render(fit(fmt.Sprintf("  … %d more lines: Space in FILES shows all", len(lines)-pictureLimit), m.width)))
			break
		}
		out = append(out, "  "+l)
	}
	return out
}

// The screen capture (S, Jørgen 14:51): the frame as you see it goes to the
// clipboard as plain text, so it pastes cleanly into a chat with an agent,
// and S asks to attach it to the task in view as a coloured .ans file.

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
// plain text to the clipboard, the coloured frame kept for attaching.
func (m *model) capture() {
	frame := m.View()
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
	t, ok := m.selected()
	if !ok {
		m.message = fmt.Sprintf("%s (%d lines) · no task in view to attach it to", copied, n)
		return
	}
	name := "screen-" + captureNow().Format("1504") + ".ans"
	for i := 2; m.hasFile(t, name); i++ {
		name = fmt.Sprintf("screen-%s-%d.ans", captureNow().Format("1504"), i)
	}
	m.captured = strings.TrimRight(frame, "\n") + "\n"
	m.capturedNote = fmt.Sprintf("%s (%d lines)", copied, n)
	m.confirm = captureKey
	m.confirmPath = t.Path
	m.pendingValue = name
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

// attachCapture attaches the captured frame (the confirm's Enter).
func (m *model) attachCapture(t task.Task, name string) {
	body := []byte(m.captured)
	note := m.capturedNote
	m.captured, m.capturedNote = "", ""
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
