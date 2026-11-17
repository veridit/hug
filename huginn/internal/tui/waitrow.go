package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// Rows that wait on something take two lines (89y0b7, Jørgen 14:55-14:58,
// mock waiting-rows.md on the ticket): line 1 is the task as always, with
// its title; line 2 hangs under the id and says what it waits on, the
// reason right after the colon, and since when:
//
//	? ○ 3    bxqtkt  spike    waiting  Contributions: agent feedback…  kestrel
//	         └─ waiting on you: approve the design ▸ bxqtkt.md    since 14:42
//
// A person is named by their short name, "you" for the TUI user, never by
// initials typed into the text. Active, ready and closed rows stay one line.

// noteStages are the stages whose rows take a second line.
var noteStages = map[string]bool{"waiting": true, "review": true, "design": true, "depends": true, "deferred": true}

// stageOf is t's stage as rows show it: task.Stage, with depends when a
// condition holds the task back (16dxsr), "" for done and dropped.
func (m model) stageOf(t task.Task) string {
	g := task.Stage(t, m.all)
	if (g == "ready" || g == "active" || g == "draft") && m.blocked[t.Path] {
		g = "depends"
	}
	return g
}

// rowHeight is how many lines t's row takes: 2 when it waits on
// something, else 1.
func (m model) rowHeight(t task.Task) int {
	if noteStages[m.stageOf(t)] {
		return 2
	}
	return 1
}

// heightOf is rowHeight by index into visible.
func (m model) heightOf(i int) int {
	if i < 0 || i >= len(m.visible) {
		return 1
	}
	return m.rowHeight(m.visible[i])
}

// linesOf is how many lines rows indices take.
func (m model) linesOf(indices []int) int {
	n := 0
	for _, i := range indices {
		n += m.heightOf(i)
	}
	return n
}

// name is how a row names who: "you" for the TUI user, a listed person by
// their short name, an email by the part before the @, else as written.
func (m model) name(who string) string {
	who = strings.TrimPrefix(strings.TrimSpace(who), "@")
	switch {
	case who == "":
		return ""
	case m.isMe(who):
		return "you"
	case strings.Contains(who, "@"):
		if p, ok := task.PersonByEmail(m.people, who); ok {
			return p.Short
		}
		return task.EmailShort(who)
	}
	if p, ok := task.PersonByName(m.people, who); ok {
		return p.Short
	}
	return oneLine(who)
}

// note is the second line's text for t: what it waits on, and the stamp
// it counts from ("" when there is none). "" text: a one-line row.
func (m model) note(t task.Task) (head, text, stamp string) {
	f := func(k string) string { return oneLine(t.Fields[k]) }
	switch m.stageOf(t) {
	case "waiting":
		who, what := m.waitingFor(t)
		if who == "" {
			return "waiting on", m.attachments(t, oneLine(what)), t.Fields["waiting_since"]
		}
		return "waiting on " + m.name(who) + ":", m.attachments(t, oneLine(what)), t.Fields["waiting_since"]
	case "review":
		who := m.name(t.Fields["review_by"])
		if t.Status == "doing" {
			if who == "" {
				return "in review", "", t.Fields["review_since"]
			}
			return "in review by " + who, "", t.Fields["review_since"]
		}
		if who == "" {
			return "waiting for review", "", t.Fields["review_since"]
		}
		return "waiting for review by " + who, "", t.Fields["review_since"]
	case "design":
		head := "question"
		if who := m.name(t.Fields["design_for"]); who != "" {
			head += " for " + who
		}
		return head + ":", m.attachments(t, f("design_question")), t.Fields["design_since"]
	case "deferred":
		text := f("deferred_reason")
		if u := f("deferred_until"); u != "" {
			text += " · until " + u
		}
		return "deferred:", text, ""
	case "depends":
		return "depends on", m.dependsNote(t), ""
	}
	return "", "", ""
}

// dependsNote is what a depends row waits for: the open dependencies and
// conditions, then the bar of how many are done.
func (m model) dependsNote(t task.Task) string {
	var open []string
	done, total := 0, 0
	for _, d := range task.DependencyDetails(t, m.all) {
		total++
		if !d.Blocking {
			done++
			continue
		}
		id := d.ID
		if id == "" {
			id = "?"
		}
		open = append(open, oneLine(id))
	}
	for _, w := range task.Waits(t, m.all) {
		total++
		if w.Met {
			done++
			continue
		}
		open = append(open, "when "+oneLine(w.Text))
	}
	text := strings.Join(open, ", ")
	if bar := dependsBar(done, total); bar != "" {
		text += "  " + bar
	}
	return text
}

// attachments marks a file of t that text names (file:NAME, or the name
// or its path) as "▸ NAME", so the reader sees there is something to look
// at, and Tab in the task lands on it (89y0b7).
func (m model) attachments(t task.Task, text string) string {
	files := m.filesOf(t)
	if len(files) == 0 {
		return text
	}
	words := strings.Fields(text)
	for i, w := range words {
		trimmed := strings.TrimRight(w, ".,;:)")
		tail := w[len(trimmed):]
		bare := strings.TrimPrefix(trimmed, "file:")
		for _, f := range files {
			if bare == f || strings.HasSuffix(bare, "/"+f) {
				words[i] = "▸ " + f + tail
				break
			}
		}
	}
	return strings.Join(words, " ")
}

// stampAt reads a stamp as hug writes it, UTC with Z or the older local
// form without a zone.
func stampAt(stamp string) (time.Time, bool) {
	stamp = strings.Trim(strings.TrimSpace(stamp), `"'`)
	for _, layout := range []string{"2006-01-02T15:04:05Z", time.RFC3339, "2006-01-02T15:04"} {
		loc := time.Local
		if strings.HasSuffix(stamp, "Z") || strings.Contains(stamp, "+") {
			loc = time.UTC
		}
		if at, e := time.ParseInLocation(layout, stamp, loc); e == nil {
			return at, true
		}
	}
	return time.Time{}, false
}

// sinceText is "since 14:42" for a stamp of today, "since 15 Nov" for an
// older one, in local time; "" when it does not read.
func sinceText(stamp string) string {
	at, ok := stampAt(stamp)
	if !ok {
		return ""
	}
	at, today := at.In(time.Local), now().In(time.Local)
	if at.Year() == today.Year() && at.YearDay() == today.YearDay() {
		return "since " + at.Format("15:04")
	}
	return "since " + at.Format("2 Jan")
}

// noteRow draws t's second line width cells wide, or "" for a one-line
// row. It hangs under the id with └─; "since" sits at the right edge and
// the reason is cut first (102bc0: each line fits).
func (m model) noteRow(t task.Task, width int, selected bool) string {
	head, text, stamp := m.note(t)
	if head == "" {
		return ""
	}
	indent := 2 + m.progressWidth()
	if width < 40 {
		indent = 2
	}
	right := sinceText(stamp)
	left := strings.Repeat(" ", indent) + "└─ " + head
	if text != "" {
		left += " " + text
	}
	room := width
	if right != "" && width-lipgloss.Width(right)-2 >= indent+10 {
		room = width - lipgloss.Width(right) - 2
	} else {
		right = ""
	}
	left = ansi.Truncate(left, room, "…")
	line := left + strings.Repeat(" ", max(0, width-lipgloss.Width(left)-lipgloss.Width(right))) + right
	if selected {
		return m.styles.selected.Render(fit(line, width))
	}
	// The head names the stage in its colour (warning: someone waits);
	// you in bold, so the line that is yours stands out.
	style := m.styles.warning
	if m.stageOf(t) == "depends" {
		style = m.styles.status["depends"]
	} else if m.stageOf(t) == "deferred" {
		style = m.styles.dim
	}
	if m.forYou(t) {
		style = style.Bold(true)
	}
	pad := strings.Repeat(" ", indent)
	body := strings.TrimPrefix(left, pad+"└─ "+head)
	out := pad + m.styles.dim.Render("└─ ") + style.Render(head) + m.styles.dim.Render(body)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	return out + strings.Repeat(" ", max(0, gap)) + m.styles.dim.Render(right)
}

// window is the rows of indices shown in lines lines, starting at start
// as far as it can: a section keeps its scroll (h9zghj) and moves it only
// as far as needed to keep the cursor in view. Rows take one or two
// lines (89y0b7); a row never shows cut in half, and the cursor's row is
// shown even when it alone is taller than lines. A cursor of -1 is none.
func (m model) window(indices []int, cursor, start, lines int) (int, int) {
	n := len(indices)
	lines = max(0, lines)
	if n == 0 {
		return 0, 0
	}
	if m.linesOf(indices) <= lines {
		return 0, n
	}
	start = max(0, min(start, n-1))
	end := func(from int) int {
		used, e := 0, from
		for e < n && used+m.heightOf(indices[e]) <= lines {
			used += m.heightOf(indices[e])
			e++
		}
		return max(e, min(from+1, n))
	}
	p := -1
	for k, i := range indices {
		if i == cursor {
			p = k
			break
		}
	}
	if p >= 0 && p < start {
		start = p
	}
	if p >= 0 && p >= end(start) {
		// Scroll down just far enough for the cursor's row.
		used, s := 0, p
		for s >= 0 && used+m.heightOf(indices[s]) <= lines {
			used += m.heightOf(indices[s])
			s--
		}
		start = min(s+1, p)
	}
	// No blank lines under the last row while rows above are hidden.
	for e := end(start); e == n && start > 0; e = end(start) {
		used := 0
		for _, i := range indices[start:e] {
			used += m.heightOf(i)
		}
		if used+m.heightOf(indices[start-1]) > lines {
			break
		}
		start--
	}
	return start, end(start)
}

// shownText is the "a–b of n" a section title gives when rows are hidden.
func shownText(start, end int) string { return fmt.Sprintf(" · %d–%d shown", start+1, end) }

// waitsOn is who t's stage waits on, as written: waiting_for, design_for
// or review_by; "" when it waits on a thing.
func (m model) waitsOn(t task.Task) string {
	switch m.stageOf(t) {
	case "waiting":
		who, _ := m.waitingFor(t)
		return who
	case "design":
		return t.Fields["design_for"]
	case "review":
		return t.Fields["review_by"]
	}
	return ""
}

// withLongName puts the person's long name after their short name in a
// note's head: "waiting on jhf (Jørgen H. Fjeld):" (the detail; rows keep
// the short name).
func (m model) withLongName(t task.Task, head string) string {
	p, ok := task.PersonByName(m.people, m.waitsOn(t))
	if !ok || p.Name == "" {
		return head
	}
	short := m.name(p.Short)
	colon := strings.HasSuffix(head, ":")
	head = strings.TrimSuffix(head, ":")
	if strings.HasSuffix(head, " "+short) {
		head += " (" + p.Name + ")"
	}
	if colon {
		head += ":"
	}
	return head
}
