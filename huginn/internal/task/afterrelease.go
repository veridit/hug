package task

import (
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
