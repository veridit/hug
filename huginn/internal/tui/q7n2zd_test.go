package tui

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// q7n2zd (Jørgen 2026-11-20 10:23Z: "press a paste key that takes ...
// whatever is in the current paste buffer and attach it to something a
// bit like the screenshot"): V reads the system clipboard itself, since a
// terminal paste carries text only. Inside a task it attaches there; in a
// list it asks as S does. No test reads the real clipboard: readClipboard
// is replaced here, and TestMain makes it say "no clipboard tool".

// pasteModel is captureModel (000002 open, clock 14:52, commits recorded)
// with the clipboard holding c, or failing with err.
func pasteModel(t *testing.T, w int, c clip, err error) (model, *[]string) {
	t.Helper()
	m, commits := captureModel(t, w)
	old := readClipboard
	readClipboard = func(func(string) string) (clip, error) { return c, err }
	t.Cleanup(func() { readClipboard = old })
	return m, commits
}

// pasteFiles is every paste-* file on the board.
func pasteFiles(t *testing.T, m model) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(m.store.Root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), "paste-") {
			out = append(out, filepath.ToSlash(p))
		}
		return nil
	})
	return out
}

// taskText is task id's file as on disk.
func taskText(t *testing.T, m model, id string) string {
	t.Helper()
	tk, err := task.Resolve(m.all, id)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(m.store.Root, tk.Path))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Check 2: inside a task V attaches at once: an image as paste-HHMM.png,
// text with colour as .ans, other text as .txt, each with hug task
// attach's note (name, size, sha256), one commit each, and FILES shows it
// in the frame drawn right after.
func TestPasteInATaskAttachesThere(t *testing.T) {
	for _, c := range []struct {
		name string
		clip clip
		said string
	}{
		{"paste-1452.png", clip{body: png1, image: true}, "clipboard image"},
		{"paste-1452.ans", clip{body: []byte("\x1b[31mred\x1b[0m line\nplain line\n")}, "clipboard text (2 lines)"},
		{"paste-1452.txt", clip{body: []byte("one\ntwo\nthree\n")}, "clipboard text (3 lines)"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m, commits := pasteModel(t, 120, c.clip, nil)
			done := press(m, "V")
			if done.prompt != "" || done.confirm != "" {
				t.Fatalf("V in a task asked: prompt %q confirm %q", done.prompt, done.confirm)
			}
			if !attachedTo(t, done, "000002", c.name) {
				t.Fatalf("%s not attached: %v message %q", c.name, pasteFiles(t, done), done.message)
			}
			tk, _ := task.Resolve(done.all, "000002")
			b, _ := os.ReadFile(filepath.Join(done.store.Root, task.FolderOf(tk), c.name))
			if string(b) != string(c.clip.body) {
				t.Fatalf("%s holds %q, want the clipboard %q", c.name, b, c.clip.body)
			}
			if len(*commits) != 1 || (*commits)[0] != "hug task attach 000002" {
				t.Fatalf("commits %v", *commits)
			}
			if !strings.Contains(done.message, c.said) || !strings.Contains(done.message, "attached as "+c.name+" to 000002") {
				t.Fatalf("message %q, want %q and the file", done.message, c.said)
			}
			// hug task attach's note: name, size, sha256, then why.
			note := regexp.MustCompile(`file:` + regexp.QuoteMeta(c.name) + ` \(\d+ bytes, sha256:[0-9a-f]{8}\) pasted from the clipboard in hug tui`)
			if !note.MatchString(taskText(t, done, "000002")) {
				t.Fatalf("no attach note for %s:\n%s", c.name, taskText(t, done, "000002"))
			}
			if screen := ansi.Strip(done.View()); !strings.Contains(screen, c.name) {
				t.Fatalf("FILES does not show %s at once:\n%s", c.name, screen)
			}
		})
	}
	// A second paste in the same minute takes the next number.
	m, _ := pasteModel(t, 120, clip{body: []byte("first\n")}, nil)
	m = press(m, "V")
	readClipboard = func(func(string) string) (clip, error) { return clip{body: []byte("second\n")}, nil }
	again := press(m, "V")
	if !attachedTo(t, again, "000002", "paste-1452.txt") || !attachedTo(t, again, "000002", "paste-1452-2.txt") {
		t.Fatalf("a second paste: %v message %q", pasteFiles(t, again), again.message)
	}
}

// V in a file shown from FILES attaches to that task too, as S works there.
func TestPasteInAFileAttachesToItsTask(t *testing.T) {
	m, commits := pasteModel(t, 120, clip{body: []byte("log line\n")}, nil)
	shown := press(toFiles(t, m), " ")
	if !shown.glance {
		t.Fatal("Space did not show the file")
	}
	done := press(shown, "V")
	if !attachedTo(t, done, "000002", "paste-1452.txt") || len(*commits) != 1 {
		t.Fatalf("V in a file: %v commits %v message %q", pasteFiles(t, done), *commits, done.message)
	}
}

// Check 3: in a list V asks where, as S does (attach to ID · select ·
// create): Enter attaches to the offered row, s picks a row, c creates a
// task, and Esc attaches nothing.
func TestPasteInAListAsksAsSDoes(t *testing.T) {
	text := clip{body: []byte("from a chat\n")}
	list := func() (model, *[]string) {
		m, commits := pasteModel(t, 120, text, nil)
		return press(m, "esc"), commits
	}
	m, commits := list()
	row := mustSelected(t, m)
	asked := press(m, "V")
	if asked.prompt != capturePrompt || asked.input.Value() != row.ID {
		t.Fatalf("V in the list: prompt %q value %q message %q", asked.prompt, asked.input.Value(), asked.message)
	}
	bar := ansi.Strip(asked.View())
	for _, want := range []string{"Attach the clipboard to: " + row.ID, "select", "create", "clipboard text (1 line)"} {
		if !strings.Contains(bar, want) {
			t.Fatalf("the prompt lacks %q:\n%s", want, bar)
		}
	}
	if strings.Contains(bar, "screen copied") || strings.Contains(bar, "this screen") {
		t.Fatalf("the paste prompt speaks of the screen:\n%s", bar)
	}
	if len(pasteFiles(t, asked)) != 0 || len(*commits) != 0 {
		t.Fatalf("V wrote before Enter: %v %v", pasteFiles(t, asked), *commits)
	}
	done := press(asked, "enter")
	if !attachedTo(t, done, row.ID, "paste-1452.txt") || len(*commits) != 1 || (*commits)[0] != "hug task attach "+row.ID {
		t.Fatalf("Enter: files %v commits %v message %q", pasteFiles(t, done), *commits, done.message)
	}

	// Esc: nothing attached, and it says so.
	m, commits = list()
	kept := press(m, "V", "esc")
	if kept.prompt != "" || kept.captured != "" || len(pasteFiles(t, kept)) != 0 || len(*commits) != 0 || !strings.Contains(kept.message, "not attached") {
		t.Fatalf("Esc: prompt %q files %v commits %v message %q", kept.prompt, pasteFiles(t, kept), *commits, kept.message)
	}

	// s picks a row; Esc while picking attaches nothing.
	m, commits = list()
	picking := press(m, "V", "s")
	if !picking.picking || !strings.Contains(ansi.Strip(picking.View()), "pick the task for the clipboard") {
		t.Fatalf("s: picking %v\n%s", picking.picking, ansi.Strip(picking.View()))
	}
	if after := press(picking, "V"); !after.picking || after.prompt != "" || after.message == "" {
		t.Fatalf("V while picking: picking %v prompt %q", after.picking, after.prompt)
	}
	if back := press(picking, "esc"); back.picking || len(pasteFiles(t, back)) != 0 || !strings.Contains(back.message, "not attached") {
		t.Fatalf("Esc while picking: %v %q", pasteFiles(t, back), back.message)
	}
	moved := press(picking, "j")
	target := mustSelected(t, moved)
	picked := press(moved, "enter")
	if !attachedTo(t, picked, target.ID, "paste-1452.txt") || len(*commits) != 1 {
		t.Fatalf("Enter while picking: %v %v %q", pasteFiles(t, picked), *commits, picked.message)
	}

	// c creates a task with the paste in its Files.
	m, commits = list()
	asking := press(m, "V", "c")
	if asking.prompt != captureTitlePrompt || !strings.Contains(asking.input.Prompt, "clipboard") {
		t.Fatalf("c: prompt %q %q", asking.prompt, asking.input.Prompt)
	}
	created := press(asking, "N", "o", "t", "e", "s", "enter")
	sel := mustSelected(t, created)
	if sel.Title != "Notes" || !attachedTo(t, created, sel.ID, "paste-1452.txt") || len(*commits) != 1 || (*commits)[0] != "hug task new "+sel.ID {
		t.Fatalf("c: %q files %v commits %v message %q", sel.Title, pasteFiles(t, created), *commits, created.message)
	}
}

// Check 4: an empty clipboard, no clipboard tool, an SSH session, or text
// that looks like a secret: a message whose last line is a fix: line,
// nothing attached, no prompt, no commit. In a task and in a list.
func TestPasteRefusesWithAFix(t *testing.T) {
	secret := clip{body: []byte("aws_" + "secret_access_key=" + "wJalr" + strings.Repeat("Xy7", 10) + "\n")}
	for _, c := range []struct {
		name string
		clip clip
		err  error
		says string
	}{
		{"empty", clip{}, errClipEmpty, "empty"},
		{"no tool", clip{}, errClipNoTool, "no clipboard tool"},
		{"ssh", clip{}, errClipSSH, "SSH"},
		{"secret", secret, nil, "secret"},
		{"not text", clip{body: []byte{0xff, 0xfe, 0}}, nil, "not text"},
	} {
		for _, where := range []string{"task", "list"} {
			t.Run(c.name+"/"+where, func(t *testing.T) {
				m, commits := pasteModel(t, 120, c.clip, c.err)
				if where == "list" {
					m = press(m, "esc")
				}
				id := mustSelected(t, m).ID
				after := press(m, "V")
				if after.prompt != "" || after.confirm != "" || len(pasteFiles(t, after)) != 0 || len(*commits) != 0 {
					t.Fatalf("prompt %q files %v commits %v message %q", after.prompt, pasteFiles(t, after), *commits, after.message)
				}
				lines := strings.Split(after.message, "\n")
				if !strings.Contains(after.message, c.says) || lines[len(lines)-1] != "fix: hug task attach "+id+" FILE" && !strings.HasPrefix(lines[len(lines)-1], "fix: hug task attach "+id+" FILE ") {
					t.Fatalf("message %q: want %q and a last line fix: hug task attach %s FILE", after.message, c.says, id)
				}
				if screen := ansi.Strip(after.View()); !strings.Contains(screen, "fix: hug task attach "+id+" FILE") {
					t.Fatalf("the fix line is not drawn:\n%s", screen)
				}
			})
		}
	}
}

// The commands that read the clipboard, by platform: none over SSH, none
// on Windows, and a fake one a test names in HUG_TEST_CLIPBOARD.
func TestClipboardSource(t *testing.T) {
	has := func(names ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, x := range names {
				if x == n {
					return "/usr/bin/" + n, nil
				}
			}
			return "", errors.New("not found")
		}
	}
	envOf := func(kv ...string) func(string) string {
		return func(k string) string {
			for i := 0; i+1 < len(kv); i += 2 {
				if kv[i] == k {
					return kv[i+1]
				}
			}
			return ""
		}
	}
	all := has("osascript", "pbpaste", "wl-paste", "xclip", "/tmp/fake-clip")
	for _, c := range []struct {
		name, goos  string
		env         func(string) string
		look        func(string) (string, error)
		err         error
		image, text string
	}{
		{"ssh connection", "darwin", envOf("SSH_CONNECTION", "1.2.3.4 5 6.7.8.9 22"), all, errClipSSH, "", ""},
		{"ssh tty", "linux", envOf("SSH_TTY", "/dev/pts/1", "WAYLAND_DISPLAY", "wayland-0"), all, errClipSSH, "", ""},
		{"ssh beats the fake", "linux", envOf("SSH_TTY", "/dev/pts/1", "HUG_TEST_CLIPBOARD", "/tmp/fake-clip"), all, errClipSSH, "", ""},
		{"windows", "windows", envOf(), all, errClipNoTool, "", ""},
		{"macos", "darwin", envOf(), all, nil, "osascript -e the clipboard as «class PNGf»", "pbpaste"},
		{"macos without pbpaste", "darwin", envOf(), has("osascript"), errClipNoTool, "", ""},
		{"macos without osascript", "darwin", envOf(), has("pbpaste"), nil, "", "pbpaste"},
		{"wayland", "linux", envOf("WAYLAND_DISPLAY", "wayland-0", "DISPLAY", ":0"), all, nil, "wl-paste --no-newline --type image/png", "wl-paste --no-newline"},
		{"x11", "linux", envOf("DISPLAY", ":0"), all, nil, "xclip -selection clipboard -target image/png -out", "xclip -selection clipboard -out"},
		{"wayland, only xclip", "linux", envOf("WAYLAND_DISPLAY", "wayland-0", "DISPLAY", ":0"), has("xclip"), nil, "xclip -selection clipboard -target image/png -out", "xclip -selection clipboard -out"},
		{"linux, no display", "linux", envOf(), all, errClipNoTool, "", ""},
		{"linux, no tool", "linux", envOf("DISPLAY", ":0"), has(), errClipNoTool, "", ""},
		{"the test's fake", "windows", envOf("HUG_TEST_CLIPBOARD", "/tmp/fake-clip"), all, nil, "/tmp/fake-clip image", "/tmp/fake-clip text"},
	} {
		t.Run(c.name, func(t *testing.T) {
			src, err := clipboardSource(c.goos, c.env, c.look)
			if !errors.Is(err, c.err) || (err == nil) != (c.err == nil) {
				t.Fatalf("err %v, want %v", err, c.err)
			}
			if err != nil {
				return
			}
			if got := strings.Join(src.image, " "); got != c.image {
				t.Fatalf("image %q, want %q", got, c.image)
			}
			if got := strings.Join(src.text, " "); got != c.text {
				t.Fatalf("text %q, want %q", got, c.text)
			}
		})
	}
}

// read tries the image first, then text: an image the command cannot give
// (no PNG on the clipboard) falls to text; empty text is an empty
// clipboard; osascript's «data PNGf…» decodes to the PNG.
func TestClipboardRead(t *testing.T) {
	fake := func(outs map[string]string) func([]string) ([]byte, error) {
		return func(argv []string) ([]byte, error) {
			out, ok := outs[argv[len(argv)-1]]
			if !ok {
				return nil, errors.New("exit status 1")
			}
			return []byte(out), nil
		}
	}
	src := clipSource{image: []string{"fake", "image"}, text: []string{"fake", "text"}}
	if c, err := src.read(fake(map[string]string{"image": string(png1), "text": "words"})); err != nil || !c.image || string(c.body) != string(png1) {
		t.Fatalf("an image: %v %v %q", err, c.image, c.body)
	}
	if c, err := src.read(fake(map[string]string{"text": "words\n"})); err != nil || c.image || string(c.body) != "words\n" {
		t.Fatalf("text: %v %v %q", err, c.image, c.body)
	}
	// Bytes from the image command that are no PNG are not an image.
	if c, err := src.read(fake(map[string]string{"image": "words", "text": "words"})); err != nil || c.image {
		t.Fatalf("no PNG: %v %v", err, c.image)
	}
	for name, outs := range map[string]map[string]string{"nothing": {}, "blank": {"text": " \n\t\n"}, "empty": {"text": ""}} {
		if _, err := src.read(fake(outs)); !errors.Is(err, errClipEmpty) {
			t.Fatalf("%s: %v, want empty", name, err)
		}
	}
	// No image command (macOS without osascript): text only.
	if c, err := (clipSource{text: []string{"fake", "text"}}).read(fake(map[string]string{"text": "t"})); err != nil || string(c.body) != "t" {
		t.Fatalf("text only: %v %q", err, c.body)
	}
	hex := "«data PNGf" + strings.ToUpper(hexOf(png1)) + "»\n"
	got, err := decodeOSAScript([]byte(hex))
	if err != nil || string(got) != string(png1) {
		t.Fatalf("decode: %v %q", err, got)
	}
	if _, err := decodeOSAScript([]byte("«data TIFF00»")); err == nil {
		t.Fatal("decoded something that is no «data PNGf»")
	}
	osa := clipSource{image: []string{"osascript", "image"}, text: []string{"fake", "text"}, decode: decodeOSAScript}
	if c, err := osa.read(fake(map[string]string{"image": hex, "text": "t"})); err != nil || !c.image || string(c.body) != string(png1) {
		t.Fatalf("osascript's image: %v %v", err, c.image)
	}
}

func hexOf(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, 2*len(b))
	for _, x := range b {
		out = append(out, digits[x>>4], digits[x&15])
	}
	return string(out)
}

// V is named where S is: the task's key bar at 140, and ? help.
func TestPasteKeyIsNamed(t *testing.T) {
	m, _ := pasteModel(t, 140, clip{body: []byte("x\n")}, nil)
	if !barNames(m, "V") {
		t.Fatalf("the task's key bar at 140 does not name V: %s", ansi.Strip(m.keyBar()))
	}
	if !barNames(m, "S") {
		t.Fatalf("the bar lost S: %s", ansi.Strip(m.keyBar()))
	}
	for _, where := range []model{m, press(m, "esc")} {
		help := ansi.Strip(press(where, "?").helpText())
		if !regexp.MustCompile(`(?m)^.*\bV\s+paste the clipboard`).MatchString(help) {
			t.Fatalf("? help does not list V:\n%s", help)
		}
	}
}
