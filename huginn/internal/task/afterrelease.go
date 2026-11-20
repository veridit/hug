package task

import (
	"regexp"
	"strconv"
	"strings"
)

// After a release (yf39ya check 3, Jørgen 2026-11-18: "things that have
// shipped have a validation failure, so they show up in lint"): a shipped
// task whose every open check is a person's (the agxqax reading: "Jørgen
// confirms on his board after release") is asked of that person, one check
// at a time, through the public Ask, so it shows in their ASKED instead of
// as shipped-not-done. Their tick answers it (qsqhpe); AfterAnswer then asks
// the next such check or, when every check and child holds, closes the
// task. A text answer closes nothing: it may say the work is broken. The
// ask code itself is not touched (q2adr8 owns it).

// personChecks returns, for a doing task (active or waiting) whose every
// open check addresses a person on the board, the first such check and who
// it is for. ok is false when any open check is not a person's, or none is
// open.
func personChecks(t Task, people []Person) (q Question, ok bool) {
	if t.State != "doing" || (t.Stage != "active" && t.Stage != "waiting") {
		return Question{}, false
	}
	for i, a := range t.Acceptance {
		if a.Settled() {
			continue
		}
		w, is := AddressesPerson(people, a.Text, len(t.Acceptance))
		if !is {
			return Question{}, false
		}
		if q.For == "" {
			q = Question{What: strings.TrimSpace(a.Text), For: w.For, Check: i + 1}
		}
	}
	return q, q.For != ""
}

// askAfterRelease asks the person of id's first person check, when every
// open check is a person's (the release's shipped note was just written).
// The asker is the board's coordinator, else the owner.
func (s *Store) askAfterRelease(id string) (Question, bool, error) {
	people, e := People(s.Root)
	if e != nil || len(people) == 0 {
		return Question{}, false, nil
	}
	d, _, e := s.load(id)
	if e != nil {
		return Question{}, false, nil
	}
	q, ok := personChecks(d.Task, people)
	if !ok {
		return Question{}, false, nil
	}
	by, _ := Coordinator(s.Root)
	if by == "" && len(d.Task.Assignee) > 0 {
		by = d.Task.Assignee[0]
	}
	if _, e := s.Ask(id, q, by); e != nil {
		return Question{}, false, nil // refused: shipped-not-done stands and says why
	}
	return q, true, nil
}

// releaseOf is the release a task's newest shipped note names ("" none).
func releaseOf(d *Document) string {
	m := shippedLine.FindAllStringSubmatch(d.SectionText("Notes"), -1)
	if len(m) == 0 {
		return ""
	}
	return m[len(m)-1][1]
}

// AfterAnswer runs after a write that answered a task's question by
// ticking check n (Result.Answered from CheckWith; the CLI check, MCP check
// and the TUI's Space all call it). On a shipped task it asks the next
// person check, or closes the task when every check and child holds and
// done goes through. who is the person who ticked. It reports what it did
// ("" for nothing).
func (s *Store) AfterAnswer(id string, n int, who string) (string, error) {
	d, ts, e := s.load(id)
	if e != nil {
		return "", nil
	}
	tag := releaseOf(d)
	if tag == "" || d.Task.State != "doing" {
		return "", nil
	}
	for _, a := range d.Task.Acceptance {
		if !a.Settled() {
			if q, ok, e := s.askAfterRelease(id); e != nil || !ok {
				return "", e
			} else {
				return "asked " + q.For + " about check " + strconv.Itoa(q.Check), nil
			}
		}
	}
	if len(OpenChildren(d.Task, ts)) > 0 || len(Blocked(d.Task, ts)) > 0 || len(Unmet(d.Task, ts, true)) > 0 {
		return "", nil
	}
	if _, e := s.transitionWhy(id, "done", ""); e != nil {
		return "", nil // refused: lint says why
	}
	if e := s.releaseDone(id, tag, AnswerDoneLine(who, n, tag)); e != nil {
		return "", e
	}
	return "done", nil
}

// AnswerDoneLine is the History reason when a person's tick closes a
// shipped task (crab 19:59).
func AnswerDoneLine(who string, n int, tag string) string {
	return "done after " + who + " ticked check " + strconv.Itoa(n) + " (release " + tag + ")"
}

// After a release, the open check that is not a person's (yf39ya check 8,
// x8zzzn c3 and ynh84t c4 on 2026-11-20: crab's own checks, open after the
// release, left as shipped-not-done): the release step hands it to the
// task's owner as active work, with a note naming the release, the check
// and the owner. The task stays doing/active; lint reads the note and asks
// nothing (doing-idle still asks if nobody works it). The owner's tick of
// the check runs the step again (AfterTick): the next person check is
// asked, or the task closes when every check holds.

// handedOnLine is the note handOn writes: "handed on after v0.7.87: check 3
// open, crab's active work".
var handedOnLine = regexp.MustCompile(`(?m)^handed on after (\S+): check (\d+) open, `)

// HandedOnNote is that note.
func HandedOnNote(tag string, n int, owner string) string {
	return "handed on after " + tag + ": check " + strconv.Itoa(n) + " open, " + owner + "'s active work"
}

// handedOn reports whether notes hand the task on after tag.
func handedOn(notes, tag string) bool {
	for _, m := range handedOnLine.FindAllStringSubmatch(notes, -1) {
		if m[1] == tag {
			return true
		}
	}
	return false
}

// newestShipped is the release the newest shipped note in notes names.
func newestShipped(notes string) string {
	m := shippedLine.FindAllStringSubmatch(notes, -1)
	if len(m) == 0 {
		return ""
	}
	return m[len(m)-1][1]
}

// handOn hands id, shipped in tag, to its owner (the assignee, else the
// board's coordinator): a doing/active task with an open check that is
// not a person's. It returns the check and the owner, n 0 for nothing
// done. Once per tag.
func (s *Store) handOn(id, tag string) (n int, owner string, err error) {
	d, _, e := s.load(id)
	if e != nil || d.Task.State != "doing" || d.Task.Stage != "active" || handedOn(d.SectionText("Notes"), tag) {
		return 0, "", nil
	}
	people, _ := People(s.Root)
	for i, a := range d.Task.Acceptance {
		if a.Settled() {
			continue
		}
		if _, is := AddressesPerson(people, a.Text, len(d.Task.Acceptance)); !is {
			n = i + 1
			break
		}
	}
	if n == 0 {
		return 0, "", nil
	}
	if len(d.Task.Assignee) > 0 {
		owner = strings.Join(d.Task.Assignee, ", ")
	} else if owner, _ = Coordinator(s.Root); owner == "" {
		return 0, "", nil
	}
	d.Note(HandedOnNote(tag, n, owner))
	if _, e := s.finish(d, true); e != nil {
		return 0, "", e
	}
	return n, owner, nil
}

// afterShipped is the release step for id, shipped in tag, run when the
// release names it, and again when what held it back goes (a dependency
// done, yf39ya check 9): close it when every check holds, else ask the
// person when every open check is theirs, else hand the open check to its
// owner. It reports what it did ("" nothing) and the fix that undoes it.
func (s *Store) afterShipped(id, tag string) (did, fix string, err error) {
	closed, e := s.closeShipped(id, tag)
	if e != nil {
		return "", "", e
	}
	if closed {
		return "done (every check holds)", "hug task set " + id + " --state todo", nil
	}
	if q, ok, e := s.askAfterRelease(id); e != nil {
		return "", "", e
	} else if ok {
		return "asked " + q.For + " about check " + strconv.Itoa(q.Check), "hug task resume " + id, nil
	}
	n, owner, e := s.handOn(id, tag)
	if e != nil || n == 0 {
		return "", "", e
	}
	return "handed on: check " + strconv.Itoa(n) + " open, " + owner + "'s active work", "hug task check " + id + " " + strconv.Itoa(n), nil
}

// AfterTick runs after a tick of check n on id by who (the CLI and MCP
// check, the TUI's Space). answered: the tick answered a question asked of
// who (qsqhpe). On a shipped doing task, after an answer or on a task the
// release handed on (yf39ya check 8: the owner's tick ends the work it was
// handed), it asks the next person check or closes the task when every
// check and child holds (AfterAnswer). It reports what it did.
func (s *Store) AfterTick(id string, n int, who string, answered bool) (string, error) {
	if !answered {
		d, _, e := s.load(id)
		if e != nil {
			return "", nil
		}
		notes := d.SectionText("Notes")
		if tag := newestShipped(notes); tag == "" || !handedOn(notes, tag) {
			return "", nil
		}
	}
	return s.AfterAnswer(id, n, who)
}
