package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/internal/tmuxtest"
)

// The test binary doubles as $VISUAL (as in huginn/internal/cli, 4emme8):
// with HUG_TEST_TUI_EDIT set it appends that text to the file it is given,
// or, given a folder, to every file in it, and exits.
const tuiEdit = "HUG_TEST_TUI_EDIT"

func TestMain(m *testing.M) {
	fakeHug() // j2vwk0: the test binary as a hug that answers "version"
	// No test writes the real clipboard or the terminal (5z4zt4).
	clipboard = func(string, bool) error { return nil }
	// No test opens Preview (33v71x): the opener is a no-op, and every
	// screen names the macOS viewer, so goldens read the same on each OS;
	// TestPictureOpensInTheViewer checks each OS's argv.
	startOpener = func([]string) error { return nil }
	openerOS = "darwin"
	if text, ok := os.LookupEnv(tuiEdit); ok && len(os.Args) > 1 {
		os.Exit(fakeAppend(os.Args[len(os.Args)-1], text))
	}
	// A tmux server left by a test run that died (wacc33) ends here.
	if runtime.GOOS != "windows" {
		if reaped, err := tmuxtest.Reap(); err != nil {
			fmt.Fprintln(os.Stderr, "tmux orphan sweep:", err)
		} else if len(reaped) > 0 {
			fmt.Fprintf(os.Stderr, "tmux orphan sweep: SIGTERM to %v\n", reaped)
		}
	}
	os.Exit(m.Run())
}

func fakeAppend(path, text string) int {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	targets := []string{path}
	if info.IsDir() {
		targets = nil
		filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				targets = append(targets, p)
			}
			return nil
		})
	}
	for _, p := range targets {
		b, err := os.ReadFile(p)
		if err == nil {
			err = os.WriteFile(p, append(b, text...), 0o644)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return 0
}

// useFakeEditor makes the test binary the editor, appending text.
func useFakeEditor(t *testing.T, text string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", exe)
	t.Setenv(tuiEdit, text)
}

// filesModel is the seeded board with 000001 (two files) open.
func filesModel(t *testing.T) (model, *[]string) {
	t.Helper()
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	var commits []string
	m.opts.Commit = func(msg string) error { commits = append(commits, msg); return nil }
	return m, &commits
}

func openTask(t *testing.T, m model, id string) model {
	t.Helper()
	m = press(m, "#", id, "enter")
	if sel, ok := m.selected(); !ok || sel.ID != id || !m.detail {
		t.Fatalf("could not open %s: %q", id, m.message)
	}
	return m
}

// runEditorCmd runs the editor command the model started and delivers the
// result as Bubble Tea would.
func runEditorCmd(t *testing.T, m model) model {
	t.Helper()
	if m.lastExec == nil {
		t.Fatalf("no editor started: %q", m.message)
	}
	cmd := m.lastExec
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fake editor: %v\n%s", err, out)
	}
	sel, _ := m.selected()
	if m.lastEdit.path != sel.Path {
		t.Fatalf("editor reports %q, the open task is %q", m.lastEdit.path, sel.Path)
	}
	next, _ := m.Update(m.lastEdit)
	return next.(model)
}

// zgr01t check 1 (model level; the goldens show it on screen): 000001 has
// two files and shows ⎘2 in overview, list and board; no other row has a
// mark.
func TestFileMarkOnRows(t *testing.T) {
	m, _ := filesModel(t)
	for _, view := range []string{"overview", "list", "board"} {
		m.setView(view)
		text := ansi.Strip(m.View())
		marked := 0
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, fileMark) {
				marked++
				if !strings.Contains(line, "000001") || !strings.Contains(line, fileMark+"2 ") {
					t.Fatalf("%s: mark on the wrong row: %q", view, line)
				}
			}
		}
		if marked == 0 {
			t.Fatalf("%s: no %s2 mark:\n%s", view, fileMark, text)
		}
	}
}

// zgr01t check 2: the files are listed inside the task; Enter on one opens
// it in $EDITOR and the change is committed when the editor closes.
func TestFilesOpenInEditorAndCommit(t *testing.T) {
	m, commits := filesModel(t)
	m.width = 160 // the whole key bar, Edit files included (e3sevr: E marked in its word)
	m.resize()
	m = openTask(t, m, "000001")
	sel0, _ := m.selected()
	// The whole page: FILES sits below the first screen once Purpose,
	// Principle and Details come first (jnaspy).
	text := ansi.Strip(m.detailText(sel0)) + "\n" + ansi.Strip(m.keyBar())
	for _, want := range []string{"FILES (2)", "⎘ bench/README.md", "⎘ bench/results.tsv", "Edit files"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	for m.detailPane != focusFiles {
		m = press(m, "l") // sections are h/l; Tab steps task ids (4bwcgp)
	}
	m = press(m, "j") // bench/results.tsv
	if !strings.Contains(ansi.Strip(m.detailText(sel0)), "> ⎘ bench/results.tsv") {
		t.Fatalf("j did not mark the second file:\n%s", ansi.Strip(m.detailText(sel0)))
	}
	useFakeEditor(t, "2\t40\n")
	m, _ = key(m, "enter")
	m = runEditorCmd(t, m)
	sel, _ := m.selected()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(filepath.Join(m.store.Root, sel.Path)), sel.ID, "bench", "results.tsv"))
	if err != nil || !strings.HasSuffix(string(b), "1\t42\n2\t40\n") {
		t.Fatalf("the editor's change is not in the file: %q %v", b, err)
	}
	if len(*commits) != 1 || (*commits)[0] != "hug task edit 000001 file bench/results.tsv" {
		t.Fatalf("commits %q", *commits)
	}
	if m.message != "editor closed; re-linted" {
		t.Fatalf("message %q", m.message)
	}
}

// zgr01t check 3: E opens the task's folder in the editor; a task without
// files does not show the key and E there only says so.
func TestFolderKeyOnlyWithFiles(t *testing.T) {
	m, commits := filesModel(t)
	m = openTask(t, m, "000001")
	useFakeEditor(t, "edited\n")
	m, _ = key(m, "E")
	if m.lastExec == nil {
		t.Fatalf("E started no editor: %q", m.message)
	}
	folder := m.lastExec.Args[len(m.lastExec.Args)-1]
	if fi, err := os.Stat(folder); err != nil || !fi.IsDir() || filepath.Base(folder) != "000001" {
		t.Fatalf("E opened %q, not the task folder", folder)
	}
	m = runEditorCmd(t, m)
	if len(*commits) != 1 || (*commits)[0] != "hug task edit 000001 files" {
		t.Fatalf("commits %q", *commits)
	}
	m, _ = filesModel(t)
	m = openTask(t, m, "000002")
	text := ansi.Strip(m.View())
	if strings.Contains(text, "Edit files") || strings.Contains(text, "FILES") {
		t.Fatalf("a task without files shows the files key or section:\n%s", text)
	}
	m, cmd := key(m, "E")
	if cmd != nil || m.lastExec != nil || !strings.Contains(m.message, "no files") {
		t.Fatalf("E without files: cmd %v message %q", cmd != nil, m.message)
	}
}
