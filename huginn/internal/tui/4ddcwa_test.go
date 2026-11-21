package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// 4ddcwa (Jørgen 2026-11-20 11:32Z, request 1 on qpp30y, screen-1130.ans):
// a task's page sits between two rules. The rule under the title gets a
// small ↑ at its left end when the page is scrolled down. Right above the
// global keys, one bottom rule: '↓── ' (the ↓ only when more is below),
// the page's own status ('Task 1 of 1 · 0% · next · previous · Esc back'),
// then the rule on to the edge. It replaces the status line and the blank
// line under it. A message ('ID moved to X', a find's 'match 1 of 3')
// takes the status's place on that rule, as it took the status line's.

// pageFrame is a task's screen cut into its parts: the crumb, the header
// over the top rule, the top rule, the page rows, the bottom rule and the
// keys. It fails when the screen is not framed that way: no blank line
// and nothing else between the page, the bottom rule and the keys.
type pageFrame struct {
	top, bottom, keys string
	page              []string
	above             []string // lines between the page and the bottom rule: a prompt, a confirm
}

func framedPage(t *testing.T, what string, m model, above int) pageFrame {
	t.Helper()
	screen := ansi.Strip(m.View())
	lines := strings.Split(screen, "\n")
	if len(lines) != m.height {
		t.Fatalf("%s: %d lines, want %d\n%s", what, len(lines), m.height, screen)
	}
	head := len(m.detailHeader(mustSelected(t, m)))
	f := pageFrame{top: lines[1+head], bottom: lines[m.height-2], keys: lines[m.height-1]}
	f.page = lines[2+head : m.height-2-above]
	f.above = lines[m.height-2-above : m.height-2]
	rule := strings.Repeat("─", m.width)
	if f.top != rule && f.top != "↑"+rule[len("─"):] {
		t.Fatalf("%s: row %d, under the title, is not the top rule: %q\n%s", what, 2+head, f.top, screen)
	}
	if !strings.HasPrefix(f.bottom, "↓── ") && !strings.HasPrefix(f.bottom, "─── ") {
		t.Fatalf("%s: the row above the keys is not the bottom rule: %q\n%s", what, f.bottom, screen)
	}
	if !strings.HasSuffix(f.bottom, "─") {
		t.Fatalf("%s: the bottom rule does not run to the edge: %q\n%s", what, f.bottom, screen)
	}
	if len(f.page) != m.viewport.Height {
		t.Fatalf("%s: %d page rows between the rules, viewport %d\n%s", what, len(f.page), m.viewport.Height, screen)
	}
	return f
}

// 4ddcwa checks 2 and 3: p87ngv (five questions, a long page) at the top,
// scrolled mid-page and at the end, at 100x30 and 160x45. ↑ on the top
// rule exactly when scrolled down; ↓ on the bottom rule exactly when more
// is below; the status on the bottom rule; the page fills every row
// between the rules, its last row the page's own line, not a filler.
func TestPageFrameArrowsAndStatus(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		t.Run(fmt.Sprintf("%dx%d", w, h), func(t *testing.T) {
			m := p87ngvBoard(t, "testdata/p87ngv/five-questions.md", w, h)
			var golden []string
			for _, at := range []struct {
				name string
				keys []string
				up   bool
				down bool
				pct  string
			}{
				{"at the top", nil, false, true, "0%"},
				{"scrolled mid-page", []string{"J"}, true, true, ""},
				{"at the end", []string{">"}, true, false, "100%"},
			} {
				v := press(m, at.keys...)
				what := fmt.Sprintf("%dx%d %s", w, h, at.name)
				f := framedPage(t, what, v, 0)
				if up := strings.HasPrefix(f.top, "↑"); up != at.up || up != (v.viewport.YOffset > 0) {
					t.Fatalf("%s: ↑ on the top rule %v, want %v (offset %d): %q", what, up, at.up, v.viewport.YOffset, f.top)
				}
				if down := strings.HasPrefix(f.bottom, "↓"); down != at.down || down == v.viewport.AtBottom() {
					t.Fatalf("%s: ↓ on the bottom rule %v, want %v: %q", what, down, at.down, f.bottom)
				}
				status := fmt.Sprintf("Task 1 of 1 · %s · next · previous · Esc back ", at.pct)
				if at.pct == "" {
					status = fmt.Sprintf("Task 1 of 1 · %.0f%% · next · previous · Esc back ", v.viewport.ScrollPercent()*100)
				}
				if got := f.bottom[len("↓── "):]; !strings.HasPrefix(got, status) && !strings.HasPrefix(f.bottom[len("─── "):], status) {
					t.Fatalf("%s: the bottom rule does not carry the status %q: %q", what, status, f.bottom)
				}
				src := strings.Split(ansi.Strip(v.detailText(mustSelected(t, v))), "\n")
				last := v.viewport.YOffset + len(f.page) - 1
				if last >= len(src) || strings.TrimRight(f.page[len(f.page)-1], " ") != strings.TrimRight(src[last], " ") {
					t.Fatalf("%s: the last page row %q is not line %d of the page", what, f.page[len(f.page)-1], last+1)
				}
				// The rows the status line and the blank line took are page rows now.
				if want := h - len(v.detailHeader(mustSelected(t, v))) - 4; v.viewport.Height != want {
					t.Fatalf("%s: the page has %d rows, want %d", what, v.viewport.Height, want)
				}
				golden = append(golden, "== "+what, ansi.Strip(v.View()))
			}
			goldenFile(t, fmt.Sprintf("testdata/4ddcwa/page-frame-%dx%d.txt", w, h), strings.Join(golden, "\n")+"\n")
		})
	}
}

// 4ddcwa check 3 on a short task: the page ends above the bottom rule,
// with no ↑ and no ↓, and the rule stays right above the keys.
func TestPageFrameShortTask(t *testing.T) {
	m := slotBoard(t, false, 0)
	m = press(m, "j", "enter")
	f := framedPage(t, "short", m, 0)
	if strings.HasPrefix(f.top, "↑") || !strings.HasPrefix(f.bottom, "─── Task ") {
		t.Fatalf("short task framed with arrows: top %q bottom %q", f.top, f.bottom)
	}
}

// 4ddcwa: a message takes the status's place on the bottom rule (24403z's
// moved note), and a prompt or confirm sits on its own line above the
// rule, taking a page row while it shows; the rule stays on the keys.
func TestPageFrameMessageAndPrompt(t *testing.T) {
	m := slotBoard(t, false, 0)
	m = press(m, "j", "enter")
	m.detailPane = focusPage
	m = press(m, "t")
	f := framedPage(t, "confirm", m, 1)
	if !strings.HasPrefix(f.above[0], "▶ Confirm take bbbbbb") {
		t.Fatalf("the confirm is not right above the bottom rule: %q", f.above[0])
	}
	m = press(m, "enter")
	f = framedPage(t, "moved", m, 0)
	if want := "─── • bbbbbb moved to IN PROGRESS ─"; !strings.HasPrefix(f.bottom, want) {
		t.Fatalf("the moved note is not on the bottom rule: %q, want %q…", f.bottom, want)
	}

	p := p87ngvBoard(t, "testdata/p87ngv/five-questions.md", 100, 30)
	p = press(p, "/", "q", "u", "e", "s", "t", "i", "o", "n")
	f = framedPage(t, "find", p, 1)
	if !strings.HasPrefix(f.above[0], "/ question") || !strings.HasPrefix(f.bottom, "↓── • match 1 of ") {
		t.Fatalf("find: prompt %q, rule %q", f.above[0], f.bottom)
	}
	p = press(p, "enter")
	f = framedPage(t, "find kept", p, 0)
	if !strings.Contains(f.bottom, " match 1 of ") {
		t.Fatalf("find kept: rule %q", f.bottom)
	}
	goldenFile(t, "testdata/4ddcwa/message-100x30.txt", strings.Join([]string{"== moved", ansi.Strip(m.View()), "== find kept", ansi.Strip(p.View())}, "\n")+"\n")
}

// 4ddcwa (crab 17:37Z): mid-page, a message replaces only the status on
// the bottom rule. The ↓ stays, the page keeps its rows, the top rule
// keeps its ↑; the message is marked as a status by a dim • (xt0qvf
// option A on the task page). Golden at 100x30 and 160x45.
func TestPageFrameMessageReplacesOnlyTheStatus(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		m := p87ngvBoard(t, "testdata/p87ngv/five-questions.md", w, h)
		m = press(m, "J")
		before := framedPage(t, "mid-page", m, 0)
		msg := m
		msg.message = "p87ngv moved to IN PROGRESS" // as movedNote says it (24403z)
		after := framedPage(t, "mid-page with a message", msg, 0)
		want := "↓── " + messageSign + oneLine(plainMarks(msg.message)) + " ─"
		if !strings.HasPrefix(after.bottom, want) {
			t.Fatalf("%dx%d: rule %q, want %q…", w, h, after.bottom, want)
		}
		if strings.Contains(after.bottom, "Task 1 of 1") {
			t.Fatalf("%dx%d: the status stayed beside the message: %q", w, h, after.bottom)
		}
		if after.top != before.top || strings.Join(after.page, "\n") != strings.Join(before.page, "\n") || after.keys != before.keys {
			t.Fatalf("%dx%d: the message moved more than the status", w, h)
		}
		goldenFile(t, fmt.Sprintf("testdata/4ddcwa/message-mid-page-%dx%d.txt", w, h), ansi.Strip(msg.View())+"\n")
	}
}

// 4ddcwa check 3: the page is as tall as its frame from the first frame
// a resize draws, before any key: syncDetail sizes it between the rules,
// the two rows of the old status line and its blank line given to it.
func TestPageFrameHeightAfterResize(t *testing.T) {
	m := p87ngvBoard(t, "testdata/p87ngv/five-questions.md", 100, 30)
	for _, size := range [][2]int{{160, 45}, {100, 30}} {
		m.width, m.height = size[0], size[1]
		m.resize() // no Update: the page height is syncDetail's alone
		// The crumb, the top rule, the bottom rule and the keys.
		if want := m.height - len(m.detailHeader(mustSelected(t, m))) - 4; m.viewport.Height != want {
			t.Fatalf("%dx%d: the page has %d rows after a resize, want %d", size[0], size[1], m.viewport.Height, want)
		}
		framedPage(t, fmt.Sprintf("%dx%d resized", size[0], size[1]), m, 0)
	}
}

// 4ddcwa with q3sh69: a frame line is not a focus sign and never pulses.
// The > of the focused question dims in the pulse's off phase; the two
// rules, arrows and status are the same bytes in both phases.
func TestPageFrameDoesNotPulse(t *testing.T) {
	m := p87ngvBoard(t, "testdata/p87ngv/five-questions.md", 100, 30)
	m = press(m, "tab", "tab", "tab", "tab") // question 5, the page scrolled to it
	on := m
	on.pulse.off = false
	off := m
	off.pulse.off = true
	a, b := strings.Split(on.View(), "\n"), strings.Split(off.View(), "\n")
	head := len(m.detailHeader(mustSelected(t, m)))
	for _, row := range []int{1 + head, m.height - 2} {
		if a[row] != b[row] {
			t.Fatalf("row %d pulses:\non  %q\noff %q", row+1, a[row], b[row])
		}
	}
	if !strings.HasPrefix(ansi.Strip(a[1+head]), "↑") || !strings.HasPrefix(ansi.Strip(a[m.height-2]), "↓") {
		t.Fatalf("not mid-page: %q / %q", ansi.Strip(a[1+head]), ansi.Strip(a[m.height-2]))
	}
	differs := false
	for i := range a {
		differs = differs || a[i] != b[i]
	}
	if !differs {
		t.Skip("no > on screen to pulse")
	}
}
