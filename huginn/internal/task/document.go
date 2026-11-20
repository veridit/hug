// Package task owns lossless Markdown documents and task operations.
package task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/veridit/hug/huginn/internal/id"
	"github.com/veridit/hug/internal/sharefile"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

var StateList = []string{"todo", "doing", "done", "dropped"}
var Types = []string{"feature", "bug", "chore", "spike", "docs"}

type Finding struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Problem string `json:"problem"`
	// Code is a short stable name for the kind of problem (for filtering
	// and counting); it is derived from Problem, see FindingCode.
	Code string `json:"code"`
	// Fix is one command that repairs the problem (or opens the task to
	// repair it). Empty means the default for the code, see FixLine.
	Fix string `json:"fix"` // Fixes are alternative ways out, each one command (txjkrm: add the
	// tag to the list, or remove it from the task); Fix is the first.
	Fixes []string `json:"fixes,omitempty"`
}

// Severity is Broken or Advice, from the finding's code (pbh9xq).
func (f Finding) Severity() string { return Severity(FindingCode(f.Problem)) }

// findingCodes maps message prefixes to stable codes. Order matters: the
// first prefix that matches wins.
var findingCodes = []struct{ prefix, code string }{
	{"missing or unterminated frontmatter", "frontmatter"},
	{"malformed frontmatter", "frontmatter"},
	{"frontmatter ", "frontmatter"},
	{"duplicate section", "duplicate-section"},
	{"duplicate field", "duplicate-field"},
	{"missing field", "missing-field"},
	{"old layout", "old-layout"},
	{"invalid id", "invalid-id"},
	{"filename id differs", "id-mismatch"},
	{"unknown state", "unknown-state"},
	{"unknown stage", "unknown-stage"},
	{"node field ", "stray-node-field"},
	{"missing node field", "missing-node-field"},
	{"stage ", "stage-disagrees"},
	{"unknown type", "unknown-type"},
	{"invalid ", "invalid-field"},
	{"missing Goal", "missing-goal"},
	{"missing section", "missing-section"},
	{"sections out of order", "section-order"},
	{"Acceptance requires", "no-acceptance"},
	{"landed, but no check", "landed-unchecked"},
	{"shipped, but ", "shipped-not-done"},
	{"every check holds, but ", "checks-all-hold"},
	{"todo, but commit ", "todo-has-commit"},
	{"doing, but nothing on it", "doing-idle"},
	{"case-insensitive filename collision", "filename-collision"},
	{"duplicate id", "duplicate-id"},
	{"depends on ", "dangling-dependency"},
	{"not a dependency", "not-a-dependency"},
	{"dependency cycle", "dependency-cycle"},
	{"missing parent", "missing-parent"},
	{"task is its own parent", "self-parent"},
	{"parent ", "parent-too-deep"},
	{"title is ", "long-title"},
	{"Goal is ", "goal-too-long"},
	{"cites ", "forbidden-citation"},
	{"task file is a symlink", "file-symlink"},
	{"duplicate origin", "duplicate-origin"},
	{"branch ", "branch-gone"},
	{"unknown condition", "unknown-condition"},
	{"owner ", "stale-owner"},
	{"nobody known: ", "unknown-person"},
	{"a question waits as a wait: ", "wait-on-person"},
	{"unlisted tag: ", "unlisted-tag"},
	{"backlog leftover", "backlog-leftover"},
}

// FindingCode names the kind of problem in a finding message.
func FindingCode(problem string) string {
	for _, c := range findingCodes {
		if strings.HasPrefix(problem, c.prefix) {
			return c.code
		}
	}
	switch {
	case strings.Contains(problem, ": no task "):
		return "missing-task-ref"
	case strings.Contains(problem, ": file not found"), strings.Contains(problem, ": no attached file matches"):
		return "missing-file"
	case strings.Contains(problem, "binary"), strings.Contains(problem, "KiB"), strings.Contains(problem, "too large"):
		return "file-content"
	}
	return "other"
}

// MarshalJSON fills Code from Problem and Fix from the code, so every
// finding carries both.
func (f Finding) MarshalJSON() ([]byte, error) {
	type plain Finding
	if f.Code == "" {
		f.Code = FindingCode(f.Problem)
	}
	f.Fix = f.FixLine()
	return json.Marshal(struct {
		plain
		Severity string `json:"severity"`
	}{plain(f), Severity(f.Code)})
}

// FixLine is the command that repairs the finding: its own Fix, else the
// usual repair for its code, naming the task the file belongs to. Every
// finding has one, so a reader always has something to run.
func (f Finding) FixLine() string {
	if f.Fix != "" {
		return f.Fix
	}
	i, ok := IDOf(f.Path)
	if !ok {
		return "hug task lint"
	}
	switch FindingCode(f.Problem) {
	case "unknown-state":
		return "hug task set " + i + " --state todo"
	case "old-layout":
		return "hug upgrade"
	case "unknown-type":
		return "hug task set " + i + " --type feature"
	case "missing-parent", "self-parent":
		return "hug task set " + i + " --parent none"
	case "filename-collision", "duplicate-id", "duplicate-origin":
		return "hug task show " + i
	case "missing-goal":
		return "hug task set " + i + " --goal \"...\""
	case "no-acceptance":
		return "hug task set " + i + " --accept \"observable check\""
	}
	return "hug task edit " + i
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: %s [%s] %s", f.Path, f.Line, f.Severity(), FindingCode(f.Problem), f.Problem)
}

type Dependency struct {
	Ref  string `json:"ref"`
	Line int    `json:"line"`
	Note string `json:"note,omitempty"`
}
type Acceptance struct {
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
	Line    int    `json:"line"`
	// Struck: the check was struck out as not relevant (sfa8w4), with the
	// reason, who and the day. The file keeps the box open and the text
	// crossed through: "- [ ] ~~TEXT~~ (not relevant: REASON, NAME, DATE)".
	Struck   bool   `json:"struck,omitempty"`
	Reason   string `json:"reason,omitempty"`
	StruckBy string `json:"struck_by,omitempty"`
	StruckOn string `json:"struck_on,omitempty"`
}

// Settled: ticked or struck out with a reason. A task is done only when
// every check is settled.
func (a Acceptance) Settled() bool { return a.Checked || a.Struck }

// State is open, ticked or struck.
func (a Acceptance) State() string {
	switch {
	case a.Struck:
		return "struck"
	case a.Checked:
		return "ticked"
	}
	return "open"
}

// struckForm is a struck-out check's text: standard Markdown strikethrough,
// then the recorded decision. The reason may hold commas: who and the day
// are the last two fields.
var struckForm = regexp.MustCompile(`^~~(.+)~~ \(not relevant: (.+)\)\s*$`)

func parseAcceptance(text string, checked bool, line int) Acceptance {
	a := Acceptance{Text: text, Checked: checked, Line: line}
	m := struckForm.FindStringSubmatch(text)
	if m == nil || checked {
		return a
	}
	a.Text, a.Struck, a.Reason = m[1], true, strings.TrimSpace(m[2])
	if i := strings.LastIndex(a.Reason, ", "); i >= 0 && isDay(a.Reason[i+2:]) {
		a.StruckOn, a.Reason = a.Reason[i+2:], a.Reason[:i]
		if j := strings.LastIndex(a.Reason, ", "); j >= 0 {
			a.StruckBy, a.Reason = a.Reason[j+2:], a.Reason[:j]
		}
	}
	return a
}

func isDay(s string) bool {
	_, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	return err == nil
}

type Task struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Group string `json:"group"`
	Rank  string `json:"rank"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	State string `json:"state"`
	// Stage is the task's stage within its state (6ephqk): todo ready,
	// doing waiting ... Empty on a board hug upgrade has not migrated yet.
	Stage string `json:"stage,omitempty"`
	// Fields are the node fields in the file (doing_since, waiting_on ...)
	// by name, as written; NodeFields says which node owns each.
	Fields   map[string]string `json:"fields,omitempty"`
	Type     string            `json:"type"`
	Priority string            `json:"priority"`
	Created  string            `json:"created"`
	Updated  string            `json:"updated"`
	Assignee []string          `json:"assignee"`
	// Model is the model that took the task (claude-opus-5-5, gpt-6.1-sol,
	// or human), recorded by hug task take --model (et85h1).
	Model string `json:"model,omitempty"`
	// Effort is the effort the model runs at (high, xhigh, ...), from
	// hug task take --model ID/EFFORT (jr55s8); "" when not given.
	Effort string `json:"effort,omitempty"`
	// Branch and Worktree are where the task was taken (x1t1t1): the git
	// branch checked out there and that worktree's folder. The branch is
	// what matters; the folder only says where to look now.
	Branch   string   `json:"branch,omitempty"`
	Worktree string   `json:"worktree,omitempty"`
	Labels   []string `json:"labels"`
	Aliases  []string `json:"aliases"`
	// Parent is the epic this task belongs to (an id), or empty. An epic is
	// a task with children; it has no parent itself (three fixed levels:
	// epic, task, acceptance item).
	Parent string `json:"parent,omitempty"`
	// Sections lists the "## " headings in file order.
	Sections     []string     `json:"sections,omitempty"`
	Goal         string       `json:"goal"`
	Dependencies []Dependency `json:"dependencies"`
	// Waits are the conditions in Depends on (16dxsr): "- when PREDICATE"
	// and "- group:NAME", checked when the board is read.
	Waits      []Wait       `json:"waits,omitempty"`
	Acceptance []Acceptance `json:"acceptance"`
	// Texts are what show and ls print beside the fields: the Notes body
	// and the description sections (DescriptionSections), filled by the
	// same Parse that fills the Task, so a cached Task answers ls without
	// reading 10,000 files again (44w1ev). Kept out of the JSON payload:
	// cli builds those keys itself. Read only: copies of a Task share it.
	Texts *Texts `json:"-"`
	// Answer is the newest answer in History its asker has not seen
	// (3jtfw9): read from the file, never written as a field.
	Answer *Answer `json:"answer,omitempty"`
	// Requests are the person's requests in ## Requests (hbgybn), open
	// and resolved, in file order.
	Requests []Request `json:"requests,omitempty"`
	// Questions are the questions asked on it in ## Questions (q2adr8),
	// open and answered, in file order.
	Questions []AskedQuestion `json:"questions,omitempty"`
	// Outcome is what the asker did first after the newest answer (41hhgn
	// check 11), from its History read line; nil before it acts.
	Outcome *Outcome `json:"outcome,omitempty"`
	// AnswerCount is how many answer blocks History holds (gmzd87: the
	// newest is "answer <count>"); Answers are the ones handed to the
	// coordinator, each with its receipt. Read from History.
	AnswerCount int    `json:"answer_count,omitempty"`
	Answers     []Item `json:"answers,omitempty"`
	// Woken is whether the newest History line is a timed wait ending
	// (z2vmte): nothing has moved the task since; next --mine puts it
	// first. Read from History, never written.
	Woken bool `json:"woken,omitempty"`
}

// Texts of one task file, see Task.Texts.
type Texts struct {
	// Bodies is true when Notes and Description are filled. The cache keeps
	// only Refs in memory (the bodies are 14 MB at 10,000 tasks and only ls
	// prints them) and fills the bodies on request (44w1ev).
	Bodies bool
	Notes  string
	// Description maps each of DescriptionSections to its trimmed body.
	Description map[string]string
	// Refs are the file:NAME and task:REF/NAME references in the file
	// (FileRefs), so lint need not read every task file to check them.
	Refs []Ref
}
type Document struct {
	Task             Task
	raw              []byte
	lines            []string
	fields           map[string]int
	fieldEnds        map[string]int
	sections         map[string][2]int
	Findings         []Finding
	frontEnd         int
	frontmatterValid bool
	newline          string
	// layout3: the file is C/ID.md (w40cmk), so rank: and group: are read
	// from the front matter.
	layout3 bool
}

func ParseName(path string) (r, i, s string, err error) {
	n := strings.TrimSuffix(filepath.Base(path), ".md")
	p := strings.Split(n, ".")
	switch len(p) {
	case 2:
		i, s = p[0], p[1]
	case 3:
		r, i, s = p[0], p[1], p[2]
	default:
		return "", "", "", fmt.Errorf("expected [rank.]id.slug.md")
	}
	if !strings.HasSuffix(path, ".md") || !id.Valid(i) || s == "" {
		err = fmt.Errorf("expected [rank.]id.slug.md with a six-character Crockford id")
	}
	return
}
func BuildName(r, i, s string) string {
	if r != "" {
		return r + "." + i + "." + s + ".md"
	}
	return i + "." + s + ".md"
}

// Slug keeps file names and list rows short: the first four words of the
// title that carry meaning (filler like "the", "of", "to" is skipped), at
// most 32 characters (2026-11-15, user: "three, four words").
func Slug(title string) string {
	words := strings.FieldsFunc(strings.ToLower(title), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var kept []string
	for _, w := range words {
		if len(kept) == 4 {
			break
		}
		if slugFiller[w] {
			continue
		}
		if n := len(strings.Join(append(kept, w), "-")); n > 32 && len(kept) > 0 {
			break
		}
		kept = append(kept, w)
	}
	if len(kept) == 0 && len(words) > 0 {
		kept = words[:1] // all filler ("A", "The"): keep the first word
	}
	if len(kept) == 0 {
		return "task"
	}
	slug := strings.Join(kept, "-")
	if len(slug) > 32 {
		slug = strings.TrimRight(slug[:32], "-")
	}
	return slug
}

var slugFiller = map[string]bool{"a": true, "an": true, "the": true, "of": true, "to": true, "in": true, "on": true, "for": true, "and": true, "or": true, "by": true, "with": true, "is": true, "are": true, "be": true, "at": true, "as": true, "from": true, "into": true, "it": true, "its": true}

func scalar(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "\"") {
		var v string
		if json.NewDecoder(strings.NewReader(s)).Decode(&v) == nil {
			return v
		}
	}
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
	}
	if n := strings.Index(s, " #"); n >= 0 {
		s = s[:n]
	}
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
	}
	return strings.TrimSpace(s)
}
func list(s string) []string {
	var out []string
	if json.Unmarshal([]byte(s), &out) == nil {
		return out
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		if n := strings.LastIndex(s, "]"); n >= 0 {
			s = s[1:n]
		}
	}
	for _, v := range strings.Split(s, ",") {
		if v = scalar(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
func Parse(path string, data []byte) *Document {
	d := &Document{raw: bytes.Clone(data), fields: map[string]int{}, fieldEnds: map[string]int{}, sections: map[string][2]int{}, frontEnd: -1, newline: "\n"}
	if bytes.Contains(data, []byte("\r\n")) {
		d.newline = "\r\n"
	}
	d.lines = strings.SplitAfter(string(data), "\n")
	d.Task.Path = path
	// Layout 3 (w40cmk): C/ID.md says only the id; rank and group are in
	// the front matter, and the slug comes from the title.
	i3, layout3 := IDPath(filepath.ToSlash(path))
	d.layout3 = layout3
	var i string
	if layout3 {
		d.Task.ID, i = i3, i3
	} else {
		d.Task.Group = filepath.ToSlash(filepath.Dir(path))
		if d.Task.Group == "." {
			d.Task.Group = ""
		}
		r, i2, s, e := ParseName(path)
		d.Task.Rank, d.Task.ID, d.Task.Slug, i = r, i2, s, i2
		if e != nil {
			d.add(1, e.Error())
		}
	}
	clean := func(s string) string {
		return strings.ToValidUTF8(strings.TrimSuffix(strings.TrimSuffix(s, "\n"), "\r"), "\ufffd")
	}
	if len(d.lines) > 0 && clean(d.lines[0]) == "---" {
		for n := 1; n < len(d.lines); n++ {
			line := clean(d.lines[n])
			if line == "---" {
				d.frontEnd = n
				break
			}
		}
	}
	if d.frontEnd < 0 {
		d.add(1, "missing or unterminated frontmatter")
	} else {
		d.parseFrontmatter()
	}
	if d.layout3 {
		// The slug is no longer in the name: it is the title's, so a
		// slug ref ("task-state-stage-hard") still finds the task.
		d.Task.Slug = Slug(d.Task.Title)
	}
	start := d.frontEnd + 1
	if start < 0 {
		start = 0
	}
	section := ""
	sectionStart := 0
	fence := ""
	seenSections := map[string]bool{}
	for n := start; n < len(d.lines); n++ {
		line := clean(d.lines[n])
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) <= 3 && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			if section == "Goal" {
				d.Task.Goal += d.lines[n]
			}
			if fence == "" {
				ch := trimmed[0]
				count := 0
				for count < len(trimmed) && trimmed[count] == ch {
					count++
				}
				fence = trimmed[:count]
			} else if strings.HasPrefix(trimmed, fence) && strings.TrimSpace(strings.TrimPrefix(trimmed, fence)) == "" {
				fence = ""
			}
			continue
		}
		if fence != "" {
			if section == "Goal" {
				d.Task.Goal += d.lines[n]
			}
			continue
		}
		if strings.HasPrefix(line, "## ") {
			if section != "" {
				d.sections[section] = [2]int{sectionStart, n}
			}
			section = strings.TrimSpace(line[3:])
			d.Task.Sections = append(d.Task.Sections, section)
			if seenSections[section] && contains([]string{"Goal", "Depends on", "Acceptance", "Notes"}, section) {
				d.add(n+1, "duplicate section "+section)
			}
			seenSections[section] = true
			sectionStart = n
			continue
		}
		switch section {
		case "Goal":
			d.Task.Goal += d.lines[n]
		case "Depends on":
			t := strings.TrimSpace(line)
			if t == "" || strings.HasPrefix(t, "<!--") {
				continue
			}
			if ref, note, ok := parseDependencyLine(t); ok {
				d.Task.Dependencies = append(d.Task.Dependencies, Dependency{ref, n + 1, note})
				continue
			}
			// A stated condition (16dxsr): "- when load below 120",
			// "- group:NAME". One hug cannot read still waits, and says so.
			if w, err, ok := parseWaitLine(t); ok {
				w.Line = n + 1
				d.Task.Waits = append(d.Task.Waits, w)
				if err != nil {
					d.Findings = append(d.Findings, unknownCondition(d.Task.ID, w.Text, n+1, d.Task.Path, err))
				}
				continue
			}
			// Prose is reported, never guessed into a dependency: an
			// unreadable line must not make the task blocked (rx74bp).
			who := d.Task.ID
			if who == "" {
				who = "ID"
			}
			d.Findings = append(d.Findings, Finding{Path: d.Task.Path, Line: n + 1,
				Problem: "not a dependency: " + quoteLine(t, 60) + " (for no dependencies, leave the section empty or remove it; a dependency line is \"- <task id>\", or run hug task dep " + who + " --add REF; prose belongs in Notes)",
				Fix:     "hug task edit " + who})
		case "Acceptance":
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "- [ ] ") || strings.HasPrefix(t, "- [x] ") || strings.HasPrefix(t, "- [X] ") {
				d.Task.Acceptance = append(d.Task.Acceptance, parseAcceptance(t[6:], t[3] != ' ', n+1))
			}
		}
	}
	if section != "" {
		d.sections[section] = [2]int{sectionStart, len(d.lines)}
	}
	// A file from before 6ephqk step 4 says status: where state: goes.
	// hug upgrade renames it; until then the task reads as its old state.
	_, hasState := d.fields["state"]
	if _, old := d.fields["status"]; old && !hasState {
		d.add(d.fieldLine("status"), "old layout: status: is now state:")
	}
	// Before 41hhgn a question to a person was a wait with waiting_for
	// (and waiting_show); hug upgrade moves it to stage asked.
	for _, k := range []string{"waiting_for", "waiting_show"} {
		if _, old := d.fields[k]; old {
			d.add(d.fieldLine(k), "old layout: "+k+": is now a question, stage asked")
		}
	}
	for _, field := range []struct{ k, v string }{{"id", d.Task.ID}, {"title", d.Task.Title}, {"state", d.Task.State}} {
		_, ok := d.fields[field.k]
		if field.k == "state" && !ok {
			if _, old := d.fields["status"]; old {
				continue // the old-layout finding says it, with hug upgrade
			}
		}
		if !ok || strings.TrimSpace(field.v) == "" {
			d.add(1, "missing field "+field.k)
		}
	}
	if !id.Valid(d.Task.ID) {
		d.add(d.fieldLine("id"), "invalid id")
	}
	if i != "" && i != d.Task.ID {
		d.add(1, "filename id differs from frontmatter id")
	}
	if !contains(StateList, d.Task.State) {
		d.add(d.fieldLine("state"), "unknown state "+d.Task.State)
	}
	d.stageFindings()
	if d.Task.Type != "" && !contains(Types, d.Task.Type) {
		d.add(d.fieldLine("type"), "unknown type "+d.Task.Type)
	}
	if _, ok := d.sections["Goal"]; !ok || strings.TrimSpace(d.Task.Goal) == "" {
		d.add(1, "missing Goal")
	}
	if len(d.Task.Acceptance) == 0 {
		d.add(1, "Acceptance requires at least one item")
	}
	// One section order everywhere (jnaspy); hug upgrade restores it.
	if !InOrder(d.Task.Sections) {
		d.Findings = append(d.Findings, Finding{Path: d.Task.Path, Line: 1,
			Problem: "sections out of order: " + strings.Join(d.Task.Sections, ", ") + "; the order is " + strings.Join(OrderedSections(d.Task.Sections), ", "),
			Fix:     "hug upgrade"})
	}
	// Backlog import leftovers on open tasks (fkpyg3); hug upgrade removes them.
	if Pattern(d.Task) {
		d.Findings = append(d.Findings, BacklogFindings(d)...)
	}
	d.Task.Goal = strings.ToValidUTF8(stripBacklogMarkers(d.Task.Goal), "\ufffd")
	d.Task.Answer, d.Task.Outcome = readAnswers(d.historyLines())
	d.Task.Woken = woken(d.historyLines())
	d.Findings = append(d.Findings, untilFindings(d.Task)...)
	d.Task.Requests = parseRequests(d.requestLines())
	d.Task.Questions = parseQuestions(d.questionLines())
	// A question only the owner can settle, from the ticket's own notes
	// (at3qep): landed with no check ticked, shipped but open, todo with a
	// commit on main. After the questions are read: one open to a person
	// says why the task is not done (yf39ya check 7).
	if q := noteQuestion(d.Task, notesText(data)); q != nil {
		d.Findings = append(d.Findings, *q)
	}
	readTargets(&d.Task, d.historyLines())
	d.Task.AnswerCount = answerCount(d.historyLines())
	readReceipts(&d.Task, d.historyLines())
	d.Task.Texts = &Texts{Bodies: true, Notes: notesText(data), Description: map[string]string{}, Refs: FileRefs(data, d.Task.ID)}
	for _, sec := range DescriptionSections {
		d.Task.Texts.Description[sec] = d.sectionText(sec)
	}
	return d
}

// notesText is the body of the first "## Notes" section (any case), trimmed:
// what hug task show prints as notes.
func notesText(b []byte) string {
	active := false
	var notes []string
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "## ") {
			if active {
				break
			}
			active = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(l, "## ")), "Notes")
			continue
		}
		if active {
			notes = append(notes, l)
		}
	}
	return strings.TrimSpace(strings.Join(notes, "\n"))
}

// stripBacklogMarkers drops Backlog.md's <!-- SECTION:...:BEGIN/END -->
// comment lines from displayed text. The file keeps them, so a re-import
// still merges against the same bytes (seid-lang feedback, 2026-11-09).
func stripBacklogMarkers(s string) string {
	if !strings.Contains(s, "<!-- SECTION:") && !strings.Contains(s, "<!-- AC:") {
		return s
	}
	var out []string
	for _, l := range strings.Split(s, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "<!-- SECTION:") || strings.HasPrefix(t, "<!-- AC:") {
			continue
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
func (d *Document) add(line int, p string) {
	d.Findings = append(d.Findings, Finding{Path: d.Task.Path, Line: line, Problem: p})
}
func (d *Document) fieldLine(k string) int {
	if n, ok := d.fields[k]; ok {
		return n + 1
	}
	return 1
}

// legacyField is the one-line value of front-matter key k that is no
// longer a node field (so Parse keeps it out of Task.Fields), for hug
// upgrade's moves; "" when absent or not one line of text.
func (d *Document) legacyField(k string) string {
	n, ok := d.fields[k]
	if !ok || n >= len(d.lines) {
		return ""
	}
	var m map[string]string
	if yaml.Unmarshal([]byte(d.lines[n]), &m) != nil {
		return ""
	}
	return strings.TrimSpace(m[k])
}
func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}
func (d *Document) Bytes() []byte { return bytes.Clone(d.raw) }
func (d *Document) replace(lines []string) {
	fresh := Parse(d.Task.Path, []byte(strings.Join(lines, "")))
	*d = *fresh
}
func (d *Document) SetField(k string, value any) error {
	if d.frontEnd < 0 || !d.frontmatterValid {
		return &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint"}
	}
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	lines := append([]string(nil), d.lines...)
	line := k + ": " + string(b) + d.newline
	if n, ok := d.fields[k]; ok {
		end := d.fieldEnds[k]
		if end <= n {
			end = n + 1
		}
		lines = append(lines[:n], append([]string{line}, lines[end:]...)...)
	} else {
		n := d.frontEnd
		lines = append(lines[:n], append([]string{line}, lines[n:]...)...)
	}
	d.replace(lines)
	return nil
}

// SetItemText rewrites acceptance item n's text, keeping its checkbox.
func (d *Document) SetItemText(n int, text string) error {
	if n < 1 || n > len(d.Task.Acceptance) {
		return &Error{Kind: "acceptance-index", ID: d.Task.ID, Value: fmt.Sprint(n), Fix: fmt.Sprintf("hug task show %s", d.Task.ID)}
	}
	a := d.Task.Acceptance[n-1]
	lines := append([]string(nil), d.lines...)
	old := lines[a.Line-1]
	i := strings.Index(old, "]")
	if i < 0 {
		return &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint"}
	}
	nl := ""
	if strings.HasSuffix(old, "\r\n") {
		nl = "\r\n"
	} else if strings.HasSuffix(old, "\n") {
		nl = "\n"
	}
	lines[a.Line-1] = old[:i+1] + " " + strings.ReplaceAll(text, "\n", " ") + nl
	d.replace(lines)
	return nil
}
func (d *Document) Check(n int) error {
	if n < 1 || n > len(d.Task.Acceptance) {
		return &Error{Kind: "acceptance-index", ID: d.Task.ID, Value: fmt.Sprint(n), Fix: fmt.Sprintf("hug task show %s", d.Task.ID)}
	}
	a := d.Task.Acceptance[n-1]
	if a.Checked {
		return nil
	}
	if a.Struck {
		return &Error{Kind: "struck-out", ID: d.Task.ID, Value: fmt.Sprintf("item %d is struck out as not relevant (%s); restore it before ticking", n, a.Reason), Fix: fmt.Sprintf("hug task strike %s %d --undo", d.Task.ID, n)}
	}
	lines := append([]string(nil), d.lines...)
	lines[a.Line-1] = strings.Replace(lines[a.Line-1], "[ ]", "[x]", 1)
	d.replace(lines)
	return nil
}

// Strike strikes item n out as not relevant, keeping its text and box and
// recording reason, who and day (sfa8w4). Striking a struck item again with
// the same reason is a no-op; a ticked item is refused (it was done).
func (d *Document) Strike(n int, reason, who, day string) error {
	if n < 1 || n > len(d.Task.Acceptance) {
		return &Error{Kind: "acceptance-index", ID: d.Task.ID, Value: fmt.Sprint(n), Fix: fmt.Sprintf("hug task show %s", d.Task.ID)}
	}
	clean := func(s string) string {
		return strings.TrimSpace(strings.ReplaceAll(strings.Join(strings.Fields(s), " "), "~~", ""))
	}
	reason, who = clean(reason), clean(who)
	if reason == "" {
		return &Error{Kind: "no-reason", ID: d.Task.ID, Value: fmt.Sprintf("striking item %d out needs a reason: why it no longer applies", n), Fix: fmt.Sprintf("hug task strike %s %d --reason \"why it no longer applies\"", d.Task.ID, n)}
	}
	if who == "" {
		return &Error{Kind: "no-name", ID: d.Task.ID, Value: "striking out records who decided: no --as, HUG_USER, agent session or model given", Fix: fmt.Sprintf("hug task strike %s %d --reason \"...\" --as NAME", d.Task.ID, n)}
	}
	a := d.Task.Acceptance[n-1]
	if a.Checked {
		return &Error{Kind: "ticked", ID: d.Task.ID, Value: fmt.Sprintf("item %d is ticked: it was done, so it is not struck out", n), Fix: fmt.Sprintf("hug task show %s", d.Task.ID)}
	}
	if a.Struck && a.Reason == reason {
		return nil
	}
	return d.setItemLine(a, "- [ ] ~~"+a.Text+"~~ (not relevant: "+reason+", "+who+", "+day+")")
}

// Unstrike restores a struck item to an open check with its text.
func (d *Document) Unstrike(n int) error {
	if n < 1 || n > len(d.Task.Acceptance) {
		return &Error{Kind: "acceptance-index", ID: d.Task.ID, Value: fmt.Sprint(n), Fix: fmt.Sprintf("hug task show %s", d.Task.ID)}
	}
	a := d.Task.Acceptance[n-1]
	if !a.Struck {
		return nil
	}
	return d.setItemLine(a, "- [ ] "+a.Text)
}

// setItemLine replaces an item's line, keeping its indent and line ending.
func (d *Document) setItemLine(a Acceptance, body string) error {
	lines := append([]string(nil), d.lines...)
	old := lines[a.Line-1]
	i := strings.Index(old, "- [")
	if i < 0 {
		return &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint"}
	}
	nl := ""
	if strings.HasSuffix(old, "\r\n") {
		nl = "\r\n"
	} else if strings.HasSuffix(old, "\n") {
		nl = "\n"
	}
	lines[a.Line-1] = old[:i] + body + nl
	d.replace(lines)
	return nil
}

func (d *Document) Note(text string) {
	lines := append([]string(nil), d.lines...)
	s, ok := d.sections["Notes"]
	if n, before := d.insertAt("Notes"); !ok && before {
		// A new section goes in its canonical place (jnaspy): Notes
		// right above History, never after.
		lines = append(lines[:n], append([]string{"## Notes" + d.newline + text + d.newline + d.newline}, lines[n:]...)...)
	} else if !ok {
		if len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "\n") {
			lines = append(lines, d.newline)
		}
		lines = append(lines, d.newline+"## Notes"+d.newline+text+d.newline)
	} else {
		n := s[1]
		insert := text + d.newline
		if n > 0 && !strings.HasSuffix(lines[n-1], "\n") {
			insert = d.newline + insert
		}
		lines = append(lines[:n], append([]string{insert}, lines[n:]...)...)
	}
	d.replace(lines)
}
func AtomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".huginn-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	mode := os.FileMode(0644)
	if st, e := os.Stat(path); e == nil {
		mode = st.Mode().Perm()
	}
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(data)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return sharefile.Rename(tmp, path)
}
