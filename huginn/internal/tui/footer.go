package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// A key letter marked inside a word (e3sevr, Jørgen 00:44: "it should be
// *n*ext, not a repeat but a highlight"). Text that goes through m.message
// or a status line carries the mark as two private-use runes around the
// letter; renderMarks draws it in the key colour, underlined. safe() keeps
// them (they are not control characters) and no task text uses them.
const (
	markOpen  = "\ue000"
	markClose = "\ue001"
)

// mk marks letter: mk("n") + "ext".
func mk(letter string) string { return markOpen + letter + markClose }

// plainMarks is s without its marks: what the screen reads, plain.
func plainMarks(s string) string {
	return strings.NewReplacer(markOpen, "", markClose, "").Replace(s)
}

// renderMarks draws s in base with each marked letter in the key colour,
// underlined.
func (m model) renderMarks(s string, base lipgloss.Style) string {
	if !strings.Contains(s, markOpen) {
		return base.Render(s)
	}
	key := m.styles.accent.Underline(true)
	var b strings.Builder
	for s != "" {
		at := strings.Index(s, markOpen)
		if at < 0 {
			b.WriteString(base.Render(s))
			break
		}
		if at > 0 {
			b.WriteString(base.Render(s[:at]))
		}
		s = s[at+len(markOpen):]
		end := strings.Index(s, markClose)
		if end < 0 {
			b.WriteString(key.Render(s))
			break
		}
		b.WriteString(key.Render(s[:end]))
		s = s[end+len(markClose):]
	}
	return b.String()
}

// markedWord is desc with key marked inside it, when key is one letter
// that the word holds: "e" "edit" reads "edit" with e marked. An upper-case
// key marks the word's first letter in upper case ("N" "new" reads "New"),
// so Shift still shows. ok is false when the key keeps its label (Enter,
// Esc, j/k, x for drop).
func markedWord(key, desc string) (string, bool) {
	r, size := utf8.DecodeRuneInString(key)
	if size != len(key) || !unicode.IsLetter(r) || desc == "" {
		return "", false
	}
	first, fsize := utf8.DecodeRuneInString(desc)
	if unicode.IsUpper(r) {
		if unicode.ToLower(r) != first {
			return "", false
		}
		return mk(string(r)) + desc[fsize:], true
	}
	word := desc
	if sp := strings.IndexByte(desc, ' '); sp >= 0 {
		word = desc[:sp]
	}
	at := strings.IndexRune(word, r)
	if at < 0 {
		return "", false
	}
	return desc[:at] + mk(string(r)) + desc[at+size:], true
}

// splitHints turns "t/d/x/e take/done/drop/edit" into one hint per key,
// so each can be marked in its own word.
func splitHints(hints []bkey.Binding) []bkey.Binding {
	var out []bkey.Binding
	for _, h := range hints {
		ks, ds := strings.Split(h.Help().Key, "/"), strings.Split(h.Help().Desc, "/")
		if len(ks) > 1 && len(ks) == len(ds) && h.Help().Key != "/" {
			one := true
			for _, k := range ks {
				if utf8.RuneCountInString(k) != 1 {
					one = false
				}
			}
			if one {
				for i := range ks {
					out = append(out, hint(ks[i], ds[i]))
				}
				continue
			}
		}
		out = append(out, h)
	}
	return out
}

// hintItem is one hint as the footer writes it: its label (empty when the
// key is marked in its word) and its words, marks included.
type hintItem struct{ label, words string }

func itemsOf(hints []bkey.Binding) []hintItem {
	var out []hintItem
	for _, h := range splitHints(hints) {
		if !h.Enabled() {
			continue
		}
		if w, ok := markedWord(h.Help().Key, h.Help().Desc); ok {
			out = append(out, hintItem{"", w})
			continue
		}
		out = append(out, hintItem{h.Help().Key, h.Help().Desc})
	}
	return out
}

// lastWord is the last space-separated word of s, plain.
func lastWord(s string) string {
	f := strings.Fields(plainMarks(s))
	if len(f) < 2 {
		return ""
	}
	return f[len(f)-1]
}

// hintText is the footer for hints: items joined by " · ", neighbours that
// share their last word joined by ", " with that word written once at the
// end and "-" on the earlier ones (Jørgen 00:45: "next-, previous file").
// Marks stay in; renderHints draws them. Width cuts at an item boundary,
// ending in " …" as before.
func (m model) renderHints(hints []bkey.Binding, width int) string {
	s, _ := m.renderHintsShown(hints, width)
	return s
}

// renderHintsShown is renderHints and the items it drew, in order (the
// rest were cut for width).
func (m model) renderHintsShown(hints []bkey.Binding, width int) (string, []hintItem) {
	items := itemsOf(hints)
	var shown []hintItem
	var groups [][]hintItem
	for _, it := range items {
		if n := len(groups); n > 0 {
			prev := groups[n-1][len(groups[n-1])-1]
			if w := lastWord(it.words); w != "" && w == lastWord(prev.words) {
				groups[n-1] = append(groups[n-1], it)
				continue
			}
		}
		groups = append(groups, []hintItem{it})
	}
	key, desc := m.styles.accent, m.styles.dim
	sep := desc.Render(" · ")
	var b strings.Builder
	total := 0
	for gi, g := range groups {
		var part strings.Builder
		if gi > 0 {
			part.WriteString(sep)
		}
		for i, it := range g {
			words := it.words
			if i < len(g)-1 {
				w := lastWord(words)
				words = strings.TrimSuffix(strings.TrimRight(words, " "), w)
				words = strings.TrimRight(words, " ") + "-"
			}
			if i > 0 {
				part.WriteString(desc.Render(", "))
			}
			if it.label != "" {
				part.WriteString(key.Render(it.label) + desc.Render(" "))
			}
			part.WriteString(m.renderMarks(words, desc))
		}
		w := lipgloss.Width(part.String())
		if width > 0 && total+w > width {
			tail := " " + desc.Render("…")
			if total+lipgloss.Width(tail) < width {
				b.WriteString(tail)
			}
			break
		}
		total += w
		b.WriteString(part.String())
		shown = append(shown, g...)
	}
	return b.String(), shown
}
