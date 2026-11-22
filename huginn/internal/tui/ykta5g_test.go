package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// jkModel opens, at 100x24, a task with three checks, three dependencies,
// three links and three files, short lines each, and long Notes after
// them, so j on the last item of any list scrolls the page past it.
func jkModel(t *testing.T) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	var ids []string
	for _, title := range []string{"Dep one", "Dep two", "Dep three"} {
		r, err := s.New(task.NewOptions{Title: title, Goal: "g", Acceptance: []string{"d"}})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, r.Task.ID)
	}
	var links []string
	for _, title := range []string{"Link one", "Link two", "Link three"} {
		r, err := s.New(task.NewOptions{Title: title, Goal: "g", Acceptance: []string{"l"}})
		if err != nil {
			t.Fatal(err)
		}
		links = append(links, "task:"+r.Task.ID)
	}
	main, err := s.New(task.NewOptions{Title: "Main", Goal: "Builds on " + strings.Join(links, ", ") + ".",
		Acceptance: []string{"one", "two", "three"}, Deps: ids, Top: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if _, _, err := s.Attach(main.Task.ID, name, []byte("x\n"), "", false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Note(main.Task.ID, strings.TrimSpace(strings.Repeat("a line of notes\n\n", 40))); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 24
	m.resize()
	m.setView("list")
	m.move(0, true)
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.Title != "Main" {
		t.Fatalf("opened %q, want Main", tk.Title)
	}
	// The notes make the page run past the screen: open, as Enter on
	// their heading leaves them (8a0ybb: a page opens with them folded).
	m = m.withUnfolded("notes")
	m.syncDetail()
	return m
}

// markedRow is the page line of the item marked > in the focused list.
func markedRow(t *testing.T, m model) int {
	t.Helper()
	tk, _ := m.selected()
	for i, line := range strings.Split(ansi.Strip(m.detailText(tk)), "\n") {
		if strings.HasPrefix(line, "> ") {
			return i
		}
	}
	t.Fatalf("no marked line on the page")
	return -1
}

// ykta5g checks 1 and 2 (Jørgen 2026-11-18 19:55: "J goes down but K
// goes to top again"): in each list of a task, j steps to the last item
// and then scrolls the page past it one line at a time; k mirrors it:
// while the marked item is above the view, k scrolls back one line with
// the item still marked, and once it is in view k steps to the previous
// item. No key moves the page more than one line.
func TestDetailJKMirrorInEveryList(t *testing.T) {
	for _, tc := range []struct {
		name string
		pane int
		at   func(m model) int
		set  func(m *model)
	}{
		{"checks", focusAcceptance, func(m model) int { return m.acceptance }, func(m *model) { m.acceptance = 0 }},
		{"dependencies", focusDeps, func(m model) int { return m.dependency }, func(m *model) { m.dependency = 0 }},
		{"links", focusLinks, func(m model) int { return m.link }, func(m *model) { m.link = 0 }},
		{"files", focusFiles, func(m model) int { return m.file }, func(m *model) { m.file = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := jkModel(t)
			m.detailPane = tc.pane
			tc.set(&m)
			m.syncDetail()
			m.viewport.GotoTop()
			m.revealAcceptance()
			// j to the last item: each j moves the page by at most a line.
			for i := 0; tc.at(m) < 2; i++ {
				y := m.viewport.YOffset
				if m = press(m, "j"); m.viewport.YOffset-y > 1 || m.viewport.YOffset < y || i > 10 {
					t.Fatalf("j to item %d: offset %d -> %d", tc.at(m), y, m.viewport.YOffset)
				}
			}
			// Past the last item: j scrolls until it is above the view.
			n := 0
			for markedRow(t, m) >= m.viewport.YOffset {
				y := m.viewport.YOffset
				if m = press(m, "j"); m.viewport.YOffset != y+1 || tc.at(m) != 2 {
					t.Fatalf("j past the last item: offset %d -> %d, item %d", y, m.viewport.YOffset, tc.at(m))
				}
				if n++; n > 60 {
					t.Fatal("the page never scrolled past the last item")
				}
			}
			// k mirrors it: back one line at a time, then step.
			scrolled := 0
			for i := 0; tc.at(m) > 0 || markedRow(t, m) < m.viewport.YOffset; i++ {
				y, at, row := m.viewport.YOffset, tc.at(m), markedRow(t, m)
				above := row < y
				m = press(m, "k")
				if d := y - m.viewport.YOffset; d < 0 || d > 1 {
					t.Fatalf("k %d (item %d, line %d, offset %d): the page jumped to %d", i+1, at, row, y, m.viewport.YOffset)
				}
				switch {
				case above && (tc.at(m) != at || m.viewport.YOffset != y-1):
					t.Fatalf("k %d with item %d above the view (line %d < %d): item %d offset %d, want item %d offset %d", i+1, at, row, y, tc.at(m), m.viewport.YOffset, at, y-1)
				case !above && tc.at(m) != at-1:
					t.Fatalf("k %d with item %d in view: item %d, want %d", i+1, at, tc.at(m), at-1)
				}
				if above {
					scrolled++
				}
				if i > 80 {
					t.Fatal("k never reached the first item")
				}
			}
			if scrolled == 0 {
				t.Fatal("k never scrolled back: the test did not get past the last item")
			}
		})
	}
}

// askedModel opens, at 100x24 as jhf, a task asked of jhf on check 2 with
// a question as long as gav9nw's (three paragraphs, a list), and a long
// description above the checks, so check 2 and its question do not fit
// the page from the top.
func askedModel(t *testing.T, question string) model {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	long := strings.TrimSpace(strings.Repeat("A paragraph of what the task is about.\n\n", 12))
	r, err := s.New(task.NewOptions{Title: "Asked", Goal: long, Acceptance: []string{"one", "two: decide which option", "three"}, Top: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.TakeAs(r.Task.ID, "crab", "m"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Ask(r.Task.ID, task.Question{What: question, For: "jhf", Check: 2}, "crab"); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "jhf"})
	m.width, m.height = 100, 24
	m.resize()
	m.setView("list")
	m.move(0, true)
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.Title != "Asked" || m.detailPane != focusAcceptance || m.acceptance != 1 {
		t.Fatalf("opened %q focus %d check %d, want Asked on check 2", tk.Title, m.detailPane, m.acceptance+1)
	}
	return m
}

// onScreen is the page as the viewport shows it, plain.
func onScreen(m model) string { return ansi.Strip(m.viewport.View()) }

// ykta5g check 4 (Jørgen 19:37Z, gav9nw request 1): when a check gets
// focus, on open or by j/k, the whole check shows, its question and who
// asked included, not only its first line; a check taller than the page
// shows from its first line.
func TestFocusedCheckShowsWhole(t *testing.T) {
	question := "Which option do we take?\n\n" +
		"A+ the extractor places at write time with personal and other-project places added.\n" +
		"B a rule at sync holds back lines that name a home path, another project or the person.\n" +
		"C only what an agent writes on purpose is shared.\n\n" +
		"A+ and B together pushed 699 with no leak seen; still a hypothesis until fresh captures confirm it."
	m := askedModel(t, question)
	whole := func(m model, when string) {
		t.Helper()
		page := onScreen(m)
		for _, want := range []string{"> ? ☐ 2. two: decide which option", "Which option do we take?", "C only what an agent writes on purpose", "still a hypothesis", "└ crab"} {
			if !strings.Contains(page, want) {
				t.Fatalf("%s: the focused check is not whole on screen, missing %q:\n%s", when, want, page)
			}
		}
	}
	whole(m, "on open")
	// By j: check 1 marked on the page's last row, j to check 2.
	m = press(m, "k")
	if m.acceptance != 0 {
		t.Fatalf("k: check %d, want 1", m.acceptance+1)
	}
	m.viewport.SetYOffset(markedRow(t, m) - m.viewport.Height + 1)
	m = press(m, "j")
	if m.acceptance != 1 {
		t.Fatalf("j: check %d, want 2", m.acceptance+1)
	}
	whole(m, "after j")
	// By k: from check 3, back to check 2.
	m = press(m, "j")
	if m.acceptance != 2 {
		t.Fatalf("j: check %d, want 3", m.acceptance+1)
	}
	m = press(m, "k")
	if m.acceptance != 1 {
		t.Fatalf("k: check %d, want 2", m.acceptance+1)
	}
	whole(m, "after k")
	// Taller than the page: from its first line, then j reads it down a
	// line at a time before it steps to check 3.
	tall := strings.TrimSpace(strings.Repeat("A line of a very long question.\n", 40))
	m = askedModel(t, tall)
	if page := onScreen(m); !strings.HasPrefix(strings.TrimLeft(strings.SplitN(page, "\n", 2)[0], " "), "> ? ☐ 2.") {
		t.Fatalf("a check taller than the page must show from its first line, page starts:\n%s", page)
	}
	for i := 0; m.acceptance == 1; i++ {
		y := m.viewport.YOffset
		m = press(m, "j")
		if m.acceptance == 1 && m.viewport.YOffset != y+1 || i > 60 {
			t.Fatalf("j %d inside the tall check: offset %d -> %d", i+1, y, m.viewport.YOffset)
		}
	}
	if !strings.Contains(onScreen(m), "> ☐ 3. three") {
		t.Fatalf("j after the tall check: check 3 not on screen:\n%s", onScreen(m))
	}
}
