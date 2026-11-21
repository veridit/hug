package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// hjbfzd (Jørgen 2026-11-18, yf39ya requests 1-2): after S the prompt asks
// where the screen goes: a task id typed (a prefix completes), s to pick
// a task from the rows, c to create one. Enter attaches, Esc keeps only
// the copy.

// captureModel is the seeded board with the clipboard and the clock
// pinned, commits recorded, and 000002 open.
func captureModel(t *testing.T, w int) (model, *[]string) {
	t.Helper()
	old, oldNow := clipboard, captureNow
	clipboard = func(string, bool) error { return nil }
	captureNow = func() time.Time { return time.Date(2026, 11, 17, 14, 52, 0, 0, time.Local) }
	t.Cleanup(func() { clipboard, captureNow = old, oldNow })
	m, _ := pictureModel(t, w, 30, nil, "")
	var commits []string
	m.opts.Commit = func(msg string) error { commits = append(commits, msg); return nil }
	return openTask(t, m, "000002"), &commits
}

// attachedTo says whether task id has the file name, read from disk.
func attachedTo(t *testing.T, m model, id, name string) bool {
	t.Helper()
	tk, err := task.Resolve(m.all, id)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(filepath.Join(m.store.Root, task.FolderOf(tk), name))
	return err == nil
}

// screenFiles is every screen-*.ans on the board.
func screenFiles(t *testing.T, m model) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(m.store.Root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), "screen-") {
			out = append(out, filepath.ToSlash(p))
		}
		return nil
	})
	return out
}

// Check 1: the prompt names where the screen goes and the keys; typing a
// task id (a prefix completes) then Enter attaches it there.
func TestCaptureAttachesToATypedTask(t *testing.T) {
	m, commits := captureModel(t, 120)
	asked := press(m, "S")
	bar := ansi.Strip(asked.View())
	for _, want := range []string{"screen copied (", "Attach this screen to: 000002", "select", "create", "Esc"} {
		if !strings.Contains(bar, want) {
			t.Fatalf("the prompt lacks %q:\n%s", want, bar)
		}
	}
	// The task in view is offered: Enter alone attaches there, as before.
	// Each branch below that writes gets its own board: the models share
	// the store's disk.
	here := press(asked, "enter")
	if !attachedTo(t, here, "000002", "screen-1452.ans") || len(*commits) != 1 || (*commits)[0] != "hug task attach 000002" {
		t.Fatalf("Enter on the offered task: commits %v message %q", *commits, here.message)
	}
	if !strings.Contains(here.message, "attached as screen-1452.ans to 000002") {
		t.Fatalf("footer: %q", here.message)
	}

	// Typing replaces the offered id; a unique prefix is enough.
	m, commits = captureModel(t, 120)
	asked = press(m, "S")
	typed := press(asked, "0", "0", "0", "0", "1", "2")
	if line := ansi.Strip(typed.View()); !strings.Contains(line, "000012") || !strings.Contains(line, "→ ") {
		t.Fatalf("the typed id and its title are not shown:\n%s", line)
	}
	done := press(typed, "enter")
	if !attachedTo(t, done, "000012", "screen-1452.ans") || attachedTo(t, done, "000002", "screen-1452.ans") {
		t.Fatalf("not attached to the typed task: %v %q", screenFiles(t, done), done.message)
	}
	if len(*commits) != 1 || (*commits)[0] != "hug task attach 000012" || !strings.Contains(done.message, "to 000012") {
		t.Fatalf("commits %v message %q", *commits, done.message)
	}
	if done.prompt != "" || done.captured != "" {
		t.Fatalf("the prompt stays open: %q", done.prompt)
	}

	// Tab completes a prefix to the id; an ambiguous one keeps the prompt
	// and says so, attaching nothing.
	m, _ = captureModel(t, 120)
	asked = press(m, "S")
	tab := press(asked, "0", "0", "0", "0", "1", "tab")
	if tab.input.Value() != "000010" {
		t.Fatalf("Tab completed %q", tab.input.Value())
	}
	vague := press(asked, "0", "0", "0", "0", "1", "enter")
	if vague.prompt != capturePrompt || len(screenFiles(t, vague)) != 0 || !strings.Contains(vague.message, "000010") {
		t.Fatalf("an ambiguous prefix: prompt %q files %v message %q", vague.prompt, screenFiles(t, vague), vague.message)
	}
	unknown := press(asked, "z", "z", "enter")
	if unknown.prompt != capturePrompt || len(screenFiles(t, unknown)) != 0 || unknown.message == "" {
		t.Fatalf("an unknown id: prompt %q message %q", unknown.prompt, unknown.message)
	}

	// Esc: nothing attached, the copy kept and said.
	kept := press(asked, "esc")
	if kept.prompt != "" || kept.captured != "" || len(screenFiles(t, kept)) != 0 || !strings.Contains(kept.message, "not attached") {
		t.Fatalf("Esc: prompt %q files %v message %q", kept.prompt, screenFiles(t, kept), kept.message)
	}
}

// With no task in view S still asks where: the field starts empty.
func TestCaptureWithNoTaskInViewAsks(t *testing.T) {
	m, commits := captureModel(t, 120)
	m = press(m, "esc")
	m.setView("overview")
	m.cursor = -1
	asked := press(m, "S")
	if asked.prompt != capturePrompt || asked.input.Value() != "" {
		t.Fatalf("S with no task: prompt %q value %q message %q", asked.prompt, asked.input.Value(), asked.message)
	}
	done := press(asked, "0", "0", "0", "0", "0", "3", "enter")
	if !attachedTo(t, done, "000003", "screen-1452.ans") || len(*commits) != 1 {
		t.Fatalf("not attached: %v %q", *commits, done.message)
	}
}

// Check 2: s picks the task from the rows: the list's own keys move,
// Enter attaches to the row and returns to the task S was pressed on.
func TestCaptureSelectPicksARow(t *testing.T) {
	m, commits := captureModel(t, 120)
	picking := press(m, "S", "s")
	if !picking.picking || picking.detail || picking.prompt != "" {
		t.Fatalf("s did not open the rows to pick: picking %v detail %v prompt %q", picking.picking, picking.detail, picking.prompt)
	}
	if bar := ansi.Strip(picking.View()); !strings.Contains(bar, "pick the task for the screen") {
		t.Fatalf("the footer does not say what Enter does:\n%s", bar)
	}
	// Writers do nothing while picking; they say so.
	for _, k := range []string{"t", "d", "x", "e", "m", "N", "S", "r", "y"} {
		after := press(picking, k)
		if after.confirm != "" || after.prompt != "" || after.grab.active || !after.picking || after.message == "" {
			t.Fatalf("%s while picking: confirm %q prompt %q grab %v picking %v", k, after.confirm, after.prompt, after.grab.active, after.picking)
		}
	}
	moved := press(picking, "j", "j")
	row, ok := moved.selected()
	if !ok {
		t.Fatal("no row selected while picking")
	}
	done := press(moved, "enter")
	if !attachedTo(t, done, row.ID, "screen-1452.ans") || len(*commits) != 1 || (*commits)[0] != "hug task attach "+row.ID {
		t.Fatalf("Enter on %s: commits %v message %q", row.ID, *commits, done.message)
	}
	if done.picking || !done.detail || mustSelected(t, done).ID != "000002" {
		t.Fatalf("not back on 000002: picking %v detail %v", done.picking, done.detail)
	}
	if !strings.Contains(done.message, "attached as screen-1452.ans to "+row.ID) {
		t.Fatalf("footer: %q", done.message)
	}
	// Esc while picking: nothing attached, back where S was pressed. (A
	// fresh board: the attach above is on this one's disk.)
	fresh, _ := captureModel(t, 120)
	back := press(fresh, "S", "s", "j", "esc")
	if back.picking || back.captured != "" || len(screenFiles(t, back)) != 0 || !strings.Contains(back.message, "not attached") || !back.detail || mustSelected(t, back).ID != "000002" {
		t.Fatalf("Esc while picking: picking %v files %v message %q", back.picking, screenFiles(t, back), back.message)
	}
	// The rows' own find and jump work while picking: # then an id finds
	// the task, Enter attaches there. Esc in the jump closes the jump only.
	fresh, commits = captureModel(t, 120)
	jumping := press(fresh, "S", "s", "#", "0", "0", "0", "0", "2", "0")
	if closed := press(jumping, "esc"); !closed.picking || closed.prompt != "" {
		t.Fatalf("Esc in the jump: picking %v prompt %q", closed.picking, closed.prompt)
	}
	jumped := press(jumping, "enter", "enter")
	if !attachedTo(t, jumped, "000020", "screen-1452.ans") || len(*commits) != 1 || jumped.picking {
		t.Fatalf("# while picking: files %v commits %v message %q", screenFiles(t, jumped), *commits, jumped.message)
	}
}

// Check 3: c asks for a title, creates the task with the capture in its
// Files, commits once and opens it.
func TestCaptureCreatesATask(t *testing.T) {
	m, commits := captureModel(t, 120)
	asking := press(m, "S", "c")
	if asking.prompt != captureTitlePrompt {
		t.Fatalf("c opened %q", asking.prompt)
	}
	// Esc on the title goes back to where, the capture kept.
	again := press(asking, "esc")
	if again.prompt != capturePrompt || again.captured == "" {
		t.Fatalf("Esc on the title: prompt %q", again.prompt)
	}
	// An empty title creates nothing.
	empty := press(asking, "enter")
	if empty.prompt != captureTitlePrompt || len(*commits) != 0 {
		t.Fatalf("an empty title: prompt %q commits %v", empty.prompt, *commits)
	}
	done := press(asking, "B", "r", "o", "k", "e", "n", " ", "b", "a", "r", "enter")
	sel := mustSelected(t, done)
	if sel.Title != "Broken bar" || !done.detail || done.prompt != "" {
		t.Fatalf("not opened on the new task: %q detail %v prompt %q", sel.Title, done.detail, done.prompt)
	}
	if !done.hasFile(sel, "screen-1452.ans") || !attachedTo(t, done, sel.ID, "screen-1452.ans") {
		t.Fatalf("the new task has no capture: files %v", done.filesOf(sel))
	}
	if len(*commits) != 1 || (*commits)[0] != "hug task new "+sel.ID {
		t.Fatalf("commits %v", *commits)
	}
	if !strings.Contains(done.message, "created "+sel.ID) || !strings.Contains(done.message, "screen-1452.ans") {
		t.Fatalf("footer: %q", done.message)
	}
	// Its Files list shows the capture.
	if screen := ansi.Strip(done.View()); !strings.Contains(screen, "screen-1452.ans") {
		t.Fatalf("Files does not show the capture:\n%s", screen)
	}
}

// s and c are keys only while the field holds the offered id or nothing:
// once typing, they are text.
func TestCaptureKeysAreTextOnceTyping(t *testing.T) {
	m, _ := captureModel(t, 120)
	typed := press(m, "S", "0", "s", "c")
	if typed.prompt != capturePrompt || typed.picking || typed.input.Value() != "0sc" {
		t.Fatalf("s and c after typing: prompt %q picking %v value %q", typed.prompt, typed.picking, typed.input.Value())
	}
}

// Check 4: the prompt at 80 and 140, as text and with its codes. One
// underline per key letter, the key colour, no background, and the field
// cursor is the one underline that blinks.
func TestCapturePromptGolden(t *testing.T) {
	var golden []string
	for _, w := range []int{80, 140} {
		m, _ := captureModel(t, w)
		m.styles = newTheme()
		m.styles.renderer.SetColorProfile(termenv.TrueColor)
		m.styles.renderer.SetHasDarkBackground(true)
		asked := press(m, "S")
		typed := press(asked, "0", "0", "0", "0", "1", "2")
		title := press(asked, "c")
		golden = append(golden, "== "+strconv.Itoa(w))
		for _, s := range []struct {
			name string
			m    model
		}{{"offered", asked}, {"typed", typed}, {"title", title}} {
			lines := strings.Split(s.m.View(), "\n")
			input, bar := lines[len(lines)-2], lines[len(lines)-1]
			if s.m.pageFramed() {
				input = lines[len(lines)-3] // over the page's bottom rule (4ddcwa)
			}
			golden = append(golden, s.name+": "+strings.TrimRight(ansi.Strip(input), " "), "  "+strconv.Quote(input), "  keys: "+strings.TrimRight(ansi.Strip(bar), " "))
			if strings.Contains(input, "48;") {
				t.Fatalf("%d %s: a background fill: %q", w, s.name, input)
			}
			// No cell bold and underlined at once (qh9ep1 check 9).
			for _, sgr := range strings.Split(input, "\x1b[")[1:] {
				params := strings.Split(sgr[:max(0, strings.IndexByte(sgr, 'm'))], ";")
				if slices.Contains(params, "1") && slices.Contains(params, "4") {
					t.Fatalf("%d %s: bold and underlined at once: %q", w, s.name, input)
				}
			}
			if ansi.StringWidth(ansi.Strip(input)) > w {
				t.Fatalf("%d %s: wider than the screen", w, s.name)
			}
		}
		picking := press(asked, "s")
		lines := strings.Split(picking.View(), "\n")
		golden = append(golden, "picking: "+strings.TrimRight(ansi.Strip(lines[len(lines)-3]), " "), "  keys: "+strings.TrimRight(ansi.Strip(lines[len(lines)-1]), " "))
	}
	goldenFile(t, "testdata/detail/capture-prompt.txt", strings.Join(golden, "\n")+"\n")
}
