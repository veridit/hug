package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// hya1st (Jørgen 2026-11-24 18:15Z on statbus kkj0mn: "it starts big and
// then shrinks, it should start small first? And DETAILS can show if not
// big?"; 18:23Z "Goal, Purpose, Principles, Details"; 18:24Z "What, Why,
// How, Notes"): a task's page reads GOAL, PURPOSE, PRINCIPLE, DETAILS,
// then ACCEPTANCE and the rest. Details is open when its text takes 6
// lines or fewer at the page's width, else it folds to one line with its
// count, in the same place.

// detailsBoard is a board of tasks with the given Details texts, titled
// by their keys, each with a Goal, Purpose and Principle, at w x h.
func detailsBoard(t *testing.T, w, h int, details map[string]string) (model, map[string]string) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	s := &task.Store{Root: t.TempDir()}
	ids := map[string]string{}
	for _, title := range sortedKeys(details) {
		r, err := s.New(task.NewOptions{Title: title, Goal: "One line goal.", Purpose: "Why it matters, a little longer than the goal.", Principle: "How we go.", Details: details[title], Acceptance: []string{"first check"}})
		if err != nil {
			t.Fatal(err)
		}
		ids[title] = r.Task.ID
	}
	m := newModel(s, Options{User: "jhf"})
	m.width, m.height = w, h
	m.resize()
	return m, ids
}

func sortedKeys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// sectionBody is the lines under heading on the page, up to the next
// heading, blank lines at its ends dropped.
func sectionBody(text, heading string) []string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) != heading {
			continue
		}
		var body []string
		for _, b := range lines[i+1:] {
			if len(headings(b)) == 1 {
				break
			}
			body = append(body, b)
		}
		for len(body) > 0 && strings.TrimSpace(body[0]) == "" {
			body = body[1:]
		}
		for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
			body = body[:len(body)-1]
		}
		return body
	}
	return nil
}

const (
	// Three paragraphs, the last wrapping to two lines at 100 columns:
	// 1 + blank + 1 + blank + 2 = 6 lines.
	sixLines = "First note.\n\nSecond note.\n\nThird note is long enough to wrap once at a hundred columns, since it says more than one line of the page can hold here."
	// Four short paragraphs: 7 lines at any width.
	sevenLines = "First note.\n\nSecond note.\n\nThird note.\n\nFourth note."
	// Three paragraphs, the last 3 lines at 100 columns (7 lines,
	// folded) and 2 at 160 (6 lines, open).
	widthBound = "First note.\n\nSecond note.\n\nThird note is long enough to wrap twice at a hundred columns, since it says rather more than two lines of a page that narrow can hold, and only once on a wide page of a hundred and sixty columns, which holds it in two lines."
)

// hya1st check 2: the order, Details open in its place when short.
func TestPageReadsGoalPurposePrincipleDetails(t *testing.T) {
	m, ids := detailsBoard(t, 100, 30, map[string]string{"Short": "Seen 2026-11-24 on the branch run."})
	m = openTask(t, m, ids["Short"])
	text := detailPlain(m)
	if got, want := strings.Join(headings(text), "|"), "GOAL|PURPOSE|PRINCIPLE|DETAILS|ACCEPTANCE"; got != want {
		t.Fatalf("headings\n %s\nwant\n %s\n%s", got, want, text)
	}
	if !strings.Contains(text, "Seen 2026-11-24 on the branch run.") {
		t.Fatalf("a one-line Details is not open on the page:\n%s", text)
	}
}

// hya1st check 2: at the threshold. 6 lines open, 7 folded to one line
// with its count, in the same place (after PRINCIPLE, before ACCEPTANCE);
// the bound is counted at the page's width.
func TestDetailsOpenAtSixLinesFoldedAtSeven(t *testing.T) {
	for _, c := range []struct {
		name, details string
		width         int
		open          bool
		count         int
	}{
		{"six", sixLines, 100, true, 3},
		{"seven", sevenLines, 100, false, 4},
		{"wide-at-100", widthBound, 100, false, 3},
		{"wide-at-160", widthBound, 160, true, 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			m, ids := detailsBoard(t, c.width, 40, map[string]string{"T": c.details})
			m = openTask(t, m, ids["T"])
			text := detailPlain(m)
			if c.open {
				if got, want := strings.Join(headings(text), "|"), "GOAL|PURPOSE|PRINCIPLE|DETAILS|ACCEPTANCE"; got != want {
					t.Fatalf("headings %s, want %s:\n%s", got, want, text)
				}
				body := sectionBody(text, "DETAILS")
				if len(body) != 6 && c.name == "six" || len(body) > 6 {
					t.Fatalf("Details open in %d lines:\n%s", len(body), strings.Join(body, "\n"))
				}
				if !strings.Contains(text, "Third note") {
					t.Fatalf("Details text not on the page:\n%s", text)
				}
				return
			}
			head := fmt.Sprintf("DETAILS (%d)", c.count)
			if got, want := strings.Join(headings(text), "|"), "GOAL|PURPOSE|PRINCIPLE|"+head+"|ACCEPTANCE"; got != want {
				t.Fatalf("headings %s, want %s:\n%s", got, want, text)
			}
			if !strings.Contains(text, "\n› "+head+"\n") || strings.Contains(text, "Second note.") {
				t.Fatalf("Details is not folded to one line:\n%s", text)
			}
			// Opened, it is the full text, measured as it is folded: 7 lines.
			open := detailPlain(m.withUnfolded("details"))
			if body := sectionBody(open, "⌄ "+head); len(body) < 7 {
				t.Fatalf("opened Details has %d lines, want 7 or more:\n%s", len(body), open)
			}
		})
	}
}

// hya1st check 2: keyboard focus follows the page. From the title, Tab
// steps Goal's id, Purpose's, Principle's, Details' (open, short), then
// the checks; a long Details' heading is the stop between Principle and
// Acceptance, Enter opens it where it stands and Acceptance follows its
// text; Shift-Tab comes back the same way.
func TestTabFollowsGoalPurposePrincipleDetails(t *testing.T) {
	m, ids := textBoard(t)
	m = openTask(t, m, ids["Mainta"])
	tk, _ := m.selected()
	if got, want := stopsOf(m, tk, ids), "text:Goalid text:Purpos text:Princi details:Detail check:Checkd asked:Askedq asked:Checkd request:Reques resolved:Resolv"; got != want {
		t.Fatalf("stops\n %s\nwant\n %s", got, want)
	}
	m.detailPane = focusPage
	for k, mark := range []string{"Builds on ▸" + ids["Goalid"], "So ▸" + ids["Purpos"], "As ▸" + ids["Princi"], "Seen in ▸" + ids["Detail"]} {
		m = press(m, "tab")
		if text := detailPlain(m); !strings.Contains(text, mark) || strings.Count(text, "▸") != 1 {
			t.Fatalf("Tab %d: no single mark %q:\n%s", k+1, mark, text)
		}
	}
	if m = press(m, "tab"); m.detailPane != focusAcceptance {
		t.Fatalf("Tab after Details: pane %d, want the checks", m.detailPane)
	}
	if m = press(m, "shift+tab"); !strings.Contains(detailPlain(m), "Seen in ▸"+ids["Detail"]) {
		t.Fatalf("Shift-Tab from the checks does not come back to Details:\n%s", detailPlain(m))
	}

	// A long Details: its heading is a stop after Principle's text.
	l, lids := detailsBoard(t, 100, 30, map[string]string{"Long": sevenLines})
	l = openTask(t, l, lids["Long"])
	l.detailPane = focusPage
	if l = press(l, "tab"); !strings.Contains(detailPlain(l), "▸ DETAILS (4) · Enter opens") {
		t.Fatalf("Tab from the title does not reach the folded DETAILS:\n%s", detailPlain(l))
	}
	l = press(l, "enter")
	text := detailPlain(l)
	d, a, s := strings.Index(text, "▸ DETAILS (4) · Enter folds"), strings.Index(text, "Fourth note."), strings.Index(text, "ACCEPTANCE")
	if d < 0 || a < d || s < a || strings.Index(text, "PRINCIPLE") > d {
		t.Fatalf("Enter did not open Details in its place (heading %d, text %d, acceptance %d):\n%s", d, a, s, text)
	}
	if l = press(l, "tab"); l.detailPane != focusAcceptance {
		t.Fatalf("Tab after the open DETAILS: pane %d, want the checks", l.detailPane)
	}
	if l = press(l, "shift+tab"); !strings.Contains(detailPlain(l), "▸ DETAILS (4)") {
		t.Fatalf("Shift-Tab from the checks does not come back to DETAILS:\n%s", detailPlain(l))
	}
}

// hya1st check 2: n and p open the next and previous task on their own
// terms: a short Details open, a long one folded (what the reader opened
// on another page does not carry over).
func TestNextPreviousKeepTheDetailsRule(t *testing.T) {
	m, ids := detailsBoard(t, 100, 30, map[string]string{"A long": sevenLines, "B short": "One note."})
	m = openTask(t, m, ids["A long"])
	m = press(m, "shift+tab")
	for n := 0; n < 10 && !strings.Contains(detailPlain(m), "▸ DETAILS"); n++ {
		m = press(m, "shift+tab")
	}
	m = press(m, "enter")
	if !strings.Contains(detailPlain(m), "Fourth note.") {
		t.Fatalf("Enter did not open the long Details:\n%s", detailPlain(m))
	}
	for _, k := range []string{"n", "p"} {
		m = press(m, k)
		tk, _ := m.selected()
		text := detailPlain(m)
		switch tk.ID {
		case ids["B short"]:
			if !strings.Contains(text, "\nDETAILS\nOne note.") {
				t.Fatalf("%s: short Details not open:\n%s", k, text)
			}
		case ids["A long"]:
			if strings.Contains(text, "Fourth note.") || !strings.Contains(text, "› DETAILS (4)") {
				t.Fatalf("%s: long Details not folded again:\n%s", k, text)
			}
		default:
			t.Fatalf("%s opened %s", k, tk.ID)
		}
	}
}

// hya1st: kkj0mn as Jørgen saw it (statbus hug/tasks 2026-11-24): a
// one-line Goal, a longer Purpose, a two-line Principle, then Details:
// one paragraph that the page draws in 8 lines at 100 columns (folded)
// and 4 at 160 (open). Golden at 100x30 and 160x45.
func TestKkj0mnPageGolden(t *testing.T) {
	for _, c := range []struct {
		w, h  int
		heads string
	}{
		{100, 30, "GOAL|PURPOSE|PRINCIPLE|DETAILS (1)|ACCEPTANCE"},
		{160, 45, "GOAL|PURPOSE|PRINCIPLE|DETAILS|ACCEPTANCE"},
	} {
		w, h := c.w, c.h
		t.Run(fmt.Sprintf("%dx%d", w, h), func(t *testing.T) {
			m := fileBoard(t, "testdata/hya1st/kkj0mn.md", "kkj0mn", w, h)
			text := detailPlain(m)
			if got, want := strings.Join(headings(text), "|"), c.heads; got != want {
				t.Fatalf("headings %s, want %s:\n%s", got, want, text)
			}
			goldenFile(t, fmt.Sprintf("testdata/hya1st/kkj0mn-%dx%d.txt", w, h), ansi.Strip(m.View())+"\n")
		})
	}
}
