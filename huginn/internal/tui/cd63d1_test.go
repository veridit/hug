package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// cd63d1 (ox 2026-11-22, 8a0ybb frame order-b): a request's text on a
// task's page is indented 3, and its wrapped lines started at column 0,
// so the request blurred into the page. The indent was put before the
// page wrapped (answerbox.go, d75c1691). One rule now for every indented
// block on the page: indent after wrapping. The fixture (testdata/cd63d1)
// has each such block long enough to wrap at 100 columns: a request, its
// account (└ crab:), a second request, a dependency's note and a wait in
// Depends on, a child, and the Questions record with its answer.

// hangBoard is cd63d1's board: cd63d1 and the task it waits on, at w x h,
// cd63d1's page open.
func hangBoard(t *testing.T, w, h int) model {
	t.Helper()
	m := fileBoard(t, "testdata/cd63d1/cd63d1.md", "cd63d1", w, h)
	b, err := os.ReadFile("testdata/cd63d1/w0rap1.md")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(m.store.Root, "w")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "w0rap1.md"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	return openTask(t, m, "cd63d1")
}

// hangLeft lists the page's lines that start left of where they hang:
// in DEPENDENCIES, CHILDREN, LINKS and FILES a row's wrapped lines under
// its text (4); in REQUESTS and QUESTIONS a text line and its wrapped
// lines (3), an account or answer under "└ " (5). Headings and a
// request's or question's head ("1. jhf · ...") start at column 0.
func hangLeft(page string) []string {
	head := regexp.MustCompile(`^\d+\. `)
	var bad []string
	section, need := "", 0
	for _, l := range strings.Split(page, "\n") {
		if strings.TrimSpace(l) == "" {
			need = 0
			continue
		}
		if name := strings.TrimLeft(l, "▸⌄› "); l == strings.TrimLeft(l, " ") && !head.MatchString(l) && name != "" && strings.ToUpper(name) == name {
			section, need = strings.Fields(name)[0], 0
			continue
		}
		lead := len(l) - len(strings.TrimLeft(l, " "))
		switch section {
		case "DEPENDENCIES", "CHILDREN", "LINKS", "FILES":
			if row := strings.TrimLeft(l, "> "); lead <= 2 && (strings.HasPrefix(row, "↳ ") || strings.HasPrefix(row, "⧗ ") || strings.HasPrefix(row, "⎘ ")) {
				need = 4
				continue
			}
		case "REQUESTS", "QUESTIONS":
			switch {
			case head.MatchString(l):
				need = 3
				continue
			case strings.HasPrefix(l, "   └ "):
				need = 5
				continue
			case lead == 3 && need == 5:
				need = 3 // the next line of the text
				continue
			}
		default:
			continue
		}
		if lead < need {
			bad = append(bad, section+": "+l)
		}
	}
	return bad
}

// cd63d1 check 1: every wrapped line of an indented block keeps its
// indent at 100 columns: a request's text 3, its account under "└ " 5,
// as the Questions record draws a question and its answer. Golden: the
// page at 100x30, opened, at its Requests, and the record opened.
func TestIndentedBlocksHangWhenWrapped(t *testing.T) {
	m := hangBoard(t, 100, 30)
	m = m.withUnfolded("questions")
	page := detailPlain(m)
	// Each block wraps at 100 (the fixture is written so): a request
	// that did not wrap would prove nothing.
	for _, s := range []string{"other task (w0rap1)?", "columns, twice.", "wide.", "this note is on", "enough to wrap on a page a hundred columns wide."} {
		if !strings.Contains(page, s) {
			t.Fatalf("no %q on the page:\n%s", s, page)
		}
	}
	if bad := hangLeft(page); len(bad) > 0 {
		t.Errorf("%d wrapped lines start left of their block's indent:\n%s\npage:\n%s", len(bad), strings.Join(bad, "\n"), page)
	}
	end := press(m, ">")
	goldenFile(t, "testdata/cd63d1/page-100x30.txt",
		"== page\n"+page+"\n== at the end\n"+ansi.Strip(end.View())+"\n")
}

// cd63d1: the id Tab is on in a wrapped request is marked where it
// stands, ▸ before it, on a line that still hangs under 3 and fits the
// page; the account's lines too. The mark is put before the wrap, so its
// column is counted.
func TestMarkedIDInWrappedRequestHangs(t *testing.T) {
	m := hangBoard(t, 100, 30)
	reached := false
	for range 40 {
		m = press(m, "tab")
		if s, ok := m.markedStop(mustSelected(t, m)); ok && s.where == stopRequest && s.shown == "w0rap1" {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatal("Tab never reaches w0rap1 in request 1's text")
	}
	page := detailPlain(m)
	var at string
	for _, l := range strings.Split(page, "\n") {
		if ansi.StringWidth(l) > 100 {
			t.Errorf("a line is wider than the page (%d): %q", ansi.StringWidth(l), l)
		}
		if strings.Contains(l, "▸w0rap1") {
			at = l
		}
	}
	if !strings.HasPrefix(at, "   ") || !strings.Contains(at, "(▸w0rap1)?") {
		t.Fatalf("w0rap1 is not marked where it stands in request 1, hung under 3: %q\n%s", at, page)
	}
	if bad := hangLeft(page); len(bad) > 0 {
		t.Errorf("marked, %d wrapped lines start left of their block's indent:\n%s", len(bad), strings.Join(bad, "\n"))
	}
}
