package tui

import (
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"

	"github.com/veridit/hug/huginn/internal/task"
)

// In a task's detail, Tab and l (Shift-Tab and h) step one focus ring in
// reading order, every task id the detail shows a stop of its own (4bwcgp,
// e3sevr: focusRing), and Enter opens the one marked; Esc or u comes back
// (Jørgen 19:43: "there is no way to tab to the dependency" in jr55s8's
// "waiting on after 46ek1f"). One rule (0ga6m0, Jørgen 11:45Z: "6ephqk"
// in a check and its question could not be reached): every id or alias
// the detail shows is a stop, marked where it stands, wherever that is:
// the parent badge, the stage line, an unread answer, the title, the
// question, the description sections, each check and the question under
// it, a dependency's note, a condition, any other section, a request and
// its account, the Notes and the History; dependencies, children and
// links are stops as rows. An id named in several places is a stop in
// each; named twice in one place, once there, the first. A task:ID is a
// link: its stop is its Links row. An id or alias that resolves to no
// task is not a stop.

// Where an id stop stands.
const (
	stopParent   = "parent"
	stopStage    = "stage"
	stopFinding  = "finding"  // the header's finding: index 0 the problem, 1 its fix
	stopAnswer   = "answer"   // the unread answer in the header
	stopTitle    = "title"    // the task's own title
	stopQuestion = "question" // a question to you that names no check
	stopText     = "text"     // Goal, Purpose, Principle: index 0..2
	stopDetails  = "details"  // Details, a long section: only while open (8a0ybb)
	stopCheck    = "check"    // acceptance check index
	stopAsked    = "asked"    // the question under check index
	stopDeps     = "deps"
	stopDepNote  = "depnote" // dependency index's note
	stopWait     = "wait"    // condition index
	stopChildren = "children"
	stopLinks    = "links"
	stopSection  = "section"  // any other section: index in m.sections
	stopRequest  = "request"  // request index's text
	stopResolved = "resolved" // request index's account
	stopRecord   = "record"   // the Questions record, open: 2i a question's text, 2i+1 its answer (zbdbtm)
	stopNotes    = "notes"
	stopHistory  = "history"
)

// idStop is one task id the detail shows: where it stands, its place in
// that list (deps, children, links), the text shown (an id or an alias)
// and the task it opens.
type idStop struct {
	where, shown string
	index        int
	target       task.Task
	// at is where the stop's word starts in the text scanned, to keep a
	// text's ids and file links in reading order; file, for a file link
	// (zbdbtm), the file it names, and then target is empty.
	at   int
	file *fileLink
}

// idWord is a word that may name a task: an id or an alias such as
// STATBUS-402. Only words that resolve exactly count (no prefix, no slug).
var idWord = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9_-]*[A-Za-z0-9]`)

// scanIDs is idsIn worked out (pageMemo keeps it per text).
func (m model) scanIDs(t task.Task, text string, linked bool) []idStop {
	var out []idStop
	seen := map[string]bool{}
	names := refWordsOf(m.all)
	for _, loc := range idWord.FindAllStringIndex(text, -1) {
		w := text[loc[0]:loc[1]]
		if len(w) < 3 || len(w) > 40 || !names[strings.ToLower(w)] || linked && taskPrefixed(text, loc[0]) {
			continue
		}
		target, err := task.ResolveDependency(m.all, w)
		if err != nil || strings.EqualFold(target.ID, t.ID) || seen[strings.ToLower(target.ID)] {
			continue
		}
		seen[strings.ToLower(target.ID)] = true
		out = append(out, idStop{shown: w, target: target, at: loc[0]})
	}
	return out
}

// taskPrefixed: the word at i in s is written task:ID.
func taskPrefixed(s string, i int) bool {
	return i >= 5 && strings.EqualFold(s[i-5:i], "task:")
}

// refWords is every id and alias on the board, lower case. A word not
// in it names no task, decided without ResolveDependency, which scans
// the board for a word it does not know; idStops reads every word the
// detail shows. Kept for the last board, keyed as task.refIndexFor keys
// its index: a reload makes a new slice.
var refWords struct {
	sync.Mutex
	first *task.Task
	n     int
	set   map[string]bool
}

func refWordsOf(all []task.Task) map[string]bool {
	if len(all) == 0 {
		return nil
	}
	refWords.Lock()
	defer refWords.Unlock()
	if refWords.first != &all[0] || refWords.n != len(all) {
		set := make(map[string]bool, len(all))
		for _, t := range all {
			set[strings.ToLower(t.ID)] = true
			for _, a := range t.Aliases {
				set[strings.ToLower(a)] = true
			}
		}
		refWords.first, refWords.n, refWords.set = &all[0], len(all), set
	}
	return refWords.set
}

// children are the tasks whose parent is t, in board order.
func (m model) children(t task.Task) []task.Task {
	var out []task.Task
	for _, c := range m.all {
		if c.Parent != "" && strings.EqualFold(task.BareRef(c.Parent), t.ID) {
			out = append(out, c)
		}
	}
	return out
}

// stageWords is the stage line's text where it may name a task: what a
// wait, a question or a deferral says. A depends line lists dependencies,
// which have their own stops.
func (m model) stageWords(t task.Task) string {
	switch m.stageOf(t) {
	case "waiting", "asked", "design", "review", "deferred":
		_, text, _ := m.note(t)
		return text
	}
	return ""
}

// idStops are the task ids t's detail shows, in reading order: each
// place the detail draws text, scanned as it is drawn (0ga6m0).
func (m model) idStops(t task.Task) []idStop {
	var out []idStop
	add := func(where string, index int, text string, linked bool) {
		// The ids and the file links (zbdbtm) of one text, in the order
		// it reads: a slice of its own, since the memo's are sorted here.
		found := append(append([]idStop(nil), m.idsIn(t, text, linked)...), m.fileLinksIn(t, text)...)
		sort.SliceStable(found, func(i, j int) bool { return found[i].at < found[j].at })
		for _, s := range found {
			s.where, s.index = where, index
			out = append(out, s)
		}
	}
	// The header: the epic badge, the stage's line (asked of you it is
	// not drawn, 0gztsy), the finding, an unread answer, the title.
	if p := strings.TrimSpace(t.Parent); p != "" {
		if target, err := task.ResolveDependency(m.all, p); err == nil {
			out = append(out, idStop{where: stopParent, shown: target.ID, target: target})
		}
	}
	_, mine := m.askedOf(t)
	if !mine {
		add(stopStage, 0, m.stageWords(t), false)
	}
	if warning, fix, ok := m.headFinding(t); ok {
		add(stopFinding, 0, warning, false)
		add(stopFinding, 1, fix, false)
	}
	if a := t.Answer; a != nil {
		add(stopAnswer, 0, safe(a.Text), false)
	}
	add(stopTitle, 0, oneLine(t.Title), false)
	// The details, top down, as detailText draws them.
	for k, b := range m.blockQuestions(t) {
		add(stopQuestion, k, questionText(b), false)
	}
	sections := m.sections(t)
	text := func(name string) string { return sectionText(sections, name) }
	// Purpose, Principle, then the Goal, as the page reads (8a0ybb);
	// Details is a long section, scanned below only while it is open.
	add(stopText, 1, text("purpose"), true)
	add(stopText, 2, text("principle"), true)
	add(stopText, 0, text("description"), true)
	for i, a := range t.Acceptance {
		add(stopCheck, i, checkText(a), true)
		if cq, ok := m.checkQuestion(t, i+1); ok {
			add(stopAsked, i, strings.TrimSpace(cq.what), false)
		}
	}
	for i, d := range task.DependencyDetails(t, m.all) {
		if !d.Missing {
			if target, err := task.Resolve(m.all, d.ID); err == nil {
				out = append(out, idStop{where: stopDeps, index: i, shown: target.ID, target: target})
			}
		}
		add(stopDepNote, i, oneLine(d.Note), false)
	}
	// A condition as written; what hug saw when it checked is not
	// scanned: checking runs git, and this runs on every frame.
	for i, w := range t.Waits {
		add(stopWait, i, oneLine(w.Text), false)
	}
	for i, c := range m.children(t) {
		out = append(out, idStop{where: stopChildren, index: i, shown: c.ID, target: c})
	}
	for i, l := range m.links(t) {
		if l.Missing {
			continue
		}
		if target, ok := m.taskAt(l.Path); ok {
			out = append(out, idStop{where: stopLinks, index: i, shown: l.ID, target: target})
		}
	}
	for i, s := range sections {
		if otherSection(s.name) {
			add(stopSection, i, s.text, true)
		}
	}
	for i, r := range t.Requests {
		add(stopRequest, i, requestStopText(m.shownRequest(r)), true) // a link's home and the targets too (ec6z6j)
		if r.Resolved != nil {
			add(stopResolved, i, r.Resolved.Text, true)
		}
	}
	// The long sections: their ids are stops only while they are open
	// (8a0ybb); folded, the page shows none of their text.
	open := func(key string) bool { return m.unfolded[key] }
	if open("details") {
		add(stopDetails, 0, text("details"), true)
	}
	if open("questions") {
		for k, q := range m.recordQuestions(t) {
			add(stopRecord, 2*k, safe(strings.TrimRight(q.Text, "\n")), false)
			if q.Answer != nil {
				add(stopRecord, 2*k+1, safe(strings.TrimRight(q.Answer.Text, "\n")), false)
			}
		}
	}
	if open("notes") {
		add(stopNotes, 0, text("notes"), true)
	}
	if open("history") {
		// History is not scanned for links (links), so task:ID there is a stop.
		add(stopHistory, 0, text("history"), false)
	}
	return out
}

// idAtFocus is the stop the detail marks now, or -1: a dependency or a
// link marked > is the stop it shows; with focusIDs, m.idAt.
func (m model) idAtFocus(stops []idStop) int {
	for k, s := range stops {
		switch {
		case m.detailPane == focusIDs && k == m.idAt:
			return k
		case m.detailPane == focusDeps && s.where == stopDeps && s.index == m.dependency:
			return k
		case m.detailPane == focusLinks && s.where == stopLinks && s.index == m.link:
			return k
		}
	}
	return -1
}

// markedStop is the stop marked in the header or the text, if one is.
func (m model) markedStop(t task.Task) (idStop, bool) {
	if m.detailPane != focusIDs {
		return idStop{}, false
	}
	stops := m.idStops(t)
	if m.idAt < 0 || m.idAt >= len(stops) {
		return idStop{}, false
	}
	return stops[m.idAt], true
}

// followID opens the task id marked in the header or the text; Esc comes
// back, as after following a dependency.
func (m *model) followID() {
	t, ok := m.selected()
	if !ok {
		return
	}
	s, ok := m.markedStop(t)
	if !ok {
		m.message = "no task id marked: Tab or l steps to the next one"
		return
	}
	if s.file != nil {
		// A file link opens its file in the file frame (zbdbtm).
		m.openFileLink(t, s)
		return
	}
	m.detailStack = append(m.detailStack, t.Path)
	m.detailPath = s.target.Path
	m.startFocus()
	m.syncDetail()
	m.openTop()
}

// marked is the word to mark at a place (where, index): the marked
// stop's, if it stands there, else "".
func (m model) marked(t task.Task) func(where string, index int) string {
	s, ok := m.markedStop(t)
	return func(where string, index int) string {
		if ok && s.where == where && s.index == index {
			return s.shown
		}
		return ""
	}
}

// visible is s without its escape sequences, and for each byte of it
// the byte's offset in s.
func visible(s string) (string, []int) {
	var b strings.Builder
	var at []int
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) {
			j := i + 2
			switch s[i+1] {
			case '[': // CSI: parameters, then one final byte
				for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
					j++
				}
				j++
			case ']': // OSC: up to BEL or ST
				for j < len(s) && s[j] != 0x07 && !(s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\') {
					j++
				}
				if j < len(s) && s[j] == 0x1b {
					j++
				}
				j++
			}
			i = min(j, len(s))
			continue
		}
		b.WriteByte(s[i])
		at = append(at, i)
		i++
	}
	return b.String(), at
}

// wordAt is the offset in s of the first word w as idsIn reads words
// (idWord: in "x-abc123" abc123 is no word of its own), past escape
// sequences, skipping task:ID when linked; -1 when there is none.
func wordAt(s, w string, linked bool) int {
	at, _, _ := markSpan(s, w, linked)
	return at
}

// markSpan is where in s the stop shown as w stands: its offset, its
// length in s (escape sequences inside it included) and the text it
// shows there. A task id is a word as idsIn reads it; a file link
// (zbdbtm) is a word as fileWords reads it, and when the page wrapped
// it, its part on the first line. at is -1 when it is not in s.
func markSpan(s, w string, linked bool) (at, n int, shown string) {
	if w == "" {
		return -1, 0, ""
	}
	return spanIn(scanVisible(s), w, linked)
}

// visibleText is a text's visible part (visible) and the words in it
// that may name a task or a file, which markSpan looks for a stop in.
type visibleText struct {
	v     string
	pos   []int
	ids   [][]int
	files []fileWord
}

func scanVisible(s string) *visibleText {
	v, pos := visible(s)
	return &visibleText{v: v, pos: pos, ids: idWord.FindAllStringIndex(v, -1), files: fileWords(v)}
}

// spanIn is markSpan in a scanned text.
func spanIn(x *visibleText, w string, linked bool) (at, n int, shown string) {
	v, pos := x.v, x.pos
	if idWord.FindString(w) != w {
		var part *fileWord
		for _, fw := range x.files {
			if fw.shown == w {
				part = &fw
				break
			}
			// Wrapped, a section link shows its first line's part here:
			// the name, # and the section's first words.
			if part == nil && fw.section != "" && strings.HasPrefix(w, fw.shown) {
				fw := fw
				part = &fw
			}
		}
		if part == nil {
			return -1, 0, ""
		}
		start, end := pos[part.at], pos[part.at+len(part.shown)-1]+1
		return start, end - start, part.shown
	}
	for _, loc := range x.ids {
		if v[loc[0]:loc[1]] != w || linked && taskPrefixed(v, loc[0]) {
			continue
		}
		if i := pos[loc[0]]; pos[loc[1]-1]-i == len(w)-1 {
			return i, len(w), w
		}
	}
	return -1, 0, ""
}

// markedRun is where in line l the mark ▸w stands, from ▸ to the end of
// w, or of the part of w the line holds (a wrapped file link); i is -1
// when l holds no ▸w.
func markedRun(l, w string) (i, j int) {
	if i = strings.Index(l, "▸"+w); i >= 0 {
		return i, i + len("▸"+w)
	}
	first := strings.SplitN(w, " ", 2)[0]
	if first == w {
		return -1, 0
	}
	if i = strings.Index(l, "▸"+first); i < 0 {
		return -1, 0
	}
	j = i + len("▸")
	for k := 0; k < len(w) && j < len(l) && l[j] == w[k]; k++ {
		j++
	}
	return i, j
}

// markWord draws the first word w in s marked: ▸ before it and the
// selection tint, so it shows without colour too. rest styles the text
// around it.
func (m model) markWord(s, w string, rest func(string) string) string {
	at, n, shown := markSpan(s, w, false)
	if at < 0 {
		return rest(s)
	}
	return rest(s[:at]) + m.styles.selected.Render("▸"+shown) + rest(s[at+n:])
}

// markPlain puts ▸ before the first word w in plain text s, before it is
// wrapped or fit, so the mark's column is counted; styleMarked then
// tints it.
func markPlain(s, w string, linked bool) string {
	at := wordAt(s, w, linked)
	if at < 0 {
		return s
	}
	return s[:at] + "▸" + s[at:]
}

// styleMarked styles each line of s, the ▸w markPlain put there in the
// selection tint. Lines are styled one by one only when one is marked,
// so an unmarked block renders as before.
func (m model) styleMarked(s, w string, style lipgloss.Style) string {
	if w == "" || !strings.Contains(s, "▸"+strings.SplitN(w, " ", 2)[0]) {
		return style.Render(s)
	}
	lines := strings.Split(s, "\n")
	for k, l := range lines {
		if i, j := markedRun(l, w); i >= 0 {
			lines[k] = renderSome(style, l[:i]) + m.styles.selected.Render(l[i:j]) + renderSome(style, l[j:])
		} else {
			lines[k] = style.Render(l)
		}
	}
	return strings.Join(lines, "\n")
}

func renderSome(style lipgloss.Style, s string) string {
	if s == "" {
		return ""
	}
	return style.Render(s)
}

// markStyled marks the first word w in s, text already styled (rendered
// Markdown, a request): ▸ and the selection tint, then the style in
// force before it again.
func (m model) markStyled(s, w string, linked bool) string {
	at, n, shown := m.markSpanOf(s, w, linked)
	if at < 0 {
		return s
	}
	return s[:at] + m.styles.selected.Render("▸"+shown) + lastSGR(s[:at]) + s[at+n:]
}

// lastSGR is the last colour sequence in s, "" after a reset.
func lastSGR(s string) string {
	i := strings.LastIndex(s, "\x1b[")
	if i < 0 {
		return ""
	}
	j := strings.IndexByte(s[i:], 'm')
	if j < 0 {
		return ""
	}
	seq := s[i : i+j+1]
	if seq == "\x1b[0m" || seq == "\x1b[m" {
		return ""
	}
	return seq
}
