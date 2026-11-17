package task

import (
	"regexp"
	"strconv"
	"strings"
)

// Answering a question in place (3jtfw9, crab 13:26). A question to a person
// is a stage's field: design_question (todo/design), waiting_on (doing/waiting)
// or review_by (review). Answering writes no new field: the stage's fields go
// to History in one line that holds the question, the answer, who answered
// and for whom, and the stage re-derives (design -> ready, waiting -> active,
// review -> ready or active; done stays its own step):
//
//	- 2026-11-17T14:30 answered design: "type stays?" -> "drop it", by jhf, for nautilus (todo/design -> todo/ready)
//
// The asker sees the newest answer first (show, next --mine, the recall
// board cue) until it writes anything on the task after it; that write
// appends "NAME read the answer", which counts as seen.

// Answer is the newest answer on a task its asker has not seen yet, read
// from History when the file is parsed. Derived, never written as a field.
type Answer struct {
	Stage    string `json:"stage"`
	Question string `json:"question"`
	Text     string `json:"text"`
	By       string `json:"by"`
	For      string `json:"for,omitempty"`
	At       string `json:"at"`
}

var (
	answeredLine = regexp.MustCompile(`^- (\S+) answered (\w+): ("(?:[^"\\]|\\.)*") -> ("(?:[^"\\]|\\.)*"), by (.+?)(?:, for (.+?))? \(`)
	readLine     = regexp.MustCompile(`^- \S+ (.+) read the answer$`)
)

// unseenAnswer is the newest answered line in history whose asker has not
// written on the task since, or nil. An answer the asker gave themselves is
// seen.
func unseenAnswer(history []string) *Answer {
	var cur *Answer
	for _, line := range history {
		line = strings.TrimRight(line, "\r\n")
		if m := answeredLine.FindStringSubmatch(line); m != nil {
			q, e1 := strconv.Unquote(m[3])
			a, e2 := strconv.Unquote(m[4])
			if e1 != nil || e2 != nil {
				continue
			}
			cur = &Answer{At: m[1], Stage: m[2], Question: q, Text: a, By: m[5], For: m[6]}
			if cur.For == "" || strings.EqualFold(cur.For, cur.By) {
				cur = nil
			}
			continue
		}
		if m := readLine.FindStringSubmatch(line); m != nil && cur != nil && strings.EqualFold(m[1], cur.For) {
			cur = nil
		}
	}
	return cur
}

// historyLines are the lines of the ## History section.
func (d *Document) historyLines() []string {
	s, ok := d.sections["History"]
	if !ok {
		return nil
	}
	return d.lines[s[0]+1 : s[1]]
}

// AnswerFor says whether t has an answer who has not seen.
func AnswerFor(t Task, who string) bool {
	return t.Answer != nil && who != "" && strings.EqualFold(t.Answer.For, who)
}

// question is what the stage t is in asks, the stage, and who asked: the
// design question, what a wait is on, or the review.
func question(t Task) (stage, q, asker string) {
	switch {
	case t.Status == "todo" && ownHolds(t, "design"):
		return "design", t.Fields["design_question"], t.Fields["design_asked_by"]
	case t.Status == "doing" && ownHolds(t, "waiting"):
		return "waiting", t.Fields["waiting_on"], t.Fields["waiting_asked_by"]
	case ownHolds(t, "review"):
		owner := strings.Join(t.Assignee, ", ")
		return "review", "review by " + t.Fields["review_by"], owner
	}
	return "", "", ""
}

// Answer answers the question ref's stage asks (design, waiting or
// review) with text, as by. The stage's fields go to History in one
// answered line; the stage re-derives.
func (s *Store) Answer(ref, text, by string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	text, by = strings.TrimSpace(text), strings.TrimSpace(by)
	stage, q, asker := question(d.Task)
	if stage == "" {
		return Result{}, usageErr(d.Task.ID, d.Task.ID+" asks nothing: its stage is not design, waiting or review", "hug task show "+d.Task.ID)
	}
	if text == "" {
		return Result{}, usageErr(d.Task.ID, "answer needs the answer", "hug task answer "+d.Task.ID+` "the answer"`)
	}
	if by == "" {
		return Result{}, usageErr(d.Task.ID, "answer needs who answers: --as NAME, or HUG_USER", "hug task answer "+d.Task.ID+` "`+text+`" --as NAME`)
	}
	var clear []string
	switch stage {
	case "design":
		clear = []string{"design_question", "design_since", "design_asked_by", "design_for"}
	case "waiting":
		clear = []string{"waiting_on", "waiting_for", "waiting_since", "waiting_asked_by"}
	case "review":
		clear = []string{"review_by", "review_since"}
	}
	line := "answered " + stage + ": " + strconv.Quote(q) + " -> " + strconv.Quote(text) + ", by " + by
	if asker = strings.TrimSpace(asker); asker != "" {
		line += ", for " + asker
	}
	if e = applyMove(d, ts, move{state: d.Task.Status, clear: clear, line: line}); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// markRead appends "NAME read the answer" when actor, writing on d, is who
// its newest unseen answer is for (crab 13:26: any write by the asker after
// the answer counts as seen).
func markRead(d *Document, actor string) {
	if actor = strings.TrimSpace(actor); actor == "" || !AnswerFor(d.Task, actor) {
		return
	}
	d.History(stamp() + " " + actor + " read the answer")
}
