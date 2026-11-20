package task

import (
	"fmt"
	"regexp"
	"strings"
)

// promoteRef matches an acceptance item that waits on another task.
var promoteRef = regexp.MustCompile(`(?i)\bdone when task:([0-9a-z]{6}) is done\b`)

// Promote turns acceptance item n of ref into its own task: a sibling under
// the same epic (standalone when the source has no parent), ranked right
// after the source, in the same group. The item becomes "done when
// task:<id> is done" and is ticked when that task is done.
func (s *Store) Promote(ref string, n int) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	src := d.Task
	if n < 1 || n > len(src.Acceptance) {
		return Result{}, &Error{Kind: "acceptance-index", ID: src.ID, Value: fmt.Sprint(n), Fix: "hug task show " + src.ID}
	}
	item := src.Acceptance[n-1]
	if item.Checked {
		return Result{}, &Error{Kind: "already-done", ID: src.ID, Value: fmt.Sprintf("item %d is ticked; nothing to promote", n), Fix: "hug task show " + src.ID}
	}
	if item.Struck {
		return Result{}, &Error{Kind: "struck-out", ID: src.ID, Value: fmt.Sprintf("item %d is struck out as not relevant (%s); nothing to promote", n, item.Reason), Fix: fmt.Sprintf("hug task strike %s %d --undo", src.ID, n)}
	}
	if m := promoteRef.FindStringSubmatch(item.Text); m != nil {
		return Result{}, &Error{Kind: "already-promoted", ID: src.ID, Value: fmt.Sprintf("item %d already waits on task:%s", n, m[1]), Fix: "hug task show " + m[1]}
	}
	text := strings.TrimSpace(item.Text)
	title := text
	if r := []rune(title); len(r) > MaxTitle {
		title = strings.TrimSpace(string(r[:MaxTitle-1])) + "…"
	}
	parent := ""
	if src.Parent != "" {
		if p, e := ResolveDependency(ts, src.Parent); e == nil {
			parent = p.ID
		}
	} else if IsEpic(src, ts) {
		// Promoting from an epic: the new task is that epic's child.
		parent = src.ID
	}
	// The Goal is the item, one sentence (0pr03h); where it came from goes
	// to Relevant details, after the source's own.
	details := strings.TrimSpace(d.sectionText("Relevant details") + "\n\nPromoted from item " + fmt.Sprint(n) + " of task:" + src.ID + ".")
	r, e := s.New(NewOptions{
		Title:      title,
		Goal:       text,
		Acceptance: []string{text},
		Group:      src.Group,
		Type:       src.Type,
		Priority:   src.Priority,
		After:      src.ID,
		// The child serves the same purpose under the same principle, so
		// it starts complete and lints clean (heype5).
		Purpose:   d.sectionText("Purpose"),
		Principle: d.sectionText("Principle"),
		Details:   details,
	})
	if e != nil {
		return Result{}, e
	}
	created := r.Task
	if parent != "" {
		if _, e = s.Set(created.ID, "parent", parent); e != nil {
			return Result{}, e
		}
	}
	// Reload the source: New may have renamed files when it made rank room.
	d, _, e = s.load(src.ID)
	if e != nil {
		return Result{}, e
	}
	if e = d.SetItemText(n, "done when task:"+created.ID+" is done ("+text+")"); e != nil {
		return Result{}, e
	}
	if _, e = s.finish(d, true); e != nil {
		return Result{}, e
	}
	out, _, e := s.load(created.ID)
	if e != nil {
		return Result{}, e
	}
	return Result{Task: out.Task, Changed: true, Next: "hug task show " + created.ID, Renamed: r.Renamed, Renames: r.Renames}, nil
}

// sectionText is the body of the "## name" section, trimmed, or "".
func (d *Document) sectionText(name string) string {
	span, ok := d.sections[name]
	if !ok {
		return ""
	}
	return strings.TrimSpace(strings.Join(d.lines[span[0]+1:span[1]], ""))
}

// tickWaiting ticks every open item elsewhere that reads "done when
// task:<id> is done" for the task that just finished.
func (s *Store) tickWaiting(id string) ([]string, error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, e
	}
	var ticked []string
	for _, t := range ts {
		for n, a := range t.Acceptance {
			if a.Settled() {
				continue
			}
			if m := promoteRef.FindStringSubmatch(a.Text); m != nil && strings.EqualFold(m[1], id) {
				d, _, e := s.load(t.ID)
				if e != nil {
					return ticked, e
				}
				if e = d.Check(n + 1); e != nil {
					return ticked, e
				}
				if _, e = s.finish(d, true); e != nil {
					return ticked, e
				}
				ticked = append(ticked, fmt.Sprintf("%s item %d", t.ID, n+1))
			}
		}
	}
	return ticked, nil
}
