// Package task owns lossless Markdown documents and task operations.
package task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/veridit/hug/huginn/internal/id"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var Statuses = []string{"todo", "doing", "done", "dropped"}
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
	Fix string `json:"fix"`
}

// findingCodes maps message prefixes to stable codes. Order matters: the
// first prefix that matches wins.
var findingCodes = []struct{ prefix, code string }{
	{"missing or unterminated frontmatter", "frontmatter"},
	{"malformed frontmatter", "frontmatter"},
	{"frontmatter ", "frontmatter"},
	{"duplicate section", "duplicate-section"},
	{"duplicate field", "duplicate-field"},
	{"missing field", "missing-field"},
	{"invalid id", "invalid-id"},
	{"filename id differs", "id-mismatch"},
	{"unknown status", "unknown-status"},
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
	{"done with unchecked acceptance", "done-unchecked"},
	{"case-insensitive filename collision", "filename-collision"},
	{"duplicate id", "duplicate-id"},
	{"depends on ", "dangling-dependency"},
	{"not a dependency", "not-a-dependency"},
	{"dependency cycle", "dependency-cycle"},
	{"missing parent", "missing-parent"},
	{"task is its own parent", "self-parent"},
	{"parent ", "parent-too-deep"},
	{"title is ", "long-title"},
	{"cites ", "forbidden-citation"},
	{"task file is a symlink", "file-symlink"},
	{"duplicate origin", "duplicate-origin"},
	{"branch ", "branch-gone"},
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
	case strings.Contains(problem, ": file not found"):
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
	return json.Marshal(plain(f))
}

// FixLine is the command that repairs the finding: its own Fix, else the
// usual repair for its code, naming the task the file belongs to. Every
// finding has one, so a reader always has something to run.
func (f Finding) FixLine() string {
	if f.Fix != "" {
		return f.Fix
	}
	_, i, _, err := ParseName(f.Path)
	if err != nil || i == "" {
		return "hug task lint"
	}
	switch FindingCode(f.Problem) {
	case "unknown-status":
		return "hug task set " + i + " --status todo"
	case "unknown-type":
		return "hug task set " + i + " --type feature"
	case "missing-parent", "self-parent":
		return "hug task set " + i + " --parent none"
	case "filename-collision", "duplicate-id", "duplicate-origin":
		return "hug task show " + i
	}
	return "hug task edit " + i
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s", f.Path, f.Line, FindingCode(f.Problem), f.Problem)
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
}
type Task struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Group  string `json:"group"`
	Rank   string `json:"rank"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
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
	Acceptance   []Acceptance `json:"acceptance"`
	// Texts are what show and ls print beside the fields: the Notes body
	// and the description sections (DescriptionSections), filled by the
	// same Parse that fills the Task, so a cached Task answers ls without
	// reading 10,000 files again (44w1ev). Kept out of the JSON payload:
	// cli builds those keys itself. Read only: copies of a Task share it.
	Texts *Texts `json:"-"`
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
	d.Task.Group = filepath.ToSlash(filepath.Dir(path))
	if d.Task.Group == "." {
		d.Task.Group = ""
	}
	r, i, s, e := ParseName(path)
	d.Task.Rank, d.Task.ID, d.Task.Slug = r, i, s
	if e != nil {
		d.add(1, e.Error())
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
				d.Task.Acceptance = append(d.Task.Acceptance, Acceptance{t[6:], t[3] != ' ', n + 1})
			}
		}
	}
	if section != "" {
		d.sections[section] = [2]int{sectionStart, len(d.lines)}
	}
	for _, field := range []struct{ k, v string }{{"id", d.Task.ID}, {"title", d.Task.Title}, {"status", d.Task.Status}} {
		_, ok := d.fields[field.k]
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
	if !contains(Statuses, d.Task.Status) {
		d.add(d.fieldLine("status"), "unknown status "+d.Task.Status)
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
	if d.Task.Status == "done" {
		for n, a := range d.Task.Acceptance {
			if !a.Checked {
				d.Findings = append(d.Findings, Finding{Path: d.Task.Path, Line: a.Line, Problem: "done with unchecked acceptance", Fix: fmt.Sprintf("hug task check %s %d", d.Task.ID, n+1)})
			}
		}
	}
	d.Task.Goal = strings.ToValidUTF8(stripBacklogMarkers(d.Task.Goal), "\ufffd")
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
	lines := append([]string(nil), d.lines...)
	lines[a.Line-1] = strings.Replace(lines[a.Line-1], "[ ]", "[x]", 1)
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
	return os.Rename(tmp, path)
}
