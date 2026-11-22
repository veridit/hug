package tui

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

// V pastes the clipboard into a task as a file (q7n2zd, Jørgen
// 2026-11-20 10:23Z: "press a paste key that takes ... whatever is in the
// current paste buffer and attach it to something a bit like the
// screenshot"). A terminal paste (Cmd+V) carries text only, so V reads
// the system clipboard itself: a PNG as paste-HHMM.png, text with colour
// as .ans, other text as .txt. Inside a task it attaches there at once;
// in a list it asks where, as S does (capture.go). It refuses what hug
// task attach refuses (not text, too large, a secret), and over SSH or
// with no clipboard tool it says so, with a fix: line.

// clipKey attaches the clipboard (Jørgen 2026-11-20 11:52Z: "let's try
// uppercase V"; provisional until the key review, bfxq93).
const clipKey = "V"

// clipKind marks the capture state as V's paste, not S's screen.
const clipKind = "clipboard"

// clip is what the clipboard holds: a PNG, or text.
type clip struct {
	body  []byte
	image bool
}

// clipSource is how this machine reads its clipboard: image is the command
// that prints a PNG (nil: none), text the one that prints text, decode
// turns the image command's output into PNG bytes (nil: as is).
type clipSource struct {
	image, text []string
	decode      func([]byte) ([]byte, error)
}

// Why V read nothing.
var (
	errClipSSH    = errors.New("an SSH session")
	errClipNoTool = errors.New("no clipboard tool")
	errClipEmpty  = errors.New("the clipboard is empty")
)

// clipTimeout bounds one clipboard command: V waits for it.
const clipTimeout = 5 * time.Second

// runClip runs one clipboard command.
var runClip = func(argv []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clipTimeout)
	defer cancel()
	return exec.CommandContext(ctx, argv[0], argv[1:]...).Output()
}

// readClipboard reads the clipboard; tests replace it (TestMain: no
// clipboard tool), so no test reads the real one.
var readClipboard = func(getenv func(string) string) (clip, error) {
	src, err := clipboardSource(runtime.GOOS, getenv, exec.LookPath)
	if err != nil {
		return clip{}, err
	}
	return src.read(runClip)
}

// clipboardSource picks the commands that read the clipboard, or says why
// there are none. Over SSH the clipboard here is the remote machine's,
// not the reader's: none. HUG_TEST_CLIPBOARD names a fake a tmux test
// runs as "CMD image" and "CMD text". macOS: osascript for a PNG, pbpaste
// for text. Linux: wl-paste under Wayland, else xclip under X. Windows:
// none yet.
func clipboardSource(goos string, getenv func(string) string, look func(string) (string, error)) (clipSource, error) {
	if getenv("SSH_CONNECTION") != "" || getenv("SSH_TTY") != "" {
		return clipSource{}, errClipSSH
	}
	if fake := getenv("HUG_TEST_CLIPBOARD"); fake != "" {
		return clipSource{image: []string{fake, "image"}, text: []string{fake, "text"}}, nil
	}
	has := func(name string) bool { _, err := look(name); return err == nil }
	switch goos {
	case "windows":
		return clipSource{}, errClipNoTool
	case "darwin":
		if !has("pbpaste") {
			return clipSource{}, errClipNoTool
		}
		src := clipSource{text: []string{"pbpaste"}}
		if has("osascript") {
			src.image = []string{"osascript", "-e", "the clipboard as «class PNGf»"}
			src.decode = decodeOSAScript
		}
		return src, nil
	}
	if getenv("WAYLAND_DISPLAY") != "" && has("wl-paste") {
		return clipSource{image: []string{"wl-paste", "--no-newline", "--type", "image/png"}, text: []string{"wl-paste", "--no-newline"}}, nil
	}
	if getenv("DISPLAY") != "" && has("xclip") {
		return clipSource{image: []string{"xclip", "-selection", "clipboard", "-target", "image/png", "-out"}, text: []string{"xclip", "-selection", "clipboard", "-out"}}, nil
	}
	return clipSource{}, errClipNoTool
}

// read runs src's commands through run: the image first (a screenshot
// copied is usually also text-less), then text. An image command that
// fails or prints no PNG means no image; text that is blank, or a text
// command that fails, is an empty clipboard.
func (src clipSource) read(run func([]string) ([]byte, error)) (clip, error) {
	if len(src.image) > 0 {
		if out, err := run(src.image); err == nil {
			if src.decode != nil {
				out, err = src.decode(out)
			}
			if err == nil && task.ImageKind(out) == "png" {
				return clip{body: out, image: true}, nil
			}
		}
	}
	out, err := run(src.text)
	if err != nil {
		return clip{}, fmt.Errorf("%w (%s: %v)", errClipEmpty, filepath.Base(src.text[0]), err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return clip{}, errClipEmpty
	}
	return clip{body: out}, nil
}

// decodeOSAScript turns osascript's «data PNGf89504E47…» into the PNG's
// bytes.
func decodeOSAScript(out []byte) ([]byte, error) {
	s := strings.TrimSpace(string(out))
	if !strings.HasPrefix(s, "«data PNGf") || !strings.HasSuffix(s, "»") {
		return nil, errors.New("osascript gave no «data PNGf»")
	}
	return hex.DecodeString(strings.TrimSuffix(strings.TrimPrefix(s, "«data PNGf"), "»"))
}

// clipColour finds an SGR sequence: text with colour is an .ans file,
// shown with its colours (picture.go).
var clipColour = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

// pasteClipboard reads the clipboard: in a task (its page or one of its
// files) it attaches there at once; in a list it asks where, as S does.
// Nothing is asked when the paste would be refused anyway.
func (m *model) pasteClipboard() tea.Cmd {
	t, ok := m.selected()
	id := "ID"
	if ok {
		id = t.ID
	}
	fix := "\nfix: hug task attach " + id + " FILE"
	c, err := readClipboard(m.getenv)
	if err == nil && !c.image && len(bytes.TrimSpace(c.body)) == 0 {
		err = errClipEmpty
	}
	if err != nil {
		m.message = clipRefusal(err) + fix
		return nil
	}
	stamp := captureNow().Format("1504")
	ext, what := ".txt", "clipboard text ("+lineCount(c.body)+")"
	if kind := task.ImageKind(c.body); kind != "" {
		ext, what = map[string]string{"png": ".png", "jpeg": ".jpg", "gif": ".gif"}[kind], "clipboard image ("+humanBytes(int64(len(c.body)))+")"
	} else if clipColour.Match(c.body) {
		ext = ".ans"
	}
	// What hug task attach refuses, refused before anything is asked.
	if e := task.CheckText("paste-"+stamp+ext, c.body); e != nil {
		var te *task.Error
		why := e.Error()
		if errors.As(e, &te) {
			why = te.Value
			if te.Kind == "file-not-text" {
				why = "not text (NUL bytes or not UTF-8) and no PNG"
			}
		}
		m.message = what + " not attached: " + oneLine(why) + fix
		return nil
	}
	if ext != ".png" && ext != ".jpg" && ext != ".gif" {
		if hint := task.SecretHint(c.body); hint != "" {
			m.message = what + " not attached: it looks like it may contain a secret (" + hint + ")" + fix + ` --safe-reason "why this is safe"`
			return nil
		}
	}
	m.captured, m.capturedNote, m.captureStamp = string(c.body), what, stamp
	m.captureKind, m.captureExt = clipKind, ext
	if m.detail && ok {
		m.attachCapture(t, m.captureName(t))
		return nil
	}
	offer := ""
	if ok {
		offer = t.ID
	}
	return m.openCapture(offer)
}

// clipRefusal says why V read nothing.
func clipRefusal(err error) string {
	switch {
	case errors.Is(err, errClipSSH):
		return "the clipboard is not read in an SSH session: it would be the remote machine's, not yours; save it to a file and attach that"
	case errors.Is(err, errClipNoTool):
		if runtime.GOOS == "windows" {
			return "no clipboard tool: V reads the clipboard on macOS and Linux only; save it to a file and attach that"
		}
		return "no clipboard tool: V reads the clipboard with pbpaste and osascript (macOS), wl-paste (Wayland) or xclip (X)"
	case errors.Is(err, errClipEmpty):
		return err.Error() + ": copy a screenshot or text first"
	}
	return "the clipboard could not be read: " + oneLine(err.Error())
}

// lineCount is "1 line" or "N lines": a last line without a newline
// counts.
func lineCount(b []byte) string {
	n := bytes.Count(b, []byte("\n"))
	if len(b) > 0 && b[len(b)-1] != '\n' {
		n++
	}
	if n == 1 {
		return "1 line"
	}
	return fmt.Sprintf("%d lines", n)
}
