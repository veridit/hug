package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// 8a0ybb (Jørgen 2026-11-20, requests 1 and 2): a task's page reads short
// to long. Purpose, Principle, the Goal under GOAL (no DESCRIPTION
// heading), Acceptance, the short sections, then the long ones, Details,
// the Questions record, Notes and History, each folded to one line with
// its count until Enter on its heading opens it. The fixture is 8a0ybb's
// own file as it was when the check 12 frames were taken (2026-11-22
// 05:36), with its old "## Relevant details" heading.

// fileBoard is a board holding one task file, copied from testdata, at
// w x h with the clock pinned, its page open.
func fileBoard(t *testing.T, file, id string, w, h int) model {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	old, oldTask := now, task.Now
	at := func() time.Time { return time.Date(2026, 11, 22, 5, 36, 0, 0, time.Local) }
	now, task.Now = at, at
	t.Cleanup(func() { now, task.Now = old, oldTask })
	s := &task.Store{Root: t.TempDir()}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(s.Root, id[:1])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".md"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "jhf"})
	m.width, m.height = w, h
	m.resize()
	return openTask(t, m, id)
}

// headings are the page's section headings in order: each line that is
// a heading as sectionHeading or foldLine draws it, its sign dropped.
func headings(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(l, "▸ "), "› "), "⌄ ")
		name, _, _ := strings.Cut(l, " · ")
		name = strings.TrimSpace(name)
		if name == "" || strings.ToUpper(name) != name || strings.ContainsAny(name[:1], "0123456789>☐☑☒─↑↓") {
			continue
		}
		out = append(out, name)
	}
	return out
}

// 8a0ybb checks 11 and 13: the order, no DESCRIPTION, Relevant details
// headed DETAILS, the long sections folded to one line each with their
// count, none of their text on the page. Golden: the screen as it opens
// and at its end, at 100x30 and 160x45.
func TestPageReadsShortToLong(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		t.Run(fmt.Sprintf("%dx%d", w, h), func(t *testing.T) {
			m := fileBoard(t, "testdata/8a0ybb/8a0ybb.md", "8a0ybb", w, h)
			text := detailPlain(m)
			want := "PURPOSE|PRINCIPLE|GOAL|ACCEPTANCE|REQUESTS|DETAILS (5)|QUESTIONS (3)|NOTES (27)|HISTORY (22)"
			if got := strings.Join(headings(text), "|"); got != want {
				t.Fatalf("headings\n %s\nwant\n %s\n%s", got, want, text)
			}
			// The task's own text says "DESCRIPTION heading" (check 13): no
			// line is that heading.
			for _, l := range strings.Split(text, "\n") {
				if h := strings.TrimPrefix(l, "▸ "); strings.HasPrefix(h, "DESCRIPTION") || strings.HasPrefix(h, "RELEVANT DETAILS") {
					t.Fatalf("an old heading is on the page: %q\n%s", l, text)
				}
			}
			for _, folded := range []string{
				"Headings (Jørgen 2026-11-20 10:26Z on ddpxs8)", // Details
				"1. crab → ",                          // Questions record
				"crab 01:21Z: reviewed the mock",      // Notes
				"handed to crab: bug's session ended", // History
			} {
				if strings.Contains(text, folded) {
					t.Fatalf("folded text %q is on the page:\n%s", folded, text)
				}
			}
			for _, line := range []string{"› DETAILS (5)", "› QUESTIONS (3)", "› NOTES (27)", "› HISTORY (22)"} {
				if !strings.Contains(text, "\n"+line+"\n") && !strings.HasSuffix(text, "\n"+line) {
					t.Fatalf("no folded line %q:\n%s", line, text)
				}
			}
			end := press(m, ">")
			goldenFile(t, fmt.Sprintf("testdata/8a0ybb/page-%dx%d.txt", w, h),
				"== opens\n"+ansi.Strip(m.View())+"\n== at the end\n"+ansi.Strip(end.View())+"\n")
		})
	}
}

// 8a0ybb check 11: Shift-Tab reaches a folded heading, which shows ▸ and
// what Enter does; Enter opens it, the page shown from its heading, its
// text on the page; Enter again folds it. Each long section opens alone.
func TestEnterOpensAFoldedSection(t *testing.T) {
	m := fileBoard(t, "testdata/8a0ybb/8a0ybb.md", "8a0ybb", 100, 30)
	// text: a line of the section, folded never on the page; first: its
	// first line, on screen once opened.
	for _, c := range []struct{ head, text, first string }{
		{"NOTES (27)", "crab 01:21Z: reviewed the mock", "file:a-opens-100x30.ans (3882 bytes"},
		{"HISTORY (22)", "handed to crab: bug's session ended", "todo -> todo/design (ask)"},
		{"DETAILS (5)", "Headings (Jørgen 2026-11-20 10:26Z on ddpxs8)", "Headings (Jørgen 2026-11-20 10:26Z on ddpxs8)"},
		{"QUESTIONS (3)", "1. crab → ", "1. crab → "},
	} {
		v := m
		for n := 0; n < 80 && !strings.Contains(detailPlain(v), "▸ "+c.head); n++ {
			v = press(v, "shift+tab")
		}
		if !strings.Contains(detailPlain(v), "▸ "+c.head+" · Enter opens") {
			t.Fatalf("Shift-Tab never focused %s:\n%s", c.head, detailPlain(v))
		}
		v = press(v, "enter")
		text := detailPlain(v)
		if !strings.Contains(text, c.text) || !strings.Contains(text, "▸ "+c.head+" · Enter folds") {
			t.Fatalf("Enter did not open %s:\n%s", c.head, text)
		}
		lines := strings.Split(text, "\n")
		if v.viewport.YOffset >= len(lines) || !strings.HasPrefix(lines[v.viewport.YOffset], "▸ "+c.head) {
			t.Fatalf("opened %s: the page does not start at its heading (offset %d)", c.head, v.viewport.YOffset)
		}
		if !strings.Contains(ansi.Strip(v.View()), c.first) {
			t.Fatalf("opened %s: its text is not on screen:\n%s", c.head, ansi.Strip(v.View()))
		}
		for _, other := range []string{"crab 01:21Z: reviewed the mock", "handed to crab: bug's session ended", "Headings (Jørgen 2026-11-20 10:26Z on ddpxs8)", "1. crab → "} {
			if other != c.text && strings.Contains(text, other) {
				t.Fatalf("opening %s opened %q too", c.head, other)
			}
		}
		v = press(v, "enter")
		if text := detailPlain(v); strings.Contains(text, c.text) || !strings.Contains(text, "▸ "+c.head+" · Enter opens") {
			t.Fatalf("Enter again did not fold %s:\n%s", c.head, text)
		}
	}
	// The Notes opened: one entry a paragraph, as the count says.
	v := m
	for n := 0; n < 80 && !strings.Contains(detailPlain(v), "▸ NOTES"); n++ {
		v = press(v, "shift+tab")
	}
	v = press(v, "enter")
	goldenFile(t, "testdata/8a0ybb/notes-open-100x30.txt", ansi.Strip(v.View())+"\n")
	// A new page opens folded again.
	if v = press(v, "esc", "#", "8a0ybb", "enter"); strings.Contains(detailPlain(v), "crab 01:21Z: reviewed the mock") {
		t.Fatalf("reopened, the Notes are still open:\n%s", detailPlain(v))
	}
}

// 8a0ybb check 13: a task hug task new writes has "## Details", and its
// page heads it DETAILS, as the old "## Relevant details" reads.
func TestNewTaskDetailsHeadedDetails(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	r, err := s.New(task.NewOptions{Title: "Fresh", Goal: "Aim.", Purpose: "Why.", Principle: "How.", Details: "Facts that matter.", Acceptance: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(r.Task.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "\n## Details\nFacts that matter.\n") || strings.Contains(string(b), "Relevant details") {
		t.Fatalf("new task file:\n%s", b)
	}
	m := newModel(s, Options{User: "jhf"})
	m = openTask(t, m, r.Task.ID)
	text := detailPlain(m)
	if got := strings.Join(headings(text), "|"); got != "PURPOSE|PRINCIPLE|GOAL|ACCEPTANCE|DETAILS (1)" {
		t.Fatalf("headings %s:\n%s", got, text)
	}
	for n := 0; n < 10 && !strings.Contains(detailPlain(m), "▸ DETAILS"); n++ {
		m = press(m, "tab")
	}
	if m = press(m, "enter"); !strings.Contains(detailPlain(m), "Facts that matter.") {
		t.Fatalf("Enter on DETAILS did not open it:\n%s", detailPlain(m))
	}
}

// 8a0ybb: a find reaches a folded section too. / with text that is only
// in the Notes opens the Notes and steps to the match; the Details, with
// no match, stay folded.
func TestFindOpensTheFoldHoldingIt(t *testing.T) {
	m := fileBoard(t, "testdata/8a0ybb/8a0ybb.md", "8a0ybb", 100, 30)
	m = typeFind(m, "machine load over 10")
	text := detailPlain(m)
	if len(m.find.hits) == 0 || !strings.Contains(text, "⌄ NOTES (27)") || !strings.Contains(text, "› DETAILS (5)") {
		t.Fatalf("find did not open the Notes alone (hits %d):\n%s", len(m.find.hits), text)
	}
	if h := m.find.hits[m.find.at]; h.line < m.viewport.YOffset || h.line >= m.viewport.YOffset+m.viewport.Height {
		t.Fatalf("the match is off the page: line %d, page %d+%d", h.line, m.viewport.YOffset, m.viewport.Height)
	}
}
