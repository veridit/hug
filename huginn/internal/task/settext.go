package task

import "strings"

// Writers for set --purpose/--principle/--details/--accept (fkpyg3): every
// combined fix: line is one command an agent can run without an editor.

// SetSection replaces the body of "## name" with text, or adds the section
// in its canonical place. The heading and the blank line before the next
// heading are kept.
func (d *Document) SetSection(name, text string) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if d.newline != "\n" {
		text = strings.ReplaceAll(text, "\n", d.newline)
	}
	lines := append([]string(nil), d.lines...)
	if span, ok := d.span(name); ok {
		end := span[1]
		// Keep the trailing blank lines that separate it from the next heading.
		for end > span[0]+1 && strings.TrimSpace(lines[end-1]) == "" {
			end--
		}
		body := []string{text + d.newline}
		lines = append(lines[:span[0]+1], append(body, lines[end:]...)...)
		d.replace(lines)
		return
	}
	at, before := d.insertAt(name)
	block := "## " + name + d.newline + text + d.newline
	if before {
		block += d.newline
	} else {
		block = endSeparator(lines, at, d.newline) + block
	}
	lines = append(lines[:at], append([]string{block}, lines[at:]...)...)
	d.replace(lines)
}

// AddAcceptance appends unchecked items to Acceptance, creating the section
// in its canonical place when there is none.
func (d *Document) AddAcceptance(items []string) {
	var add strings.Builder
	for _, it := range items {
		add.WriteString("- [ ] " + strings.Join(strings.Fields(it), " ") + d.newline)
	}
	lines := append([]string(nil), d.lines...)
	if span, ok := d.sections["Acceptance"]; ok {
		at := span[1]
		for at > span[0]+1 && strings.TrimSpace(lines[at-1]) == "" {
			at--
		}
		if at > 0 && !strings.HasSuffix(lines[at-1], "\n") {
			lines[at-1] += d.newline
		}
		lines = append(lines[:at], append([]string{add.String()}, lines[at:]...)...)
		d.replace(lines)
		return
	}
	at, before := d.insertAt("Acceptance")
	block := "## Acceptance" + d.newline + add.String()
	if before {
		block += d.newline
	} else {
		block = endSeparator(lines, at, d.newline) + block
	}
	lines = append(lines[:at], append([]string{block}, lines[at:]...)...)
	d.replace(lines)
}

// endSeparator is what goes before a section appended at line at: a line
// end if the file has none, and one blank line unless there is one.
func endSeparator(lines []string, at int, nl string) string {
	if at == 0 {
		return ""
	}
	last := lines[at-1]
	if !strings.HasSuffix(last, "\n") {
		return nl + nl
	}
	if strings.TrimSpace(last) == "" {
		return ""
	}
	return nl
}

// SectionFlags are the set flags that write a description section.
var SectionFlags = map[string]string{"purpose": "Purpose", "principle": "Principle", "details": "Details", "goal": "Goal"}

// SetText writes description sections and appends acceptance items in one
// write: sections maps a section name to its new text.
func (s *Store) SetText(ref string, sections map[string]string, accept []string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	for _, a := range accept {
		if strings.TrimSpace(a) == "" {
			return Result{}, &Error{Kind: "empty-acceptance", ID: d.Task.ID, Fix: "hug task set " + d.Task.ID + " --accept \"observable check\""}
		}
	}
	// A new Goal is one sentence of at most 140 characters (0pr03h); a
	// Goal already on disk is read as it is, and lint warns on it.
	if v, ok := sections["Goal"]; ok {
		if p, bad := GoalProblem(v); bad {
			return Result{}, &Error{Kind: "goal-too-long", ID: d.Task.ID, Value: p.Text, Problems: []Problem{p}, Fix: "hug task set " + d.Task.ID + " " + goalFlag}
		}
	}
	before := string(d.Bytes())
	for _, name := range append(append([]string{}, "Goal"), DescriptionSections...) {
		if v, ok := sections[name]; ok {
			if strings.TrimSpace(v) == "" {
				return Result{}, &Error{Kind: "empty-section", ID: d.Task.ID, Value: name, Fix: "hug task set " + d.Task.ID + " --" + flagFor(name) + " \"...\""}
			}
			d.SetSection(name, v)
		}
	}
	if len(accept) > 0 {
		d.AddAcceptance(accept)
	}
	return s.finish(d, before != string(d.Bytes()))
}

func flagFor(section string) string {
	for f, s := range SectionFlags {
		if s == section {
			return f
		}
	}
	return strings.ToLower(section)
}
