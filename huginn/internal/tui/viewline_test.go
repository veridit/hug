package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// qh9ep1 checks 1 and 7: line 2 reads 'view: overview  list  board  !lint'
// with v and each view's letter marked, and the state counts sit on it,
// right-aligned after a dim │; the old counts line is gone. At 80 the
// counts drop from the right first.
func TestViewLineCarriesCounts(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m := manyModel(t)
		m.width, m.height = w, 24
		m.resize()
		lines := strings.Split(ansi.Strip(m.View()), "\n")
		line := lines[1]
		if !strings.HasPrefix(line, "view: overview  list  board  !lint") {
			t.Fatalf("at %d line 2 is %q", w, line)
		}
		if !strings.Contains(line, "│ todo ") {
			t.Fatalf("at %d the counts are not on the view line: %q", w, line)
		}
		if strings.TrimRight(line, " ") != line || len([]rune(line)) != w {
			t.Fatalf("at %d the counts are not right-aligned: %q", w, line)
		}
		if strings.HasPrefix(strings.TrimSpace(lines[2]), "todo ") {
			t.Fatalf("at %d the old counts line is still drawn: %q", w, lines[2])
		}
		if w == 80 && (strings.Contains(line, "dropped") || !strings.Contains(line, "todo")) {
			t.Fatalf("at 80 the counts did not drop from the right: %q", line)
		}
		if w == 140 && !strings.Contains(line, "dropped") {
			t.Fatalf("at 140 every count fits: %q", line)
		}
		golden = append(golden, "== "+strconv.Itoa(w), strings.Join(lines[:3], "\n"))
	}
	// The marks: v and each view letter carry their step's key style (v
	// step 1, o l b ! step 2; 911j70, qh9ep1 check 9).
	m := manyModel(t)
	m.width = 140
	m.resize()
	raw := m.viewLine(m.stateCounts())
	for _, k := range []string{"v", "o", "l", "b", "!"} {
		key := m.styles.viewKey
		if k == "v" {
			key = m.styles.viewPrefixKey
		}
		if !strings.Contains(raw, key.Render(k)) {
			t.Fatalf("%q is not marked on the view line: %q", k, raw)
		}
	}
	goldenFile(t, "testdata/header/view-line.txt", strings.Join(golden, "\n")+"\n")
}

// qh9ep1 check 2: vo, vl, vb and v! switch views from every view and from a
// task; bare o, b and ! no longer do (o orders), nor the old i; l stays
// the pane key.
func TestViewPrefixPicksEveryView(t *testing.T) {
	picks := map[string]string{"o": "overview", "l": "list", "b": "board", "!": "lint"}
	for _, from := range []string{"overview", "list", "board", "lint"} {
		for k, want := range picks {
			m := manyModel(t)
			m.setView(from)
			if got := press(m, "v", k); got.view != want || got.pendingV {
				t.Fatalf("from %s v%s: view %s pending %v", from, k, got.view, got.pendingV)
			}
			// l is the pane key (h/l Tab, e3sevr); it steps, not picks.
			if got := press(m, k); k != "l" && got.view != from {
				t.Fatalf("from %s bare %q switched to %s", from, k, got.view)
			}
		}
	}
	for k, want := range picks {
		m := press(manyModel(t), "enter")
		if !m.detail {
			t.Fatal("enter opened no task")
		}
		if got := press(m, "v", k); got.detail || got.view != want {
			t.Fatalf("from a task v%s: detail %v view %s", k, got.detail, got.view)
		}
		if got := press(m, k); !got.detail {
			t.Fatalf("in a task bare %q left it (view %s)", k, got.view)
		}
	}
	for _, gone := range []string{"i"} {
		if got := press(manyModel(t), gone); got.view != "list" {
			t.Fatalf("bare %q switched to %s", gone, got.view)
		}
		if got := press(manyModel(t), "v", gone); got.view != "list" || got.pendingV {
			t.Fatalf("v%s switched to %s", gone, got.view)
		}
	}
}

// qh9ep1 check 3: after v alone the view line shows it waits and the
// footer lists the choices; Esc or an unknown key cancels and changes
// nothing. Since a82j3n only v (solid) and the underlines of o l b !
// change (TestViewLineMarksActiveView, TestViewKeysBlinkAfterV), not the
// whole line.
func TestPendingViewPrefixShowsChoices(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := manyModel(t)
	before := state(m)
	v := press(m, "v")
	if !v.pendingV {
		t.Fatal("v did not wait for a view")
	}
	screen := ansi.Strip(v.View())
	footer := strings.TrimRight(footerLine(v), " ")
	if footer != "v: o overview · l list · b board · ! lint · Esc cancel" {
		t.Fatalf("footer while v waits: %q", footer)
	}
	lit, plain := v, m
	lit.styles, plain.styles = newTheme(), newTheme()
	lit.styles.renderer.SetColorProfile(termenv.TrueColor)
	plain.styles.renderer.SetColorProfile(termenv.TrueColor)
	if lit.viewLine(lit.stateCounts()) == plain.viewLine(plain.stateCounts()) {
		t.Fatal("the view line does not show that v waits")
	}
	for _, k := range []string{"esc", "x", "j", "q", "e", "S", "?"} {
		after := press(v, k)
		if after.pendingV {
			t.Fatalf("%s left v waiting", k)
		}
		if k == "?" {
			if !after.help {
				t.Fatal("? while v waits opened no help")
			}
			continue
		}
		after.message = ""
		if state(after) != before {
			t.Fatalf("v then %s changed something:\n%s", k, after.View())
		}
	}
	if got := press(v, "x"); !strings.Contains(got.message, "v x is no view; nothing changed") {
		t.Fatalf("an unknown key after v says %q", got.message)
	}
	// From a task too: Esc cancels the prefix, not the task.
	in := press(manyModel(t), "enter", "v", "esc")
	if !in.detail || in.pendingV {
		t.Fatalf("v Esc in a task: detail %v pending %v", in.detail, in.pendingV)
	}
	goldenFile(t, "testdata/header/view-pending.txt", screen+"\n")
}

// qh9ep1 check 5: help lists the prefix keys as two-key rows, and its find
// reaches them by either key, the keys together, or the name.
func TestHelpListsPrefixRows(t *testing.T) {
	m := manyModel(t)
	m.width = 140
	m.resize()
	help := strings.Join(strings.Fields(ansi.Strip(press(m, "?").helpText())), " ")
	for _, want := range []string{"v o view: overview", "v l view: list", "v b view: board", "v ! view: lint"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help lacks %q:\n%s", want, help)
		}
	}
	for query, want := range map[string]string{"vo": "view: overview", "vb": "view: board", "v !": "view: lint", "view: board": "view: board", "!": "view: lint", "view: lint": "view: lint"} {
		h := press(m, "?", "/")
		for _, r := range query {
			h = press(h, string(r))
		}
		if got := ansi.Strip(h.helpText()); !strings.Contains(got, want) {
			t.Fatalf("find %q in help lacks %q:\n%s", query, want, got)
		}
	}
}
