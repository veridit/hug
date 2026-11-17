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
// previous file, Esc or u go back to Files with that file marked. Plain text
// only: no highlighting, no Markdown, no search, no editing. The file is
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
	m.glanceHead, m.glanceBody = m.glanceRead(t, name)
	m.viewport.Height = max(1, m.bodyHeight()-len(m.detailHeader(t))-2)
	m.viewport.SetContent(m.glanceBody)
	m.viewport.GotoTop()
}

// glanceRead reads one file of t: the header line and the text.
func (m model) glanceRead(t task.Task, name string) (head, body string) {
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
	if info.Size() > glanceLimit {
		return head, glanceText(nil, info.Size(), m.width)
	}
	b, err := readFile(path)
	if err != nil {
		return head, err.Error()
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

// closeGlance goes back to Files with the file last shown marked.
func (m *model) closeGlance() {
	m.glance = false
	m.glanceHead, m.glanceBody = "", ""
	m.detailPane = focusFiles
	m.syncDetail()
	m.revealAcceptance()
}
