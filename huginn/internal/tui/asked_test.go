package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// askedBoard is stageBoard with aaaaaa asked of jhf about check 2, and
// cccccc asked of jhf with no check and a long, two-line question.
func askedBoard(t *testing.T) model {
	t.Helper()
	m := stageBoard(t)
	m.opts.User = "jhf"
	if _, err := m.store.Ask("aaaaaa", task.Question{What: "look at the state/stage rows on your board", For: "jhf", Check: 2}, "nautilus"); err != nil {
		t.Fatal(err)
	}
	long := "which colour should the ASKED title take on a light terminal, given that the warning colour already marks lint findings and the rows there are long?\nAnd should it differ under NO_COLOR?"
	if _, err := m.store.Ask("cccccc", task.Question{What: long, For: "jhf"}, "nautilus"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	return m
}

// detailTop is the header and the page's first n lines, plain.
func detailTop(m model, n int) string {
	tk, _ := m.selected()
	page := strings.Split(m.viewport.View(), "\n")
	return ansi.Strip(strings.Join(append(m.detailHeader(tk), page[:min(n, len(page))]...), "\n"))
}

// 0gztsy check 1: a question that names check N opens with focus on check
// N, which carries the ? marker and the same box, number and text as the
// list; the question shows there, once, and in no block of its own.
// Check 3 (crab 01:06): the header says the stage in one word, no text.
func TestAskedCheckIsFocusedOnOpen(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m := askedBoard(t)
		m.width, m.height = w, 30
		m.resize()
		m = openTask(t, m, "aaaaaa")
		if m.detailPane != focusAcceptance || m.acceptance != 1 {
			t.Fatalf("at %d: focus pane %d check %d, want the checks, check 2", w, m.detailPane, m.acceptance+1)
		}
		screen := ansi.Strip(m.View())
		if n := strings.Count(screen, "look at the state/stage rows"); n != 1 {
			t.Fatalf("at %d the question shows %d times:\n%s", w, n, screen)
		}
		if !strings.Contains(screen, "> ? ☐ 2. b") || !strings.Contains(screen, "  ☑ 1. a") {
			t.Fatalf("at %d check 2 does not carry ? and the focus:\n%s", w, screen)
		}
		head := ansi.Strip(strings.Join(m.detailHeader(mustSelected(t, m)), "\n"))
		if !strings.Contains(head, "[asked]") || strings.Contains(head, "look at") || strings.Contains(head, "└─") {
			t.Fatalf("at %d the header holds more than the stage word:\n%s", w, head)
		}
		// Space ticks it; a answers it.
		if after := press(m, " "); after.confirm != " " {
			t.Fatalf("at %d Space on the asked check: confirm %q message %q", w, after.confirm, after.message)
		}
		if after := press(m, "a"); after.prompt != answerPrompt {
			t.Fatalf("at %d a on the asked check: prompt %q", w, after.prompt)
		}
		golden = append(golden, "== aaaaaa at "+strconv.Itoa(w), screen)
	}
	goldenFile(t, "testdata/detail/asked-check.txt", strings.Join(golden, "\n")+"\n")
}

// 0gztsy check 2: a question that names no check is one block at the top
// of the details, styled like a check row, focused on open; it shows
// whole, wrapped, never cut (crab 01:06); Space says a answers it, and a
// answers.
func TestAskedWithoutCheckIsOneBlock(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m := askedBoard(t)
		m.width, m.height = w, 30
		m.resize()
		m = openTask(t, m, "cccccc")
		if m.detailPane != focusQuestion {
			t.Fatalf("at %d: focus pane %d, want the question", w, m.detailPane)
		}
		top := detailTop(m, 8)
		flat := strings.Join(strings.Fields(top), " ")
		for _, want := range []string{"> ? which colour should the ASKED title", "the rows there are long?", "And should it differ under NO_COLOR?", "└ nautilus · 3 min ago · answer"} {
			if !strings.Contains(flat, want) {
				t.Fatalf("at %d the block lacks %q:\n%s", w, want, top)
			}
		}
		if strings.Contains(top, "…") {
			t.Fatalf("at %d the question is cut:\n%s", w, top)
		}
		if strings.Contains(ansi.Strip(m.View()), "? ☐") {
			t.Fatalf("at %d a check carries ? for a question that names none", w)
		}
		if after := press(m, " "); !strings.Contains(after.message, "a answers it") || after.confirm != "" {
			t.Fatalf("at %d Space on the question: confirm %q message %q", w, after.confirm, after.message)
		}
		if after := press(m, "a"); after.prompt != answerPrompt {
			t.Fatalf("at %d a on the question: prompt %q", w, after.prompt)
		}
		if bar := ansi.Strip(m.keyBar()); !strings.HasPrefix(bar, "answer") {
			t.Fatalf("at %d the bar does not lead with answer: %q", w, bar)
		}
		golden = append(golden, "== cccccc at "+strconv.Itoa(w), top)
	}
	goldenFile(t, "testdata/detail/asked-block.txt", strings.Join(golden, "\n")+"\n")
}

// 41hhgn check 8 (TUI): the answer box keeps lines. Alt-Enter (and ctrl-j)
// add a line, a paste keeps its lines, Enter sends; the box grows with
// them; History holds every line and the asker's notice prints them all.
func TestAnswerBoxKeepsLines(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := askedBoard(t)
	m.width, m.height = 100, 30
	m.resize()
	m = openTask(t, m, "cccccc")
	m = press(m, "a")
	typeIn := func(s string) {
		for _, r := range s {
			m = press(m, string(r))
		}
	}
	typeIn(`he said: "keep it"`)
	m, _ = key(m, "alt+enter")
	typeIn("second: yes")
	// A paste of two lines, with a CRLF, as a terminal hands it over.
	next, _ := m.Update(pasteKey("pasted one\r\npasted: two"))
	m = next.(model)
	want := "he said: \"keep it\"\nsecond: yes" + "pasted one\npasted: two"
	if got := m.answerBox.Value(); got != want {
		t.Fatalf("box holds %q, want %q", got, want)
	}
	if h := m.answerBox.Height(); h < 3 {
		t.Fatalf("the box did not grow: height %d", h)
	}
	m, _ = key(m, "ctrl+j")
	typeIn("last")
	top := detailTop(m, 12)
	for _, l := range []string{`answer: he said: "keep it"`, "second: yespasted one", "pasted: two", "last"} {
		if !strings.Contains(top, l) {
			t.Fatalf("box lacks %q:\n%s", l, top)
		}
	}
	m = press(m, "enter")
	if m.prompt != "" || !strings.HasPrefix(m.message, "answer ") {
		t.Fatalf("Enter did not send: prompt %q message %q", m.prompt, m.message)
	}
	tk := mustSelected(t, m)
	b, _ := os.ReadFile(filepath.Join(m.store.Root, tk.Path))
	for _, l := range []string{"  he said: \"keep it\"\n", "  second: yespasted one\n", "  pasted: two\n", "  last\n"} {
		if !strings.Contains(string(b), l) {
			t.Fatalf("History lacks %q:\n%s", l, b)
		}
	}
	// The asker reads every line.
	m.opts.User = "nautilus"
	m.refresh("")
	m = openTask(t, m, "cccccc")
	head := ansi.Strip(strings.Join(m.detailHeader(mustSelected(t, m)), "\n"))
	for _, l := range []string{`✓ jhf answered asked: he said: "keep it"`, "  second: yespasted one", "  pasted: two", "  last", "  unread by nautilus"} {
		if !strings.Contains(head, l) {
			t.Fatalf("notice lacks %q:\n%s", l, head)
		}
	}
}
