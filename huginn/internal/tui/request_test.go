package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// hbgybn check 4 (crab 01:30: r = request, r marked in 'request'): r in a
// task and on its row opens the request box, the answer's textarea (Enter
// sends, Alt-Enter adds a line); the row shows 'req 1' while it is open;
// the detail shows each request, then its account once resolved.
func TestRequestBox(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m := stageBoard(t)
		m.opts.User = "jhf"
		m.width, m.height = w, 30
		m.resize()
		m.setView("list")
		m.clamp()
		m = selectRow(t, m, "ffffff")
		if bar := plainMarks(ansi.Strip(m.View())); w == 140 && !strings.Contains(bar, "request") {
			t.Fatalf("at %d the bar does not name r request", w)
		}
		// r on the row opens the task with the box.
		m = press(m, "r")
		if m.prompt != requestPrompt || !m.detail {
			t.Fatalf("at %d r on a row: prompt %q detail %v", w, m.prompt, m.detail)
		}
		for _, r := range "make it per board" {
			m = press(m, string(r))
		}
		m, _ = key(m, "alt+enter")
		for _, r := range `and split "the TUI part"?` {
			m = press(m, string(r))
		}
		screen := ansi.Strip(m.View())
		if !strings.Contains(screen, "request: make it per board") || !strings.Contains(screen, `split "the TUI part"?`) || m.help {
			t.Fatalf("at %d the box does not hold both lines (or ? opened help):\n%s", w, screen)
		}
		golden = append(golden, "== typing at "+strconv.Itoa(w), lastLines(screen, 12))
		m = press(m, "enter")
		if m.prompt != "" || !strings.HasPrefix(m.message, "request ") {
			t.Fatalf("at %d Enter did not send: prompt %q message %q", w, m.prompt, m.message)
		}
		tk := mustSelected(t, m)
		if len(tk.Requests) != 1 || tk.Requests[0].Text != "make it per board\nand split \"the TUI part\"?" || tk.Requests[0].By != "jhf" {
			t.Fatalf("at %d stored %+v", w, tk.Requests)
		}
		page := ansi.Strip(m.detailText(tk))
		if !strings.Contains(page, "REQUESTS") || !strings.Contains(page, "1. jhf · ") || !strings.Contains(page, "· open") {
			t.Fatalf("at %d the detail lacks the open request:\n%s", w, page)
		}
		// The row says req 1 while it is open.
		back := press(m, "esc")
		if row := ansi.Strip(back.View()); !strings.Contains(row, "req 1 Depend on") {
			t.Fatalf("at %d the row lacks 'req 1':\n%s", w, row)
		}
		// Resolved: the account under it, and no req on the row.
		if _, err := m.store.Resolve("ffffff", 1, "split into two checks; per board in the goal", "nautilus"); err != nil {
			t.Fatal(err)
		}
		m.refresh("")
		tk = mustSelected(t, m)
		page = ansi.Strip(m.detailText(tk))
		if !strings.Contains(page, "· resolved") || !strings.Contains(page, "└ nautilus: split into two checks; per board in the goal") {
			t.Fatalf("at %d the account is not shown:\n%s", w, page)
		}
		at := strings.Index(page, "REQUESTS")
		golden = append(golden, "== resolved at "+strconv.Itoa(w), strings.Join(strings.Split(page[at:], "\n")[:6], "\n"))
		if row := ansi.Strip(press(m, "esc").View()); strings.Contains(row, "req 1") {
			t.Fatalf("at %d a resolved request still marks the row", w)
		}
		// r inside a task opens it too; an empty request sends nothing.
		if e := press(m, "r", "enter"); !strings.Contains(e.message, "no request sent") {
			t.Fatalf("at %d empty request: %q", w, e.message)
		}
	}
	goldenFile(t, "testdata/detail/request.txt", strings.Join(golden, "\n")+"\n")
}

// lastLines is the last n lines of s.
func lastLines(s string, n int) string {
	l := strings.Split(s, "\n")
	return strings.Join(l[max(0, len(l)-n):], "\n")
}
