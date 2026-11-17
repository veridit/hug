package task

import (
	"fmt"
	"strings"
	"time"
)

// Moves (6ephqk check 3): the commands that change a task's state or stage
// write the node fields of where it goes, move the fields of where it was
// to ## History, and set the stage to the first that holds (Stage). Only the
// moves in Transitions are made; any other is refused with a fix: line that
// runs without an editor (crab 11:32).

// Now is the clock the moves stamp *_since and *_at fields with; tests set
// it.
var Now = time.Now

func stamp() string { return Now().UTC().Format(StampLayout) }

// StampLayout is how hug writes a time: UTC, RFC 3339 to the second, with
// Z (crab 13:33, kestrel: a local time without a zone reads an hour off on
// a clone in another zone).
const StampLayout = "2006-01-02T15:04:05Z"

// zonelessLayout is the old form, local time without a zone: hug upgrade
// converts it, lint names it.
const zonelessLayout = "2006-01-02T15:04"

// ParseStamp reads a time hug wrote: RFC 3339 with a zone, or the old
// zoneless form as local time (until hug upgrade converts it).
func ParseStamp(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if t, e := time.Parse(time.RFC3339, s); e == nil {
		return t, true
	}
	if t, e := time.ParseInLocation(zonelessLayout, s, time.Local); e == nil {
		return t, true
	}
	return time.Time{}, false
}

// TimeField is whether a node field holds a time: *_since and *_at.
func TimeField(k string) bool { return strings.HasSuffix(k, "_since") || strings.HasSuffix(k, "_at") }

// RemoveField deletes frontmatter field k (and its continuation lines), if
// present.
func (d *Document) RemoveField(k string) error {
	if d.frontEnd < 0 || !d.frontmatterValid {
		return &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint"}
	}
	n, ok := d.fields[k]
	if !ok {
		return nil
	}
	end := d.fieldEnds[k]
	if end <= n {
		end = n + 1
	}
	lines := append([]string(nil), d.lines...)
	d.replace(append(lines[:n], lines[end:]...))
	return nil
}

// History appends one line to ## History, creating the section last in the
// file (History is the last section).
func (d *Document) History(text string) {
	lines := append([]string(nil), d.lines...)
	if s, ok := d.sections["History"]; ok {
		n := s[1]
		for n > s[0]+1 && strings.TrimSpace(lines[n-1]) == "" {
			n--
		}
		insert := "- " + text + d.newline
		if n > 0 && !strings.HasSuffix(lines[n-1], "\n") {
			insert = d.newline + insert
		}
		lines = append(lines[:n], append([]string{insert}, lines[n:]...)...)
	} else {
		if len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "\n") {
			lines = append(lines, d.newline)
		}
		lines = append(lines, d.newline+"## History"+d.newline+"- "+text+d.newline)
	}
	d.replace(lines)
}

// move is one change of state or stage: the state to go to, the fields
// to write there, and which node fields to drop (to History) though their
// node holds still (resume drops waiting_on).
type move struct {
	state string
	set   map[string]string
	clear []string
	why   string // History line suffix: "take", "wait", "reopen" ...
}

// applyMove writes m on d: fields of nodes that no longer hold move to
// History with the time, m's fields are written, then the stage is set to
// the first that holds. ts is the board (dependencies' states).
func applyMove(d *Document, ts []Task, m move) error {
	from := d.Task.Status
	if d.Task.Stage != "" {
		from += "/" + d.Task.Stage
	}
	if m.state != d.Task.Status {
		if e := d.SetField("status", m.state); e != nil {
			return e
		}
	}
	for k, v := range m.set {
		if e := d.SetField(k, v); e != nil {
			return e
		}
	}
	owners := NodeFields()
	var moved []string
	for _, k := range sortedKeys(d.Task.Fields) {
		if _, set := m.set[k]; set {
			continue
		}
		if contains(m.clear, k) || !fieldHolds(d.Task, k, owners[k]) {
			moved = append(moved, k+": "+d.Task.Fields[k])
			if e := d.RemoveField(k); e != nil {
				return e
			}
		}
	}
	if e := settle(d, ts); e != nil {
		return e
	}
	to := d.Task.Status
	if d.Task.Stage != "" {
		to += "/" + d.Task.Stage
	}
	if from != to || len(moved) > 0 {
		line := stamp() + " " + from + " -> " + to
		if m.why != "" {
			line += " (" + m.why + ")"
		}
		if len(moved) > 0 {
			line += "; was " + strings.Join(moved, "; ")
		}
		d.History(line)
	}
	return nil
}

// settle sets d's stage to the first that holds, removing the stage line
// for done and dropped. Fields of nodes left behind stay: applyMove moves
// them first.
func settle(d *Document, ts []Task) error {
	t := d.Task
	for i := range ts {
		if ts[i].ID == t.ID {
			ts = append(append([]Task(nil), ts[:i]...), ts[i+1:]...)
			break
		}
	}
	want := Stage(t, append(ts, t))
	if want == "" {
		return d.RemoveField("stage")
	}
	if want == t.Stage {
		return nil
	}
	return d.SetField("stage", want)
}

// allowed says whether moving t from its state to state is one of
// Transitions. A move within the same state (a stage change) is checked by
// the command that makes it.
func allowed(t Task, state string) bool {
	from := t.Status
	for _, tr := range Transitions {
		f, _, _ := strings.Cut(tr.From, "/")
		to, _, _ := strings.Cut(tr.To, "/")
		if (tr.From == "any" || f == from) && to == state {
			return true
		}
	}
	return false
}

// refuse is the error for a move the tree does not allow, with the command
// that is allowed instead.
func refuse(t Task, to, fix string) error {
	from := t.Status
	if t.Stage != "" {
		from += "/" + t.Stage
	}
	return &Error{Kind: "not-allowed", ID: t.ID, Value: from + " -> " + to + " is not a move a command makes (hug task states)", Fix: fix}
}

// Wait moves ref, a doing task, to doing/waiting on what; by is who waits
// (waiting_asked_by), told when it is answered (3jtfw9).
func (s *Store) Wait(ref, on, by string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	on = strings.TrimSpace(on)
	if on == "" {
		return Result{}, usageErr(d.Task.ID, "wait needs what it waits on", "hug task wait "+d.Task.ID+` "CI run 123"`)
	}
	if d.Task.Status != "doing" {
		return Result{}, refuse(d.Task, "doing/waiting", "hug task take "+d.Task.ID+" --model MODEL")
	}
	if d.Task.Fields["waiting_on"] == on {
		return s.finish(d, false)
	}
	set := map[string]string{"waiting_on": on, "waiting_since": stamp()}
	if by = strings.TrimSpace(by); by != "" {
		set["waiting_asked_by"] = by
	}
	if e = applyMove(d, ts, move{state: "doing", set: set, why: "wait"}); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// Resume ends a wait: the waiting fields go to History.
func (s *Store) Resume(ref string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if d.Task.Status != "doing" || !ownHolds(d.Task, "waiting") {
		return Result{}, refuse(d.Task, "doing/active", "hug task show "+d.Task.ID)
	}
	if e = applyMove(d, ts, move{state: "doing", clear: []string{"waiting_on", "waiting_since", "waiting_asked_by"}, why: "resume"}); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// Design asks question on ref, a todo task (todo/design), of who (a
// person's name, design_for, required: crab 12:38), recording who asked;
// clear (question "") answers it: the design fields go to History.
func (s *Store) Design(ref, question, by, who string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if d.Task.Status != "todo" {
		fix := "hug task set " + d.Task.ID + " --status todo"
		return Result{}, refuse(d.Task, "todo/design", fix)
	}
	question = strings.TrimSpace(question)
	if question == "" {
		if !ownHolds(d.Task, "design") {
			return s.finish(d, false)
		}
		if e = applyMove(d, ts, move{state: "todo", clear: []string{"design_question", "design_since", "design_asked_by", "design_for"}, why: "design answered"}); e != nil {
			return Result{}, e
		}
		return s.finish(d, true)
	}
	if who = strings.TrimSpace(who); who == "" {
		return Result{}, usageErr(d.Task.ID, "design needs who must answer: --for NAME", "hug task design "+d.Task.ID+` "`+question+`" --for NAME`)
	}
	if d.Task.Fields["design_question"] == question && d.Task.Fields["design_for"] == who {
		return s.finish(d, false)
	}
	set := map[string]string{"design_question": question, "design_since": stamp(), "design_for": who}
	if by = strings.TrimSpace(by); by != "" {
		set["design_asked_by"] = by
	}
	if e = applyMove(d, ts, move{state: "todo", set: set, why: "design"}); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// DropBecause drops ref with a reason (dropped_reason).
func (s *Store) DropBecause(ref, reason string) (Result, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		id := ref
		if d, e := s.Get(ref); e == nil {
			id = d.Task.ID
		}
		return Result{}, usageErr(id, "drop needs a reason", "hug task drop "+id+` "duplicate of OTHER"`)
	}
	return s.transitionWhy(ref, "dropped", reason)
}

func usageErr(id, problem, fix string) error {
	return &Error{Kind: "usage", ID: id, Value: problem, Fix: fix}
}

// staged moves d along the tree after transitionAt's checks: the state's
// fields are written and the old node's fields go to History. user is who
// takes (doing/review sets review_by to them).
func stagedMove(d *Document, ts []Task, state, user, reason string) error {
	set := map[string]string{}
	why := ""
	switch state {
	case "doing":
		if d.Task.Status != "doing" {
			set["doing_since"] = stamp()
			why = "take"
			if d.Task.Status == "todo" && ownHolds(d.Task, "review") && user != "" {
				set["review_by"] = user
				set["review_since"] = stamp()
				why = "take for review"
			}
		}
	case "todo":
		why = "give back"
		if d.Task.Status == "done" || d.Task.Status == "dropped" {
			why = "reopen"
		}
	case "done":
		set["done_at"] = stamp()
		why = "done"
	case "dropped":
		set["dropped_reason"] = reason
		set["dropped_at"] = stamp()
		why = "drop"
	}
	var clear []string
	if state == "todo" && d.Task.Status == "doing" {
		// Giving back keeps the todo stage it came from (doing/review
		// becomes todo/review): review_by stays, the reviewer's stamp does
		// not. todo means nobody is working on it: the assignee goes too,
		// named in History, so it is back in NEXT UP.
		clear = []string{"doing_since"}
		if len(d.Task.Assignee) > 0 {
			why += "; was assignee: " + strings.Join(d.Task.Assignee, ", ")
			if e := d.SetField("assignee", []string{}); e != nil {
				return e
			}
		}
	}
	return applyMove(d, ts, move{state: state, set: set, clear: clear, why: why})
}

// cascade re-settles the tasks that depend on id directly: their depends
// stage follows id's state. One level: a stage reads only its own
// dependencies' states, so a chain settles one link per write (each link's
// state is unchanged by its stage). A task is settled once per call, which
// bounds a cycle. Only staged files are rewritten.
func (s *Store) cascade(id string) ([]string, error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, e
	}
	var touched []string
	seen := map[string]bool{id: true}
	for _, t := range ts {
		if seen[t.ID] || !Staged(t) || t.Status == "done" || t.Status == "dropped" {
			continue
		}
		dependent := false
		for _, dep := range t.Dependencies {
			if x, e := ResolveDependency(ts, dep.Ref); e == nil && x.ID == id {
				dependent = true
			}
		}
		if !dependent || Stage(t, ts) == t.Stage {
			continue
		}
		seen[t.ID] = true
		d, _, e := s.load(t.ID)
		if e != nil {
			return touched, e
		}
		if e = applyMove(d, ts, move{state: t.Status, why: "dependency " + id}); e != nil {
			return touched, e
		}
		if _, e = s.finish(d, true); e != nil {
			return touched, e
		}
		touched = append(touched, fmt.Sprintf("%s %s", t.ID, d.Task.Stage))
	}
	return touched, nil
}
