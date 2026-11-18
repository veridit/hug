package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/internal/sharefile"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

var htmlComments = regexp.MustCompile(`(?s)<!--.*?(?:-->|$)`)

func cleanMarkdown(s string) string { return htmlComments.ReplaceAllString(safe(s), "") }

// literalAngles escapes every "<" that markdown would read as an HTML tag
// or autolink, so a note like "git worktree add --detach <dir> <rev>" shows
// as written instead of losing <dir> and <rev> as raw HTML (heype5). Code
// spans and fenced blocks are left alone: there "<" is already literal and
// a backslash would show.
func literalAngles(s string) string {
	var b strings.Builder
	fence := ""
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			b.WriteString(line)
			continue
		}
		if len(line)-len(trimmed) < 4 && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			fence = trimmed[:3]
			b.WriteString(line)
			continue
		}
		for j := 0; j < len(line); j++ {
			c := line[j]
			switch {
			case c == '\\' && j+1 < len(line):
				b.WriteString(line[j : j+2]) // already escaped
				j++
			case c == '`':
				// A code span runs to the next run of as many backticks.
				run := j
				for run < len(line) && line[run] == '`' {
					run++
				}
				ticks := line[j:run]
				if end := strings.Index(line[run:], ticks); end >= 0 {
					stop := run + end + len(ticks)
					b.WriteString(line[j:stop])
					j = stop - 1
				} else {
					b.WriteString(ticks)
					j = run - 1
				}
			case c == '<' && j+1 < len(line) && tagStart(line[j+1]):
				b.WriteString(`\<`)
			default:
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

func tagStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '/' || c == '!' || c == '?'
}
func (m model) markdown(s string) string {
	return renderMarkdown(s, m.width, m.styles.renderer.HasDarkBackground(), m.styles.renderer.ColorProfile())
}

// All inputs are values: file formatting can run off the UI thread without
// reading the model or sharing its mutable terminal renderer (tmz060).
func renderMarkdown(s string, width int, dark bool, profile termenv.Profile) string {
	s = cleanMarkdown(s)
	if strings.TrimSpace(s) == "" {
		return ""
	}
	style := styles.DarkStyleConfig
	accent := "#00bfff"
	codeFg, codeBg := "#c9d6e3", "#1a2630"
	if !dark {
		style = styles.LightStyleConfig
		accent = "#006c9c"
		codeFg, codeBg = "#2b3a48", "#e8eef3"
	}
	zero := uint(0)
	style.Document.Margin = &zero
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""
	style.Heading.Color = &accent
	// Heading styles are copied, never mutate package defaults.
	style.H1.Prefix = ""
	style.H2.Prefix = ""
	style.H3.Prefix = ""
	style.H4.Prefix = ""
	style.H5.Prefix = ""
	style.H6.Prefix = ""
	style.H1.Color = &accent
	style.H2.Color = &accent
	style.H3.Color = &accent
	// Code reads as code without shouting (8bsx4y): a light blue-grey on a
	// faint background, no padding, no red (red is for blocked and lint),
	// and no syntax colours in blocks, which bring red back.
	style.Code.StylePrimitive = gansi.StylePrimitive{Color: &codeFg, BackgroundColor: &codeBg}
	style.CodeBlock.Chroma = nil
	style.CodeBlock.StylePrimitive = gansi.StylePrimitive{Color: &codeFg, BackgroundColor: &codeBg}
	style.CodeBlock.Margin = &zero
	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(style), glamour.WithWordWrap(max(1, width-2)), glamour.WithColorProfile(profile))
	if err != nil {
		return wrap(s, width)
	}
	out, err := renderer.Render(literalAngles(s))
	if err != nil {
		return wrap(s, width)
	}
	// Without syntax colours glamour ends a code block with its reset on a
	// line of its own, which shows as an extra blank line: keep it on the
	// block's last line.
	out = strings.ReplaceAll(out, "\n\x1b[0m\n", "\x1b[0m\n")
	out = strings.TrimSuffix(out, "\n\x1b[0m")
	return strings.Trim(out, "\n")
}

// stageLineStyle is the colour of a task's stage line in its detail: the
// depends red, else dim; a wait on a thing is calm (zhvkmj, crab 02:52).
// What waits on you has no stage line: its question is in the details.
func (m model) stageLineStyle(t task.Task) lipgloss.Style {
	if m.stageOf(t) == "depends" {
		return m.styles.status["depends"]
	}
	return m.styles.dim
}

func (m model) detailTitle(t task.Task) string {
	// subject: the title alone, bold; page focus shows as a ▸ before it,
	// as a focused heading shows it, so it reads without colour too. The
	// id is the breadcrumb's last crumb (header-spec.md, Jørgen 00:02).
	title := oneLine(t.Title)
	w := m.marked(t)(stopTitle, 0)
	if m.detailPane == focusPage {
		return m.styles.focus.Render("▸ ") + m.styleMarked(fit(markPlain(title, w, false), m.width-2), w, m.styles.normal.Bold(true))
	}
	return m.styleMarked(fit(markPlain(title, w, false), m.width), w, m.styles.normal.Bold(true))
}

func (m model) detailHeader(t task.Task) []string {
	title := m.tintFindTitle(m.detailTitle(t))
	badges := []string{m.styles.status[m.boardState(t)].Bold(true).Render("[" + oneLine(m.boardState(t)) + "]")}
	if _, mine := m.askedOf(t); mine {
		// One word, no question text: the question shows once, in the
		// details (0gztsy, crab 01:06).
		badges = append(badges, m.styles.forYou.Bold(true).Render("["+oneLine(m.taskStage(t))+"]"))
	}
	// An unset type or priority shows no badge, not "[]" (2026-11-16, user).
	if v := oneLine(t.Type); v != "" {
		badges = append(badges, m.styles.kind[t.Type].Render("["+v+"]"))
	}
	if v := oneLine(t.Priority); v != "" {
		badges = append(badges, m.styles.priority[t.Priority].Render("["+v+"]"))
	}
	group := t.Group
	if group == "" {
		group = "root"
	}
	owner := "assigned:" + strings.Join(t.Assignee, ", ")
	switch {
	case len(t.Assignee) == 0 && t.Model == "":
		owner = "unassigned"
	case len(t.Assignee) == 0 || len(t.Assignee) == 1 && t.Assignee[0] == t.Model:
		// Taken without a name: the model is the assignee, shown once,
		// with the effort it runs at (jr55s8).
		owner = "assigned:" + t.ModelLabel()
	case t.Model != "":
		owner += " · " + t.ModelLabel()
	}
	// Where the work happens (x1t1t1): the branch it was taken on.
	if b := oneLine(t.Branch); b != "" && owner != "unassigned" {
		owner += " · " + b
	}
	badges = append(badges, m.styles.dim.Render("group:"+oneLine(group)), m.styles.accent.Render(oneLine(owner)))
	// The epic it belongs to, one Tab away (4bwcgp).
	marked, isMarked := m.markedStop(t)
	mark := m.marked(t)
	if p := oneLine(t.Parent); p != "" {
		badge := m.styles.dim.Render("epic:") + m.styles.accent.Render(p)
		if isMarked && marked.where == stopParent {
			badge = m.styles.dim.Render("epic:") + m.styles.selected.Render("▸"+p)
		}
		badges = append(badges, badge)
	}
	// context: the badges, the dates, the stage and any finding; then the
	// subject, right above the rule that opens the details
	// (header-spec.md bands, Jørgen 00:03).
	var lines []string
	lines = append(lines, strings.Split(wrap(strings.Join(badges, "  "), m.width), "\n")...)
	lines = append(lines, m.styles.dim.Render("created "+oneLine(t.Created)+" · updated "+oneLine(t.Updated)+" · "+oneLine(t.Slug)))
	// What it waits on, as its row's second line says it, with the
	// person's long name here (89y0b7, Jørgen 15:33: rows the short name,
	// the detail the long one).
	if _, mine := m.askedOf(t); mine {
		// Asked of you: the stage is its badge; the question is in the
		// details, once (0gztsy, crab 01:06).
	} else if head, text, stamp := m.note(t); head != "" {
		line := "└─ " + m.withLongName(t, head)
		if text != "" {
			line += " " + text
		}
		if s := sinceText(stamp); s != "" {
			line += " · " + s
		}
		if isMarked && marked.where == stopStage {
			// The id it waits on, marked where it stands (4bwcgp); fit
			// after the mark, so the line keeps its width.
			line = fit(m.markWord(line, marked.shown, func(s string) string { return m.stageLineStyle(t).Render(s) }), m.width)
		} else {
			line = m.stageLineStyle(t).Render(fit(line, m.width))
		}
		lines = append(lines, line)
	}
	if warning, fix, ok := m.headFinding(t); ok {
		// Broken is a problem (red); advice only is dim (zhvkmj).
		style := m.styles.dim
		if m.headProblems(t)[0].Severity() == task.Broken {
			style = m.styles.problem
		}
		for i, l := range []string{warning, fix} {
			w := mark(stopFinding, i)
			lines = append(lines, m.styleMarked(fit(markPlain(l, w, false), m.width), w, style))
		}
	}
	// A dim rule above the subject too, so it stands out (Jørgen 00:04):
	// breadcrumb, context, rule, subject, rule, details.
	return append(append(lines, m.questionLines(t)...), m.styles.dim.Render(strings.Repeat("─", m.width)), title)
}

// headProblems are the findings the header shows for t: what the last
// edit broke first, then the rest in line order.
func (m model) headProblems(t task.Task) []task.Finding {
	var problems []task.Finding
	seen := map[string]bool{}
	var fresh []task.Finding
	if nt, fs, ok := m.noticeTask(); ok && nt.Path == t.Path {
		fresh = fs
	}
	for _, f := range append(fresh, m.findings...) {
		if f.Path == t.Path && !seen[f.Problem] {
			seen[f.Problem] = true
			problems = append(problems, f)
		}
	}
	return problems
}

// headFinding is the header's finding: what is wrong, on which line,
// then the fix on its own line so a long problem never cuts it off
// (rx74bp). ok is false when t has none.
func (m model) headFinding(t task.Task) (warning, fix string, ok bool) {
	problems := m.headProblems(t)
	if len(problems) == 0 {
		return "", "", false
	}
	f := problems[0]
	warning = "⚠ "
	if f.Line > 0 {
		warning += fmt.Sprintf("line %d: ", f.Line)
	}
	warning += oneLine(f.Problem)
	if len(problems) > 1 {
		warning += fmt.Sprintf(" (+%d more findings)", len(problems)-1)
	}
	return warning, "  fix: " + oneLine(f.FixLine()), true
}

// asked is the question t asks the viewer: what it asks, who asked,
// when, and the check it is about (0 for none). ok is false when t asks
// the viewer nothing.
type asked struct {
	stage, what, by, at string
	check               int
}

func (m model) askedOf(t task.Task) (asked, bool) {
	if !m.forYou(t) {
		return asked{}, false
	}
	q := asked{stage: m.taskStage(t)}
	switch q.stage {
	case "design":
		q.what, q.by, q.at = t.Fields["design_question"], t.Fields["design_asked_by"], t.Fields["design_since"]
	case "waiting":
		_, q.what = m.waitingFor(t)
		q.by, q.at = t.Fields["waiting_asked_by"], t.Fields["waiting_since"]
	case "asked":
		q.what, q.by, q.at = t.Fields["asked_what"], t.Fields["asked_by"], t.Fields["asked_since"]
		if n, err := strconv.Atoi(strings.TrimSpace(t.Fields["asked_check"])); err == nil && n >= 1 && n <= len(t.Acceptance) && !t.Acceptance[n-1].Checked {
			// A ticked check carries no question (qsqhpe: a tick is the
			// answer); one asked before the tick shows as its own block.
			q.check = n
		}
	case "review":
		q.what, q.by, q.at = "review it: approve, or say what to change", strings.Join(t.Assignee, ", "), t.Fields["review_since"]
	}
	return q, true
}

// questionLines are the header's line for an answer its asker has not
// read yet. The question itself is in the details, once (0gztsy).
func (m model) questionLines(t task.Task) []string {
	var out []string
	if a := t.Answer; a != nil {
		// Every line of the answer, verbatim (41hhgn check 8): the first
		// after who answered, the rest under it. An id in it is marked
		// where it stands (0ga6m0).
		w := m.marked(t)(stopAnswer, 0)
		lines := strings.Split(strings.TrimRight(safe(a.Text), "\n"), "\n")
		for i, l := range lines {
			if marked := markPlain(l, w, false); marked != l {
				lines[i] = marked // the first only
				break
			}
		}
		out = append(out, m.styleMarked(fit(fmt.Sprintf("✓ %s answered %s: %s", oneLine(a.By), a.Stage, lines[0]), m.width), w, m.styles.accent))
		for _, l := range lines[1:] {
			out = append(out, m.styleMarked(fit("  "+l, m.width), w, m.styles.accent))
		}
		out = append(out, m.styles.dim.Render(fit("  unread by "+oneLine(a.For), m.width)))
	}
	return out
}

// askedBy is the dim line under a question: who asked, how long ago, and
// a to answer, or the open answer box. indent hangs it under the text.
func (m model) askedBy(q asked, indent string) []string {
	var by []string
	if q.by != "" {
		by = append(by, oneLine(q.by))
	}
	if s := strings.TrimPrefix(since(q.at), " · "); s != "" {
		by = append(by, s+" ago")
	}
	if m.prompt == answerPrompt {
		// Typed right under the question (Jørgen 11:45), not in the footer.
		out := []string{}
		if len(by) > 0 {
			out = append(out, m.styles.dim.Render(fit(indent+"└ "+strings.Join(by, " · "), m.width)))
		}
		for _, l := range m.answerLines() {
			out = append(out, indent+strings.TrimPrefix(l, answerIndent))
		}
		return out
	}
	line := indent + "└ "
	if len(by) > 0 {
		line += strings.Join(by, " · ") + " · "
	}
	return []string{fit(m.renderMarks(line+mk("a")+"nswer", m.styles.dim), m.width)}
}

// questionBlock is a question that names no check, at the top of the
// details, drawn as a check row is (0gztsy, crab 01:06): "? " and the
// question whole, wrapped, never cut; who asked under it. Focused, it
// carries the > and the highlight a focused check has.
func (m model) questionBlock(t task.Task) []string {
	q, ok := m.askedOf(t)
	if !ok || q.check > 0 {
		return nil
	}
	focused := m.detailPane == focusQuestion
	pointer := " "
	if focused {
		pointer = ">"
	}
	prefix := pointer + " ? "
	indent := strings.Repeat(" ", len(prefix))
	w := m.marked(t)(stopQuestion, 0)
	text := markPlain(questionText(q), w, false)
	var lines []string
	for i, para := range strings.Split(text, "\n") {
		for j, l := range strings.Split(wrap(para, max(10, m.width-len(indent))), "\n") {
			if i == 0 && j == 0 {
				lines = append(lines, prefix+l)
			} else {
				lines = append(lines, indent+l)
			}
		}
	}
	label := strings.Join(lines, "\n")
	if focused {
		// Calm (kg07bt): the question in its own colour, the > alone
		// drawn as the cursor.
		label = m.cursorMark(label, m.styles.forYou)
	} else {
		label = m.styleMarked(label, w, m.styles.forYou)
	}
	return append([]string{label}, m.askedBy(q, indent)...)
}

// questionText is a question's text as its block shows it: the stage
// first unless it is a plain ask.
func questionText(q asked) string {
	text := strings.TrimSpace(q.what)
	if q.stage != "asked" {
		text = q.stage + ": " + text
	}
	return text
}

// checkText is a check's text as its row shows it: one line, and a
// struck check's decision after it (sfa8w4).
func checkText(a task.Acceptance) string {
	text := oneLine(cleanMarkdown(a.Text))
	if a.Struck {
		why := "not relevant: " + oneLine(a.Reason)
		if a.StruckBy != "" {
			why += ", " + oneLine(a.StruckBy)
		}
		if a.StruckOn != "" {
			why += ", " + oneLine(a.StruckOn)
		}
		text += " (" + why + ")"
	}
	return text
}

// sectionText is the text of every section called name, as the detail
// draws it.
func sectionText(sections []detailSection, name string) string {
	var out []string
	for _, s := range sections {
		if s.name == name && strings.TrimSpace(s.text) != "" {
			out = append(out, s.text)
		}
	}
	return strings.Join(out, "\n")
}

// otherSection: a section the detail draws as written, in file order,
// after the Files, not in a place of its own.
func otherSection(name string) bool {
	for _, n := range task.DescriptionSections {
		if strings.ToLower(n) == name {
			return false
		}
	}
	return !contains([]string{"description", "acceptance", "dependencies", "notes", "history", "requests"}, name)
}

type detailSection struct{ name, text string }

// doc is t's parsed file: from the last reload, or read now and kept until
// the next one (with Options.Lint a reload parses no file, 44w1ev).
func (m model) doc(t task.Task) *task.Document {
	if d := m.docs[t.Path]; d != nil || m.docs == nil {
		return d
	}
	b, e := sharefile.ReadFile(filepath.Join(m.store.Root, filepath.FromSlash(t.Path)))
	if e != nil {
		return nil
	}
	d := task.Parse(t.Path, b)
	m.docs[t.Path] = d
	return d
}

func (m model) sections(t task.Task) []detailSection {
	d := m.doc(t)
	if d == nil {
		return []detailSection{{"description", cleanMarkdown(t.Goal)}}
	}
	raw := strings.Split(strings.ReplaceAll(cleanMarkdown(string(d.Bytes())), "\r\n", "\n"), "\n")
	start := 0
	if len(raw) > 0 && raw[0] == "---" {
		for i := 1; i < len(raw); i++ {
			if raw[i] == "---" {
				start = i + 1
				break
			}
		}
	}
	out := []detailSection{{name: "description"}}
	fence := ""
	for _, line := range raw[start:] {
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) <= 3 && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			if fence == "" {
				n := 0
				for n < len(trimmed) && trimmed[n] == trimmed[0] {
					n++
				}
				fence = trimmed[:n]
			} else if strings.HasPrefix(trimmed, fence) && strings.TrimSpace(strings.TrimPrefix(trimmed, fence)) == "" {
				fence = ""
			}
		}
		if fence == "" && strings.HasPrefix(line, "## ") {
			name := strings.ToLower(strings.TrimSpace(line[3:]))
			switch name {
			case "goal":
				name = "description"
			case "depends on":
				name = "dependencies"
			case "acceptance criteria":
				name = "acceptance"
			}
			out = append(out, detailSection{name: name})
			continue
		}
		out[len(out)-1].text += line + "\n"
	}
	return out
}

// Detail focus: the page scrolls, or one list (acceptance checks or
// dependencies) has a selection that j/k step through. Exactly one place
// shows focus: the task id in the tab bar for the page, else the focused
// section's heading (2026-11-16, user).
const (
	focusPage = iota
	focusAcceptance
	focusDeps
	focusLinks // the task:ID references in the text (xxvm6t)
	focusFiles // the files in the task's folder (zgr01t)
	focusIDs   // a task id marked in the header, Children or Notes (4bwcgp)
	// focusQuestion: the question a task asks you that names no check, at
	// the top of the details (0gztsy). One that names a check is focused
	// on that check.
	focusQuestion
)

// startFocus is the focus a freshly opened task gets: its checks when it has
// any, otherwise the page. A question to you with a picture opens on the
// page, so the picture under the question is what you see (5z4zt4).
func (m *model) startFocus() {
	m.acceptance, m.dependency, m.link, m.file, m.idAt = 0, 0, 0, 0, 0
	m.glance, m.glanceHead, m.glanceBody = false, "", ""
	m.find = findState{} // a find is the page's: a new page starts without (4s3g5q)
	m.detailPane = focusPage
	t, ok := m.selected()
	if !ok {
		return
	}
	// Where the reader's answer goes (0gztsy): the check the question is
	// about, or the question itself.
	if q, mine := m.askedOf(t); mine {
		if q.check > 0 {
			m.detailPane, m.acceptance = focusAcceptance, q.check-1
		} else {
			m.detailPane = focusQuestion
		}
		return
	}
	if len(t.Acceptance) > 0 && m.askedPicture(t) == "" {
		m.detailPane = focusAcceptance
	}
}

// focusStop is one place the detail's focus ring stops: a pane, and for
// a list of ids (dependencies, links, ids in the header, Children or
// Notes) which one.
type focusStop struct{ pane, index int }

// focusRing is every place h/l and Tab/Shift-Tab stop in t's detail, in
// reading order (e3sevr, Jørgen 23:28: Tab = l, Shift-Tab = h, one ring):
// the ids in the header, the page, the ids in the title and the
// description, the checks and the ids in them, each dependency, each
// child, each link, the files, and the ids in the text below (0ga6m0).
// Enter opens a stop that is an id.
func (m model) focusRing(t task.Task) []focusStop {
	stops := m.idStops(t)
	var ring []focusStop
	ids := func(where ...string) {
		for k, s := range stops {
			if contains(where, s.where) {
				ring = append(ring, focusStop{focusIDs, k})
			}
		}
	}
	// The header, then the title (the page stop), the details top down;
	// an id stands where idStops found it (0ga6m0).
	ids(stopParent, stopStage, stopFinding, stopAnswer)
	ring = append(ring, focusStop{focusPage, 0})
	ids(stopTitle)
	if q, mine := m.askedOf(t); mine && q.check == 0 {
		ring = append(ring, focusStop{focusQuestion, 0})
	}
	ids(stopQuestion, stopText)
	if len(t.Acceptance) > 0 {
		ring = append(ring, focusStop{focusAcceptance, 0})
	}
	ids(stopCheck, stopAsked)
	for i := range t.Dependencies {
		ring = append(ring, focusStop{focusDeps, i})
		for k, s := range stops {
			if s.where == stopDepNote && s.index == i {
				ring = append(ring, focusStop{focusIDs, k})
			}
		}
	}
	ids(stopWait, stopChildren)
	for i, l := range m.links(t) {
		if !l.Missing { // an id that is no task is not a stop
			ring = append(ring, focusStop{focusLinks, i})
		}
	}
	if len(m.filesOf(t)) > 0 {
		ring = append(ring, focusStop{focusFiles, 0})
	}
	ids(stopSection, stopRequest, stopResolved, stopNotes, stopHistory)
	return ring
}

// ringAt is where the focus is in ring, or -1.
func (m model) ringAt(ring []focusStop) int {
	for k, s := range ring {
		if s.pane != m.detailPane {
			continue
		}
		switch s.pane {
		case focusDeps:
			if s.index == m.dependency {
				return k
			}
		case focusLinks:
			if s.index == m.link {
				return k
			}
		case focusIDs:
			if s.index == m.idAt {
				return k
			}
		default:
			return k
		}
	}
	return -1
}

// cycleFocus moves focus one stop along the ring: l and Tab forward (step
// 1), h and Shift-Tab back (-1), around the end.
func (m *model) cycleFocus(step int) {
	t, ok := m.selected()
	if !ok {
		return
	}
	ring := m.focusRing(t)
	if len(ring) == 1 {
		m.message = "nothing to focus: this task has no checks, task ids, links or files"
		return
	}
	at := m.ringAt(ring)
	switch {
	case at < 0 && step > 0:
		at = 0
	case at < 0:
		at = len(ring) - 1
	default:
		at = (at + step + len(ring)) % len(ring)
	}
	s := ring[at]
	m.detailPane = s.pane
	switch s.pane {
	case focusDeps:
		m.dependency = s.index
	case focusLinks:
		m.link = s.index
	case focusIDs:
		m.idAt = s.index
	}
}

// sectionHeading is a section's name in capitals. The focused section gets
// the highlight, a ▸ that shows without colour, and the keys that work in it;
// keys are never upper-cased, since J and K are other keys.
func (m model) sectionHeading(title, hint string, focused bool) string {
	head := strings.ToUpper(title)
	if !focused {
		return m.styles.accent.Render(head)
	}
	// Focused: the heading as every other one, a ▸ before it and its keys
	// dim after, no fill (kg07bt: the block competed with the question).
	out := m.styles.accent.Render("▸ " + head)
	if hint != "" {
		out += m.styles.dim.Render(fit(" · "+hint, max(0, m.width-len([]rune(head))-2)))
	}
	return out
}
func (m model) detailText(t task.Task) string {
	sections := m.sections(t)
	var b strings.Builder
	// The picture the question asks you to look at opens the page, right
	// under the question in the header (5z4zt4): what --show named, else a
	// task file the question names. Read here, when the page is built, not
	// on every frame.
	if block := m.questionBlock(t); len(block) > 0 && m.detail {
		b.WriteString(strings.Join(block, "\n") + "\n\n")
	}
	if name := m.askedPicture(t); name != "" && m.detail {
		b.WriteString(strings.Join(m.pictureLines(t, name), "\n") + "\n\n")
	}
	section := func(heading, body, empty string) {
		if strings.TrimSpace(body) == "" {
			if empty != "" {
				b.WriteString(m.styles.dim.Render(empty) + "\n\n")
			}
			return
		}
		b.WriteString(heading + "\n" + body + "\n\n")
	}
	plain := func(title string) string { return m.sectionHeading(title, "", false) }
	text := func(name string) string { return sectionText(sections, name) }
	// Every id the page shows is marked where it stands when Tab is on it
	// (0ga6m0): rendered text is marked after rendering, plain text before
	// it is wrapped, so the mark's column is counted.
	markAt := m.marked(t)
	markdown := func(where string, index int, s string) string {
		return m.markStyled(m.markdown(s), markAt(where, index), true)
	}
	section(plain("Description"), markdown(stopText, 0, text("description")), "no description yet")
	// What the task is about comes first (jnaspy): Purpose, Principle and
	// Relevant details right under the goal, above the checks.
	for i, name := range task.DescriptionSections {
		section(plain(name), markdown(stopText, i+1, text(strings.ToLower(name))), "")
	}
	var checks []string
	q, mine := m.askedOf(t)
	for i, a := range t.Acceptance {
		box, pointer := "☐", " "
		if a.Checked {
			box = "☑"
		}
		if a.Struck {
			box = "☒"
		}
		if m.detailPane == focusAcceptance && i == m.acceptance {
			pointer = ">"
		}
		// Wrap to the room left after the prefix and hang continuation lines
		// under the text, so each item reads as one block (2026-11-15).
		mark := ""
		if mine && q.check == i+1 {
			// The check the question is about carries it (0gztsy): the
			// same box, number and text as the list, ? before them.
			mark = "? "
		}
		prefix := fmt.Sprintf("%s %s%s %d. ", pointer, mark, box, i+1)
		indent := strings.Repeat(" ", lipgloss.Width(prefix))
		w := markAt(stopCheck, i)
		body := wrap(markPlain(checkText(a), w, true), max(10, m.width-len(indent)))
		lines := strings.Split(body, "\n")
		for j := range lines {
			if j == 0 {
				lines[j] = prefix + lines[j]
			} else {
				lines[j] = indent + lines[j]
			}
		}
		label := strings.Join(lines, "\n")
		if m.detailPane == focusAcceptance && i == m.acceptance && mark != "" {
			// The check asked of you, focused (kg07bt, Jørgen 14:58: "too
			// much visual noise"): a normal row, the > the one cursor.
			checks = append(checks, m.cursorMark(label, m.checkStyle(a.Checked)))
		} else if m.detailPane == focusAcceptance && i == m.acceptance {
			checks = append(checks, m.styles.selected.Render(label))
		} else if a.Struck {
			// Struck out with a reason (sfa8w4): the text crossed through
			// and dim, the decision after it.
			checks = append(checks, m.styleMarked(label, w, m.styles.dim.Strikethrough(true)))
		} else {
			checks = append(checks, m.styleMarked(label, w, m.checkStyle(a.Checked)))
		}
		if mark != "" {
			// The question once, under its check: what it asks, then who.
			asked := markAt(stopAsked, i)
			if what := markPlain(strings.TrimSpace(q.what), asked, false); what != "" {
				for _, para := range strings.Split(what, "\n") {
					for _, l := range strings.Split(wrap(para, max(10, m.width-len(indent))), "\n") {
						checks = append(checks, m.styleMarked(indent+l, asked, m.styles.forYou))
					}
				}
			}
			checks = append(checks, m.askedBy(q, indent)...)
		}
	}
	section(m.sectionHeading("Acceptance", "j/k or [ ] step · Space ticks · - strikes out", m.detailPane == focusAcceptance), strings.Join(checks, "\n"), "no acceptance items yet")
	var dependencies []string
	for i, d := range task.DependencyDetails(t, m.all) {
		state := d.State
		if d.Missing {
			state = "depends"
		}
		pointer := " "
		if m.detailPane == focusDeps && i == m.dependency {
			pointer = ">"
		}
		flag := ""
		if d.Blocking {
			flag = " open"
		}
		if d.OpenChildren > 0 {
			flag += fmt.Sprintf(" (epic: %d open)", d.OpenChildren)
		}
		line := fmt.Sprintf("%s ↳ %s %s [%s]%s %s", pointer, m.styles.status[state].Render(oneLine(d.ID)), oneLine(d.Title), oneLine(state), flag, m.markStyled(oneLine(d.Note), markAt(stopDepNote, i), false))
		if pointer == ">" {
			line = m.styles.selected.Render(ansi.Strip(line))
		}
		dependencies = append(dependencies, line)
	}
	// Conditions (16dxsr), after the task dependencies: what it waits for
	// and what hug saw when it checked. Not focusable: there is nothing to
	// follow.
	for i, w := range task.Waits(t, m.all) {
		flag, style := " WAITING", m.styles.status["blocked"]
		if w.Met {
			flag, style = " met", m.styles.status["done"]
		}
		line := fmt.Sprintf("  ⧗ when %s%s · %s", m.markStyled(oneLine(w.Text), markAt(stopWait, i), false), style.Render(flag), m.styles.dim.Render(oneLine(w.Seen)))
		if w.Wakes != "" {
			line += m.styles.dim.Render(" · wakes " + oneLine(w.Wakes))
		}
		dependencies = append(dependencies, line)
	}
	section(m.sectionHeading("Dependencies", "j/k or [ ] step · Enter follows", m.detailPane == focusDeps), strings.Join(dependencies, "\n"), "no dependencies")
	// Children: an epic's tasks, each one Tab away (4bwcgp).
	var children []string
	for i, c := range m.children(t) {
		pointer := " "
		marked, ok := m.markedStop(t)
		if ok && marked.where == stopChildren && marked.index == i {
			pointer = ">"
		}
		line := fmt.Sprintf("%s ↳ %s %s [%s]", pointer, m.styles.status[m.place(c)].Render(oneLine(c.ID)), oneLine(c.Title), oneLine(m.place(c)))
		if pointer == ">" {
			line = m.styles.selected.Render(ansi.Strip(line))
		}
		children = append(children, line)
	}
	section(plain(fmt.Sprintf("Children (%d)", len(children))), strings.Join(children, "\n"), "")
	// Links: every task:ID the text names, to follow with Enter (xxvm6t).
	var links []string
	for i, l := range m.links(t) {
		pointer := " "
		if m.detailPane == focusLinks && i == m.link {
			pointer = ">"
		}
		state, title := l.State, l.Title
		if l.Missing {
			state, title = "depends", "no task has this id"
		}
		line := fmt.Sprintf("%s ↳ %s %s [%s]", pointer, m.styles.status[state].Render(oneLine(l.ID)), oneLine(title), oneLine(state))
		if l.Missing {
			line = fmt.Sprintf("%s ↳ %s %s", pointer, m.styles.status[state].Render(oneLine(l.ID)), title)
		}
		if pointer == ">" {
			line = m.styles.selected.Render(ansi.Strip(line))
		}
		links = append(links, line)
	}
	section(m.sectionHeading("Links", "j/k or [ ] step · Enter follows", m.detailPane == focusLinks), strings.Join(links, "\n"), "")
	// Files: the task's folder, each opening in the editor (zgr01t).
	section(m.sectionHeading(fmt.Sprintf("Files (%d)", len(m.filesOf(t))), "j/k or [ ] step · Space shows · Enter edits · E folder", m.detailPane == focusFiles), strings.Join(m.filesLines(t), "\n"), "")
	// Any other section, in file order, then Notes and History last.
	for i, s := range sections {
		if otherSection(s.name) {
			section(plain(s.name), markdown(stopSection, i, s.text), "")
		}
	}
	// Requests (hbgybn): each verbatim, with its account once resolved;
	// the box r opens sits at its end.
	if reqs := m.requestLines(t); len(reqs) > 0 {
		m.markRequests(t, reqs, markAt)
		heading := plain("Requests")
		if m.prompt == requestPrompt {
			heading = m.sectionHeading("Requests", "Enter sends · Alt-Enter a new line · Esc cancels", true)
		}
		section(heading, strings.Join(reqs, "\n"), "")
	}
	section(plain("Notes"), markdown(stopNotes, 0, text("notes")), "")
	// History has no Links (links), so a task:ID there is marked too.
	section(plain("History"), m.markStyled(m.markdown(text("history")), markAt(stopHistory, 0), false), "")
	return wrap(strings.TrimRight(b.String(), "\n"), m.width)
}

// markRequests marks, in requestLines' lines, the id the focus is on in
// a request's text or its account: request i takes its head line, a line
// per line of its text, then one per line of its account.
func (m model) markRequests(t task.Task, lines []string, markAt func(string, int) string) {
	at := 0
	markIn := func(n int, w string) {
		for k := at; k < min(at+n, len(lines)); k++ {
			if marked := m.markStyled(lines[k], w, true); marked != lines[k] {
				lines[k] = marked
				break
			}
		}
		at += n
	}
	for i, r := range t.Requests {
		at++ // who and when
		markIn(len(strings.Split(strings.TrimRight(r.Text, "\n"), "\n")), markAt(stopRequest, i))
		if a := r.Resolved; a != nil {
			markIn(len(strings.Split(strings.TrimRight(a.Text, "\n"), "\n")), markAt(stopResolved, i))
		}
	}
}

// openTop shows a just-opened task from the top, scrolled only as far as
// needed to keep the focused check in view: the header can grow (a finding
// and its fix take two lines, rx74bp) and must not hide the focus.
func (m *model) openTop() {
	m.viewport.GotoTop()
	if m.detailPane == focusPage {
		return
	}
	if t, ok := m.selected(); ok {
		for i, line := range strings.Split(ansi.Strip(m.detailText(t)), "\n") {
			if strings.HasPrefix(line, "> ") {
				if i >= m.viewport.Height {
					m.viewport.SetYOffset(i - m.viewport.Height + 1)
				}
				return
			}
		}
	}
}
func (m *model) revealAcceptance() {
	if m.detailPane == focusPage {
		return // nothing is marked; the page keeps its scroll
	}
	t, ok := m.selected()
	if !ok {
		return
	}
	// An id is marked ▸ where it stands, in the text (0ga6m0); in the
	// header it is always on screen.
	marker := func(line string) bool { return strings.HasPrefix(line, "> ") }
	if s, ok := m.markedStop(t); ok {
		marker = func(line string) bool { return strings.Contains(line, "▸"+s.shown) }
	}
	for i, line := range strings.Split(ansi.Strip(m.detailText(t)), "\n") {
		if marker(line) {
			if i < m.viewport.YOffset || i >= m.viewport.YOffset+m.viewport.Height {
				m.viewport.SetYOffset(max(0, i-m.viewport.Height/2))
			}
			break
		}
	}
}
func (m *model) followDependency() {
	t, ok := m.selected()
	if !ok {
		return
	}
	deps := task.DependencyDetails(t, m.all)
	if len(deps) == 0 {
		m.message = "no dependency to follow"
		return
	}
	d := deps[min(m.dependency, len(deps)-1)]
	if d.Missing {
		m.message = "dependency not found: " + d.ID
		return
	}
	target, err := task.Resolve(m.all, d.ID)
	if err != nil {
		m.message = err.Error()
		return
	}
	m.detailStack = append(m.detailStack, t.Path)
	m.detailPath = target.Path
	m.startFocus()
	m.syncDetail()
	m.openTop()
}

// cursorMark draws a focused row calm (kg07bt): its text in style, as it
// reads unfocused, and its leading > as the cursor, the one underlined
// accent cell on the page, so nothing else competes with it.
func (m model) cursorMark(label string, style lipgloss.Style) string {
	lines := strings.Split(label, "\n")
	for i, l := range lines {
		if i == 0 && strings.HasPrefix(l, ">") {
			lines[i] = m.styles.accent.Underline(true).Render(">") + style.Render(l[1:])
		} else {
			lines[i] = style.Render(l)
		}
	}
	return strings.Join(lines, "\n")
}

// checkStyle colours an acceptance check: ticked ones green, open ones in the
// plain text colour, so they do not read as headings, which are blue
// (2026-11-16, user).
func (m model) checkStyle(checked bool) lipgloss.Style {
	if checked {
		return m.styles.status["done"]
	}
	return m.styles.normal
}
