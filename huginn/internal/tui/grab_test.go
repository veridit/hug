package tui

import (
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// The 3xcesq board: six tasks in rank order, in the list.
func grabBoard(t *testing.T) (model, []string, *int) {
	t.Helper()
	m, ids, commits := moveBoard(t, "A", "B", "C", "D", "E", "F")
	m.setView("list")
	var order []string
	for _, title := range []string{"A", "B", "C", "D", "E", "F"} {
		order = append(order, ids[title])
	}
	return m, order, commits
}

func pick(order []string, picks ...int) string {
	var out []string
	for _, p := range picks {
		out = append(out, order[p])
	}
	return strings.Join(out, " ")
}

// rankOrder is the order on disk, read fresh from the store.
func rankOrder(t *testing.T, m model) string {
	t.Helper()
	all, err := m.store.Tasks()
	if err != nil {
		t.Fatal(err)
	}
	task.Sort(all)
	var out []string
	for _, v := range all {
		out = append(out, v.ID)
	}
	return strings.Join(out, " ")
}

// Check: m, several j, Enter moves a task far with exactly one commit; the
// rows move on screen at every press and nothing is written until Enter.
func TestGrabMovesFarWithOneCommit(t *testing.T) {
	m, o, commits := grabBoard(t)
	m = selectRow(t, m, o[0])
	m = press(m, "m")
	if !m.grab.active || !strings.Contains(m.keyBar(), "Enter drop") || strings.Contains(m.keyBar(), "Enter open") {
		t.Fatalf("grab did not start or the key bar shows keys that do not work:\n%s", m.keyBar())
	}
	m = press(m, "j", "j", "j")
	if got := strings.Join(shown(m), " "); got != pick(o, 1, 2, 3, 0, 4, 5) {
		t.Fatalf("on screen after 3 j: %s", got)
	}
	if got := strings.Join(screenOrder(m, o...), " "); got != pick(o, 1, 2, 3, 0, 4, 5) {
		t.Fatalf("screen does not show the grabbed row moving: %s\n%s", got, m.View())
	}
	view := m.View()
	if !strings.Contains(view, "≡") || !strings.Contains(view, "Moving "+o[0]+" · row 4 of 6 in the list (was 1) · nothing written until Enter") {
		t.Fatalf("grabbed row or footer not shown:\n%s", view)
	}
	if *commits != 0 || rankOrder(t, m) != pick(o, 0, 1, 2, 3, 4, 5) {
		t.Fatalf("written before Enter: commits %d, order %s", *commits, rankOrder(t, m))
	}
	m = press(m, "enter")
	if *commits != 1 || m.grab.active {
		t.Fatalf("Enter: commits %d, still grabbed %v", *commits, m.grab.active)
	}
	if got := rankOrder(t, m); got != pick(o, 1, 2, 3, 0, 4, 5) {
		t.Fatalf("order on disk: %s", got)
	}
	if want := "moved " + o[0] + " below " + o[3] + ", row 4 of 6 in the list"; m.message != want {
		t.Fatalf("message %q, want %q", m.message, want)
	}
	if sel, _ := m.selected(); sel.ID != o[0] {
		t.Fatal("selection did not stay on the moved task", sel.ID)
	}
}

// Check 1, literally: m, five presses of k, Enter moves the last task five
// rows up with one commit and no confirm.
func TestGrabFiveUpOneCommit(t *testing.T) {
	m, o, commits := grabBoard(t)
	m = selectRow(t, m, o[5])
	m = press(m, "m", "k", "k", "k", "k", "k", "enter")
	if m.confirm != "" || m.grab.active || *commits != 1 {
		t.Fatalf("confirm %q grabbed %v commits %d", m.confirm, m.grab.active, *commits)
	}
	if got := rankOrder(t, m); got != pick(o, 5, 0, 1, 2, 3, 4) {
		t.Fatalf("order on disk: %s", got)
	}
	if want := "moved " + o[5] + " above " + o[0] + ", row 1 of 6 in the list"; m.message != want {
		t.Fatalf("message %q, want %q", m.message, want)
	}
}

// Check: Esc puts the task back with no commit and says so.
func TestGrabEscCancels(t *testing.T) {
	m, o, commits := grabBoard(t)
	m = selectRow(t, m, o[4])
	m = press(m, "m", "k", "k", "k", "esc")
	if m.grab.active || *commits != 0 || m.message != "move cancelled; nothing changed" {
		t.Fatalf("Esc: grabbed %v commits %d message %q", m.grab.active, *commits, m.message)
	}
	if got := strings.Join(shown(m), " "); got != pick(o, 0, 1, 2, 3, 4, 5) {
		t.Fatalf("screen after Esc: %s", got)
	}
	if sel, _ := m.selected(); sel.ID != o[4] {
		t.Fatal("Esc lost the selection", sel.ID)
	}
	// Enter where it started writes nothing either.
	m = press(m, "m", "j", "k", "enter")
	if *commits != 0 || m.message != "dropped where it was; nothing changed" {
		t.Fatalf("drop in place: commits %d message %q", *commits, m.message)
	}
}

// Check: a count, < and > move far; moving up lands above the row below.
func TestGrabCountsAndEnds(t *testing.T) {
	m, o, commits := grabBoard(t)
	m = selectRow(t, m, o[5])
	m = press(m, "m", "3", "k")
	if got := strings.Join(shown(m), " "); got != pick(o, 0, 1, 5, 2, 3, 4) {
		t.Fatalf("3k: %s", got)
	}
	m = press(m, "<", "enter")
	if got := rankOrder(t, m); got != pick(o, 5, 0, 1, 2, 3, 4) || *commits != 1 {
		t.Fatalf("< Enter: %s, commits %d", got, *commits)
	}
	if want := "moved " + o[5] + " above " + o[0] + ", row 1 of 6 in the list"; m.message != want {
		t.Fatalf("message %q, want %q", m.message, want)
	}
	m = press(m, "m", ">", "enter")
	if got := rankOrder(t, m); got != pick(o, 0, 1, 2, 3, 4, 5) || *commits != 2 {
		t.Fatalf("> Enter: %s, commits %d", got, *commits)
	}
	// Past the end clamps to the last row.
	m = selectRow(t, m, o[0])
	m = press(m, "m", "9", "9", "j")
	if got := strings.Join(shown(m), " "); got != pick(o, 1, 2, 3, 4, 5, 0) {
		t.Fatalf("99j: %s", got)
	}
	m = press(m, "q")
	if !m.grab.active || !strings.Contains(m.message, "Enter drop · Esc cancel") {
		t.Fatalf("other keys must say what works: %q", m.message)
	}
	m = press(m, "l")
	if !m.grab.active || m.message != o[0]+" moves within the list: Enter drops it, Esc cancels" {
		t.Fatalf("pane keys: %q", m.message)
	}
}

// Check: y then P/p moves the yanked task above/below the selected row with
// one commit; Esc forgets the yank.
func TestYankAndPaste(t *testing.T) {
	m, o, commits := grabBoard(t)
	m = selectRow(t, m, o[5])
	m = press(m, "y")
	if m.yanked != o[5] || !strings.Contains(m.message, "P puts it above the selected row, p below") {
		t.Fatalf("yank: %q %q", m.yanked, m.message)
	}
	if bar := m.keyBar(); !strings.Contains(plainMarks(bar), "Paste above") || !strings.Contains(bar, "Esc forget "+o[5]) {
		t.Fatalf("key bar while yanked:\n%s", bar)
	}
	m = selectRow(t, m, o[1])
	m = press(m, "P")
	if got := rankOrder(t, m); got != pick(o, 0, 5, 1, 2, 3, 4) || *commits != 1 || m.yanked != "" {
		t.Fatalf("P: %s, commits %d, yanked %q", got, *commits, m.yanked)
	}
	if want := "moved " + o[5] + " above " + o[1] + ", row 2 of 6 in the list"; m.message != want {
		t.Fatalf("message %q, want %q", m.message, want)
	}
	m = selectRow(t, m, o[0])
	m = press(m, "y")
	m = selectRow(t, m, o[4])
	m = press(m, "p")
	if got := rankOrder(t, m); got != pick(o, 5, 1, 2, 3, 4, 0) || *commits != 2 {
		t.Fatalf("p: %s, commits %d", got, *commits)
	}
	// Without a yank, p and P say so and do nothing else; k moves up.
	cursor := m.cursor
	m = press(m, "p")
	if m.cursor != cursor || *commits != 2 || m.message != "nothing yanked: y on a task first" {
		t.Fatalf("p without a yank: cursor %d -> %d, commits %d, message %q", cursor, m.cursor, *commits, m.message)
	}
	if bar := m.keyBar(); strings.Contains(bar, "paste") {
		t.Fatalf("paste keys shown with nothing yanked:\n%s", bar)
	}
	m = press(m, "y", "esc")
	if m.yanked != "" || !strings.Contains(m.message, "nothing changed") || *commits != 2 {
		t.Fatalf("Esc after yank: %q %q", m.yanked, m.message)
	}
	// Pasting next to itself says what to do.
	m = press(m, "y", "p")
	if *commits != 2 || !strings.Contains(m.message, "select the row to paste it next to") {
		t.Fatalf("paste on itself: %q", m.message)
	}
}

// Check: a yanked task not shown in this section is not pasted; RECENTLY
// DONE refuses grab and yank; J and K move with no confirm.
func TestMoveFarRefusals(t *testing.T) {
	m, ids, commits := moveBoard(t, "Alpha one", "Beta", "Alpha two", "done: Old")
	m.setView("list")
	m = selectRow(t, m, ids["Beta"])
	m = press(m, "y", "/", "Alpha", "tab")
	m = selectRow(t, m, ids["Alpha two"])
	m = press(m, "p")
	if *commits != 0 || m.message != ids["Beta"]+" is not shown in the list (filtered): paste it where it is shown, or Esc forgets it" {
		t.Fatalf("paste of a hidden task: commits %d %q", *commits, m.message)
	}
	m = press(m, "esc", "F") // Esc forgets the yank, F the kept search (5cjy88)
	m.setView("overview")
	m.pane = sectionRecent
	m.clamp()
	if len(m.indices()) == 0 {
		t.Fatal("RECENTLY DONE is empty")
	}
	for _, k := range []string{"m", "y"} {
		m = press(m, k)
		if m.grab.active || m.yanked != "" || m.confirm != "" || *commits != 0 || !strings.HasPrefix(m.message, "RECENTLY DONE is ordered by close time") {
			t.Fatalf("%s in RECENTLY DONE: %q", k, m.message)
		}
	}
}

// Check: a change on disk during a grab waits for the drop, so the rows do
// not jump under the reader.
func TestGrabDefersReload(t *testing.T) {
	m, o, _ := grabBoard(t)
	m = selectRow(t, m, o[0])
	m = press(m, "m", "j")
	if !m.busy() {
		t.Fatal("a grab must hold back a reload")
	}
	m = press(m, "esc")
	if m.busy() {
		t.Fatal("after Esc the reload must run")
	}
}
