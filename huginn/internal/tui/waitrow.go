package tui

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// The row under the cursor opens a second line when it waits on
// something (89y0b7, Jørgen 14:55-14:58, mock waiting-rows.md on the
// ticket; tje4sx, Jørgen 20:00: only the focused row, every other row is
// one line): line 1 is the task as always, with its title; line 2 hangs
// under the id and says what it waits on, the reason right after the
// colon, and since when:
//
//	> ○ 3    bxqtkt  spike    waiting  Contributions: agent feedback…  kestrel
//	         └─ waiting on you: approve the design ▸ bxqtkt.md    since 14:42
//
// A person is named by their short name, "you" for the TUI user, never by
// initials typed into the text. A row that waits on you carries ? when the
// cursor is elsewhere, so it stands out on its one line.

// noteStages are the stages whose rows take a second line.
var noteStages = map[string]bool{"waiting": true, "asked": true, "review": true, "design": true, "depends": true, "deferred": true}

// stageOf is t's stage as rows show it: task.Stage, with depends when a
// condition holds the task back (16dxsr), "" for done and dropped.
func (m model) stageOf(t task.Task) string {
	g := m.taskStage(t)
	if (g == "ready" || g == "active" || g == "draft") && m.blocked[t.Path] {
		g = "depends"
	}
	return g
}

// rowHeight is how many lines t's row takes: 2 when it is the selected
// row and waits on something, else 1 (tje4sx).
func (m model) rowHeight(t task.Task, selected bool) int {
	if selected && noteStages[m.stageOf(t)] {
		return 2
	}
	return 1
}

// heightOf is rowHeight by index into visible; sel is the index drawn
// selected in the section, or -1 for none.
func (m model) heightOf(i, sel int) int {
	heightLookups.Add(1)
	if i < 0 || i >= len(m.visible) {
		return 1
	}
	return m.rowHeight(m.visible[i], i == sel)
}

// heightLookups counts heightOf calls (46ek1f: a frame's count must not
// grow with the board).
var heightLookups atomic.Int64

// linesOf is how many lines rows indices take, sel drawn selected.
func (m model) linesOf(indices []int, sel int) int {
	return m.linesUpTo(indices, sel, -1)
}

// linesUpTo is linesOf, but stops once the sum passes limit (a negative
// limit: never): a frame asks whether rows fit the screen, and a section
// of 2,000 rows need not be summed to learn it does not (46ek1f). Past
// limit the result is more than limit and at most limit+2.
func (m model) linesUpTo(indices []int, sel, limit int) int {
	n := 0
	for _, i := range indices {
		n += m.heightOf(i, sel)
		if limit >= 0 && n > limit {
			return n
		}
	}
	return n
}

// selIn is the row section p draws selected: the cursor in the focused
// section, none elsewhere (a task in LINT is also in its own section, and
// only one of them is under the cursor).
func (m model) selIn(p int) int {
	if p == m.pane {
		return m.cursor
	}
	return -1
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
	case "asked":
		// The question in its own words (41hhgn check 10, header-spec.md
		// 'Overview sections'): "asked: check 9 · <question> · nautilus".
		// Asked of the viewer it sits in ASKED, so "you" is said by the
		// section; asked of someone else the line names them.
		var parts []string
		if n := f("asked_check"); n != "" && !tickedCheck(t, n) {
			parts = append(parts, "check "+n)
		}
		parts = append(parts, m.attachments(t, f("asked_what")))
		// Several open (q2adr8): the row shows the oldest and says how
		// many; the detail shows each with its own answer.
		if n := len(task.OpenQuestions(t)); n > 1 {
			parts = append(parts, strconv.Itoa(n)+" questions")
		}
		if by := m.name(t.Fields["asked_by"]); by != "" && by != "you" {
			parts = append(parts, by)
		}
		head := "asked:"
		if who := m.name(t.Fields["asked_for"]); who != "" && who != "you" {
			head = "asked " + who + ":"
		}
		return head, strings.Join(parts, " · "), t.Fields["asked_since"]
	case "waiting":
		// A wait for a date leads with it (atex3d, Jørgen 2026-11-20
		// 13:05): "waiting until 2026-11-24 · then ask jhf", never "on
		// the date ..." or a second "until".
		if until, rest, ok := task.DatedWait(t.Fields); ok {
			if rest != "" {
				rest = "· " + m.attachments(t, oneLine(rest))
			}
			return "waiting until " + oneLine(until), rest, t.Fields["waiting_since"]
		}
		who, what := m.waitingFor(t)
		at := t.Fields["waiting_since"]
		if v := t.Fields["asked_since"]; v != "" {
			at = v
		}
		if who == "" {
			return "waiting on", m.attachments(t, oneLine(what)), at
		}
		return "waiting on " + m.name(who) + ":", m.attachments(t, oneLine(what)), at
	case "review":
		who := m.name(t.Fields["review_by"])
		if t.State == "doing" {
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
	indent := rowHead // under the id, the row header (911j70)
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
	// No bar under the cursor row (911j70): the note reads in its own
	// colours, as the row above it does (selected is kept for callers).
	// The head names the stage in its colour: amber and bold when it
	// waits on you, the calm depends colour, else dim (zhvkmj, crab
	// 02:52; check 6: depends is no problem, not red).
	style := m.styles.dim
	if m.stageOf(t) == "depends" {
		style = m.styles.status["depends"]
	}
	if m.forYou(t) {
		style = m.styles.forYou.Bold(true)
	}
	pad := strings.Repeat(" ", indent)
	body := strings.TrimPrefix(left, pad+"└─ "+head)
	out := pad + m.styles.dim.Render("└─ ") + style.Render(head) + m.styles.dim.Render(body)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	return out + strings.Repeat(" ", max(0, gap)) + m.styles.dim.Render(right)
}

// window is the rows of indices shown in lines lines, starting at start
// as far as it can: a section keeps its scroll (h9zghj) and moves it only
// as far as needed to keep the cursor in view. The selected row sel takes
// two lines when it waits (tje4sx), every other row one; a row never
// shows cut in half, and the cursor's row is shown even when it alone is
// taller than lines. A cursor or sel of -1 is none.
func (m model) window(indices []int, cursor, start, lines, sel int) (int, int) {
	n := len(indices)
	lines = max(0, lines)
	if n == 0 {
		return 0, 0
	}
	if m.linesUpTo(indices, sel, lines) <= lines {
		return 0, n
	}
	start = max(0, min(start, n-1))
	end := func(from int) int {
		used, e := 0, from
		for e < n && used+m.heightOf(indices[e], sel) <= lines {
			used += m.heightOf(indices[e], sel)
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
		for s >= 0 && used+m.heightOf(indices[s], sel) <= lines {
			used += m.heightOf(indices[s], sel)
			s--
		}
		start = min(s+1, p)
	}
	// No blank lines under the last row while rows above are hidden.
	for e := end(start); e == n && start > 0; e = end(start) {
		used := 0
		for _, i := range indices[start:e] {
			used += m.heightOf(i, sel)
		}
		if used+m.heightOf(indices[start-1], sel) > lines {
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
	case "waiting", "asked":
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

// untilCell is a wait's STAGE cell when it waits for a date: "til 11-24"
// (atex3d, crab 23:07Z: the cell keeps its 9 cells, so no "until" and no
// year; the second line has the whole date). "" when v is no date.
func untilCell(v string) string {
	d, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(v), time.Local)
	if err != nil {
		return ""
	}
	return "til " + d.Format("01-02")
}

// tickedCheck says whether check n (as written in asked_check) of t is
// ticked: a ticked check carries no question (qsqhpe).
func tickedCheck(t task.Task, n string) bool {
	i, err := strconv.Atoi(strings.TrimSpace(n))
	return err == nil && i >= 1 && i <= len(t.Acceptance) && t.Acceptance[i-1].Checked
}
