package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// ec6z6j check 5 (Jørgen 2026-11-20 08:31Z, option A): with tasks marked,
// r opens the request box at the bottom of the list, the marked rows still
// in view while typing; Enter sends one request for exactly the marked
// set (task.RequestFor, as hug task request a,b,c); Esc cancels and keeps
// the marks; the marked footer offers r request. With no marks r is a
// request on the cursor row's task, as before.
func TestMarkedRequestBox(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, view := range []string{"list", "lint", "overview"} {
		t.Run(view, func(t *testing.T) {
			m, _, _ := markBoard(t)
			var subjects []string
			m.opts.Commit = func(s string) error { subjects = append(subjects, s); return nil }
			m.opts.User = "jhf"
			m.width, m.height = 80, 24
			m.resize()
			if view == "overview" {
				m = nextUp(m)
			} else {
				m.setView(view)
			}
			m.clamp()
			rows := shown(m)
			if len(rows) < 4 {
				t.Fatalf("%s shows %d rows: %v", view, len(rows), rows)
			}
			marked := rows[:3]
			m = selectRow(t, m, rows[0])
			m = press(m, " ", " ", " ")
			if f := plainMarks(footerText(m)); !strings.Contains(f, " · request · ") || !strings.HasPrefix(f, "3 marked · ") {
				t.Fatalf("the marked footer does not offer r request: %q", f)
			}
			if !strings.Contains(m.footerMarkedWord("request"), markOpen+"r"+markClose) {
				t.Fatalf("r is not the marked key in 'request': %q", m.footerMarkedWord("request"))
			}

			// r: the box at the bottom of the list, the list stays.
			m = press(m, "r")
			if m.prompt != requestPrompt || m.detail {
				t.Fatalf("r with 3 marked: prompt %q detail %v", m.prompt, m.detail)
			}
			for _, k := range "each lacks a purpose" {
				m = press(m, string(k))
			}
			lines := strings.Split(ansi.Strip(m.View()), "\n")
			head, box := -1, -1
			for i, l := range lines {
				if strings.HasPrefix(l, "▸ REQUEST FOR 3 MARKED · "+strings.Join(marked, " ")) {
					head = i
				}
				if strings.HasPrefix(strings.TrimSpace(l), "request: each lacks a purpose") {
					box = i
				}
			}
			if head < 0 || box != head+1 {
				t.Fatalf("no heading naming the 3 marked with the box under it (head %d box %d):\n%s", head, box, strings.Join(lines, "\n"))
			}
			if box < len(lines)-5 {
				t.Fatalf("the box is not at the bottom (line %d of %d):\n%s", box, len(lines), strings.Join(lines, "\n"))
			}
			for _, id := range marked {
				at := -1
				for i, l := range lines[:head] {
					if strings.Contains(l, id+" ") && strings.HasPrefix(l, markSign) {
						at = i
					}
				}
				if at < 0 {
					t.Fatalf("marked %s is not in view above the box:\n%s", id, strings.Join(lines, "\n"))
				}
			}
			if bar := lines[len(lines)-1]; !strings.Contains(bar, "Enter send") || !strings.Contains(bar, "Esc cancel") {
				t.Fatalf("key bar while typing: %q", bar)
			}

			// Esc: nothing written, the marks stay.
			m = press(m, "esc")
			if m.prompt != "" || len(m.marks) != 3 || len(subjects) != 0 {
				t.Fatalf("Esc: prompt %q marks %v commits %v", m.prompt, m.marks, subjects)
			}
			for _, id := range marked {
				if got, _ := m.taskAt(pathOf(t, m, id)); len(got.Requests) != 0 {
					t.Fatalf("Esc wrote a request on %s", id)
				}
			}
			if strings.Contains(ansi.Strip(m.View()), "REQUEST FOR") {
				t.Fatalf("Esc left the box drawn:\n%s", ansi.Strip(m.View()))
			}

			// Empty Enter sends nothing and keeps the marks.
			m = press(m, "r", "enter")
			if !strings.Contains(m.message, "no request sent") || len(m.marks) != 3 || len(subjects) != 0 {
				t.Fatalf("empty Enter: %q marks %v commits %v", m.message, m.marks, subjects)
			}

			// Enter: one request for exactly the marked set.
			m = press(m, "r")
			for _, k := range "write a purpose" {
				m = press(m, string(k))
			}
			m = press(m, "enter")
			if m.prompt != "" {
				t.Fatalf("Enter left the box open: %q", m.prompt)
			}
			home, _ := m.taskAt(pathOf(t, m, marked[0]))
			if len(home.Requests) != 1 || home.Requests[0].Text != "write a purpose" || home.Requests[0].By != "jhf" {
				t.Fatalf("home %s requests %+v", marked[0], home.Requests)
			}
			var targets []string
			for _, tg := range home.Requests[0].Targets {
				targets = append(targets, tg.ID)
			}
			if strings.Join(targets, ",") != strings.Join(marked, ",") {
				t.Fatalf("targets %v, want exactly the marked %v", targets, marked)
			}
			for _, id := range marked[1:] {
				got, _ := m.taskAt(pathOf(t, m, id))
				if len(got.Requests) != 1 || got.Requests[0].Home == nil || got.Requests[0].Home.ID != marked[0] || got.Requests[0].Home.N != 1 {
					t.Fatalf("%s is not a link to %s request 1: %+v", id, marked[0], got.Requests)
				}
			}
			for _, id := range rows[3:] {
				if got, _ := m.taskAt(pathOf(t, m, id)); len(got.Requests) != 0 {
					t.Fatalf("unmarked %s got a request", id)
				}
			}
			if len(subjects) != 1 || subjects[0] != "hug task request "+strings.Join(marked, ",") {
				t.Fatalf("commits %q, want one 'hug task request %s'", subjects, strings.Join(marked, ","))
			}
			if want := "request 1 for 3 tasks, on " + marked[0]; !strings.HasPrefix(m.message, want) {
				t.Fatalf("message %q, want %q", m.message, want)
			}
			if len(m.marks) != 0 {
				t.Fatalf("marks after the request went through: %v", m.marks)
			}

			// No marks: r is a request on the cursor row's task, as before.
			m = selectRow(t, m, rows[3])
			m = press(m, "r")
			if m.prompt != requestPrompt || !m.detail {
				t.Fatalf("r with no marks: prompt %q detail %v", m.prompt, m.detail)
			}
		})
	}
}

// footerMarkedWord is word as the marked footer writes it, its key marked.
func (m model) footerMarkedWord(word string) string {
	for _, it := range itemsOf(m.markedHints()) {
		if plainMarks(it.words) == word {
			return it.words
		}
	}
	return ""
}
