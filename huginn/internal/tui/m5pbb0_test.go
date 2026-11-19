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

// m5pbb0 (Jørgen 2026-11-19 09:05Z on p87ngv: "I can't get the second
// question here"). p87ngv asked jhf three questions (q2adr8). His TUI, a
// v0.7.69 left running, answered question 1 the old way: the answer went
// to History only, design_* went, and the task fell to ready with all
// three questions still open in ## Questions. Today's TUI draws each of
// them with its own a answer (a todo task's open questions are for the
// person, yphbra), but the store refused every answer: "p87ngv asks
// nobody: its stage is not design, asked or review". Every open question
// in that file must have an answer action that answers it. The fixture is
// p87ngv as committed at 2026-11-19T10:05:26 (hug/tasks bab152ff), the
// state he wrote his request against; golden at 100x30 and 160x45.
func TestOldAnswerLeavesEveryQuestionAnswerable(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		name := fmt.Sprintf("%dx%d", w, h)
		t.Run(name, func(t *testing.T) {
			m := p87ngvBoard(t, "testdata/p87ngv/old-answer.md", w, h)
			tk := mustSelected(t, m)
			page := ansi.Strip(m.detailText(tk))
			if n := answerActions(page); n != 3 {
				t.Fatalf("%d answer actions for 3 open questions:\n%s", n, page)
			}
			golden := []string{"== opened", ansi.Strip(m.View())}
			// Tab to question 2's block; a answers it.
			m = press(m, "tab")
			if m.detailPane != focusQuestion || m.question != 1 {
				t.Fatalf("tab: pane %d question %d, want question block 2", m.detailPane, m.question)
			}
			m = press(m, "a")
			for _, r := range "both may resolve" {
				m = press(m, string(r))
			}
			golden = append(golden, "== typing under question 2", ansi.Strip(m.View()))
			m = press(m, "enter")
			b, _ := os.ReadFile(filepath.Join(m.store.Root, "p", "p87ngv.md"))
			b = []byte(strings.ReplaceAll(string(b), "\r\n", "\n")) // a Windows checkout of the fixture
			tk = mustSelected(t, m)
			if open := task.OpenQuestions(tk); len(open) != 2 || open[0].N != 1 || open[1].N != 3 || !strings.Contains(string(b), "  Who may resolve a thread: only the person who opened it, or also the agent once it has acted on it?\n  answered 2026-11-19T10:06 by jhf:\n    both may resolve\n") {
				t.Fatalf("question 2 not answered (message %q):\n%s", m.message, b)
			}
			after := ansi.Strip(m.View())
			if afterPage := ansi.Strip(m.detailText(tk)); answerActions(afterPage) != 2 || !strings.Contains(after, "✓ jhf answered question 2: both may resolve") {
				t.Fatalf("after answering 2 (message %q):\n%s", m.message, afterPage)
			}
			golden = append(golden, "== question 2 answered", after)
			goldenFile(t, "testdata/detail/p87ngv-old-answer-"+name+".txt", strings.Join(golden, "\n")+"\n")
		})
	}
}

// m5pbb0 check 2: p87ngv as committed at 2026-11-19T10:05:47 (hug/tasks
// 19977d7d, two seconds before Jørgen's request): five questions open to
// jhf, four struck checks. Each has its own answer action, each is a Tab
// stop, and a on each answers that one alone. Golden at 100x30 and 160x45.
func TestFiveQuestionsEachReachable(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		name := fmt.Sprintf("%dx%d", w, h)
		t.Run(name, func(t *testing.T) {
			m := p87ngvBoard(t, "testdata/p87ngv/five-questions.md", w, h)
			tk := mustSelected(t, m)
			page := ansi.Strip(m.detailText(tk))
			if n := answerActions(page); n != 5 {
				t.Fatalf("%d answer actions for 5 open questions:\n%s", n, page)
			}
			golden := []string{"== opened", ansi.Strip(m.View())}
			// Each block is a stop; the fifth is reached and shown.
			for k := 1; k < 5; k++ {
				m = press(m, "tab")
				if m.detailPane != focusQuestion || m.question != k {
					t.Fatalf("tab %d: pane %d question %d", k, m.detailPane, m.question)
				}
				if k == 1 {
					golden = append(golden, "== question 2 focused", ansi.Strip(m.View()))
				}
			}
			if screen := ansi.Strip(m.View()); !strings.Contains(screen, "> ? History file:") || !strings.Contains(screen, "└ question 5") {
				t.Fatalf("question 5 focused but not on screen:\n%s", screen)
			}
			golden = append(golden, "== question 5 focused", ansi.Strip(m.View()))
			// a on question 2 answers it alone.
			m = press(m, "shift+tab", "shift+tab", "shift+tab", "a")
			for _, r := range "both" {
				m = press(m, string(r))
			}
			m = press(m, "enter")
			tk = mustSelected(t, m)
			var open []int
			for _, q := range task.OpenQuestions(tk) {
				open = append(open, q.N)
			}
			if len(open) != 4 || open[0] != 1 || open[1] != 3 {
				t.Fatalf("open after answering 2: %v (message %q)", open, m.message)
			}
			goldenFile(t, "testdata/detail/p87ngv-five-questions-"+name+".txt", strings.Join(golden, "\n")+"\n")
		})
	}
}

// p87ngvBoard is a board holding one task, p87ngv, from file, opened as
// jhf at w x h, the clocks at 2026-11-19T10:06.
func p87ngvBoard(t *testing.T, file string, w, h int) model {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	old, oldTask := now, task.Now
	at := func() time.Time { return time.Date(2026, 11, 19, 10, 6, 0, 0, time.Local) }
	now, task.Now = at, at
	t.Cleanup(func() { now, task.Now = old, oldTask })
	s := &task.Store{Root: t.TempDir()}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(s.Root, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "p", "p87ngv.md"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "jhf"})
	m.width, m.height = w, h
	m.resize()
	return openTask(t, m, "p87ngv")
}
