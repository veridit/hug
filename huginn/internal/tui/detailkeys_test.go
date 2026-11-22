package tui

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// x1ra0t check 1: in a task, J and K scroll half a page and never change
// the rank (Jørgen 14:49 pressed them in zhvkmj's detail to scroll and
// moved it above vzghj8).
func TestDetailJKScrollAndNeverMove(t *testing.T) {
	m, main := focusModel(t)
	before := rankOrder(t, m)
	m.detailPane = focusPage
	m.viewport.GotoTop()
	half := m.viewport.Height / 2
	m = press(m, "J")
	if !m.detail {
		t.Fatal("J left the task")
	}
	if m.viewport.YOffset != half {
		t.Fatalf("J scrolled to %d, want half a page (%d)", m.viewport.YOffset, half)
	}
	m = press(m, "J", "K")
	if m.viewport.YOffset != half {
		t.Fatalf("J K came back to %d, want %d", m.viewport.YOffset, half)
	}
	if strings.Contains(m.message, "moved") {
		t.Fatalf("message says a move: %q", m.message)
	}
	// Every focus: none of them moves the task.
	for _, pane := range panesOf(m, main) {
		mm := m
		mm.detailPane = pane
		mm.syncDetail()
		mm = press(mm, "K", "K", "J")
		if mm.confirm != "" || strings.Contains(mm.message, "moved") {
			t.Fatalf("pane %d: J/K acted: confirm %q message %q", pane, mm.confirm, mm.message)
		}
	}
	if after := rankOrder(t, m); before != after {
		t.Fatalf("rank changed from inside the task:\nbefore %v\nafter  %v", before, after)
	}
	// The key bar names it, so the reader need not guess.
	m.width = 140
	m.resize()
	if bar := ansi.Strip(m.keyBar()); !strings.Contains(bar, "J/K ½ page") {
		t.Fatalf("key bar lacks J/K: %q", bar)
	}
}

// e3sevr check 1 (reverses x1ra0t check 2): J and K on a list row page,
// they never move the task; m grabs to move.
func TestRowJKPageNeverMove(t *testing.T) {
	m := manyModel(t)
	before := shown(m)
	m = press(m, "J")
	if m.cursor != m.bodyHeight() || !slices.Equal(before, shown(m)) || m.confirm != "" {
		t.Fatalf("J: cursor %d (want a page, %d), rows changed %v", m.cursor, m.bodyHeight(), !slices.Equal(before, shown(m)))
	}
	m = press(m, "K")
	if m.cursor != 0 || !slices.Equal(before, shown(m)) {
		t.Fatalf("K: cursor %d, rows changed %v", m.cursor, !slices.Equal(before, shown(m)))
	}
	if m = press(m, "m"); !m.grab.active {
		t.Fatal("m does not grab")
	}
}

// x1ra0t check 3: every key that writes in the detail is named in its key
// bar, at every width the bar is drawn for, and asks before it writes.
// The writers are enumerated here; a new one must join the bar and this
// list (TestFocusHintsAreHonest presses each hint the bar shows).
func TestDetailWritingKeysAreNamed(t *testing.T) {
	writers := map[string]string{"t": "t/d/x/e", "d": "t/d/x/e", "x": "t/d/x/e", "e": "t/d/x/e", " ": "Space", "-": "-", "N": "N", "S": "S", "V": "V"}
	for _, w := range []int{80, 100, 140} {
		m, main := focusModel(t)
		m.width = w
		m.resize()
		for _, pane := range panesOf(m, main) {
			m.detailPane = pane
			m.syncDetail()
			// The bar as drawn, after the help model cut what does not fit.
			bar := ansi.Strip(m.keyBar())
			for k, label := range writers {
				// Space and - write only with the checks focused, and
				// there they are named; elsewhere they only explain.
				if (k == " " || k == "-") && pane != focusAcceptance {
					after := press(m, k)
					if after.confirm != "" || after.prompt != "" {
						t.Fatalf("pane %d: %q asks to write with no check marked", pane, k)
					}
					continue
				}
				if k == "-" && w < 120 || k == "N" && w < 100 || (k == "S" || k == "V") && w < 140 {
					// - is named in the ACCEPTANCE heading below 120
					// (sfa8w4); N opens a prompt that says what it makes.
					continue
				}
				if pane == focusPage && label == "t/d/x/e" {
					// On the task itself each writer has its own hint,
					// those that apply to its state (e3sevr check 9).
					tk, _ := m.selected()
					if k == "d" && tk.State != "doing" || k == "t" && tk.State != "todo" {
						continue
					}
				}
				if !barNames(m, k) {
					t.Fatalf("width %d pane %d: %q writes but the bar shows %q", w, pane, k, bar)
				}
				before := rankOrder(t, m)
				after := press(m, k)
				if after.confirm == "" && after.prompt == "" && after.message == "" {
					t.Fatalf("width %d pane %d: %q acted without a confirm, prompt or message", w, pane, k)
				}
				if before != rankOrder(t, m) {
					t.Fatalf("%q changed the rank before any confirm", k)
				}
			}
		}
	}
}

// panesOf is the panes t's focus ring stops in, each once, in ring order.
func panesOf(m model, t task.Task) []int {
	var out []int
	seen := map[int]bool{}
	for _, s := range m.focusRing(t) {
		if !seen[s.pane] {
			seen[s.pane] = true
			out = append(out, s.pane)
		}
	}
	return out
}

// barNames says whether the key bar as drawn names key k: as a label
// before its words, or marked inside a word (e3sevr check 14).
func barNames(m model, k string) bool {
	_, shown := m.renderHintsShown(m.barHints(), m.width)
	for _, it := range shown {
		if it.label == "" && strings.Contains(it.words, mk(k)) || it.label == k || k == " " && it.label == "Space" || it.label == "-" && k == "-" {
			return true
		}
	}
	return false
}
