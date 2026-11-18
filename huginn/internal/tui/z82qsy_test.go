package tui

import (
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
			}
		}
		if at < 0 || at+2 >= len(lines) {
			t.Fatalf("%s: no file header:\n%s", view, strings.Join(lines, "\n"))
		}
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
