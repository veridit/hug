package tui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Find in a task (4s3g5q, Jørgen 23:29): in a task's detail or a file it
// shows, / finds the typed text in what is on the page, like less or vim,
// never tasks. Every match is tinted, the first at or after the top of
// the page scrolls into view, and n / N step through them, wrapping. The
// list's / (tasks, search.go) is untouched; a find never becomes a list
// filter: it is its own prompt, "find", and never writes m.query.
//
// Smart case, as vim: all lower case ignores case; an upper-case letter
// makes it exact. A match is found in the page's lines as drawn, so a
// match a wrap split across two lines is found as two pieces, each
// tinted (wrap keeps the words; the find looks for the text with the
// line ends taken out).

const findPrompt = "find"

// findSpan is the cells a match covers on one drawn line.
type findSpan struct{ line, from, to int }

// findHit is one occurrence, even when wrapping splits it into spans.
type findHit struct {
	line  int
	spans []findSpan
}

// findState is the find of the open page: what was typed, where it is.
type findState struct {
	text string
	hits []findHit
	at   int // the hit n / N stand on
}

// openFind starts a find on the page: the old one is forgotten.
func (m *model) openFind() {
	m.find = findState{}
	m.openInput(findPrompt, "")
	m.input.Prompt = "/ "
}

// findTyped finds what is typed so far and shows the first match at or
// after the top of the page.
func (m *model) findTyped() {
	m.find.text = m.input.Value()
	m.find.at = 0
	if !m.glance {
		m.openFoldsHolding(m.find.text)
	}
	m.refind()
	start := m.viewport.YOffset
	if start == 0 {
		start = -1 // the fixed title comes before the viewport
	}
	for i, h := range m.find.hits {
		if h.line >= start {
			m.find.at = i
			break
		}
	}
	m.viewport.SetContent(m.tintHits(m.pageText()))
	m.showHit()
	m.message = m.findSays()
}

// refind finds the text in the page as drawn now and tints it there;
// syncDetail calls it after the page is drawn again (a resize, a write).
func (m *model) refind() {
	body := m.pageText()
	y := m.viewport.YOffset
	defer m.viewport.SetYOffset(y)
	m.find.hits = findIn(body, m.find.text)
	if !m.glance {
		if t, ok := m.selected(); ok {
			titleHits := findIn(m.detailTitle(t), m.find.text)
			for i := range titleHits {
				titleHits[i].line = -1
				for j := range titleHits[i].spans {
					titleHits[i].spans[j].line = -1
				}
			}
			m.find.hits = append(titleHits, m.find.hits...)
		}
	}
	m.find.at = min(m.find.at, max(0, len(m.find.hits)-1))
	m.viewport.SetContent(m.tintHits(body))
}

// pageText is the page the viewport shows: the file in a file, the
// task's detail in a task.
func (m model) pageText() string {
	if m.glance {
		return m.glanceBody
	}
	if t, ok := m.selected(); ok {
		return m.detailText(t)
	}
	return ""
}

// findIn is every match of text in body's lines, in reading order. A
// match may run over a line end (a wrapped line), but is still one hit.
func findIn(body, text string) []findHit {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	fold := strings.ToLower(text) == text // smart case
	lines := strings.Split(body, "\n")
	var flat strings.Builder
	// where is each rune's line and cells (from, to): wide runes take
	// two, and ansi.Cut counts cells.
	type at struct{ line, from, to int }
	var where []at
	widths := make([]int, len(lines)) // each line's text, without its padding
	for n, l := range lines {
		cell := 0
		// The page pads each line to its width: the padding is not text,
		// so a wrap's 'row' + padding + 'layout' reads 'row layout'.
		for _, r := range strings.TrimRight(ansi.Strip(l), " ") {
			w := ansi.StringWidth(string(r))
			if fold {
				r = unicode.ToLower(r)
			}
			flat.WriteRune(r)
			where = append(where, at{n, cell, cell + w})
			cell += w
		}
		// A line end reads as one space, so 'row lay' finds 'row' at one
		// line's end and 'lay…' at the next one's start.
		widths[n] = cell
		flat.WriteRune(' ')
		where = append(where, at{n, cell, cell})
	}
	hay := []rune(flat.String())
	needle := []rune(text)
	var hits []findHit
	for i := 0; i+len(needle) <= len(hay); i++ {
		if string(hay[i:i+len(needle)]) != string(needle) {
			continue
		}
		start, end := where[i], where[i+len(needle)-1]
		hit := findHit{line: start.line}
		for line := start.line; line <= end.line; line++ {
			from, to := 0, widths[line]
			if line == start.line {
				from = start.from
			}
			if line == end.line {
				to = end.to
			}
			if to > from {
				hit.spans = append(hit.spans, findSpan{line, from, to})
			}
		}
		if len(hit.spans) > 0 {
			hits = append(hits, hit)
		}
		i += len(needle) - 1
	}
	return hits
}

// tintHits draws body with every hit tinted, the current one stronger.
func (m model) tintHits(body string) string {
	if len(m.find.hits) == 0 {
		return body
	}
	lines := strings.Split(body, "\n")
	type tintedSpan struct {
		findSpan
		current bool
	}
	byLine := map[int][]tintedSpan{}
	for i, h := range m.find.hits {
		for _, span := range h.spans {
			byLine[span.line] = append(byLine[span.line], tintedSpan{span, i == m.find.at})
		}
	}
	for line, idx := range byLine {
		if line < 0 || line >= len(lines) {
			continue // title hits are drawn separately in the fixed band
		}
		l := lines[line]
		// Right to left, so earlier cells keep their place.
		for k := len(idx) - 1; k >= 0; k-- {
			h := idx[k]
			style := m.styles.column
			if h.current {
				style = m.styles.columnCursor
			}
			cell := ansi.Strip(ansi.Cut(l, h.from, h.to))
			l = ansi.Cut(l, 0, h.from) + style.Render(cell) + ansi.Cut(l, h.to, lipgloss.Width(l))
		}
		lines[line] = l
	}
	return strings.Join(lines, "\n")
}

// The title stays above the scrolling page. Negative lines identify its
// matches without changing viewport offsets or repeating the title.
func (m model) tintFindTitle(title string) string {
	if m.glance {
		return title
	}
	var hits []findHit
	at := -1
	for i, h := range m.find.hits {
		if h.line != -1 {
			continue
		}
		if i == m.find.at {
			at = len(hits)
		}
		spans := append([]findSpan(nil), h.spans...)
		for j := range spans {
			spans[j].line = 0
		}
		hits = append(hits, findHit{line: 0, spans: spans})
	}
	m.find.hits, m.find.at = hits, at
	return m.tintHits(title)
}

// showHit scrolls the current hit into view, a few lines from the top,
// and says where it is.
func (m *model) showHit() {
	if len(m.find.hits) == 0 {
		return
	}
	h := m.find.hits[m.find.at]
	if h.line < m.viewport.YOffset || h.line >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(max(0, h.line-2))
	}
}

// stepHit is n (1) and N (-1): the next or previous match, wrapping.
func (m *model) stepHit(step int) {
	n := len(m.find.hits)
	if n == 0 {
		return
	}
	m.find.at = (m.find.at + step + n) % n
	m.viewport.SetContent(m.tintHits(m.pageText()))
	m.showHit()
	m.message = m.findSays()
}

// openFoldsHolding opens each folded long section of the page whose text
// holds text, as a find matches it (8a0ybb): a find reaches all the page,
// a folded section too, and shows it open. A section opened stays open.
func (m *model) openFoldsHolding(text string) {
	t, ok := m.selected()
	if !ok || strings.TrimSpace(text) == "" {
		return
	}
	changed := false
	for _, f := range m.folds(t) {
		if m.unfolded[f.key] {
			continue
		}
		open := m.withUnfolded(f.key)
		if len(findIn(open.foldText(t, f.key), text)) > 0 {
			if m.unfolded == nil {
				m.unfolded = map[string]bool{}
			}
			m.unfolded[f.key], changed = true, true
		}
	}
	if changed {
		y := m.viewport.YOffset
		m.syncDetail()
		m.viewport.SetYOffset(y)
	}
}

// withUnfolded is m with the long section key open, the rest as they are.
func (m model) withUnfolded(key string) model {
	open := map[string]bool{key: true}
	for k, v := range m.unfolded {
		open[k] = v
	}
	m.unfolded = open
	return m
}

// findSays is the footer of a find: 'match K of M · n next · N previous ·
// Esc clear', or with no match '0 matches in ID · Enter searches all
// tasks', so the task search stays one key away.
func (m model) findSays() string {
	if strings.TrimSpace(m.find.text) == "" {
		return "find in this " + m.pageNoun() + ": type · Esc back"
	}
	if len(m.find.hits) == 0 {
		return fmt.Sprintf("0 matches in %s · Enter searches all tasks", m.pageName())
	}
	return fmt.Sprintf("match %d of %d · %sext · %s previous · Esc clear", m.find.at+1, len(m.find.hits), mk("n"), mk("N"))
}

func (m model) pageNoun() string {
	if m.glance {
		return "file"
	}
	return "task"
}

// pageName names the page: the task's id, or the file in a file.
func (m model) pageName() string {
	if t, ok := m.selected(); ok {
		if m.glance {
			name, _, _ := m.glanceFile(t)
			return oneLine(name)
		}
		return t.ID
	}
	return "this " + m.pageNoun()
}

// updateFind handles a key in the find prompt: Enter keeps the matches
// for n / N (or, with none, runs the task search with the text); Esc,
// in back(), clears them. Typing finds as it goes.
func (m model) updateFind(k string) (model, bool) {
	switch k {
	case "enter":
		m.prompt = ""
		m.input.Blur()
		if len(m.find.hits) == 0 && strings.TrimSpace(m.find.text) != "" {
			// No match here: the task search, today's /, with the text.
			text := m.find.text
			m.clearFind()
			m.detail, m.glance = false, false
			m.openSearch(text)
			m.input.SetValue(text)
			m.searchTyped()
			return m, true
		}
		m.message = m.findSays()
		return m, true
	case "tab", "shift+tab":
		// Tab keeps a task search as a list filter; a find has no list
		// to filter (4s3g5q check 7), so it does nothing here.
		return m, true
	case "down", "ctrl+n":
		m.stepHit(1)
		return m, true
	case "up", "ctrl+p":
		m.stepHit(-1)
		return m, true
	}
	return m, false
}

// clearFind takes the tints out and forgets the find; the page stays
// where it is scrolled.
func (m *model) clearFind() {
	if m.find.text == "" && len(m.find.hits) == 0 {
		return
	}
	y := m.viewport.YOffset
	m.find = findState{}
	m.viewport.SetContent(m.pageText())
	m.viewport.SetYOffset(y)
}
