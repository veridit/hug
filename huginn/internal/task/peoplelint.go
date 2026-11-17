package task

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Who a question is for must be someone hug can find (89y0b7, crab 15:28):
// asked_for, design_for and review_by name a person by their short name
// in hug.toml [people], or an agent by the name it works under on this
// board. Free text there would never reach anyone: nobody's TUI shows it
// as "you".

// personFields are the fields that name who must act.
var personFields = []string{"asked_for", "design_for", "review_by"}

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
		for _, k := range []string{"waiting_asked_by", "asked_by", "design_asked_by"} {
			add(t.Fields[k])
		}
	}
	return known
}

// PeopleFindings flags an open task whose asked_for, design_for or
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

// checkPrefix is a leading "check N:" in a question (0gztsy wrote them so
// before asked_check, 41hhgn).
var checkPrefix = regexp.MustCompile(`^(?i)check\s+(\d+)\s*[:,-]\s*`)

// MoveToAsked is hug upgrade's move for questions written as waits
// (41hhgn, Jørgen 23:48): a doing/waiting with waiting_for, or (before
// 89y0b7) a waiting_on "NAME: what" whose NAME is a listed person, becomes
// doing/asked. asked_what is the question, with a leading "check N:" (N a
// check of the task) moved to asked_check; asked_for, asked_since,
// asked_by and asked_show come from waiting_for, waiting_since,
// waiting_asked_by and waiting_show. Waits on things stay. It returns the
// changed paths and a line per change.
func (s *Store) MoveToAsked(people []Person, dryRun bool) (changed, lines []string, err error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, nil, e
	}
	for _, t := range ts {
		on := strings.TrimSpace(t.Fields["waiting_on"])
		if on == "" {
			continue
		}
		d, ts2, e := s.load(t.ID)
		if e != nil {
			return changed, lines, e
		}
		// waiting_for and waiting_show are no longer node fields (41hhgn),
		// so Parse leaves them out of Task.Fields: read them off the file.
		who, show := d.legacyField("waiting_for"), d.legacyField("waiting_show")
		what := on
		if who == "" {
			name, rest, ok := strings.Cut(on, ":")
			p, listed := PersonByName(people, name)
			if !ok || !listed || strings.TrimSpace(rest) == "" {
				continue
			}
			who, what = p.Short, strings.TrimSpace(rest)
		}
		check := ""
		if m := checkPrefix.FindStringSubmatch(what); m != nil {
			if n, _ := strconv.Atoi(m[1]); n >= 1 && n <= len(t.Acceptance) {
				check, what = m[1], strings.TrimSpace(what[len(m[0]):])
			}
		}
		set := map[string]string{"asked_what": what, "asked_for": who}
		for from, to := range map[string]string{"waiting_since": "asked_since", "waiting_asked_by": "asked_by"} {
			if v := strings.TrimSpace(t.Fields[from]); v != "" {
				set[to] = v
			}
		}
		if show != "" {
			set["asked_show"] = show
		}
		if check != "" {
			set["asked_check"] = check
		}
		for _, k := range sortedKeys(set) {
			if e = d.SetField(k, set[k]); e != nil {
				return changed, lines, e
			}
		}
		for _, k := range []string{"waiting_on", "waiting_for", "waiting_since", "waiting_asked_by", "waiting_show"} {
			if _, ok := d.fields[k]; ok {
				if e = d.RemoveField(k); e != nil {
					return changed, lines, e
				}
			}
		}
		if e = settle(d, ts2); e != nil {
			return changed, lines, e
		}
		line := t.ID + ": doing/waiting -> doing/asked, asked_for " + who
		if check != "" {
			line += ", asked_check " + check
		}
		line += ", asked_what " + strconv.Quote(what)
		d.History(stamp() + " doing/waiting -> doing/asked (hug upgrade, 41hhgn: a question to a person is its own stage)")
		changed = append(changed, t.Path)
		lines = append(lines, line)
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
