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
// calm depends colour (zhvkmj check 6: not red), else dim; a wait on a
// thing is calm (zhvkmj, crab 02:52).
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
	// n is the question's number on the task (q2adr8), 0 for a stage
	// question with none (design, review, an ask written before q2adr8).
	n int
	// several: the task asks the viewer more than one, so each says its
	// number.
	several bool
	// file is the task file the question asks you to look at (--show,
	// else one its text names), "" for none. It is a link line under the
	// question; Enter shows it in the file frame (qcq3tm).
	file string
}

// askedOf is the first question t asks the viewer, the oldest.
func (m model) askedOf(t task.Task) (asked, bool) {
	all := m.askedAll(t)
	if len(all) == 0 {
		return asked{}, false
	}
	return all[0], true
}

// askedAll is every open question t asks the viewer, oldest first
// (q2adr8: an agent asks one question per ask; each has its own answer).
func (m model) askedAll(t task.Task) []asked {
	if !m.forYou(t) {
		return nil
	}
	if g := m.taskStage(t); (g == "asked" || t.State == "todo") && len(t.Questions) > 0 {
		if g != "asked" {
			g = "asked" // a plain ask (yphbra): the block shows no stage word
		}
		var out []asked
		for _, q := range task.OpenQuestions(t) {
			if !m.isMe(q.To) {
				continue
			}
			a := asked{stage: g, what: q.Text, by: q.By, at: q.At, n: q.N, file: m.questionPictureOf(t, q.Show, q.Text)}
			if n := q.Check; n >= 1 && n <= len(t.Acceptance) && !t.Acceptance[n-1].Checked {
				a.check = n
			}
			out = append(out, a)
		}
		for i := range out {
			out[i].several = len(out) > 1
		}
		return out
	}
	q, _ := m.stageQuestion(t)
	return []asked{q}
}

// blockQuestions are the questions t asks the viewer that name no check:
// each a block at the top of the details, a stop of its own.
func (m model) blockQuestions(t task.Task) []asked {
	var out []asked
	for _, q := range m.askedAll(t) {
		if q.check == 0 {
			out = append(out, q)
		}
	}
	return out
}

// checkQuestion is the question t asks the viewer about check n (1-based).
func (m model) checkQuestion(t task.Task, n int) (asked, bool) {
	for _, q := range m.askedAll(t) {
		if q.check == n {
			return q, true
		}
	}
	return asked{}, false
}

// stageQuestion is the one question the stage's fields ask.
func (m model) stageQuestion(t task.Task) (asked, bool) {
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
	if q.stage == "review" {
		q.file = m.questionPicture(t, q.stage, "")
	} else {
		q.file = m.questionPicture(t, q.stage, q.what)
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
		out = append(out, m.styleMarked(fit(fmt.Sprintf("✓ %s answered %s: %s", oneLine(a.By), answeredWhat(*a), lines[0]), m.width), w, m.styles.accent))
		for _, l := range lines[1:] {
			out = append(out, m.styleMarked(fit("  "+l, m.width), w, m.styles.accent))
		}
		out = append(out, m.styles.dim.Render(fit("  unread by "+oneLine(a.For), m.width)))
	}
	return out
}

// fileLink is the dim line under a question that names its file
// (qcq3tm, Jørgen: "the file as a link"): never the file drawn on the
// page; Enter on the question shows it in the file frame.
func (m model) fileLink(q asked, indent string) []string {
	if q.file == "" {
		return nil
	}
	return []string{m.styles.dim.Render(fit(indent+"→ "+oneLine(q.file)+" · Enter shows", m.width))}
}

// askedBy is the dim line under a question: who asked, how long ago, and
// a to answer, or the open answer box. indent hangs it under the text.
func (m model) askedBy(q asked, indent string) []string {
	var by []string
	if q.several {
		by = append(by, "question "+strconv.Itoa(q.n))
	}
	if q.by != "" {
		by = append(by, oneLine(q.by))
	}
	if s := strings.TrimPrefix(since(q.at), " · "); s != "" {
		by = append(by, s+" ago")
	}
	if m.prompt == answerPrompt && q.n == m.answerN {
		// Typed right under the question (Jørgen 11:45), not in the footer;
		// under the one it answers when there are several (q2adr8).
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

// questionBlock is the questions that name no check, at the top of the
// details, each drawn as a check row is (0gztsy, crab 01:06): "? " and the
// question whole, wrapped, never cut; who asked under it. The focused one
// carries the > and the highlight a focused check has. Several questions
// (q2adr8) are several blocks, each its own stop and its own answer.
func (m model) questionBlock(t task.Task) []string {
	var out []string
	for k, q := range m.blockQuestions(t) {
		if k > 0 {
			out = append(out, "")
		}
		focused := m.detailPane == focusQuestion && m.question == k
		pointer := " "
		if focused {
			pointer = ">"
		}
		prefix := pointer + " ? "
		indent := strings.Repeat(" ", len(prefix))
		w := m.marked(t)(stopQuestion, k)
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
		out = append(append(append(out, label), m.fileLink(q, indent)...), m.askedBy(q, indent)...)
	}
	return out
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
// draws it. A file with both "## Details" and the old "## Relevant
// details" (hxqqdp) shows both, the Details text first, as hug task show.
func sectionText(sections []detailSection, name string) string {
	var out []string
	for _, old := range []bool{false, true} {
		for _, s := range sections {
			if s.name == name && s.old == old && strings.TrimSpace(s.text) != "" {
				out = append(out, s.text)
			}
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
	return !contains([]string{"description", "acceptance", "dependencies", "notes", "history", "requests", "questions"}, name)
}

type detailSection struct {
	name, text string
	old        bool // headed by the name before 8a0ybb ("Relevant details")
}

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

// sections are t's file split into its sections, as the page draws them
// (worked out once per file: pageMemo).
func (m model) sections(t task.Task) []detailSection { return m.facts(t).sections }

func parseSections(t task.Task, d *task.Document) []detailSection {
	if d == nil {
		return []detailSection{{name: "description", text: cleanMarkdown(t.Goal)}}
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
			name, old := strings.ToLower(strings.TrimSpace(line[3:])), false
			switch name {
			case "goal":
				name = "description"
			case "relevant details":
				// The name before 8a0ybb: the same section (task.OldDetails).
				name, old = "details", true
			case "depends on":
				name = "dependencies"
			case "acceptance criteria":
				name = "acceptance"
			}
			out = append(out, detailSection{name: name, old: old})
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
	// focusFold: a long section's heading, folded to one line until Enter
	// opens it (8a0ybb).
	focusFold
)

// startFocus is the focus a freshly opened task gets: the question it
// asks you, on its check or its own block, whatever file it names
// (qcq3tm); else its checks when it has any, otherwise the page.
func (m *model) startFocus() {
	m.acceptance, m.dependency, m.link, m.file, m.idAt, m.fold = 0, 0, 0, 0, 0, 0
	m.unfolded = nil // a new page opens with its long sections folded (8a0ybb)
	m.glance, m.glanceHead, m.glanceBody = false, "", ""
	m.glanceReturn = nil
	m.glanceRepo, m.glanceSection, m.inline = "", "", nil
	m.find = findState{} // a find is the page's: a new page starts without (4s3g5q)
	m.detailPane = focusPage
	t, ok := m.selected()
	if !ok {
		return
	}
	// Where the reader's answer goes (0gztsy): the check the question is
	// about, or the question itself.
	if q, mine := m.askedOf(t); mine {
		m.focusOn(t, q)
		return
	}
	if len(t.Acceptance) > 0 {
		m.detailPane = focusAcceptance
	}
}

// focusOn puts the focus on question q: on the check it is about, else
// on its own block.
func (m *model) focusOn(t task.Task, q asked) {
	if q.check > 0 {
		m.detailPane, m.acceptance = focusAcceptance, q.check-1
		return
	}
	m.detailPane, m.question = focusQuestion, 0
	for k, b := range m.blockQuestions(t) {
		if b.n == q.n {
			m.question = k
		}
	}
}

// focusedQuestion is the question asked of you where the focus is: the
// focused block, or the focused check's question.
func (m model) focusedQuestion(t task.Task) (asked, bool) {
	switch m.detailPane {
	case focusQuestion:
		if bs := m.blockQuestions(t); m.question >= 0 && m.question < len(bs) {
			return bs[m.question], true
		}
	case focusAcceptance:
		return m.checkQuestion(t, m.acceptance+1)
	}
	return asked{}, false
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
	for k := range m.blockQuestions(t) {
		ring = append(ring, focusStop{focusQuestion, k})
	}
	ids(stopQuestion, stopText)
	// Details after the Principle (hya1st): short, its ids; long, its
	// heading, then its ids while open.
	folds := m.folds(t)
	if len(folds) > 0 && folds[0].key == "details" {
		ring = append(ring, focusStop{focusFold, 0})
	}
	ids(stopDetails)
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
	ids(stopSection, stopRequest, stopResolved)
	// The long sections, folded (8a0ybb): each heading a stop, and the
	// ids in one only while it is open (idStops scans no folded text).
	for k, f := range folds {
		if f.key == "details" {
			continue // after the Principle, above
		}
		ring = append(ring, focusStop{focusFold, k})
		ids(f.stop)
	}
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
		case focusQuestion:
			if s.index == m.question {
				return k
			}
		case focusFold:
			if s.index == m.fold {
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
	m.inline = nil // a file shown in line belongs to the link it was shown from (zbdbtm)
	m.detailPane = s.pane
	switch s.pane {
	case focusDeps:
		m.dependency = s.index
	case focusLinks:
		m.link = s.index
	case focusIDs:
		m.idAt = s.index
	case focusQuestion:
		m.question = s.index
	case focusFold:
		m.fold = s.index
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
	// The questions that name no check open the details. A file a
	// question asks you to look at is a link line under it, never drawn
	// on the page (qcq3tm: drawn above the page it hid the question).
	if block := m.questionBlock(t); len(block) > 0 && m.detail {
		b.WriteString(strings.Join(block, "\n") + "\n\n")
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
	// The page reads small to large (hya1st, Jørgen 2026-11-24 18:23Z:
	// "Goal, Purpose, Principles, Details"; 18:24Z "What, Why, How,
	// Notes"): the Goal (what), Purpose (why), Principle (how), then
	// Details (the notes that explain them), open when short, else folded
	// in this place; then the checks, the short sections, and last the
	// long ones folded.
	section(plain("Goal"), markdown(stopText, 0, text("description")), "no goal yet")
	section(plain("Purpose"), markdown(stopText, 1, text("purpose")), "")
	section(plain("Principle"), markdown(stopText, 2, text("principle")), "")
	folds := m.folds(t)
	foldAt := func(k int) {
		f := folds[k]
		open := m.unfolded[f.key]
		b.WriteString(m.foldLine(f, k, open) + "\n")
		if !open {
			b.WriteString("\n")
			return
		}
		b.WriteString(m.foldBody(t, f.key, sections, markAt) + "\n\n")
	}
	if m.detailsShort(t) {
		section(plain("Details"), m.foldBody(t, "details", sections, markAt), "")
	} else if len(folds) > 0 && folds[0].key == "details" {
		foldAt(0)
	}
	var checks []string
	for i, a := range t.Acceptance {
		q, mine := m.checkQuestion(t, i+1)
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
			checks = append(checks, m.focusRow(label))
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
			checks = append(append(checks, m.fileLink(q, indent)...), m.askedBy(q, indent)...)
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
		line := m.hangRow(pointer+" ↳ ", fmt.Sprintf("%s %s [%s]%s %s", m.styles.status[state].Render(oneLine(d.ID)), oneLine(d.Title), oneLine(state), flag, m.markStyled(oneLine(d.Note), markAt(stopDepNote, i), false)))
		if pointer == ">" {
			line = m.focusRow(ansi.Strip(line))
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
		line := fmt.Sprintf("when %s%s · %s", m.markStyled(oneLine(w.Text), markAt(stopWait, i), false), style.Render(flag), m.styles.dim.Render(oneLine(w.Seen)))
		if w.Wakes != "" {
			line += m.styles.dim.Render(" · wakes " + oneLine(w.Wakes))
		}
		dependencies = append(dependencies, m.hangRow("  ⧗ ", line))
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
		line := m.hangRow(pointer+" ↳ ", fmt.Sprintf("%s %s [%s]", m.styles.status[m.place(c)].Render(oneLine(c.ID)), oneLine(c.Title), oneLine(m.place(c))))
		if pointer == ">" {
			line = m.focusRow(ansi.Strip(line))
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
		line := m.hangRow(pointer+" ↳ ", fmt.Sprintf("%s %s [%s]", m.styles.status[state].Render(oneLine(l.ID)), oneLine(title), oneLine(state)))
		if l.Missing {
			line = m.hangRow(pointer+" ↳ ", fmt.Sprintf("%s %s", m.styles.status[state].Render(oneLine(l.ID)), title))
		}
		if pointer == ">" {
			line = m.focusRow(ansi.Strip(line))
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
	if reqs := m.requestLines(t, markAt); len(reqs) > 0 {
		heading := plain("Requests")
		if m.prompt == requestPrompt {
			heading = m.sectionHeading("Requests", "Enter sends · Alt-Enter a new line · Esc cancels", true)
		}
		section(heading, strings.Join(reqs, "\n"), "")
	}
	// The long sections last, each folded to its heading line until Enter
	// opens it (8a0ybb): the Questions record (q2adr8: each with its
	// answer; one open and asked of you is at the top or under its check,
	// not here), Notes, History. A long Details folded above, after the
	// Principle (hya1st).
	for k, f := range folds {
		if f.key != "details" {
			foldAt(k)
		}
	}
	// A file Space shows in line, under the line naming it (zbdbtm).
	return m.withInline(t, m.wrapPage(strings.TrimRight(b.String(), "\n"), m.width))
}

// hangRow is a Dependencies, Children, Links or Files row: prefix (the
// pointer and its sign, 4 columns) then text, wrapped to the page and
// hung under the text (cd63d1), so a long note never starts a line at
// column 0.
func (m model) hangRow(prefix, text string) string {
	return strings.Join(hang(prefix, text, lipgloss.Width(prefix), m.width), "\n")
}

// foldBody is a long section's text as the page draws it open.
func (m model) foldBody(t task.Task, key string, sections []detailSection, markAt func(string, int) string) string {
	text := func(name string) string { return sectionText(sections, name) }
	switch key {
	case "details":
		return m.markStyled(m.markdown(text("details")), markAt(stopDetails, 0), true)
	case "questions":
		return strings.Join(m.questionRecord(t, markAt), "\n")
	case "notes":
		// One entry a paragraph, as the heading counts them: written
		// one a line, markdown would run them together.
		return m.markStyled(m.markdown(entryParagraphs(text("notes"))), markAt(stopNotes, 0), true)
	case "history":
		// History has no Links (links), so a task:ID there is marked too.
		return m.markStyled(m.markdown(text("history")), markAt(stopHistory, 0), false)
	}
	return ""
}

// foldText is a long section's text as drawn open, wrapped to the page,
// for a find to look in (openFoldsHolding).
func (m model) foldText(t task.Task, key string) string {
	return wrap(m.foldBody(t, key, m.sections(t), func(string, int) string { return "" }), m.width)
}

// detailsOpenLines is the most lines a Details text may take, wrapped at
// the page's width, to show open (hya1st, Jørgen 2026-11-24: "DETAILS can
// show if not big?"; 6 is provisional); a longer one folds to one line.
const detailsOpenLines = 6

// detailsShort: t's Details has text and fits in detailsOpenLines at the
// page's width, so the page shows it open, under its plain heading, and
// it is no fold.
func (m model) detailsShort(t task.Task) bool {
	f := m.facts(t)
	if f.details == "" {
		return false
	}
	text := strings.TrimRight(m.foldText(t, "details"), "\n ")
	return len(strings.Split(text, "\n")) <= detailsOpenLines
}

// fold is one of a page's long sections (8a0ybb): Details when it is over
// detailsOpenLines (hya1st), the Questions record, Notes and History,
// whenever they have text; no other section folds. key names it in
// m.unfolded, heading is what its line says, count how many entries it
// holds, stop where its ids stand.
type fold struct {
	key, heading, stop string
	count              int
}

// folds are t's long sections with text, in page order.
func (m model) folds(t task.Task) []fold {
	f := m.facts(t) // the texts and their counts, once per file (pageMemo)
	var out []fold
	if f.details != "" && !m.detailsShort(t) {
		out = append(out, fold{key: "details", heading: "Details", stop: stopDetails, count: f.detailsParagraphs})
	}
	if n := m.recordCount(t); n > 0 {
		out = append(out, fold{key: "questions", heading: "Questions", stop: stopRecord, count: n})
	}
	if f.notes != "" {
		out = append(out, fold{key: "notes", heading: "Notes", stop: stopNotes, count: f.notesEntries})
	}
	if f.history != "" {
		out = append(out, fold{key: "history", heading: "History", stop: stopHistory, count: f.historyEntries})
	}
	return out
}

// paragraphs counts s's paragraphs: runs of lines between blank ones.
func paragraphs(s string) int {
	n, in := 0, false
	for _, l := range strings.Split(s, "\n") {
		if blank := strings.TrimSpace(l) == ""; !blank && !in {
			n++
			in = true
		} else if blank {
			in = false
		}
	}
	return n
}

// entries counts a Notes or History text's entries: each line that does
// not hang under the one before (an indented or blank line continues it).
func entries(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" && !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") {
			n++
		}
	}
	return n
}

// entryParagraphs puts a blank line before each entry of a Notes text
// (a line entries counts), outside fenced code, so each reads as its own
// paragraph.
func entryParagraphs(s string) string {
	var out []string
	fence := false
	for i, l := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = !fence
		}
		if i > 0 && !fence && trimmed != "" && !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") && out[len(out)-1] != "" {
			out = append(out, "")
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

// foldLine is a long section's heading with its count, NOTES (11): ›
// before it folded, ⌄ open; focused, the ▸ every focused heading has and
// what Enter does, dim.
func (m model) foldLine(f fold, k int, open bool) string {
	head := fmt.Sprintf("%s (%d)", strings.ToUpper(f.heading), f.count)
	if m.detailPane == focusFold && m.fold == k {
		hint := "Enter opens"
		if open {
			hint = "Enter folds"
		}
		return m.sectionHeading(head, hint, true)
	}
	sign := "› "
	if open {
		sign = "⌄ "
	}
	return m.styles.accent.Render(sign + head)
}

// toggleSection opens the focused long section, or folds it again; opened,
// the page shows it from its heading (8a0ybb).
func (m *model) toggleSection() {
	t, ok := m.selected()
	if !ok {
		return
	}
	fs := m.folds(t)
	if m.fold < 0 || m.fold >= len(fs) {
		return
	}
	if m.unfolded == nil {
		m.unfolded = map[string]bool{}
	}
	key := fs[m.fold].key
	m.unfolded[key] = !m.unfolded[key]
	m.syncDetail()
	if s, _, ok := m.markedBlock(); ok {
		if m.unfolded[key] {
			m.viewport.SetYOffset(s)
		} else {
			m.showMarked()
		}
	}
}

// openTop shows a just-opened task from the top, scrolled only as far as
// needed to keep the focus in view: the question it asks you, wherever it
// sits (qcq3tm), or the focused check. The header can grow (a finding
// and its fix take two lines, rx74bp) and must not hide the focus.
func (m *model) openTop() {
	m.viewport.GotoTop()
	// The whole focused check, its question and all (ykta5g check 4),
	// scrolled no further than that: the header can grow (a finding and
	// its fix take two lines, rx74bp) and must not hide the focus.
	m.showMarked()
}
func (m *model) revealAcceptance() {
	s, e, ok := m.markedBlock()
	if !ok || s >= m.viewport.YOffset && e < m.viewport.YOffset+m.viewport.Height {
		return
	}
	// Centred when it fits; a block taller than the view from its
	// first line (ykta5g check 4).
	m.viewport.SetYOffset(max(0, min(s, s-(m.viewport.Height-(e-s+1))/2)))
}

// showMarked scrolls the page only as far as the marked block needs to
// be in view whole (ykta5g check 4, Jørgen 19:37: the check with its
// question, not only its first line); a block taller than the view shows
// from its first line.
func (m *model) showMarked() {
	s, e, ok := m.markedBlock()
	if !ok {
		return
	}
	y := m.viewport.YOffset
	if e >= y+m.viewport.Height {
		y = e - m.viewport.Height + 1
	}
	if s < y {
		y = s
	}
	m.viewport.SetYOffset(max(0, y))
}

// markedBlock is the first and last page line of what the focus marks:
// a list item with the lines that hang under it (a check's wrapped text,
// the question asked on it and who asked), or the line of an id marked
// ▸ in the text.
func (m model) markedBlock() (int, int, bool) {
	if m.detailPane == focusPage {
		return 0, 0, false // nothing is marked; the page keeps its scroll
	}
	t, ok := m.selected()
	if !ok {
		return 0, 0, false
	}
	if m.detailPane == focusFold {
		// The focused long section's heading line (8a0ybb).
		fs := m.folds(t)
		if m.fold < 0 || m.fold >= len(fs) {
			return 0, 0, false
		}
		head := fmt.Sprintf("▸ %s (%d)", strings.ToUpper(fs[m.fold].heading), fs[m.fold].count)
		for i, l := range m.pageLines(t) {
			if strings.HasPrefix(l, head) {
				return i, i, true
			}
		}
		return 0, 0, false
	}
	// An id is marked ▸ where it stands, in the text (0ga6m0); in the
	// header it is always on screen.
	marker := func(line string) bool { return strings.HasPrefix(line, "> ") }
	id := false
	if s, ok := m.markedStop(t); ok {
		marker = func(line string) bool { i, _ := markedRun(line, s.shown); return i >= 0 }
		id = true
	}
	lines := m.pageLines(t)
	// The > is looked for under the focused section's ▸ heading: text
	// above it with its own "> " rows is not the focus (zhvkmj check 5:
	// Files had the focus, the page scrolled to a capture's rows, and
	// Files was never in sight).
	from := 0
	if heading := m.focusedHeading(); heading != "" {
		for i, line := range lines {
			if strings.HasPrefix(line, "▸ "+heading) {
				from = i
				break
			}
		}
	}
	for i := from; i < len(lines); i++ {
		if !marker(lines[i]) {
			continue
		}
		end := i
		if id && m.inline != nil {
			// A file shown in line under its link (zbdbtm) is part of
			// the marked block: its box from ┌ to └, so the page shows it
			// whole. A link the page wrapped has its box a line lower.
			for j := i + 1; j < len(lines) && j <= i+2+inlineMax*4; j++ {
				t := strings.TrimSpace(lines[j])
				if strings.HasPrefix(t, "┌ ") || strings.HasPrefix(t, "│") {
					continue
				}
				if strings.HasPrefix(t, "└ ") {
					end = j
					break
				}
				if j > i+1 {
					break
				}
			}
		} // What hangs under an item is indented past the next item's
		// pointer column (2): wrapped text, the question (its blank
		// lines between paragraphs are indented too), └ who asked.
		for j := i + 1; !id && j < len(lines) && strings.HasPrefix(lines[j], "    "); j++ {
			if strings.TrimSpace(lines[j]) != "" {
				end = j
			}
		}
		return i, end, true
	}
	return 0, 0, false
}

// focusedHeading is the capitalised heading of the section that has the
// focus, as sectionHeading draws it after "▸ ", or "" when the focus has
// no section of its own (the page, an id, the question at the top).
func (m model) focusedHeading() string {
	switch m.detailPane {
	case focusAcceptance:
		return "ACCEPTANCE"
	case focusDeps:
		return "DEPENDENCIES"
	case focusLinks:
		return "LINKS"
	case focusFiles:
		return "FILES"
	}
	return ""
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
// reads unfocused, and its leading > the focus sign, no underline (an
// underline reads as a key, q3sh69). The sign is drawn on; it pulses
// (pulsePage) until a opens the box, whose cursor then blinks alone.
func (m model) cursorMark(label string, style lipgloss.Style) string {
	lines := strings.Split(label, "\n")
	for i, l := range lines {
		if i == 0 && strings.HasPrefix(l, ">") {
			lines[i] = m.signIn(">", false) + style.Render(l[1:])
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

// recordCount is how many questions the Questions record holds: every
// one but those open and asked of the viewer (questionRecord).
func (m model) recordCount(t task.Task) int { return len(m.recordQuestions(t)) }

// recordQuestions are the questions the Questions record shows, in order.
func (m model) recordQuestions(t task.Task) []task.AskedQuestion {
	mine := map[int]bool{}
	for _, q := range m.askedAll(t) {
		mine[q.n] = q.n > 0
	}
	var out []task.AskedQuestion
	for _, q := range t.Questions {
		if !mine[q.N] {
			out = append(out, q)
		}
	}
	return out
}

// questionRecord is the Questions section's lines: every question but
// those open and asked of the viewer, each with its answer, or withdrawn.
func (m model) questionRecord(t task.Task, markAt func(string, int) string) []string {
	var out []string
	for k, q := range m.recordQuestions(t) {
		head := fmt.Sprintf("%d. %s → %s", q.N, oneLine(q.By), oneLine(m.name(q.To)))
		if q.Check > 0 {
			head += fmt.Sprintf(" · check %d", q.Check)
		}
		if s := strings.TrimPrefix(since(q.At), " · "); s != "" {
			head += " · " + s + " ago"
		}
		switch {
		case q.Answer != nil:
			head += " · answered"
		case q.Withdrawn != nil:
			// nanvnt: withdrawn, when; its text dim too, so it never
			// reads as a question still waiting.
			head += " · withdrawn " + strings.Replace(q.Withdrawn.At, "T", " ", 1)
		default:
			head += " · open"
		}
		out = append(out, m.styles.dim.Render(head))
		// Each block hangs under its indent as a request's does (cd63d1).
		style := m.styles.normal
		if q.Answer == nil && q.Withdrawn != nil {
			style = m.styles.dim
		}
		// A file it names is a stop (zbdbtm), marked where it stands.
		w := markAt(stopRecord, 2*k)
		out = append(out, m.hangStyled("   ", markPlain(safe(strings.TrimRight(q.Text, "\n")), w, false), 3, w, style)...)
		if a := q.Answer; a != nil {
			w := markAt(stopRecord, 2*k+1)
			out = append(out, m.hangStyled("   └ "+oneLine(a.By)+": ", markPlain(safe(strings.TrimRight(a.Text, "\n")), w, false), 5, w, m.styles.status["done"])...)
		}
	}
	return out
}

// answeredWhat is what an answer answered: "question N" when the task
// asked several (q2adr8), else its stage.
func answeredWhat(a task.Answer) string {
	if a.QuestionN > 0 {
		return "question " + strconv.Itoa(a.QuestionN)
	}
	return a.Stage
}
