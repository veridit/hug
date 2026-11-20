package task

import (
	"regexp"
	"strings"
)

// Deducing from git (g9dg32, epic 47sxn2): when a commit that names a ticket
// in its subject reaches main, or a release tag first contains it, hug writes
// that onto the ticket and tells its owner. Only certain facts: the id in the
// subject and the commit reachable from main or the tag. It never ticks a
// check and never marks a task done.

// Commit is one commit seen on main: its full hash, subject and commit date
// (YYYY-MM-DD).
type Commit struct {
	Hash, Subject, Date string
}

// Short is the abbreviated hash hug writes: 7 characters, as git log shows.
func (c Commit) Short() string {
	if len(c.Hash) > 7 {
		return c.Hash[:7]
	}
	return c.Hash
}

// trailingGroup is the last parenthesised group of a subject, at its end.
var trailingGroup = regexp.MustCompile(`\(([^()]*)\)\s*$`)

// idWord is a task id: 6 characters of 0-9 a-z (as task ids are).
var idWord = regexp.MustCompile(`^[0-9a-z]{6}$`)

// SubjectIDs returns the ticket ids a commit subject names: the trailing
// "(id)" or "(id, id ...)" group, each comma or semicolon part whose first
// word is an id ("(x1t1t1 check 4)" names x1t1t1; "(qawmnb, CI run 43936088478)"
// names qawmnb only). Repeated ids are returned once, in order.
func SubjectIDs(subject string) []string {
	m := trailingGroup.FindStringSubmatch(subject)
	if m == nil {
		return nil
	}
	var out []string
	for _, part := range strings.FieldsFunc(m[1], func(r rune) bool { return r == ',' || r == ';' }) {
		f := strings.Fields(part)
		if len(f) == 0 || !idWord.MatchString(f[0]) {
			continue
		}
		id := f[0]
		if !contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

// Deduction is one fact hug writes onto a ticket, and how it tells the owner.
type Deduction struct {
	// ID is the ticket, Owner who has it (assignee, may be empty).
	ID, Owner string
	// Observed is what hug saw; Marked what it wrote; Fix the command that
	// undoes or corrects it.
	Observed, Marked, Fix string
	// Note is the line written to the ticket's Notes; empty when the ticket
	// already has it (then nothing is written and nothing reported).
	Note string
	// Closed is set by ApplyDeductions when the release closed the task
	// (yf39ya check 2): every check and child held and done went through.
	Closed bool
}

// Report is the line shown to the owner: "observed: ...; marked: ...; if
// wrong: <fix>" (Jørgen 2026-11-17 11:58).
func (d Deduction) Report() string {
	return "observed: " + d.Observed + "; marked: " + d.Marked + "; if wrong: " + d.Fix
}

// landedSubject is how much of a subject a landed line keeps: subjects run
// to 300-400 characters, and the whole is one git show away (crab
// 2026-11-17).
const landedSubject = 80

// LandedNote is the note a commit reaching main writes: the subject cut to
// at most 80 characters, at a word, with an ellipsis when cut.
func LandedNote(c Commit, branch string) string {
	subject := c.Subject
	if r := []rune(subject); len(r) > landedSubject {
		cut := string(r[:landedSubject])
		// At a word: the last space in the 80, unless that leaves little.
		if i := strings.LastIndex(cut, " "); i > len(cut)/2 && r[landedSubject] != ' ' {
			cut = cut[:i]
		}
		subject = strings.TrimRight(cut, " ,;:") + "…"
	}
	return "landed " + c.Short() + " on " + branch + " " + c.Date + ": " + subject
}

// ShippedNote is the note a release tag writes.
func ShippedNote(tag string, c Commit) string {
	return "shipped in " + tag + " (" + c.Short() + ")"
}

// hasNote reports whether a document's Notes already hold a line starting
// with prefix: the idempotence check, so passes and clones write a fact once.
func hasNote(d *Document, prefix string) bool {
	for _, line := range strings.Split(d.SectionText("Notes"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

// PlanLanded returns what the commits reaching branch say about the board:
// one deduction per (commit, named ticket) whose ticket exists and does not
// have the note yet. get loads a ticket by id (nil when there is none).
func PlanLanded(commits []Commit, branch string, get func(id string) *Document) []Deduction {
	var out []Deduction
	for _, c := range commits {
		for _, id := range SubjectIDs(c.Subject) {
			d := get(id)
			if d == nil || hasNote(d, "landed "+c.Short()) {
				continue
			}
			out = append(out, Deduction{
				ID: d.Task.ID, Owner: strings.Join(d.Task.Assignee, ", "),
				Observed: "commit " + c.Short() + " naming " + d.Task.ID + " reached " + branch,
				Marked:   "landed " + c.Short(),
				Fix:      "hug task note " + d.Task.ID + ` "not landed: ` + c.Short() + ` (why)"`,
				Note:     LandedNote(c, branch),
			})
		}
	}
	return out
}

// PlanShipped returns what a new release tag says: each ticket named by a
// commit new in the tag gets "shipped in TAG", once per tag.
func PlanShipped(tag string, commits []Commit, get func(id string) *Document) []Deduction {
	var out []Deduction
	seen := map[string]bool{}
	for _, c := range commits {
		for _, id := range SubjectIDs(c.Subject) {
			d := get(id)
			if d == nil || seen[d.Task.ID] || hasNote(d, "shipped in "+tag+" ") {
				continue
			}
			seen[d.Task.ID] = true
			out = append(out, Deduction{
				ID: d.Task.ID, Owner: strings.Join(d.Task.Assignee, ", "),
				Observed: "release " + tag + " contains commit " + c.Short() + " naming " + d.Task.ID,
				Marked:   "shipped in " + tag,
				Fix:      "hug task note " + d.Task.ID + ` "not in ` + tag + ` (why)"`,
				Note:     ShippedNote(tag, c),
			})
		}
	}
	return out
}

// ApplyDeductions writes each deduction's note on its ticket (and, for a
// shipped done ticket, done_release), checking the Notes again first so a
// line another clone wrote meanwhile is not written twice. It returns the
// deductions written; the caller commits them in one commit.
func (s *Store) ApplyDeductions(ds []Deduction) ([]Deduction, error) {
	var done []Deduction
	for _, x := range ds {
		d, _, e := s.load(x.ID)
		if e != nil {
			continue // the ticket is gone: nothing to mark
		}
		prefix := x.Marked
		if strings.HasPrefix(prefix, "shipped in ") {
			prefix += " "
		}
		if hasNote(d, prefix) {
			continue
		}
		d.Note(x.Note)
		if tag, ok := strings.CutPrefix(x.Marked, "shipped in "); ok && d.Task.State == "done" && d.Task.Fields["done_release"] == "" {
			if e := d.SetField("done_release", tag); e != nil {
				return done, e
			}
		}
		if _, e := s.finish(d, true); e != nil {
			return done, e
		}
		if tag, ok := strings.CutPrefix(x.Marked, "shipped in "); ok {
			// Close it, ask the person, or hand the open check to its
			// owner (yf39ya checks 2, 3 and 8).
			did, fix, e := s.afterShipped(x.ID, tag)
			if e != nil {
				return done, e
			}
			x.Closed = strings.HasPrefix(did, "done")
			if did != "" {
				x.Marked += ", " + did
				x.Fix = fix
			}
		}
		done = append(done, x)
	}
	return done, nil
}

// ReleaseLine is the History reason a release close writes (yf39ya).
func ReleaseLine(tag string) string { return "done, by release " + tag + ": every check holds" }

// closeShipped closes id after its shipped note for tag (yf39ya check 2,
// Jørgen 2026-11-18: shipped work left the board full of lint): a doing
// task, active or waiting (asked and review wait on a person), whose every
// check and child holds is closed by the ordinary done, so whatever hug
// task done refuses (an open dependency, a board condition) leaves it as
// it was, and its finding stands. done_release is the tag. A todo task is
// never closed: nobody took it. Only on the pass that writes the shipped
// note, so a clone that finds the note already written closes nothing and
// two clones give one done.
func (s *Store) closeShipped(id, tag string) (bool, error) {
	d, ts, e := s.load(id)
	if e != nil || d.Task.State != "doing" || (d.Task.Stage != "active" && d.Task.Stage != "waiting") || len(d.Task.Acceptance) == 0 {
		return false, nil
	}
	for _, a := range d.Task.Acceptance {
		if !a.Settled() {
			return false, nil
		}
	}
	if len(OpenChildren(d.Task, ts)) > 0 || len(Blocked(d.Task, ts)) > 0 || len(Unmet(d.Task, ts, true)) > 0 {
		return false, nil
	}
	if _, e := s.transitionWhy(id, "done", ""); e != nil {
		return false, nil // refused: the finding says why
	}
	return true, s.releaseDone(id, tag, ReleaseLine(tag))
}

// releaseDone sets done_release and replaces the done History line's
// reason with why, on a task just closed.
func (s *Store) releaseDone(id, tag, why string) error {
	d, _, e := s.load(id)
	if e != nil {
		return e
	}
	if e := d.SetField("done_release", tag); e != nil {
		return e
	}
	d.retagLastHistory("(done)", "("+why+")")
	_, e = s.finish(d, true)
	return e
}
