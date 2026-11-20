package tui

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/veridit/hug/huginn/internal/task"
)

// A glance at a task file (scrxak, Jørgen 11:05): Space on a file in Files
// shows its text in the detail pane, read only, under a header "path ·
// size · i of N"; j/k and PgUp/PgDn scroll, n/p step to the next and
// previous file, Esc or u go back to Files with that file marked. Markdown
// uses the detail renderer, M shows its source, and / finds text. The file is
// read when it is shown, never on list or detail load, and its bytes never
// reach the terminal raw: control characters and escape sequences are
// shown escaped.

// glanceLimit is the largest file shown inline; a larger one shows its
// size and "Enter opens it".
const glanceLimit = 1 << 20

// readFile is how a glance reads a file; tests count the reads.
var readFile = os.ReadFile

// glanceText is name's text as the glance shows it: escaped and wrapped to
// width, or one line for a binary or too large file.
func glanceText(b []byte, size int64, width int) string {
	if size > glanceLimit {
		return fmt.Sprintf("too large to show here, %s: Enter opens it", humanBytes(size))
	}
	if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) {
		return fmt.Sprintf("binary, %s: Enter opens it", humanBytes(size))
	}
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	var out []string
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		out = append(out, wrap(escapeControls(strings.ReplaceAll(line, "\t", "    ")), width))
	}
	return strings.Join(out, "\n")
}

// escapeControls writes every control character (ESC, CR, BEL, DEL, C1)
// as Go would quote it, so a file cannot move the cursor, clear the screen
// or set a title.
func escapeControls(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			q := strconv.QuoteRune(r)
			b.WriteString(q[1 : len(q)-1])
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// glanceFile is the file the glance shows: the one marked > in Files.
func (m model) glanceFile(t task.Task) (name string, i, n int) {
	files := m.filesOf(t)
	if len(files) == 0 {
		return "", 0, 0
	}
	i = max(0, min(m.file, len(files)-1))
	return files[i], i, len(files)
}

// openGlance shows the marked file (Space in Files).
func (m *model) openGlance() {
	t, ok := m.selected()
	if !ok {
		return
	}
	if len(m.filesOf(t)) == 0 {
		m.message = "this task has no files: hug task attach " + t.ID + " FILE"
		return
	}
	m.glance = true
	m.showGlance()
}

// showGlance reads the marked file and puts it in the viewport, at the top.
func (m *model) showGlance() {
	t, ok := m.selected()
	if !ok {
		m.glance = false
		return
	}
	name, _, _ := m.glanceFile(t)
	m.glanceGen++
	m.glanceMD = glanceMarkdown{}
	m.glanceImage = nil
	// Each file opens fitted (33v71x).
	m.picZoom, m.picX, m.picY = 0, 0, 0
	m.find = findState{} // the file's own find (4s3g5q)
	m.glanceHead, m.glanceBody = m.glanceRead(t, name)
	m.viewport.Height = m.glanceHeight()
	m.viewport.SetContent(m.glanceBody)
	m.viewport.GotoTop()
}

// glanceRead reads one file of t: the header line and the text.
func (m *model) glanceRead(t task.Task, name string) (head, body string) {
	_, i, n := m.glanceFile(t)
	path, err := m.insideFolder(t, name)
	if err != nil {
		return oneLine(name), err.Error()
	}
	info, err := os.Stat(path)
	if err != nil {
		return oneLine(name), err.Error()
	}
	head = fmt.Sprintf("%s · %s · %d of %d", oneLine(task.FolderOf(t)+"/"+name), humanBytes(info.Size()), i+1, n)
	if task.IsImageName(name) {
		// An image shows in the frame too (2qndsd): the terminal draws it
		// over the frame's rows (imageOverlay), or one line says why not.
		return head, m.glanceReadImage(path, info.Size())
	}
	if info.Size() > glanceLimit {
		return head, glanceText(nil, info.Size(), m.width)
	}
	b, err := readFile(path)
	if err != nil {
		return head, err.Error()
	}
	if len(b) > glanceLimit {
		return head, glanceText(nil, int64(len(b)), m.width)
	}
	if ansName(name) && utf8.Valid(b) {
		// A screen capture shows with its colours, nothing else of its
		// escapes (5z4zt4).
		return head, colourOnly(b, m.width, os.Getenv("NO_COLOR") == "")
	}
	if markdownName(name) && bytes.IndexByte(b, 0) < 0 && utf8.Valid(b) {
		m.beginGlanceMarkdown(escapedFileSource(b))
		return head, "formatting Markdown · M shows raw source"
	}
	return head, glanceText(b, info.Size(), m.width)
}

// stepGlance shows the next (1) or previous (-1) file, stopping at the ends.
func (m *model) stepGlance(step int) {
	t, ok := m.selected()
	if !ok {
		return
	}
	_, i, n := m.glanceFile(t)
	next := i + step
	if next < 0 || next >= n {
		m.message = fmt.Sprintf("%s file: %d of %d", map[bool]string{true: "first", false: "last"}[step < 0], i+1, n)
		return
	}
	m.file = next
	m.showGlance()
}

// closeGlance goes back to Files with the file last shown marked, or to
// the question whose link line showed it (qcq3tm), where it was on the
// page.
func (m *model) closeGlance() {
	back := m.glanceReturn
	m.glanceReturn = nil
	m.glance = false
	m.glanceImage = nil
	m.picZoom, m.picX, m.picY = 0, 0, 0
	m.glanceHead, m.glanceBody = "", ""
	m.glanceMD = glanceMarkdown{}
	m.find = findState{}
	if back != nil {
		m.detailPane, m.acceptance, m.question = back.pane, back.acceptance, back.question
		m.syncDetail()
		m.viewport.SetYOffset(back.offset)
		m.showMarked()
		return
	}
	m.detailPane = focusFiles
	m.syncDetail()
	m.revealAcceptance()
}

// glanceReturn is the question a file was shown from: its focus and the
// page's scroll.
type glanceReturn struct {
	pane, acceptance, question, offset int
}

// showQuestionFile shows, in the file frame, the file the focused
// question names (its link line, qcq3tm). false when the focus is on no
// question with a file.
func (m *model) showQuestionFile() bool {
	t, ok := m.selected()
	if !ok {
		return false
	}
	q, ok := m.focusedQuestion(t)
	if !ok || q.file == "" {
		return false
	}
	for i, f := range m.filesOf(t) {
		if f == q.file {
			m.glanceReturn = &glanceReturn{pane: m.detailPane, acceptance: m.acceptance, question: m.question, offset: m.viewport.YOffset}
			m.file = i
			m.glance = true
			m.showGlance()
			return true
		}
	}
	return false
}

// glanceHeight is how many lines the shown file gets (z82qsy check 4):
// the body less the note line (when the file has a note) and the two rules
// that frame it, plus the message and input lines while they are blank
// (yp3c09). The crumb above is the task screen's one head line.
func (m model) glanceHeight() int {
	chrome := 2
	if m.glanceNote() != "" {
		chrome++
	}
	_, spare := m.footRows()
	return max(1, m.bodyHeight()+spare-chrome)
}

// fitGlance makes the file as tall as the rows it has now: a message
// coming takes a row from it, its going gives the row back (yp3c09).
// Scrolled to the end, it stays at the end.
func (m *model) fitGlance() {
	h := m.glanceHeight()
	if !m.glance || m.viewport.Height == h {
		return
	}
	end := m.viewport.AtBottom() && m.viewport.YOffset > 0
	m.viewport.Height = h
	if end {
		m.viewport.GotoBottom()
		return
	}
	m.viewport.SetYOffset(m.viewport.YOffset)
}

// glanceNote is what the task's notes say about the shown file when it
// was attached, "file:NAME (N bytes, sha256:H) WHY": the WHY of the
// latest such note, without a "replaces sha256:H". "" with none.
func (m model) glanceNote() string {
	t, ok := m.selected()
	if !ok {
		return ""
	}
	name, _, _ := m.glanceFile(t)
	d := m.doc(t)
	if name == "" || d == nil || d.Task.Texts == nil {
		return ""
	}
	return attachmentNote(d.Task.Texts.Notes, name)
}

// attachmentNote finds the note an attach left for name in notes.
func attachmentNote(notes, name string) string {
	note := ""
	prefix := "file:" + name + " ("
	for _, l := range strings.Split(notes, "\n") {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, prefix) {
			continue
		}
		rest := l[len(prefix):]
		close := strings.Index(rest, ")")
		if close < 0 {
			continue
		}
		why := strings.TrimSpace(rest[close+1:])
		if strings.HasPrefix(why, "replaces sha256:") {
			why = strings.TrimSpace(strings.TrimPrefix(why, "replaces sha256:"))
			if sp := strings.IndexByte(why, ' '); sp >= 0 {
				why = strings.TrimSpace(why[sp+1:])
			} else {
				why = ""
			}
		}
		note = oneLine(why) // the latest attach says what the file is now
	}
	return note
}
