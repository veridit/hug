package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// File links (zbdbtm, Jørgen 2026-11-20, hpqt99 request 1 and p87ngv
// request 4): a file a task names opens from the task. Three forms are
// links, wherever the detail shows text (the places idStops scans):
//   - file:NAME, a file attached to the task;
//   - NAME, the plain name of a file attached to the task (MOCK.md);
//   - PATH, a file of the board's checkout (docs/DESIGN.md).
//
// Each may end in #Section (docs/DESIGN.md#Imported data). A name is a
// link only when the task has that file or the checkout has that path,
// so ordinary words never are. Each link is a Tab stop as a task id is
// (0ga6m0); Space shows the file in line under the line that names it,
// Enter opens it in the file frame scrolled to the section, and Esc
// comes back to the link (Jørgen's answer 1, keys provisional until
// bfxq93). A branch (the p87ngv .ref mark) is not read: the mark syntax
// is not built, and no other form names one.

// fileLink is the file a link names: an attachment of the task, or a
// path in the checkout (repo), and the section after #.
type fileLink struct {
	name, section string
	repo          bool
}

// fileCandidate is a word that may name a file: an optional file:, a
// name with an extension, maybe a path, then an optional #Section that
// runs to the end of the line or a closing mark.
var fileCandidate = regexp.MustCompile(`(file:)?((?:[A-Za-z0-9_+@-][A-Za-z0-9_.+@-]*/)*[A-Za-z0-9_+@-][A-Za-z0-9_.+@-]*\.[A-Za-z0-9]+)(#[^\s#()\[\]{}<>` + "`" + `"',;:|*][^\n#()\[\]{}<>` + "`" + `"',;:|*]*)?`)

// fileWord is one candidate in text: its offset (past file:), what the
// reader sees (name and #section, as the mark shows it), the name, the
// section and whether file: was written.
type fileWord struct {
	at                   int
	shown, name, section string
	prefixed             bool
}

// fileWords are the candidates in s, in order. A word inside a longer
// one (after /, :, a letter or a dot) is none: a URL, an absolute path,
// task:ID/NAME.
func fileWords(s string) []fileWord {
	var out []fileWord
	for _, loc := range fileCandidate.FindAllStringSubmatchIndex(s, -1) {
		if loc[0] > 0 {
			r, _ := utf8.DecodeLastRuneInString(s[:loc[0]])
			if r == '/' || r == ':' || r == '.' || r == '~' || r == '\\' || r == '_' || r == '-' || r < utf8.RuneSelf && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				continue
			}
		}
		if loc[5] < len(s) && loc[6] < 0 {
			// Nor is the front of a longer word: history.sdo-to-be.
			if c := s[loc[5]]; c == '-' || c == '_' || c == '/' || c == '@' || c == '+' {
				continue
			}
		}
		w := fileWord{at: loc[4], name: s[loc[4]:loc[5]], prefixed: loc[2] >= 0}
		end := loc[5]
		if loc[6] >= 0 {
			// A section ends before a sentence's full stop and its spaces.
			sec := strings.TrimRight(s[loc[6]+1:loc[7]], " \t.")
			if sec != "" {
				w.section = sec
				end = loc[6] + 1 + len(sec)
			}
		}
		w.shown = s[w.at:end]
		out = append(out, w)
	}
	return out
}

// fileLinksIn are the files text names that are t's or the checkout's,
// in order, each shown text once.
func (m model) fileLinksIn(t task.Task, text string) []idStop {
	if !strings.Contains(text, ".") {
		return nil
	}
	var out []idStop
	seen := map[string]bool{}
	for _, w := range fileWords(text) {
		if seen[w.shown] {
			continue
		}
		l, ok := m.resolveFile(t, w)
		if !ok {
			continue
		}
		seen[w.shown] = true
		out = append(out, idStop{shown: w.shown, at: w.at, file: &l})
	}
	return out
}

// resolveFile says which file w names: one of t's files (by its name, or
// a path that ends in its folder and name), else, unless written file:,
// a regular file of the checkout.
func (m model) resolveFile(t task.Task, w fileWord) (fileLink, bool) {
	folder := task.FolderOf(t) + "/"
	for _, f := range m.filesOf(t) {
		if w.name == f || strings.HasSuffix(w.name, "/"+folder+f) || w.name == folder+f {
			return fileLink{name: f, section: w.section}, true
		}
	}
	if w.prefixed || m.opts.Checkout == "" {
		return fileLink{}, false
	}
	if _, ok := repoFile(m.opts.Checkout, w.name); ok {
		return fileLink{name: w.name, section: w.section, repo: true}, true
	}
	return fileLink{}, false
}

// repoFiles remembers, for a few seconds, which paths are regular files
// of a checkout: idStops runs on every frame, a stat each word would add
// up on a long page.
var repoFiles struct {
	sync.Mutex
	seen map[string]repoSeen
}

type repoSeen struct {
	abs string
	ok  bool
	at  time.Time
}

// repoFile is rel's path in checkout when it is a regular file there,
// not outside it, not in .git or the task branch's folder.
func repoFile(checkout, rel string) (string, bool) {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if rel == "" || filepath.IsAbs(rel) || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, ".git/") || strings.HasPrefix(clean, ".hug/") {
		return "", false
	}
	key := checkout + "\x00" + clean
	repoFiles.Lock()
	defer repoFiles.Unlock()
	if s, ok := repoFiles.seen[key]; ok && time.Since(s.at) < 3*time.Second {
		return s.abs, s.ok
	}
	if repoFiles.seen == nil || len(repoFiles.seen) > 4096 {
		repoFiles.seen = map[string]repoSeen{}
	}
	abs := filepath.Join(checkout, filepath.FromSlash(clean))
	info, err := os.Lstat(abs)
	s := repoSeen{abs: abs, ok: err == nil && info.Mode().IsRegular(), at: time.Now()}
	repoFiles.seen[key] = s
	return s.abs, s.ok
}

// filePath is where l's file is on disk, checked: an attachment inside
// its task's folder, a repo file inside the checkout.
func (m model) filePath(t task.Task, l fileLink) (string, error) {
	if !l.repo {
		return m.insideFolder(t, l.name)
	}
	abs, ok := repoFile(m.opts.Checkout, l.name)
	if !ok {
		return "", fmt.Errorf("%s is no file of the checkout %s\nfix: hug task show %s", l.name, m.opts.Checkout, t.ID)
	}
	root, err := filepath.EvalSymlinks(m.opts.Checkout)
	if err != nil {
		return "", err
	}
	actual, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	if r, err := filepath.Rel(root, actual); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s leads out of the checkout\nfix: hug task show %s", l.name, t.ID)
	}
	return abs, nil
}

// markedFile is the file link the focus marks, if one is.
func (m model) markedFile(t task.Task) (idStop, bool) {
	s, ok := m.markedStop(t)
	if !ok || s.file == nil {
		return idStop{}, false
	}
	return s, true
}

// openFileLink shows the marked link's file in the file frame, scrolled
// to its section; Esc comes back to the link (glanceReturn).
func (m *model) openFileLink(t task.Task, s idStop) {
	l := *s.file
	if _, err := m.filePath(t, l); err != nil {
		m.message = err.Error()
		return
	}
	m.glanceReturn = &glanceReturn{pane: m.detailPane, acceptance: m.acceptance, question: m.question, idAt: m.idAt, offset: m.viewport.YOffset, link: true}
	m.glanceRepo = ""
	if l.repo {
		m.glanceRepo = l.name
	} else {
		for i, f := range m.filesOf(t) {
			if f == l.name {
				m.file = i
			}
		}
	}
	m.glanceSection = l.section
	m.glance = true
	m.showGlance()
}

// headingText is a line as a heading reads: without the #s that open it
// in Markdown, the spaces around it, and its colours.
func headingText(line string) string {
	s := strings.TrimSpace(ansi.Strip(line))
	return strings.TrimSpace(strings.TrimLeft(s, "#"))
}

// slug is s as a Markdown anchor: lower case, words joined by -.
func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= utf8.RuneSelf:
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		default:
			dash = true
		}
	}
	return b.String()
}

// sectionLine is the line of source where section starts: a Markdown
// heading (or, in a file that is no Markdown, any line) that begins with
// it, or whose anchor is it. A section written in running text may run
// on into the sentence ("#Imported data and more"): the longest leading
// run of its words that names a heading. -1 for none.
func sectionLine(lines []string, section string, markdown bool) (int, string) {
	words := strings.Fields(section)
	for n := len(words); n > 0; n-- {
		want := strings.ToLower(strings.Join(words[:n], " "))
		for i, l := range lines {
			t := strings.TrimSpace(l)
			if markdown && !strings.HasPrefix(t, "#") {
				continue
			}
			h := headingText(t)
			if h == "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(h), want) || slug(h) == slug(want) {
				return i, h
			}
		}
	}
	return -1, ""
}

// seekSection scrolls the file frame to the section a link named, once
// the text is drawn: a Markdown file after it is formatted. The heading
// is found in the source, then its line among the lines drawn.
func (m *model) seekSection() {
	if m.glanceSection == "" || !m.glance || m.glanceMD.loading && !m.glanceMD.raw {
		return
	}
	section := m.glanceSection
	m.glanceSection = ""
	source := m.glanceBody
	if m.glanceMD.eligible {
		source = m.glanceMD.source
	}
	_, heading := sectionLine(strings.Split(source, "\n"), section, m.glanceMD.eligible)
	if heading == "" {
		m.message = fmt.Sprintf("no heading %q in this file: it opens at the top", section)
		return
	}
	h := strings.ToLower(heading)
	body := strings.Split(m.glanceBody, "\n")
	for i, l := range body {
		got := strings.ToLower(headingText(l))
		if got != "" && (strings.HasPrefix(got, h) || len(got) >= min(len(h), 12) && strings.HasPrefix(h, got)) {
			if short := i + m.viewport.Height - len(body); short > 0 {
				// A file too short to scroll that far ends in blank lines,
				// so the heading still stands first, where the link said.
				m.viewport.SetContent(m.glanceBody + strings.Repeat("\n", short))
			}
			m.viewport.SetYOffset(i)
			return
		}
	}
}

// inlineMax is how many lines of a file Space shows in line; Enter
// shows the whole file.
const inlineMax = 12

// inlineFile is a file Space shows in line under the link that names it:
// the stop it belongs to, and its lines from the section on, read once.
type inlineFile struct {
	where, shown string
	index        int
	head         string
	lines        []string
	more         int
}

// toggleInline shows the marked link's file in line, or folds it away.
func (m *model) toggleInline(t task.Task, s idStop) {
	if m.inlineShown(s) {
		m.inline = nil
		m.syncDetail()
		return
	}
	l := *s.file
	in := &inlineFile{where: s.where, index: s.index, shown: s.shown}
	path, err := m.filePath(t, l)
	if err != nil {
		m.message = err.Error()
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		m.message = err.Error()
		return
	}
	in.head = oneLine(l.name) + " · " + humanBytes(info.Size())
	switch {
	case task.IsImageName(l.name):
		in.lines = []string{"a picture: Enter shows it"}
	case info.Size() > glanceLimit:
		in.lines = []string{"too large to show here: Enter opens it"}
	default:
		b, err := readFile(path)
		if err != nil {
			m.message = err.Error()
			return
		}
		if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) {
			in.lines = []string{"binary: Enter opens it"}
			break
		}
		lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n"), "\n")
		if l.section != "" {
			if i, _ := sectionLine(lines, l.section, markdownName(l.name)); i >= 0 {
				lines = lines[i:]
				in.head += " · #" + oneLine(l.section)
			} else {
				in.head += " · no heading " + oneLine(l.section)
			}
		}
		if len(lines) > inlineMax {
			in.more = len(lines) - inlineMax
			lines = lines[:inlineMax]
		}
		for i, line := range lines {
			if ansName(l.name) {
				lines[i] = colourOnly([]byte(line), max(10, m.width-6), os.Getenv("NO_COLOR") == "")
			} else {
				lines[i] = escapeControls(strings.ReplaceAll(line, "\t", "    "))
			}
		}
		in.lines = lines
	}
	m.inline = in
	m.syncDetail()
	m.showMarked()
}

// inlineShown says whether the file of stop s is shown in line now.
func (m model) inlineShown(s idStop) bool {
	return m.inline != nil && m.inline.where == s.where && m.inline.index == s.index && m.inline.shown == s.shown
}

// inlineBox is the shown file's lines, hung under indent: a head with its
// name and size, the text behind a bar, a foot with the keys.
func (m model) inlineBox(indent int) []string {
	in := m.inline
	pad := strings.Repeat(" ", indent)
	width := max(10, m.width-indent-2)
	out := []string{m.styles.dim.Render(fit(pad+"┌ "+in.head, m.width))}
	more := in.more
	for k, l := range in.lines {
		rows := strings.Split(wrap(l, width), "\n")
		if len(out)-1+len(rows) > inlineMax {
			// At most inlineMax rows drawn, wrapped lines counted: the
			// rest is Enter's.
			more += len(in.lines) - k
			break
		}
		for _, w := range rows {
			out = append(out, m.styles.dim.Render(pad+"│ ")+w)
		}
	}
	foot := "└ Enter opens it · Esc or Space closes"
	if more > 0 {
		foot = fmt.Sprintf("└ %d more lines · Enter opens it · Esc or Space closes", more)
	}
	return append(out, m.styles.dim.Render(fit(pad+foot, m.width)))
}

// withInline puts the file shown in line under the page line that holds
// its marked link (where the link ends, when it wraps).
func (m model) withInline(t task.Task, page string) string {
	s, ok := m.markedFile(t)
	if !ok || !m.inlineShown(s) {
		return page
	}
	lines := strings.Split(page, "\n")
	first := "▸" + strings.Fields(s.shown + " ")[0]
	for i, l := range lines {
		plain := ansi.Strip(l)
		if !strings.Contains(plain, first) {
			continue
		}
		at := i
		if !strings.Contains(plain, "▸"+s.shown) && i+1 < len(lines) {
			at = i + 1
		}
		indent := len(plain) - len(strings.TrimLeft(plain, " "))
		box := m.inlineBox(indent + 2)
		return strings.Join(append(append(append([]string{}, lines[:at+1]...), box...), lines[at+1:]...), "\n")
	}
	return page
}
