package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// fileScreen opens 000001's first file (bench/README.md) from view, by
// Enter on its row there, and returns the screen's lines.
func fileScreen(t *testing.T, view string) (model, []string) {
	t.Helper()
	m, _ := glanceModel(t, 80, 24)
	m.setView(view)
	found := false
	for pane := 0; pane < maxSections && !found; pane++ {
		m.pane = pane
		for _, i := range m.indices() {
			if m.visible[i].ID == "000001" {
				m.cursor, found = i, true
				break
			}
		}
	}
	if !found {
		t.Fatalf("%s: no row for 000001", view)
	}
	m = press(m, "enter")
	if sel, ok := m.selected(); !ok || sel.ID != "000001" || !m.detail || m.view != view {
		t.Fatalf("%s: Enter opened %v in view %s", view, sel.ID, m.view)
	}
	m = press(toFiles(t, m), " ")
	if !m.glance {
		t.Fatalf("%s: Space did not show the file", view)
	}
	return m, strings.Split(ansi.Strip(m.View()), "\n")
}

// z82qsy check 1: a rule separates the file's header line (path · size ·
// i of N) from its contents, in every view the task was opened from, in
// the detail header's own rule.
func TestFileViewerRuleUnderHeader(t *testing.T) {
	var golden []string
	for _, view := range []string{"overview", "list", "board"} {
		m, lines := fileScreen(t, view)
		at := -1
		for i, l := range lines {
			if strings.Contains(l, "bench/README.md · 32 B · 1 of 5") {
				at = i
				break
			}
		}
		if at < 0 || at+2 >= len(lines) {
			t.Fatalf("%s: no file header:\n%s", view, strings.Join(lines, "\n"))
		}
		// check 4 put the header in the crumb, line 0; the rule follows.
		rule := strings.Repeat("─", m.width)
		if lines[at+1] != rule || strings.TrimRight(lines[at+2], " ") != "Bench" {
			t.Fatalf("%s: under the file header %q / %q, want the rule then the contents", view, lines[at+1], lines[at+2])
		}
		if len(lines) != m.height {
			t.Fatalf("%s: the screen is %d lines, want %d", view, len(lines), m.height)
		}
		golden = append(golden, "== "+view, strings.Join(lines, "\n"))
	}
	goldenFile(t, "testdata/detail/file-rule.txt", strings.Join(golden, "\n")+"\n")
}

// z82qsy check 2: a result stays while you stay; opening another task, a
// file, or another view clears it.
func TestActionMessageClearsOnMovingOn(t *testing.T) {
	m, _ := glanceModel(t, 120, 30)
	m = openTask(t, m, "000001")
	m = press(m, "t", "enter")
	if !strings.Contains(m.message, "000001") {
		t.Fatalf("t Enter on 000001: message %q", m.message)
	}
	result := m.message
	if stay := press(m, "j"); stay.message != result {
		t.Fatalf("moving within the same task cleared the result: %q", stay.message)
	}
	for name, step := range map[string]func(model) model{
		"another task": func(m model) model { return press(m, "#", "000002", "enter") },
		"a file":       func(m model) model { return press(toFiles(t, m), " ") },
		"another view": func(m model) model { return press(m, "esc", "v", "b") },
	} {
		got := step(m)
		if name == "a file" && !got.glance {
			t.Fatalf("a file: Space did not show it (pane %v)", got.detailPane)
		}
		if strings.Contains(got.message, result) || strings.Contains(ansi.Strip(got.View()), result) {
			t.Fatalf("%s: the old result stays: %q", name, got.message)
		}
	}
}

// z82qsy check 3: a result reads as done, with the id ("answered ID"),
// not as the command that did it.
func TestActionResultIsPastTense(t *testing.T) {
	for action, want := range map[string]string{
		"take": "took", "give back": "gave back", "done": "marked done", "drop": "dropped",
		"wait": "parked", "answer": "answered", "request": "requested", "strike": "struck",
		"restore": "restored", "check": "checked", "uncheck": "unchecked", "new": "created", "move": "moved",
	} {
		if got := pastTense(action); got != want {
			t.Errorf("%s: %q, want %q", action, got, want)
		}
	}
	m, _ := glanceModel(t, 80, 24)
	m = openTask(t, m, "000001")
	if got := press(m, "t", "enter"); got.message != "took 000001" {
		t.Fatalf("t Enter: %q, want %q", got.message, "took 000001")
	}
}

// attachedViewer is task 000001 with screen.ans and notes.md attached
// through the store, each with its note, at width w; Files focused.
func attachedViewer(t *testing.T, w int) model {
	t.Helper()
	m, _ := glanceModel(t, w, 24)
	ans := "\x1b[1;38;2;0;191;255mHUG TASKS\x1b[0m · list\n\x1b[38;2;135;149;168mrow one\x1b[0m\nrow two\n"
	for _, f := range []struct{ name, body, why string }{
		{"screen.ans", ans, "screen capture from hug tui"},
		{"notes.md", "# Notes heading\n\nA paragraph about the mock.\n", "detail mock"},
	} {
		if _, _, err := m.store.Attach("000001", f.name, []byte(f.body), f.why, false); err != nil {
			t.Fatal(err)
		}
	}
	m.refresh("")
	m = openTask(t, m, "000001")
	return toFiles(t, m)
}

// z82qsy check 4: viewing a file, only the file shows: on top the crumb
// (task id › file path · size · n of m) and the attachment's note, under
// it a full-width rule; the file; a full-width rule; then outside the
// frame the message line and the keys. No chips, dates, lint or task
// title; a .md's own heading is not repeated as the note.
func TestFileViewerFramesOnlyTheFile(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m := attachedViewer(t, w)
		files := m.filesOf(mustSelected(t, m))
		for _, want := range []struct{ name, note, first string }{
			{"screen.ans", "screen capture from hug tui", "HUG TASKS · list"},
			{"notes.md", "detail mock", "Notes heading"},
		} {
			at := -1
			for i, f := range files {
				if f == want.name {
					at = i
				}
			}
			if at < 0 {
				t.Fatalf("no %s in %v", want.name, files)
			}
			v := m
			v.file = at
			v = press(v, " ")
			if v.glanceMD.eligible {
				v = finishMarkdown(v)
			}
			screen := ansi.Strip(v.View())
			lines := strings.Split(screen, "\n")
			trim := func(i int) string { return strings.TrimRight(lines[i], " ") }
			rule := strings.Repeat("─", w)
			crumb := "HUG · list › 000001 › compiler/000001/" + want.name
			if !strings.HasPrefix(trim(0), crumb+" · ") || !strings.HasSuffix(trim(0), fmt.Sprintf("%d of %d", at+1, len(files))) {
				t.Fatalf("at %d %s: crumb %q, want %q · size · n of m", w, want.name, trim(0), crumb)
			}
			if trim(1) != want.note || lines[2] != rule || trim(3) != want.first {
				t.Fatalf("at %d %s: top reads %q / %q / %q, want the note, a rule, then the file\n%s", w, want.name, trim(1), lines[2], trim(3), screen)
			}
			// Under the frame: the keys, with no blank row between (yp3c09);
			// a message or an input line would sit between when there is one.
			if lines[len(lines)-2] != rule {
				t.Fatalf("at %d %s: no rule right above the keys:\n%s", w, want.name, screen)
			}
			for _, gone := range []string{"[todo]", "created 2026", "⚠ line", "Feature task 01", "File · "} {
				if strings.Contains(screen, gone) {
					t.Fatalf("at %d %s: %q still shows:\n%s", w, want.name, gone, screen)
				}
			}
			if strings.Count(screen, "Notes heading") > 1 {
				t.Fatalf("at %d: the .md heading shows twice:\n%s", w, screen)
			}
			if len(lines) != v.height {
				t.Fatalf("at %d %s: %d lines, want %d", w, want.name, len(lines), v.height)
			}
			golden = append(golden, fmt.Sprintf("== %s at %d", want.name, w), screen)
		}
	}
	goldenFile(t, "testdata/detail/file-frame.txt", strings.Join(golden, "\n")+"\n")
}

// z82qsy check 4: a file with no attachment note has no note line; the
// rule is right under the crumb.
func TestFileViewerNoNoteNoLine(t *testing.T) {
	_, lines := fileScreen(t, "list")
	if !strings.HasPrefix(lines[0], "HUG · list › 000001 › compiler/000001/bench/README.md · 32 B · 1 of 5") || lines[1] != strings.Repeat("─", 80) || strings.TrimRight(lines[2], " ") != "Bench" {
		t.Fatalf("no note: %q / %q / %q", lines[0], lines[1], lines[2])
	}
}
