package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/huginn/internal/tui/testfixture"
)

// png1 is a 1x1 PNG: the smallest real image.
var png1 = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")

// capture is an .ans screen capture: colours, plus a clear-screen and a
// title escape that must never reach the terminal.
const captureAns = "\x1b[1;38;2;0;191;255mHUG TASKS\x1b[0m · overview\n\x1b[38;5;203mIN PROGRESS\x1b[0m rows\x1b[2J\x1b]0;pwned\a\nplain line\n"

// pictureModel is the seeded board where 000002 (doing, alice) waits on
// the TUI user, jhf, to look at a capture and a screenshot it carries.
func pictureModel(t *testing.T, w, h int, env map[string]string, show string) (model, *[]string) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	root := t.TempDir()
	if err := testfixture.Seed(root); err != nil {
		t.Fatal(err)
	}
	// Both clocks pinned: asked at 14:42, seen at 14:52.
	oldTask, oldTUI := task.Now, now
	task.Now = func() time.Time { return time.Date(2026, 11, 17, 14, 42, 0, 0, time.Local) }
	now = func() time.Time { return time.Date(2026, 11, 17, 14, 52, 0, 0, time.Local) }
	t.Cleanup(func() { task.Now, now = oldTask, oldTUI })
	s := &task.Store{Root: root}
	// In a fixed order: NOTES lists the attach lines as written.
	for _, f := range []struct {
		name string
		body []byte
	}{{"rows.ans", []byte(captureAns)}, {"shot.png", png1}} {
		if _, _, err := s.Attach("000002", f.name, f.body, "", false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Ask("000002", task.Question{What: "look at the IN PROGRESS rows", For: "jhf", Show: show}, "alice"); err != nil {
		t.Fatal(err)
	}
	var reads []string
	old := readFile
	readFile = func(p string) ([]byte, error) { reads = append(reads, filepath.ToSlash(p)); return os.ReadFile(p) }
	t.Cleanup(func() { readFile = old })
	m := newModel(s, Options{User: "jhf", Getenv: func(k string) string { return env[k] }})
	m.width, m.height = w, h
	m.resize()
	m.setView("list")
	return m, &reads
}

// 5z4zt4: a PNG, JPEG or GIF is a task file when its name says so; a
// renamed binary or a binary with an image name is not.
func TestImagesAreTaskFiles(t *testing.T) {
	if e := task.CheckText("shot.png", png1); e != nil {
		t.Fatalf("a PNG refused: %v", e)
	}
	if e := task.CheckText("shot.txt", png1); e == nil || !strings.Contains(e.Error(), "not named .png") {
		t.Fatalf("a PNG named .txt: %v", e)
	}
	if e := task.CheckText("shot.png", []byte{0x7f, 'E', 'L', 'F', 0}); e == nil {
		t.Fatal("a binary named .png was accepted")
	}
	big := append(append([]byte{}, png1...), make([]byte, task.MaxImageBytes)...)
	if e := task.CheckText("big.png", big); e == nil || !strings.Contains(e.Error(), "2 MiB") {
		t.Fatalf("an image over 2 MiB: %v", e)
	}
}

// 5z4zt4 checks 2 and 3, as qcq3tm changed them: the file a question names
// (--show) is one link line under the question, never drawn on the page
// (Jørgen: drawn above the page it hid the question). The list and the
// page read no file; Enter on the question shows it in the file frame, an
// .ans with its colours and nothing else of its escapes, and Esc goes back
// to the question (goldens at 80 and 140).
func TestQuestionShowsItsPicture(t *testing.T) {
	var got []string
	for _, w := range []int{80, 140} {
		m, reads := pictureModel(t, w, 30, nil, "rows.ans")
		m.View()
		if len(*reads) != 0 {
			t.Fatalf("the list read files: %v", *reads)
		}
		m = openTask(t, m, "000002")
		if len(*reads) != 0 {
			t.Fatalf("the page read %v, want no file until Enter", *reads)
		}
		screen := ansi.Strip(m.View())
		q := strings.Index(screen, "> ? look at the IN PROGRESS rows") // in the details, focused (0gztsy)
		l := strings.Index(screen, "→ rows.ans · Enter shows")
		if q < 0 || l < q || strings.Contains(screen, "HUG TASKS · overview") {
			t.Fatalf("at %d the question has no link line, or the file is drawn:\n%s", w, screen)
		}
		for _, line := range strings.Split(m.View(), "\n") {
			if n := ansi.StringWidth(line); n > w {
				t.Fatalf("at %d a line is %d wide: %q", w, n, line)
			}
		}
		got = append(got, "== 000002 at "+strconv.Itoa(w), screen)
		f := press(m, "enter")
		file := ansi.Strip(f.View())
		if !f.glance || len(*reads) != 1 || !strings.HasSuffix((*reads)[0], "/000002/rows.ans") || !strings.Contains(file, "HUG TASKS · overview") || !strings.Contains(file, `\x1b[2J`) {
			t.Fatalf("at %d Enter did not show rows.ans (reads %v):\n%s", w, *reads, file)
		}
		got = append(got, "== 000002 Enter at "+strconv.Itoa(w), file)
		back := press(f, "esc")
		if back.glance || back.detailPane != focusQuestion || !strings.Contains(ansi.Strip(back.View()), "> ? look at the IN PROGRESS rows") {
			t.Fatalf("at %d Esc did not go back to the question:\n%s", w, ansi.Strip(back.View()))
		}
	}
	text := strings.Join(got, "\n") + "\n"
	path := filepath.Join("testdata", "detail", "picture.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestQuestionShowsItsPicture ./huginn/internal/tui/", err)
	}
	if text != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("picture differs from %s:\n%s", path, text)
	}

	// Without --show, a task file the question names is its link; an
	// image too, never read until shown.
	m, reads := pictureModel(t, 100, 30, nil, "")
	if _, err := m.store.Ask("000002", task.Question{What: "compare shot.png with the rows", For: "jhf"}, "alice"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	m = openTask(t, m, "000002")
	screen := ansi.Strip(m.View())
	if !strings.Contains(screen, "→ shot.png · Enter shows") || len(*reads) != 0 {
		t.Fatalf("image link under the question, reads %v:\n%s", *reads, screen)
	}
	// --show must name a task file.
	if _, err := m.store.Ask("000002", task.Question{What: "look", For: "jhf", Show: "nope.ans"}, "alice"); err == nil || !strings.Contains(err.Error(), "hug task attach 000002 FILE --name nope.ans") {
		t.Fatalf("--show of a missing file: %v", err)
	}
	// A review asked with --show (6w0gve) links its file under the review
	// question for the reviewer. 000006 is doing and not waiting.
	if _, _, err := m.store.Attach("000006", "rows.ans", []byte(captureAns), "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.store.Review("000006", "jhf", "rows.ans"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	m = openTask(t, m, "000006")
	screen = ansi.Strip(m.View())
	q, p := strings.Index(screen, "? review: review it"), strings.Index(screen, "→ rows.ans · Enter shows")
	if q < 0 || p < q || strings.Contains(screen, "HUG TASKS · overview") {
		t.Fatalf("the review file is not a link under the review question:\n%s", screen)
	}
}

// colourOnly keeps colours and escapes the rest, with colours on.
func TestCaptureKeepsOnlyColours(t *testing.T) {
	got := colourOnly([]byte(captureAns), 80, true)
	if !strings.Contains(got, "\x1b[1;38;2;0;191;255mHUG TASKS\x1b[0m") || !strings.Contains(got, "\x1b[38;5;203m") {
		t.Fatalf("colours lost: %q", got)
	}
	for _, bad := range []string{"\x1b[2J", "\x1b]0;", "\a"} {
		if strings.Contains(got, bad) {
			t.Fatalf("%q reached the terminal: %q", bad, got)
		}
	}
	if !strings.Contains(got, `\x1b[2J`) || !strings.Contains(got, `\x1b]0;pwned\a`) {
		t.Fatalf("other escapes not shown escaped: %q", got)
	}
	// A full-width frame row: its padding goes, so it is not cut.
	if got := ansi.Strip(colourOnly([]byte("\x1b[32mdone\x1b[0m"+strings.Repeat(" ", 100)+"\n\x1b[1mbold   "+strings.Repeat(" ", 90)+"\x1b[0m\n"), 20, true)); got != "done\nbold" {
		t.Fatalf("padding kept or cut: %q", got)
	}
	for _, line := range strings.Split(colourOnly([]byte(strings.Repeat("\x1b[31mwide ", 40)), 20, true), "\n") {
		if n := ansi.StringWidth(line); n > 20 || !strings.HasSuffix(line, "\x1b[0m") {
			t.Fatalf("line %d wide or colour left open: %q", n, line)
		}
	}
}

// 5z4zt4 check 1, in the file frame since 2qndsd: Space on an image draws
// it where the terminal can (iTerm2's OSC 1337, kitty's graphics protocol,
// wrapped for tmux), and says in one line what it is elsewhere, without
// reading it.
func TestImageShownWhereTheTerminalCan(t *testing.T) {
	m, reads := pictureModel(t, 100, 30, nil, "")
	m = openTask(t, m, "000002")
	m = toFiles(t, m)
	for i := 0; i < 5; i++ {
		if name, _, _ := m.glanceFile(mustSelected(t, m)); name == "shot.png" {
			break
		}
		m = press(m, "j")
	}
	if name, _, _ := m.glanceFile(mustSelected(t, m)); name != "shot.png" {
		t.Fatalf("marked %q", name)
	}
	next := press(m, " ")
	if !next.glance || next.glanceImage != nil || !strings.Contains(ansi.Strip(next.View()), "image, 67 B: this terminal draws no images (iTerm2 and kitty do): Enter opens it") || len(*reads) != 0 {
		t.Fatalf("plain terminal: frame %v image %v reads %v", next.glance, next.glanceImage != nil, *reads)
	}
	for _, c := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"TERM_PROGRAM": "iTerm.app"}, "\x1b]1337;File=name=c2hvdC5wbmc=;size=67;inline=1"},
		{map[string]string{"TERM": "xterm-kitty"}, "\x1b_Ga=T,f=100,m=0;"},
		{map[string]string{"TERM_PROGRAM": "iTerm.app", "TMUX": "/tmp/tmux-1/default,1,0"}, "\x1bPtmux;\x1b\x1b]1337;File="},
	} {
		m.opts.Getenv = func(k string) string { return c.env[k] }
		*reads = nil
		shown := press(m, " ")
		if shown.glanceImage == nil || len(*reads) != 1 || !strings.Contains(shown.View(), c.want[:12]) {
			t.Fatalf("%v: no image drawn (reads %v)", c.env, *reads)
		}
		seq := imageSequenceSized(imageTerminal(m.getenv), "shot.png", png1, m.getenv("TMUX") != "", 0, 0)
		if !strings.HasPrefix(seq, c.want) {
			t.Fatalf("%v: sequence starts %q", c.env, seq[:min(60, len(seq))])
		}
	}
}

func mustSelected(t *testing.T, m model) task.Task {
	t.Helper()
	sel, ok := m.selected()
	if !ok {
		t.Fatal("nothing selected")
	}
	return sel
}

// 5z4zt4 (Jørgen 14:51): S copies the screen as plain text at once, then
// asks where to attach the coloured frame, offering the task in view;
// Enter attaches and commits screen-HHMM.ans, Esc keeps only the copy
// (hjbfzd: the prompt, not a confirm).
func TestScreenCaptureCopiesThenAsks(t *testing.T) {
	var copied []string
	old, oldNow := clipboard, captureNow
	clipboard = func(text string, _ bool) error { copied = append(copied, text); return nil }
	captureNow = func() time.Time { return time.Date(2026, 11, 17, 14, 52, 0, 0, time.Local) }
	t.Cleanup(func() { clipboard, captureNow = old, oldNow })
	m, _ := pictureModel(t, 100, 30, nil, "")
	var commits []string
	m.opts.Commit = func(msg string) error { commits = append(commits, msg); return nil }
	m = openTask(t, m, "000002")
	before := ansi.Strip(m.View())

	asked := press(m, "S")
	if len(copied) != 1 || strings.Contains(copied[0], "\x1b") || !strings.Contains(copied[0], "? look at the IN PROGRESS rows") {
		t.Fatalf("clipboard got %q", copied)
	}
	if first := strings.SplitN(copied[0], "\n", 2)[0]; !strings.HasPrefix(before, first) {
		t.Fatalf("the copy is not the screen as seen: %q", first)
	}
	bar := ansi.Strip(asked.View())
	if asked.prompt != capturePrompt || !strings.Contains(bar, "Attach this screen to: 000002") {
		t.Fatalf("S did not ask:\n%s", bar)
	}
	if kept := press(asked, "esc"); kept.prompt != "" || kept.hasFile(mustSelected(t, kept), "screen-1452.ans") || !strings.Contains(kept.message, "screen copied (") || !strings.Contains(kept.message, "not attached") {
		t.Fatalf("Esc: prompt %q message %q", kept.prompt, kept.message)
	}
	done := press(asked, "enter")
	sel := mustSelected(t, done)
	if !done.hasFile(sel, "screen-1452.ans") || len(commits) != 1 || commits[0] != "hug task attach 000002" {
		t.Fatalf("Enter: files %v commits %v message %q", done.filesOf(sel), commits, done.message)
	}
	if !strings.Contains(done.message, "screen copied (") || !strings.Contains(done.message, "attached as screen-1452.ans to 000002") {
		t.Fatalf("footer: %q", done.message)
	}
	b, err := os.ReadFile(filepath.Join(done.store.Root, task.FolderOf(sel), "screen-1452.ans"))
	if err != nil || !strings.Contains(string(b), "? look at the IN PROGRESS rows") {
		t.Fatalf("attached capture: %v\n%s", err, b)
	}
	// A second capture in the same minute gets its own name.
	again := press(done, "S", "enter")
	if !again.hasFile(mustSelected(t, again), "screen-1452-2.ans") {
		t.Fatalf("second capture: files %v message %q", again.filesOf(mustSelected(t, again)), again.message)
	}
}
