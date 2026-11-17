package task

import (
	"sort"
	"strconv"
	"strings"
)

// Who a question is for must be someone hug can find (89y0b7, crab 15:28):
// waiting_for, design_for and review_by name a person by their short name
// in hug.toml [people], or an agent by the name it works under on this
// board. Free text there would never reach anyone: nobody's TUI shows it
// as "you".

// personFields are the fields that name who must act.
var personFields = []string{"waiting_for", "design_for", "review_by"}

// KnownNames are the names a question may be for on this board: every
// listed person's short name and every agent or person the board shows
// working (assignees, the asker of a wait or a question), lower case.
func KnownNames(people []Person, ts []Task) map[string]bool {
	known := map[string]bool{}
	add := func(n string) {
		if n = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(n), "@")); n != "" {
			known[n] = true
		}
	}
	for _, p := range people {
		add(p.Short)
	}
	for _, t := range ts {
		for _, a := range t.Assignee {
			add(a)
		}
		for _, k := range []string{"waiting_asked_by", "design_asked_by"} {
			add(t.Fields[k])
		}
	}
	return known
}

// PeopleFindings flags an open task whose waiting_for, design_for or
// review_by names nobody known: not a [people] short name and no agent on
// the board. The fix adds the person; an email names its entry.
func PeopleFindings(people []Person, ts []Task) []Finding {
	known := KnownNames(people, ts)
	var out []Finding
	for _, t := range ts {
		if t.State == "done" || t.State == "dropped" {
			continue
		}
		for _, k := range personFields {
			who := strings.TrimSpace(t.Fields[k])
			if who == "" || known[strings.ToLower(strings.TrimPrefix(who, "@"))] {
				continue
			}
			short, email := who, "EMAIL"
			if strings.Contains(who, "@") {
				short, email = EmailShort(who), who
			}
			out = append(out, Finding{Path: t.Path, Line: 1, Code: "unknown-person",
				Problem: "nobody known: " + k + " " + who + " is not in hug.toml [people] and works on no task here, so nobody sees it as theirs",
				Fix:     "hug task people add " + short + " " + email})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// MoveWaitingFor is hug upgrade's move for waits written before
// waiting_for (89y0b7, crab 15:28): a waiting_on "NAME: what" whose NAME
// is a listed person becomes waiting_for NAME and waiting_on what. Only
// listed people: "CI run: 123" or "landing turn: x" stay as they are.
// It returns the changed paths and a line per change.
func (s *Store) MoveWaitingFor(people []Person, dryRun bool) (changed, lines []string, err error) {
	if len(people) == 0 {
		return nil, nil, nil
	}
	ts, e := s.Tasks()
	if e != nil {
		return nil, nil, e
	}
	for _, t := range ts {
		on := strings.TrimSpace(t.Fields["waiting_on"])
		if on == "" || strings.TrimSpace(t.Fields["waiting_for"]) != "" {
			continue
		}
		name, what, ok := strings.Cut(on, ":")
		p, listed := PersonByName(people, name)
		if !ok || !listed || strings.TrimSpace(what) == "" {
			continue
		}
		d, _, e := s.load(t.ID)
		if e != nil {
			return changed, lines, e
		}
		if e = d.SetField("waiting_for", p.Short); e != nil {
			return changed, lines, e
		}
		if e = d.SetField("waiting_on", strings.TrimSpace(what)); e != nil {
			return changed, lines, e
		}
		changed = append(changed, t.Path)
		lines = append(lines, t.ID+": waiting_on "+strconv.Quote(on)+" -> waiting_for "+strconv.Quote(p.Short)+", waiting_on "+strconv.Quote(strings.TrimSpace(what)))
		if dryRun {
			continue
		}
		path, e := s.diskPath(t.Path)
		if e != nil {
			return changed, lines, e
		}
		s.Invalidate()
		if e = AtomicWrite(path, d.Bytes()); e != nil {
			return changed, lines, e
		}
	}
	return changed, lines, nil
}
