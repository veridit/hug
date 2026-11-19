package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/huginn/internal/tui/testfixture"
)

// glanceModel is the seeded board with 000001's folder holding, besides
// bench/README.md and bench/results.tsv, a file with escape sequences and
// a long line, a binary file and one over 1 MiB; reads counts glance reads.
func glanceModel(t *testing.T, w, h int) (model, *[]string) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	root := t.TempDir()
	if err := testfixture.Seed(root); err != nil {
		t.Fatal(err)
	}
	var dir string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && d.Name() == "000001" {
			dir = p
		}
		return nil
	})
	if dir == "" {
		t.Fatal("no 000001 folder in the seed")
	}
	for name, body := range map[string][]byte{
		"evidence/escapes.log": []byte("ok line\n\x1b[2Jcleared?\x07 bell\ttab\r\nnext " + strings.Repeat("long ", 40) + "end\n"),
		"evidence/blob.bin":    {0x7f, 'E', 'L', 'F', 0, 1, 2, 3},
		"evidence/huge.txt":    bytes.Repeat([]byte("x"), 1<<20+1),
	} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var reads []string
	old := readFile
	readFile = func(p string) ([]byte, error) { reads = append(reads, filepath.ToSlash(p)); return os.ReadFile(p) }
	t.Cleanup(func() { readFile = old })
	m := newModel(&task.Store{Root: root}, Options{User: "tester"})
	m.width, m.height = w, h
	m.resize()
	m.setView("list")
	return m, &reads
}

// toFiles focuses Files in the open task.
func toFiles(t *testing.T, m model) model {
	t.Helper()
	for i := 0; m.detailPane != focusFiles; i++ {
		if i > 6 {
			t.Fatal("Files never focused")
		}
		m = press(m, "l")
	}
	return m
}

// scrxak checks 1 and 4: Space on a file shows its text under "path · size
// · i of N"; Esc and u go back to Files with that file marked; opening the
// list and the task read no file, showing one reads only it (goldens at 80
// and 140).
func TestGlanceShowsFile(t *testing.T) {
	var got []string
	for _, w := range []int{80, 140} {
		m, reads := glanceModel(t, w, 24)
		m = openTask(t, m, "000001")
		m = toFiles(t, m)
		if len(*reads) != 0 {
			t.Fatalf("list and detail read files: %v", *reads)
		}
		// Files: bench/README.md, bench/results.tsv, evidence/blob.bin, evidence/escapes.log, evidence/huge.txt
		m = press(m, " ")
		if !m.glance || len(*reads) != 1 || !strings.HasSuffix((*reads)[0], "/000001/bench/README.md") {
			t.Fatalf("Space: glance %v reads %v", m.glance, *reads)
		}
		screen := ansi.Strip(m.View())
		if !strings.Contains(screen, "000001/bench/README.md · 32 B · 1 of 5") || !strings.Contains(screen, "Synthetic bench notes.") {
			t.Fatalf("at %d:\n%s", w, screen)
		}
		got = append(got, "== README at "+strconv.Itoa(w), screen)
		m = press(m, "n", "n", "n") // escapes.log
		screen = ansi.Strip(m.View())
		got = append(got, "== escapes.log at "+strconv.Itoa(w), screen)
		for _, line := range strings.Split(m.View(), "\n") {
			if n := ansi.StringWidth(line); n > w {
				t.Fatalf("at %d a line is %d wide: %q", w, n, line)
			}
		}
		// Back: Files with escapes.log marked; u does the same as Esc.
		back := press(m, "esc")
		sel, _ := back.selected()
		if back.glance || back.detailPane != focusFiles || !strings.Contains(ansi.Strip(back.detailText(sel)), "> ⎘ evidence/escapes.log") {
			t.Fatalf("Esc: glance %v pane %v\n%s", back.glance, back.detailPane, ansi.Strip(back.detailText(sel)))
		}
		if up := press(m, "u"); up.glance || up.detailPane != focusFiles || up.file != back.file {
			t.Fatalf("u: glance %v pane %v file %d", up.glance, up.detailPane, up.file)
		}
		if !back.detail {
			t.Fatal("Esc from the file left the task")
		}
	}
	text := strings.Join(got, "\n") + "\n"
	path := filepath.Join("testdata", "detail", "glance.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestGlanceShowsFile ./huginn/internal/tui/", err)
	}
	if text != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("glance differs from %s:\n%s", path, text)
	}
}

// scrxak check 2: n/p step files and stop at the ends with a note; j/k and
// PgDn/PgUp scroll.
func TestGlanceStepsAndScrolls(t *testing.T) {
	m, _ := glanceModel(t, 80, 7) // the frame shows 3 of escapes.log's 5 lines (z82qsy: the file gets the whole body; yp3c09: and the blank message and input rows)
	m = toFiles(t, openTask(t, m, "000001"))
	m = press(m, " ", "p")
	if m.file != 0 || !strings.Contains(m.message, "first file: 1 of 5") {
		t.Fatalf("p at the first: file %d message %q", m.file, m.message)
	}
	// The footer spells the keys out (Jørgen 00:08: nobody guesses n/p).
	// n and p marked in their words, "file" written once (e3sevr checks
	// 14 and 16, Jørgen 00:44-00:45).
	if bar := ansi.Strip(m.keyBar()); !strings.Contains(bar, "M raw source · next-, previous file · Enter edit") {
		t.Fatalf("file footer: %q", bar)
	}
	m = press(m, "n", "n", "n", "n", "n")
	if m.file != 4 || !strings.Contains(m.message, "last file: 5 of 5") {
		t.Fatalf("n past the last: file %d message %q", m.file, m.message)
	}
	m = press(m, "p") // escapes.log: long enough to scroll at 80x9
	if m.file != 3 || !strings.Contains(m.glanceHead, "evidence/escapes.log") {
		t.Fatalf("p: %d %q", m.file, m.glanceHead)
	}
	if m.viewport.YOffset != 0 {
		t.Fatal("a new file does not start at the top")
	}
	// Longer than the frame (z82qsy: the README now fits): j/k scroll.
	if bar := ansi.Strip(m.keyBar()); !strings.Contains(bar, "j/k scroll · next-, previous file") {
		t.Fatalf("long file footer: %q", bar)
	}
	m = press(m, "j")
	if m.viewport.YOffset != 1 {
		t.Fatalf("j: offset %d", m.viewport.YOffset)
	}
	m = press(m, "k", "pgdown")
	if m.viewport.YOffset == 0 {
		t.Fatal("pgdown did not scroll")
	}
	m = press(m, "pgup")
	if m.viewport.YOffset != 0 {
		t.Fatalf("pgup: offset %d", m.viewport.YOffset)
	}
	// Space in a file edits nothing.
	if m2 := press(m, " "); !strings.Contains(m2.message, "read only") {
		t.Fatalf("Space in a file: %q", m2.message)
	}
}

// scrxak check 3: a binary file and one over 1 MiB show one line, not
// their bytes (the big one is not read at all); ESC and control characters
// are shown escaped.
func TestGlanceNeverPrintsRawBytes(t *testing.T) {
	m, reads := glanceModel(t, 140, 30)
	m = toFiles(t, openTask(t, m, "000001"))
	m = press(m, "j", "j", " ") // blob.bin
	if m.glanceBody != "binary, 8 B: Enter opens it" {
		t.Fatalf("binary: %q", m.glanceBody)
	}
	m = press(m, "n") // escapes.log
	raw := m.View()
	for _, bad := range []string{"\x1b[2J", "\x07", "\r"} {
		if strings.Contains(raw, bad) {
			t.Fatalf("raw %q reached the screen", bad)
		}
	}
	if !strings.Contains(m.glanceBody, `\x1b[2Jcleared?\a bell    tab`) {
		t.Fatalf("escaped:\n%s", m.glanceBody)
	}
	before := len(*reads)
	m = press(m, "n") // huge.txt
	if m.glanceBody != "too large to show here, 1.0 MiB: Enter opens it" || len(*reads) != before {
		t.Fatalf("huge: %q, reads %v", m.glanceBody, (*reads)[before:])
	}
}
