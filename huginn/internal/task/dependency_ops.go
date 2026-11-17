package task

import (
	"errors"
	"strings"

	"github.com/veridit/hug/internal/cond"
)

// AddDependencies resolves refs once to canonical ids and adds only missing
// edges. Existing source lines, including aliases and human notes, are untouched.
// A proposed cycle rejects the entire operation before any file is written.
func (s *Store) AddDependencies(ref string, refs []string) (Result, error) {
	d, tasks, err := s.load(ref)
	if err != nil {
		return Result{}, err
	}
	id := d.Task.ID
	// Conditions (16dxsr): "when PREDICATE" and "group:NAME" are written
	// as their own lines; a branch condition records the branch's tip, so
	// it still reads as merged after the branch is deleted.
	var waits []string
	var plain []string
	for _, r := range refs {
		line, isWait, e := s.waitLine(id, d.Task.Waits, tasks, r)
		if e != nil {
			return Result{}, e
		}
		if isWait {
			if line != "" {
				waits = append(waits, line)
			}
		} else {
			plain = append(plain, r)
		}
	}
	refs = plain
	present := map[string]bool{}
	for _, dep := range d.Task.Dependencies {
		if target, err := ResolveDependency(tasks, dep.Ref); err == nil {
			present[target.ID] = true
		}
	}
	var added []string
	for _, ref := range refs {
		target, err := Resolve(tasks, ref)
		if err != nil {
			return Result{}, err
		}
		if present[target.ID] {
			continue
		}
		if path := dependencyPath(tasks, target.ID, id); len(path) > 0 {
			cycle := append([]string{id}, path...)
			// Prefer an existing edge as the repair, since the proposed edge
			// has not been written. Self-dependency has no existing edge.
			from, to := id, target.ID
			if len(path) > 1 {
				from, to = path[0], path[1]
			}
			var involved []Task
			seen := map[string]bool{}
			for _, id := range cycle {
				if seen[id] {
					continue
				}
				seen[id] = true
				for _, task := range tasks {
					if task.ID == id {
						involved = append(involved, task)
						break
					}
				}
			}
			return Result{}, &Error{Kind: "blocked", ID: id, Field: "dependencies", Value: "dependency cycle: " + strings.Join(cycle, " -> "), Cycle: cycle, Dependencies: involved, Fix: "hug task dep " + from + " --remove " + to}
		}
		present[target.ID] = true
		added = append(added, target.ID)
	}
	if len(added) == 0 && len(waits) == 0 {
		return s.finish(d, false)
	}
	d.appendDependencies(append(added, waits...))
	for _, id := range added {
		found := false
		for _, dep := range d.Task.Dependencies {
			found = found || dep.Ref == id
		}
		if !found {
			return Result{}, &Error{Kind: "broken-file", ID: d.Task.ID, Field: "dependencies", Fix: "hug task lint"}
		}
	}
	return s.finish(d, true)
}

// RemoveDependencies removes all matching source lines without rewriting any
// remaining text. A dangling source token can also be removed by its exact
// spelling (case-insensitively), so deleting a target never prevents repair.
func (s *Store) RemoveDependencies(ref string, refs []string) (Result, error) {
	d, tasks, err := s.load(ref)
	if err != nil {
		return Result{}, err
	}
	ids, raw := map[string]bool{}, map[string]bool{}
	waits := map[string]bool{}
	for _, ref := range refs {
		// A condition is removed by what it says ("when load below 120"),
		// matched on its canonical text, or exactly as written.
		if w, _, ok := parseWaitLine(ref); ok {
			waits[waitKey(w.Text)] = true
			continue
		}
		target, err := Resolve(tasks, ref)
		if err == nil {
			ids[target.ID] = true
			continue
		}
		var typed *Error
		if !errors.As(err, &typed) || typed.Kind != "not-found" {
			return Result{}, err
		}
		// Missing targets are safe no-ops when absent, and exact dangling
		// tokens are removable. Never guess an ambiguous resolver match.
		raw[strings.ToLower(ref)] = true
	}
	remove := map[int]bool{}
	for _, dep := range d.Task.Dependencies {
		matched := raw[strings.ToLower(dep.Ref)]
		if target, err := ResolveDependency(tasks, dep.Ref); err == nil {
			matched = matched || ids[target.ID]
		}
		if matched {
			remove[dep.Line-1] = true
		}
	}
	for _, w := range d.Task.Waits {
		if waits[waitKey(w.Text)] {
			remove[w.Line-1] = true
		}
	}
	if len(remove) == 0 {
		return s.finish(d, false)
	}
	lines := make([]string, 0, len(d.lines)-len(remove))
	for n, line := range d.lines {
		if !remove[n] {
			lines = append(lines, line)
		}
	}
	d.replace(lines)
	return s.finish(d, true)
}

func (d *Document) appendDependencies(ids []string) {
	lines := append([]string(nil), d.lines...)
	var text strings.Builder
	section, exists := d.sections["Depends on"]
	at := len(lines)
	before := false
	if exists {
		at = section[1]
	} else {
		// A new section goes in its canonical place (jnaspy): before
		// Acceptance and Notes, not at the end of the file.
		at, before = d.insertAt("Depends on")
	}
	if !before && at > 0 && lines[at-1] != "" && !strings.HasSuffix(lines[at-1], "\n") {
		text.WriteString(d.newline)
	}
	if !exists && !before {
		text.WriteString(d.newline + "## Depends on" + d.newline)
	} else if !exists {
		text.WriteString("## Depends on" + d.newline)
	}
	for _, id := range ids {
		text.WriteString("- " + id + d.newline)
	}
	if before {
		text.WriteString(d.newline)
	}
	lines = append(lines[:at], append([]string{text.String()}, lines[at:]...)...)
	d.replace(lines)
}

// dependencyPath returns an actual canonical-id path from start to goal.
// Existing dangling/ambiguous edges are left for lint, not guessed here.
func dependencyPath(tasks []Task, start, goal string) []string {
	byID := map[string]Task{}
	for _, task := range tasks {
		byID[task.ID] = task
	}
	visited := map[string]bool{}
	var walk func(string) []string
	walk = func(id string) []string {
		if id == goal {
			return []string{id}
		}
		if visited[id] {
			return nil
		}
		visited[id] = true
		for _, dep := range byID[id].Dependencies {
			if target, err := ResolveDependency(tasks, dep.Ref); err == nil {
				if path := walk(target.ID); len(path) > 0 {
					return append([]string{id}, path...)
				}
			}
		}
		return nil
	}
	return walk(start)
}

// waitLine reads one --add value as a condition. isWait is false for a
// task reference. A condition hug cannot read is refused here with what it
// can read; a group with no task is refused too, since it could never be
// checked. The line written is canonical: "when load below 120", "when
// group auth closed", "when branch feat/x merged at 1a2b3c4d5e6f".
func (s *Store) waitLine(id string, have []Wait, tasks []Task, ref string) (line string, isWait bool, err error) {
	w, perr, ok := parseWaitLine(ref)
	if !ok {
		return "", false, nil
	}
	fix := "hug task dep " + id + " --add " + shellQuote("when load below 120") + "  (or: " + cond.Kinds + ")"
	if perr != nil {
		return "", true, &Error{Kind: "invalid-condition", ID: id, Value: perr.Error(), Fix: fix}
	}
	c, _ := cond.Parse(w.Text)
	switch c.Kind {
	case cond.Group:
		if _, known := boardState(cond.State{}, tasks).GroupOpen(c.Ref); !known {
			return "", true, &Error{Kind: "invalid-condition", ID: id, Value: "no task is in group " + c.Ref, Fix: "hug task ls --group " + c.Ref}
		}
	case cond.Task:
		target, e := Resolve(tasks, c.Ref)
		if e != nil {
			return "", true, e
		}
		c.Ref, c.Text = target.ID, "task:"+target.ID+" closed"
	case cond.Branch:
		if c.At == "" {
			c.At = cond.Tip(s.Root, c.Ref)
		}
	}
	line = "when " + c.Written()
	if w.Wakes != "" {
		line += " wakes " + w.Wakes
	}
	for _, w := range have {
		if hc, e := cond.Parse(w.Text); e == nil && hc.Text == c.Text {
			return "", true, nil // already waits on it: a no-op
		}
	}
	return line, true, nil
}

// waitKey is what two spellings of one condition share: its canonical text
// without a recorded branch tip, else the text as written.
func waitKey(text string) string {
	if c, e := cond.Parse(text); e == nil {
		return c.Text
	}
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}
