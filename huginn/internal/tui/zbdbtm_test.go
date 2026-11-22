package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// zbdbtm: a file a task names opens from the task. Its own TRY IT HERE
// block names four: a plain name of an attached file, a file:NAME link,
// a section in an attached file, and a repo file with a section, read
// from the board's checkout. Two names that are no file (an attachment
// the task lacks, a repo path that does not exist) are no links.
const tryItHere = `TRY IT HERE:
- A plain-text name: open sample.txt from here.
- An attachment link: file:sample-notes.md
- A section in it: sample-notes.md#Section two
- A repo file with a section: docs/DESIGN.md#Personal vs shared memories

Not links: notattached.txt and docs/NOPE.md#Nothing.`

var sampleNotes = "# Sample notes\n\nA short file.\n\n## Section one\n\n" + strings.Repeat("Line of section one.\n\n", 30) + "## Section two\n\nThis is the section a link names.\n" + strings.Repeat("\nMore of section two.\n", 30)

// fileLinkModel opens such a task at 100x30, its Details open, the
// checkout a folder holding docs/DESIGN.md.
func fileLinkModel(t *testing.T) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	r, err := s.New(task.NewOptions{Title: "Main", Goal: "g", Details: tryItHere, Acceptance: []string{"a"}, Top: true})
	if err != nil {
		t.Fatal(err)
	}
	id := r.Task.ID
	if _, _, err := s.Attach(id, "sample.txt", []byte("line 1: sample text file for zbdbtm\nline 2\nline 3\n"), "", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Attach(id, "sample-notes.md", []byte(sampleNotes), "", false); err != nil {
		t.Fatal(err)
	}
	checkout := t.TempDir()
	if err := os.MkdirAll(filepath.Join(checkout, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	design := "# Design\n\n" + strings.Repeat("filler line\n\n", 40) + "## Personal vs shared memories (gav9nw, decided 2026-11-18)\n\nThe section a link names.\n\n" + strings.Repeat("after\n\n", 40)
	if err := os.WriteFile(filepath.Join(checkout, "docs", "DESIGN.md"), []byte(design), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester", Checkout: checkout})
	m.width, m.height = 100, 30
	m.resize()
	m = openID(t, m, id)
	m = m.withUnfolded("details")
	m.syncDetail()
	return m
}

func pageText(m model) string {
	tk, _ := m.selected()
	return ansi.Strip(m.detailText(tk))
}

// The four links of TRY IT HERE, as each is marked when focused.
var tryLinks = []string{"▸sample.txt", "▸sample-notes.md", "▸sample-notes.md#Section two", "▸docs/DESIGN.md#Personal vs shared memories"}

// tabTo presses Tab until the page marks want, at most 40 times.
func tabTo(t *testing.T, m model, want string) model {
	t.Helper()
	for i := 0; i < 40; i++ {
		m = press(m, "tab")
		if marked(pageText(m), want) {
			return m
		}
	}
	t.Fatalf("Tab never marked %q:\n%s", want, pageText(m))
	return m
}

// marked: page shows want marked, and not as the front of a longer link.
func marked(page, want string) bool {
	for _, l := range strings.Split(page, "\n") {
		if i := strings.Index(l, want); i >= 0 {
			rest := l[i+len(want):]
			if !strings.HasPrefix(rest, "#") {
				return true
			}
		}
	}
	return false
}

// zbdbtm check 3: Tab and Shift+Tab stop on every file link, in reading
// order; a name that is no file is no stop.
func TestTabStopsOnEveryFileLink(t *testing.T) {
	for _, dir := range []string{"tab", "shift+tab"} {
		m := fileLinkModel(t)
		var seen []string
		for i := 0; i < 40; i++ {
			m = press(m, dir)
			page := pageText(m)
			for _, w := range tryLinks {
				if marked(page, w) && (len(seen) == 0 || seen[len(seen)-1] != w) {
					seen = append(seen, w)
				}
			}
			for _, no := range []string{"▸notattached.txt", "▸docs/NOPE.md"} {
				if strings.Contains(page, no) {
					t.Fatalf("%s marked %s, a name that is no file:\n%s", dir, no, page)
				}
			}
		}
		for _, w := range tryLinks {
			found := false
			for _, s := range seen {
				found = found || s == w
			}
			if !found {
				t.Errorf("%s never stopped on %s (stopped on %v)", dir, w, seen)
			}
		}
	}
}

// zbdbtm check 1: Enter on a link to an attached file opens it in the
// file frame; Esc comes back to the link, still marked.
func TestEnterOnAttachedFileLinkOpensFrame(t *testing.T) {
	m := tabTo(t, fileLinkModel(t), "▸sample.txt")
	m = press(m, "enter")
	if !m.glance || !strings.Contains(m.glanceHead, "sample.txt") || !strings.Contains(m.viewport.View(), "line 1: sample text file for zbdbtm") {
		t.Fatalf("Enter on sample.txt: glance %v head %q\n%s", m.glance, m.glanceHead, m.viewport.View())
	}
	m = press(m, "esc")
	if m.glance || !marked(pageText(m), "▸sample.txt") {
		t.Fatalf("Esc did not come back to the link: glance %v\n%s", m.glance, pageText(m))
	}
}

// zbdbtm checks 1, 2 and 6: Enter on a section link opens the file
// scrolled to that heading, an attachment's and a repo file's.
func TestEnterOnSectionLinkOpensAtHeading(t *testing.T) {
	for _, c := range []struct{ link, head, heading string }{
		{"▸sample-notes.md#Section two", "sample-notes.md", "Section two"},
		{"▸docs/DESIGN.md#Personal vs shared memories", "docs/DESIGN.md", "Personal vs shared memories"},
	} {
		m := tabTo(t, fileLinkModel(t), c.link)
		m = press(m, "enter")
		top := strings.SplitN(ansi.Strip(m.viewport.View()), "\n", 2)[0]
		if !m.glance || !strings.Contains(m.glanceHead, c.head) || !strings.Contains(top, c.heading) {
			t.Fatalf("Enter on %s: glance %v head %q, first line %q\n%s", c.link, m.glance, m.glanceHead, top, m.viewport.View())
		}
		m = press(m, "esc")
		if m.glance || !marked(pageText(m), c.link) {
			t.Fatalf("Esc from %s did not come back to the link\n%s", c.link, pageText(m))
		}
	}
}

// zbdbtm checks 5 and 6: Space shows the file in line, under the line
// that names it; Space again (or Esc) folds it away; the focus stays.
func TestSpaceShowsFileInLine(t *testing.T) {
	m := tabTo(t, fileLinkModel(t), "▸sample.txt")
	m = press(m, " ")
	page := pageText(m)
	at := strings.Index(page, "▸sample.txt")
	in := strings.Index(page, "line 1: sample text file for zbdbtm")
	next := strings.Index(page, "An attachment link")
	if m.glance || at < 0 || in < at || next < in {
		t.Fatalf("Space on sample.txt: not in line under its line (glance %v):\n%s", m.glance, page)
	}
	m = press(m, "esc")
	if page := pageText(m); strings.Contains(page, "line 1: sample text file") || !marked(page, "▸sample.txt") {
		t.Fatalf("Esc did not fold the file away and keep the link:\n%s", page)
	}
}

// zbdbtm: which words may name a file. A name inside a longer word (a
// URL, an absolute path, task:ID/NAME, the front of history.sdo-to-be)
// is none; a section runs to the end of the line, less a full stop.
func TestFileWords(t *testing.T) {
	for _, c := range []struct {
		text string
		want []string
	}{
		{"open sample.txt from here.", []string{"sample.txt"}},
		{"An attachment link: file:sample-notes.md", []string{"sample-notes.md"}},
		{"A section in it: sample-notes.md#Section two", []string{"sample-notes.md#Section two"}},
		{"see docs/DESIGN.md#Imported data.", []string{"docs/DESIGN.md#Imported data"}},
		{"(before.md, after.md, history.sdo-to-be, cli.txt)", []string{"before.md", "after.md", "cli.txt"}},
		{"https://example.com/a.md and /abs/b.md and task:abc123/c.md", nil},
		// Candidates only: "0.7" and "e.g" are words that might name a
		// file; resolveFile makes them links only when the file exists.
		{"version 0.7 and e.g. nothing", []string{"0.7", "e.g"}},
	} {
		var got []string
		for _, w := range fileWords(c.text) {
			got = append(got, w.shown)
		}
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("fileWords(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}
