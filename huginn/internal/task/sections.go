package task

import (
	"github.com/veridit/hug/internal/sharefile"
	"strings"
)

// Canonical section order (jnaspy, Jørgen 2026-11-16): opening a task shows
// what it is about first. Goal, Purpose, Principle, Details, then
// Depends on and Acceptance, then any other section in the order the file
// has them, then Notes (appended, oldest first) and History last. The file
// writer, hug task show, the TUI and MCP all use this order; hug upgrade
// puts existing files in it without changing any section's text.
var SectionOrder = []string{"Goal", "Purpose", "Principle", "Details", "Depends on", "Acceptance"}

// OldDetails is the Details section's name before 8a0ybb (Jørgen
// 2026-11-20: "Relevant details" is shortened to "Details"). A file with it
// still reads: the section is Details wherever hug looks for it.
const OldDetails = "Relevant details"

// sectionName is the name hug knows a "## " heading by: Details for the
// old "Relevant details", else the heading as written.
func sectionName(heading string) string {
	if strings.EqualFold(heading, OldDetails) {
		return "Details"
	}
	return heading
}

// TailSections close a task, in this order, after every other section.
var TailSections = []string{"Notes", "History"}

// sectionRank places a "## " heading: its index in SectionOrder, then other
// sections, then TailSections.
func sectionRank(name string) int {
	name = sectionName(name)
	for i, s := range SectionOrder {
		if strings.EqualFold(s, name) {
			return i
		}
	}
	for i, s := range TailSections {
		if strings.EqualFold(s, name) {
			return len(SectionOrder) + 1 + i
		}
	}
	return len(SectionOrder) // another section: before Notes
}

// InOrder reports whether headings (in file order) follow SectionOrder.
func InOrder(headings []string) bool {
	for i := 1; i < len(headings); i++ {
		if sectionRank(headings[i-1]) > sectionRank(headings[i]) {
			return false
		}
	}
	return true
}

// insertAt is the line where a new "## name" section goes so the file stays
// in canonical order: the start of the first section that ranks after name,
// or the end of the file when none does. Every writer that adds a section
// uses it (Depends on, Notes), so writing never puts a file out of order.
func (d *Document) insertAt(name string) (at int, beforeHeading bool) {
	at, rank := len(d.lines), sectionRank(name)
	for s, span := range d.sections {
		if sectionRank(s) > rank && span[0] < at {
			at, beforeHeading = span[0], true
		}
	}
	return at, beforeHeading
}

// OrderedSections are the canonical order of headings: a stable sort by
// sectionRank, so other sections keep their order among themselves.
func OrderedSections(headings []string) []string {
	out := append([]string(nil), headings...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && sectionRank(out[j-1]) > sectionRank(out[j]); j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// Canonical is the file with its sections in canonical order. Each section
// moves as a block, its heading and every line up to the next heading, with
// its text unchanged; frontmatter and anything before the first heading stay
// first. A block that did not end in a blank line before the next heading
// gets one when it moves, so headings never run into text. changed is false
// when the file was already in order (the bytes are then the same).
func (d *Document) Canonical() (data []byte, changed bool) {
	if InOrder(d.Task.Sections) {
		return d.Bytes(), false
	}
	type block struct {
		name  string
		lines []string
	}
	start := d.frontEnd + 1
	if start < 0 {
		start = 0
	}
	// Headings outside fences, as Parse found them: d.sections holds each
	// name's last span, so walk the lines again with the same fence rule.
	var head []string
	var blocks []block
	fence := ""
	for n := 0; n < len(d.lines); n++ {
		line := strings.TrimSuffix(strings.TrimSuffix(d.lines[n], "\n"), "\r")
		if n >= start {
			trimmed := strings.TrimLeft(line, " ")
			if len(line)-len(trimmed) <= 3 && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
				if fence == "" {
					count := 0
					for count < len(trimmed) && trimmed[count] == trimmed[0] {
						count++
					}
					fence = trimmed[:count]
				} else if strings.HasPrefix(trimmed, fence) && strings.TrimSpace(strings.TrimPrefix(trimmed, fence)) == "" {
					fence = ""
				}
			} else if fence == "" && strings.HasPrefix(line, "## ") {
				blocks = append(blocks, block{name: strings.TrimSpace(line[3:])})
			}
		}
		if len(blocks) == 0 {
			head = append(head, d.lines[n])
		} else {
			b := &blocks[len(blocks)-1]
			b.lines = append(b.lines, d.lines[n])
		}
	}
	// Stable insertion sort by rank.
	for i := 1; i < len(blocks); i++ {
		for j := i; j > 0 && sectionRank(blocks[j-1].name) > sectionRank(blocks[j].name); j-- {
			blocks[j-1], blocks[j] = blocks[j], blocks[j-1]
		}
	}
	blank := func(s string) bool { return strings.TrimSpace(s) == "" }
	out := append([]string(nil), head...)
	for i, b := range blocks {
		lines := append([]string(nil), b.lines...)
		last := len(lines) - 1
		if !strings.HasSuffix(lines[last], "\n") {
			lines[last] += d.newline
		}
		if i < len(blocks)-1 && !blank(lines[last]) {
			lines = append(lines, d.newline)
		}
		out = append(out, lines...)
	}
	// The last block keeps the file's own ending: trim a separator blank
	// line it gained, and the newline when the file had none.
	text := strings.Join(out, "")
	orig := string(d.raw)
	if !strings.HasSuffix(orig, "\n") {
		text = strings.TrimRight(text, "\r\n")
	} else {
		for strings.HasSuffix(text, d.newline+d.newline) && !strings.HasSuffix(orig, d.newline+d.newline) {
			text = strings.TrimSuffix(text, d.newline)
		}
	}
	return []byte(text), text != orig
}

// span is the lines of the "## name" section: Details is the old
// "## Relevant details" in a file written before 8a0ybb.
func (d *Document) span(name string) ([2]int, bool) {
	if s, ok := d.sections[name]; ok {
		return s, true
	}
	if name == "Details" {
		s, ok := d.sections[OldDetails]
		return s, ok
	}
	return [2]int{}, false
}

// SectionText is the body of the "## name" section, trimmed, or "".
func (d *Document) SectionText(name string) string { return d.sectionText(name) }

// Reorder tidies every task file for hug upgrade, the mechanical part only
// (fkpyg3): backlog leftovers removed (StripBacklog), then sections put in
// canonical order, no section's text changed. It returns the paths it
// rewrote; a second run rewrites nothing.
func (s *Store) Reorder() ([]string, error) {
	changed, _, e := s.Tidy(false)
	return changed, e
}

// Tidy is Reorder that also returns the diff of each rewritten file; with
// dryRun nothing is written.
func (s *Store) Tidy(dryRun bool) (changed []string, diff string, err error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, "", e
	}
	var b strings.Builder
	for _, t := range ts {
		path, e := s.diskPath(t.Path)
		if e != nil {
			return changed, b.String(), e
		}
		raw, e := sharefile.ReadFile(path)
		if e != nil {
			return changed, b.String(), e
		}
		stripped, didStrip := Parse(t.Path, raw).StripBacklog()
		out, didOrder := Parse(t.Path, stripped).Canonical()
		if !didStrip && !didOrder {
			continue
		}
		b.WriteString(TidyDiff(t.Path, raw, out))
		changed = append(changed, t.Path)
		if dryRun {
			continue
		}
		s.Invalidate()
		if e := AtomicWrite(path, out); e != nil {
			return changed, b.String(), e
		}
	}
	return changed, b.String(), nil
}

// TidyDiff shows what upgrade changed in one file: each removed line as
// "- line", and the old and new section order when it moved sections.
func TidyDiff(path string, before, after []byte) string {
	var b strings.Builder
	b.WriteString("--- " + path + "\n")
	kept := map[string]int{}
	for _, l := range strings.Split(string(after), "\n") {
		kept[strings.TrimRight(l, "\r")]++
	}
	for _, l := range strings.Split(string(before), "\n") {
		l = strings.TrimRight(l, "\r")
		if kept[l] > 0 {
			kept[l]--
			continue
		}
		b.WriteString("- " + l + "\n")
	}
	old, now := Parse(path, before).Task.Sections, Parse(path, after).Task.Sections
	if strings.Join(old, "\x00") != strings.Join(now, "\x00") {
		b.WriteString("  sections: " + strings.Join(old, ", ") + "\n  now:      " + strings.Join(now, ", ") + "\n")
	}
	return b.String()
}
