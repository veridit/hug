package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// withdrawnBoard is askedBoard with cccccc's question withdrawn (hug task
// resume) and asked again, corrected: question 1 withdrawn, question 2
// open (hpqt99 2026-11-18).
func withdrawnBoard(t *testing.T) model {
	t.Helper()
	m := askedBoard(t)
	if _, err := m.store.Resume("cccccc"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.store.Ask("cccccc", task.Question{What: "should the ASKED title take the warning colour?", For: "jhf"}, "nautilus"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	return m
}

// nanvnt check 2: the detail draws a withdrawn question dim, its head and
// every line of its text, with "withdrawn <at>"; it is never a block with
// an answer action, and the open question asked again is.
func TestWithdrawnQuestionDrawnDim(t *testing.T) {
	m := withdrawnBoard(t)
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.width, m.height = 100, 30
	m.resize()
	m = openTask(t, m, "cccccc")
	tk := mustSelected(t, m)
	if open := task.OpenQuestions(tk); len(open) != 1 || open[0].N != 2 {
		t.Fatalf("open questions %+v, want question 2 alone", open)
	}
	// The withdrawn question is in the Questions record, a long section:
	// opened here (8a0ybb), folded it is QUESTIONS (1).
	if folded := ansi.Strip(m.detailText(tk)); !strings.Contains(folded, "› QUESTIONS (1)") || strings.Contains(folded, "withdrawn 2026-11-17") {
		t.Fatalf("the record is not folded to QUESTIONS (1):\n%s", folded)
	}
	m = m.withUnfolded("questions")
	color := m.detailText(tk)
	page := ansi.Strip(color)
	if !strings.Contains(page, "1. nautilus → you · 3 min ago · withdrawn 2026-11-17 09:50") {
		t.Fatalf("question 1 not marked withdrawn with its time:\n%s", page)
	}
	if !strings.Contains(page, "? should the ASKED title take the warning colour?") || answerActions(page) != 1 {
		t.Fatalf("question 2 is not the one open block with its answer:\n%s", page)
	}
	// The record's own lines, wrapped and hung under 3 (cd63d1): every
	// line of the withdrawn text dim.
	record := m.questionRecord(tk, m.marked(tk))
	for _, l := range []string{
		"   which colour should the ASKED title take on a light terminal, given that the warning colour",
		"   already marks lint findings and the rows there are long?",
		"   And should it differ under NO_COLOR?",
	} {
		dim := m.styles.dim.Render(l)
		found := false
		for _, r := range record {
			found = found || r == dim
		}
		if dim == l || !found {
			t.Fatalf("withdrawn text line not dim: %q\n%q", l, record)
		}
	}
}

// nanvnt check 3: ASKED neither lists nor counts a withdrawn question,
// also when the task holds a second open one: cccccc counts once, for
// question 2, and a task whose only question is withdrawn is not in
// ASKED at all.
func TestAskedSkipsWithdrawnQuestion(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := withdrawnBoard(t)
	if n := m.countForYou(); n != 3 {
		t.Fatalf("for you %d, want 3 (aaaaaa, cccccc's question 2, eeeeee)", n)
	}
	view := focusAt(t, m, 100, "cccccc")
	asked := section(view, "ASKED")
	if !strings.Contains(asked, "ASKED · for you  [3]") || strings.Count(asked, "cccccc") != 1 {
		t.Fatalf("ASKED does not count cccccc once:\n%s", view)
	}
	if strings.Contains(view, "which colour should the ASKED title") && strings.Contains(view, "─ asked: which colour") {
		t.Fatalf("ASKED shows the withdrawn question:\n%s", view)
	}
	// Withdraw aaaaaa's only question: it leaves ASKED.
	if _, err := m.store.Resume("aaaaaa"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	if n := m.countForYou(); n != 2 {
		t.Fatalf("for you %d after withdrawing aaaaaa's only question, want 2", n)
	}
	if asked := section(focusAt(t, m, 100, "cccccc"), "ASKED"); strings.Contains(asked, "aaaaaa") {
		t.Fatalf("aaaaaa's withdrawn question is still in ASKED:\n%s", asked)
	}
}
