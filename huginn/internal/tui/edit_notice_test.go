package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// jorgenEdit is the exact line Jørgen added to qawmnb's Depends on (rx74bp).
const jorgenEdit = "Is there a task:asdfasdf or ticket:asdlfkjasldkfj?"

// editWith goes through the TUI's own edit path (e, Enter), lets "the
// editor" rewrite the selected task file with change, then delivers the
// editor's result as Bubble Tea would.
func editWith(t *testing.T, m model, change func(string) string) model {
	t.Helper()
	t.Setenv("VISUAL", "true")
	sel, ok := m.selected()
	if !ok {
		t.Fatal("nothing selected")
	}
	m, cmd := key(m, "e")
	if cmd == nil {
		// The ordinary edit asks first; with the notice up, e goes at once.
		m, cmd = key(m, "enter")
	}
	if cmd == nil {
		t.Fatal("no editor command: " + m.message)
	}
	file := filepath.Join(m.store.Root, sel.Path)
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	after := change(string(b))
	if after == string(b) {
		t.Fatal("edit changed nothing")
	}
	if err := os.WriteFile(file, []byte(after), 0600); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(editorMsg{path: sel.Path})
	return next.(model)
}
func addDep(line string) func(string) string {
	return func(s string) string {
		return strings.Replace(s, "## Depends on\n", "## Depends on\n"+line+"\n", 1)
	}
}

// After the editor closes on Jørgen's edit, the TUI says what is wrong and
// how to fix it, e reopens the editor on that task without a question, the
// task is not blocked and shows no dependency row; Esc dismisses.
func TestEditNoticeShowsFindingFixAndReopens(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 160, 30
	m.resize()
	target, _ := m.selected()
	m = editWith(t, m, addDep(jorgenEdit))
	view := ansi.Strip(m.View())
	t.Log("\n" + view)
	for _, want := range []string{
		"⚠ " + target.ID + " line ",
		`not a dependency: "Is there a task:asdfasdf or ticket:asdlfkjasldkfj?"`,
		"▶ fix: hug task edit " + target.ID + " · e edit again · Esc dismiss",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in:\n%s", want, view)
		}
	}
	// Only what the edit introduced: the fixture's older missing-section
	// finding is not the notice.
	if strings.Contains(view, "(+") {
		t.Fatal("notice counts findings the edit did not introduce:\n" + view)
	}
	if strings.Contains(view, "dangling") || strings.Contains(view, "re-linted") {
		t.Fatal("old wording:\n" + view)
	}
	if m.blocked[target.Path] {
		t.Fatal("prose made the task blocked")
	}
	again, cmd := key(m, "e")
	if cmd == nil || again.confirm != "" {
		t.Fatal("e must reopen the editor at once while the notice shows")
	}
	// The detail header carries the finding, its line and its fix; there
	// is no dependency row and nothing open.
	d := press(m, "enter")
	detail := ansi.Strip(d.View())
	t.Log("\n" + detail)
	for _, want := range []string{"⚠ line ", `not a dependency: "Is there`, "  fix: hug task edit " + target.ID, "no dependencies"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail missing %q:\n%s", want, detail)
		}
	}
	if strings.Contains(detail, "↳") || strings.Contains(detail, "] open") {
		t.Fatalf("prose shown as a dependency:\n%s", detail)
	}
	m = press(m, "esc")
	if strings.Contains(ansi.Strip(m.View()), "e edit again") {
		t.Fatal("notice survived Esc")
	}
	if _, cmd := key(m, "e"); cmd != nil {
		t.Fatal("after Esc, e is the ordinary edit that asks first")
	}
}

// Undoing the edit clears the notice and its key.
func TestEditNoticeGoneWhenClean(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 160, 30
	m.resize()
	m = editWith(t, m, addDep(jorgenEdit))
	if !strings.Contains(ansi.Strip(m.View()), "e edit again") {
		t.Fatal("no notice")
	}
	m = editWith(t, m, func(s string) string { return strings.Replace(s, jorgenEdit+"\n", "", 1) })
	v := ansi.Strip(m.View())
	if strings.Contains(v, "e edit again") || !strings.Contains(v, "editor closed; re-linted") {
		t.Fatalf("clean edit:\n%s", v)
	}
}

// A missing dependency gets the notice too, with its remove command.
func TestEditNoticeMissingDependency(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 160, 30
	m.resize()
	target, _ := m.selected()
	m = editWith(t, m, addDep("- zz9zz9"))
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "depends on zz9zz9, but no task has that id or alias") || !strings.Contains(view, "▶ fix: hug task dep "+target.ID+" --remove zz9zz9 · e edit again") {
		t.Fatal(view)
	}
}

// Narrow terminals cut the fix, never the key that works.
func TestEditNoticeNarrowKeepsKey(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 60, 20
	m.resize()
	m = editWith(t, m, addDep(jorgenEdit))
	if !strings.Contains(ansi.Strip(m.View()), "· e edit again · Esc dismiss") {
		t.Fatal(ansi.Strip(m.View()))
	}
}

// Findings on closed tasks are hidden from the header count, the lint view
// and the ! marks, and the overview names how many; the task's own detail
// still shows them (6neax1).
func TestClosedTaskFindingsHidden(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 160, 30
	m.resize()
	target, _ := m.selected()
	// A finding that stays on a closed task: a Depends on line that is not
	// one (missing-section and long-title skip closed tasks already).
	file := filepath.Join(m.store.Root, target.Path)
	b, _ := os.ReadFile(file)
	if err := os.WriteFile(file, []byte(addDep("Nothing")(string(b))), 0600); err != nil {
		t.Fatal(err)
	}
	m.refresh(target.Path)
	before := len(m.findings)
	if !m.linted[target.Path] {
		t.Fatal("fixture task has no finding")
	}
	if _, err := m.store.Take(target.ID, "tester"); err != nil { // take before done (6ephqk)
		t.Fatal(err)
	}
	if _, err := m.store.Done(target.ID); err != nil {
		// Done refuses unchecked items; tick them first.
		for i := range target.Acceptance {
			if _, e := m.store.Check(target.ID, i+1); e != nil {
				t.Fatal(e)
			}
		}
		if _, err := m.store.Take(target.ID, "tester"); err != nil { // take before done (6ephqk)
			t.Fatal(err)
		}
		if _, err = m.store.Done(target.ID); err != nil {
			t.Fatal(err)
		}
	}
	m.refresh(target.Path)
	if m.linted[target.Path] || len(m.findings) >= before || len(m.closedFindings) == 0 {
		t.Fatalf("closed task still counted: %d of %d, closed %d", len(m.findings), before, len(m.closedFindings))
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, fmt.Sprintf("! %d findings", len(m.findings))) {
		t.Fatal(view)
	}
	m.setView("overview")
	if v := ansi.Strip(m.View()); !strings.Contains(v, fmt.Sprintf("(%d on closed tasks hidden)", len(m.closedFindings))) {
		t.Fatal(v)
	}
	m.setView("lint")
	for _, i := range m.indices() {
		if m.visible[i].Path == target.Path {
			t.Fatal("closed task in the lint view")
		}
	}
	m.detailPath, m.detail = target.Path, true
	if h := strings.Join(m.detailHeader(target), "\n"); !strings.Contains(h, "⚠") {
		t.Fatal("closed task's own detail lost its finding: " + h)
	}
}
