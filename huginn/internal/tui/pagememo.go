package tui

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// pageMemo keeps what a task's page costs to work out, for the next key
// and frame (1tjq11, Jørgen 2026-11-22 21:37Z: on p39w4a, 668 KB with 759
// notes, one j took 0.8 s, since every key split the file into sections,
// scanned all its text for ids, links and file names and rendered its
// markdown again, several times over).
//
// One memo, its slots each kept for exactly the inputs it was worked out
// from, compared in full on every use: a slot whose inputs differ is
// worked out again, never shown stale.
//
//   - facts: the sections, the task:ID links and the long sections'
//     counts, for the task's parsed file: the *task.Document the load
//     made. Every load parses every file again (loadSnapshot, or doc on
//     first use), and apply sets the board and the files together, so a
//     change on disk, to this task or any other, is a new Document. A
//     task whose file could not be read has none and is worked out every
//     time.
//   - md: a text rendered as markdown, for the text, the width and the
//     theme (dark background, colour profile).
//   - ids: the task ids a text names, for the text, whether its task:ID
//     are links, the task and the board.
//   - words: the words of a text that may name a file, for the text.
//     Whether each is a file is decided on every use (repoFile).
//   - files: the files a text names, for the text, the task's folder
//     and files and the checkout, until the oldest answer from the
//     checkout it used goes stale (repoFileFor, as repoFile's own).
//   - scans: a styled text's visible part and the words in it that may
//     name a task or a file, for the text (markSpan).
//   - wraps: a piece of the page wrapped, for the piece and the width.
//   - lines: the page as plain lines, for the page drawn.
//
// What the focus marks, the open box, the time ago and anything else
// that changes between keys is drawn fresh on every frame from these.
// Model copies share the memo; a model without one (nil) works it all
// out every time.
type pageMemo struct {
	mu    sync.Mutex
	facts struct {
		doc *task.Document
		v   *pageFacts
	}
	md    sizedCache[mdKey, string]
	ids   sizedCache[idsKey, []idStop]
	words sizedCache[string, []fileWord]
	scans sizedCache[string, *visibleText]
	files sizedCache[filesKey, fileLinks]
	wraps sizedCache[wrapKey, string]
	lines struct {
		page string
		v    []string
	}
}

func newPageMemo() *pageMemo { return &pageMemo{} }

// boardSerial numbers the loads: apply gives each its own, so a memo
// keyed on it never mistakes one load's board for another's, in this
// model or a copy of it.
var boardSerial atomic.Int64

// pageFacts are a task file's sections and what follows from them alone
// (and the board, for links).
type pageFacts struct {
	sections []detailSection
	links    []linkDetail
	// The long sections (folds): their text trimmed, and how many
	// paragraphs or entries each holds.
	details, notes, history                         string
	detailsParagraphs, notesEntries, historyEntries int
}

type mdKey struct {
	text    string
	width   int
	dark    bool
	profile termenv.Profile
}

type idsKey struct {
	text   string
	linked bool
	id     string
	board  int64
}

// sizedCache is a map that empties itself once what it holds passes
// sizedCacheMax bytes, so a long session does not grow it without end;
// a 700 KB task's page fits several times over. A key is put cloned
// (strings.Clone): a key cut from a page would keep the whole page.
type sizedCache[K comparable, V any] struct {
	m     map[K]V
	bytes int
}

const sizedCacheMax = 16 << 20

func (c *sizedCache[K, V]) get(k K) (V, bool) {
	v, ok := c.m[k]
	return v, ok
}

func (c *sizedCache[K, V]) put(k K, size int, v V) {
	if c.m == nil || c.bytes+size > sizedCacheMax {
		c.m, c.bytes = map[K]V{}, 0
	}
	c.m[k] = v
	c.bytes += size
}

// facts are t's page facts, worked out once per parsed file.
func (m model) facts(t task.Task) *pageFacts {
	d := m.doc(t)
	if m.page == nil || d == nil {
		return m.workFacts(t, d)
	}
	m.page.mu.Lock()
	if m.page.facts.v != nil && m.page.facts.doc == d {
		v := m.page.facts.v
		m.page.mu.Unlock()
		return v
	}
	m.page.mu.Unlock()
	v := m.workFacts(t, d)
	m.page.mu.Lock()
	m.page.facts.doc, m.page.facts.v = d, v
	m.page.mu.Unlock()
	return v
}

func (m model) workFacts(t task.Task, d *task.Document) *pageFacts {
	f := &pageFacts{sections: parseSections(t, d)}
	f.links = m.linksOf(t, f.sections)
	text := func(name string) string { return strings.TrimSpace(sectionText(f.sections, name)) }
	f.details, f.notes, f.history = text("details"), text("notes"), text("history")
	f.detailsParagraphs, f.notesEntries, f.historyEntries = paragraphs(f.details), entries(f.notes), entries(f.history)
	return f
}

// markdown is s rendered as markdown at the page's width in its theme.
func (m model) markdown(s string) string {
	dark, profile := m.styles.renderer.HasDarkBackground(), m.styles.renderer.ColorProfile()
	if m.page == nil {
		return renderMarkdown(s, m.width, dark, profile)
	}
	key := mdKey{text: s, width: m.width, dark: dark, profile: profile}
	m.page.mu.Lock()
	out, ok := m.page.md.get(key)
	m.page.mu.Unlock()
	if ok {
		return out
	}
	out = renderMarkdown(s, m.width, dark, profile)
	m.page.mu.Lock()
	key.text = strings.Clone(key.text)
	m.page.md.put(key, len(s)+len(out), out)
	m.page.mu.Unlock()
	return out
}

// idsIn are the tasks text names by id or alias, in order, each once,
// never t itself. linked: the text is a section whose task:ID is a link
// (links), so a task:ID there is its Links row, not a stop here. What
// it returns is the memo's: read it, never change it.
func (m model) idsIn(t task.Task, text string, linked bool) []idStop {
	if m.page == nil {
		return m.scanIDs(t, text, linked)
	}
	key := idsKey{text: text, linked: linked, id: t.ID, board: m.board}
	m.page.mu.Lock()
	out, ok := m.page.ids.get(key)
	m.page.mu.Unlock()
	if !ok {
		out = m.scanIDs(t, text, linked)
		m.page.mu.Lock()
		key.text = strings.Clone(key.text)
		m.page.ids.put(key, len(text), out)
		m.page.mu.Unlock()
	}
	return out
}

// fileWordsOf are the words of s that may name a file, each the first
// time it stands there (as written, with or without file:), worked out
// once per text. Which file a word names follows from what it shows and
// whether file: was written, so a word again names what it named the
// first time: fileLinksIn, which keeps each link once, asks once.
func (m model) fileWordsOf(s string) []fileWord {
	if m.page == nil {
		return fileWords(s) // as before the memo: the tests' oracle
	}
	m.page.mu.Lock()
	out, ok := m.page.words.get(s)
	m.page.mu.Unlock()
	if !ok {
		out = firstFileWords(s)
		m.page.mu.Lock()
		m.page.words.put(strings.Clone(s), len(s), out)
		m.page.mu.Unlock()
	}
	return out
}

// firstFileWords are fileWords(s), each shown text (with or without
// file:) the first time only.
func firstFileWords(s string) []fileWord {
	type seen struct {
		shown    string
		prefixed bool
	}
	done := map[seen]bool{}
	var out []fileWord
	for _, w := range fileWords(s) {
		if k := (seen{w.shown, w.prefixed}); !done[k] {
			done[k] = true
			out = append(out, w)
		}
	}
	return out
}

// pageLines are t's page as drawn, without its colours, one string a
// line: what markedBlock and the boxes look for their line in.
func (m model) pageLines(t task.Task) []string {
	page := m.detailText(t)
	if m.page == nil {
		return strings.Split(ansi.Strip(page), "\n")
	}
	m.page.mu.Lock()
	if m.page.lines.v != nil && m.page.lines.page == page {
		v := m.page.lines.v
		m.page.mu.Unlock()
		return v
	}
	m.page.mu.Unlock()
	v := strings.Split(ansi.Strip(page), "\n")
	m.page.mu.Lock()
	m.page.lines.page, m.page.lines.v = page, v
	m.page.mu.Unlock()
	return v
}

// markSpanOf is markSpan(s, w, linked), the scan of s kept per text: a
// Tab through the ids in the open Notes marks a new one in the same
// megabyte of text each time.
func (m model) markSpanOf(s, w string, linked bool) (at, n int, shown string) {
	if w == "" {
		return -1, 0, ""
	}
	if m.page == nil {
		return markSpan(s, w, linked)
	}
	m.page.mu.Lock()
	x, ok := m.page.scans.get(s)
	m.page.mu.Unlock()
	if !ok {
		x = scanVisible(s)
		m.page.mu.Lock()
		m.page.scans.put(strings.Clone(s), 10*len(s), x) // the text, and an offset a byte
		m.page.mu.Unlock()
	}
	return spanIn(x, w, linked)
}

// wrapChunk is how many bytes of a page wrapPage wraps as one piece.
const wrapChunk = 8 << 10

// wrapPage is wrap(s, width), worked out in pieces of whole lines, each
// kept: wrap treats each line on its own (a line break ends every run
// it measures), so the pieces joined are the page wrapped whole, and a
// key that changes one line (the focus mark, the box) wraps one piece.
func (m model) wrapPage(s string, width int) string {
	if m.page == nil || len(s) <= wrapChunk {
		return wrap(s, width)
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/8)
	for len(s) > 0 {
		piece := s
		if len(s) > wrapChunk {
			if i := strings.IndexByte(s[wrapChunk:], '\n'); i >= 0 {
				piece = s[:wrapChunk+i]
			}
		}
		s = s[len(piece):]
		key := wrapKey{text: piece, width: width}
		m.page.mu.Lock()
		out, ok := m.page.wraps.get(key)
		m.page.mu.Unlock()
		if !ok {
			out = wrap(piece, width)
			m.page.mu.Lock()
			key.text = strings.Clone(piece)
			m.page.wraps.put(key, len(piece)+len(out), out)
			m.page.mu.Unlock()
		}
		b.WriteString(out)
		if len(s) > 0 {
			b.WriteByte('\n') // the line break the piece ended before
			s = s[1:]
		}
	}
	return b.String()
}

type wrapKey struct {
	text  string
	width int
}

type filesKey struct {
	text, folder, files, checkout string
}

type fileLinks struct {
	stops []idStop
	until time.Time // zero: holds until the key changes
}

// fileLinksOf is scanFileLinks kept per text (fileLinksIn). What it
// returns is the memo's: read it, never change it.
func (m model) fileLinksOf(folder string, files []string, text string) []idStop {
	if m.page == nil {
		out, _ := m.scanFileLinks(folder, files, text)
		return out
	}
	key := filesKey{text: text, folder: folder, files: strings.Join(files, "\x00"), checkout: m.opts.Checkout}
	m.page.mu.Lock()
	v, ok := m.page.files.get(key)
	m.page.mu.Unlock()
	if !ok || !v.until.IsZero() && !time.Now().Before(v.until) {
		v.stops, v.until = m.scanFileLinks(folder, files, text)
		m.page.mu.Lock()
		key.text = strings.Clone(text)
		m.page.files.put(key, len(text), v)
		m.page.mu.Unlock()
	}
	return v.stops
}
