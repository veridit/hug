package tui

import (
	"errors"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// V pastes the clipboard into a task as a file (q7n2zd). Stubs: the tests
// come first.

// clipKey attaches the clipboard.
const clipKey = "V"

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

// runClip runs one clipboard command.
var runClip = func(argv []string) ([]byte, error) { return exec.Command(argv[0], argv[1:]...).Output() }

// readClipboard reads the clipboard; tests replace it.
var readClipboard = func(getenv func(string) string) (clip, error) { return clip{}, errClipEmpty }

// clipboardSource picks the commands that read the clipboard, or says why
// there are none.
func clipboardSource(goos string, getenv func(string) string, look func(string) (string, error)) (clipSource, error) {
	return clipSource{}, nil
}

// read runs src's commands through run.
func (src clipSource) read(run func([]string) ([]byte, error)) (clip, error) {
	return clip{}, nil
}

// decodeOSAScript turns osascript's «data PNGf…» into the PNG's bytes.
func decodeOSAScript(out []byte) ([]byte, error) { return nil, nil }

// pasteClipboard reads the clipboard and attaches it (not yet).
func (m *model) pasteClipboard() tea.Cmd { return nil }
