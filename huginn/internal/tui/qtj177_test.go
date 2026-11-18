package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// qtj177 check 1 (Jørgen 02:31, zhvkmj's detail on v0.7.52): after H/L
// or o on the rows, a task opened, or two stacked, draws no list hint:
// no "H/L column", no "order by", at 80 and 140. Back on the rows the
// next H shows it again.
func TestDetailDrawsNoColumnHint(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m, ids := idBoard(t)
		m.width, m.height = w, 24
		m.resize()
		m.setView("list")
		m.clamp()
		m = selectRow(t, m, ids["Main"])
		m = press(m, "L", "L", "o")
		if rows := plainMarks(ansi.Strip(m.View())); !strings.Contains(rows, "order by") {
			t.Fatalf("at %d the rows lack the column hint:\n%s", w, rows)
		}
		one := press(m, "enter")
		two := press(one, "shift+tab", "shift+tab", "enter") // the stage's id: a second task on the stack
		if len(two.detailStack) == 0 {
			t.Fatalf("at %d no second task stacked", w)
		}
		for name, d := range map[string]model{"one": one, "two": two} {
			screen := plainMarks(ansi.Strip(d.View()))
			if strings.Contains(screen, "H/L column") || strings.Contains(screen, "order by") || strings.Contains(screen, "H/L pick") {
				t.Fatalf("at %d, %s open, a list hint shows:\n%s", w, name, screen)
			}
		}
		back := press(two, "esc", "esc")
		if back.detail {
			t.Fatalf("at %d Esc Esc did not reach the rows", w)
		}
		if rows := plainMarks(ansi.Strip(press(back, "H").View())); !strings.Contains(rows, "order by") {
			t.Fatalf("at %d H on the rows again shows no hint:\n%s", w, rows)
		}
		// The footer and bar of two stacked; ids are random: each becomes
		// its title, as wide.
		lines := strings.Split(ansi.Strip(two.View()), "\n")
		text := strings.Join(lines[len(lines)-3:], "\n")
		for title, id := range ids {
			text = strings.ReplaceAll(text, id, (strings.ToUpper(title) + "######")[:6])
		}
		golden = append(golden, "== two stacked at "+strconv.Itoa(w), text)
	}
	text := strings.Join(golden, "\n") + "\n"
	goldenFile(t, "testdata/detail/no-column-hint.txt", text)
}

// shownKeys are the keys a footer or key bar item names: its label split
// on "/" and spaces (j/k, H/L, t/d/x/e, "h/l Tab"), or the letter marked
// in its word.
func shownKeys(it hintItem) []string {
	if it.label == "" {
		var out []string
		for w := it.words; ; {
			i := strings.Index(w, markOpen)
			if i < 0 {
				return out
			}
			w = w[i+len(markOpen):]
			r := []rune(w)
			out = append(out, string(r[0]))
		}
	}
	names := map[string]string{"Enter": "enter", "Esc": "esc", "Space": " ", "Tab": "tab", "⌫": "backspace", "PgUp": "pgup", "PgDn": "pgdown", "F1": "f1", "↑↓": "down", "^n": "ctrl+n", "^p": "ctrl+p", "</>": ">", "< >": ">", "Alt-Enter": "alt+enter", "u/Esc": "u"}
	if k, ok := names[it.label]; ok {
		return []string{k}
	}
	var out []string
	for _, f := range strings.Fields(it.label) {
		if k, ok := names[f]; ok {
			out = append(out, k)
			continue
		}
		for _, p := range strings.Split(f, "/") {
			if k, ok := names[p]; ok {
				out = append(out, k)
			} else if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// qtj177 check 2: in every place (list, overview, board, lint, a task in
// each focus, a file, each prompt), each key the key bar names does
// something there: changes what is shown or starts a command. A hint
// whose key is dead there fails, naming the place.
func TestEveryShownKeyActsWhereShown(t *testing.T) {
	// Words that are no key: "letter" in the sort builder, "type" where
	// typing narrows.
	words := map[string]bool{"letter": true, "type": true}
	for name, m := range helpContexts(t) {
		for _, w := range []int{80, 140} {
			m.width = w
			m.resize()
			m.syncDetail()
			_, shown := m.renderHintsShown(m.barHints(), m.width)
			for _, it := range shown {
				if words[it.label] {
					continue
				}
				for _, k := range shownKeys(it) {
					if k == "q" {
						if _, cmd := key(m, "q"); cmd == nil {
							t.Errorf("%s at %d: q shown, does not quit", name, w)
						}
						continue
					}
					if _, ok := pressed(m, k); !ok {
						t.Errorf("%s at %d: %q shown (%q %q) does nothing there", name, w, k, it.label, plainMarks(it.words))
					}
				}
			}
		}
	}
}

// footerLine is the footer as drawn: the line above the input and key bar.
func footerLine(m model) string {
	lines := strings.Split(m.View(), "\n")
	return ansi.Strip(lines[len(lines)-3])
}
