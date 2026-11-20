package task

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Several questions on one task, each answered on its own (q2adr8, Jørgen
// on rxr4z8 2026-11-18 19:27Z: "If you ask three questions, I should have
// three questions to answer in the ticket"). Every hug task ask adds a
// question to the "## Questions" section, numbered once from 1 and never
// reused; its answer is written under it:
//
//	## Questions
//	- 2026-11-18T19:50 question 1 by nautilus to jhf (check 1, show mock.md):
//	  Is the mark sign right?
//	  answered 2026-11-18T20:01 by jhf:
//	    keep the diamond
//	- 2026-11-18T19:51 question 2 by nautilus to jhf (check 3):
//	  Keep folding on grouped headings?
//
// The task stays doing/asked while any question is open. The stage's
// asked_* fields mirror the oldest open question (crab 19:49), so every
// reader of the stage (the TUI's ASKED rows, lint, upgrade, MCP) keeps
// working; answering it moves them to the next open one. History keeps one
// answer block per answer, its head naming the question:
// "answered asked (question 2, check 3), by jhf, for nautilus (...)".
//
// A task asked before q2adr8 has asked_* and no Questions section: it is
// one open question with no number, answered as before.

// QuestionsSection is the heading the questions live under.
const QuestionsSection = "Questions"

// Question is one question asked on a task.
type AskedQuestion struct {
	N     int    `json:"n"`
	By    string `json:"by"`
	To    string `json:"to"`
	At    string `json:"at"`
	Check int    `json:"check,omitempty"`
	Show  string `json:"show,omitempty"`
	Text  string `json:"text"`
	// Answer is its answer, nil while it is open.
	Answer *Resolution `json:"answer,omitempty"`
	// Withdrawn is when the task left asked with it unanswered (resume,
	// wait, done, drop), and the move: it no longer waits.
	Withdrawn *Resolution `json:"withdrawn,omitempty"`
}

// Open says whether the question waits for its answer.
func (q AskedQuestion) Open() bool { return q.Answer == nil && q.Withdrawn == nil }

var (
	questionHead = regexp.MustCompile(`^- (\S+) question (\d+) by (.+?) to (.+?)(?: \(([^()]*)\))?:$`)
	answerUnder  = regexp.MustCompile(`^  answered (\S+) by (.+):$`)
	// withdrawnUnder closes a question nobody answered: the task left
	// asked ("  withdrawn 2026-11-18T20:10 (doing/asked -> doing/active)").
	withdrawnUnder = regexp.MustCompile(`^  withdrawn (\S+) \((.*)\)$`)
	// closedUnder is either: the end of a question's own text.
	closedUnder = regexp.MustCompile(`^  (?:answered \S+ by .+:|withdrawn \S+ \(.*\))$`)
)

// parseQuestions reads the blocks of the Questions section's lines.
func parseQuestions(lines []string) []AskedQuestion {
	var out []AskedQuestion
	for i := 0; i < len(lines); i++ {
		m := questionHead.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[2])
		q := AskedQuestion{N: n, At: m[1], By: m[3], To: m[4]}
		for _, part := range strings.Split(m[5], ", ") {
			if c, ok := strings.CutPrefix(part, "check "); ok {
				q.Check, _ = strconv.Atoi(c)
			} else if s, ok := strings.CutPrefix(part, "show "); ok {
				q.Show = s
			}
		}
		var text, answer []string
		for i+1 < len(lines) {
			l := strings.TrimRight(lines[i+1], "\r\n")
			if q.Answer == nil && q.Withdrawn == nil {
				if wm := withdrawnUnder.FindStringSubmatch(l); wm != nil {
					q.Withdrawn = &Resolution{At: wm[1], Text: wm[2]}
					i++
					continue
				}
				if am := answerUnder.FindStringSubmatch(l); am != nil {
					q.Answer = &Resolution{At: am[1], By: am[2]}
					i++
					continue
				}
				if rest, ok := strings.CutPrefix(l, "  "); ok {
					text = append(text, rest)
					i++
					continue
				}
				break
			}
			if q.Answer == nil {
				break
			}
			if rest, ok := strings.CutPrefix(l, "    "); ok {
				answer = append(answer, rest)
				i++
				continue
			}
			if (l == "" || l == "  ") && i+2 < len(lines) && strings.HasPrefix(lines[i+2], "    ") {
				answer = append(answer, "")
				i++
				continue
			}
			break
		}
		q.Text = strings.Join(text, "\n")
		if q.Answer != nil {
			q.Answer.Text = strings.Join(answer, "\n")
		}
		out = append(out, q)
	}
	return out
}

// questionLines are the lines of the Questions section, heading excluded.
func (d *Document) questionLines() []string {
	s, ok := d.sections[QuestionsSection]
	if !ok {
		return nil
	}
	return d.lines[s[0]+1 : s[1]]
}

// OpenQuestions are t's questions with no answer yet, oldest first.
func OpenQuestions(t Task) []AskedQuestion {
	var out []AskedQuestion
	for _, q := range t.Questions {
		if q.Open() {
			out = append(out, q)
		}
	}
	return out
}

// HasOpenQuestion reports whether t waits on a question to someone: an
// open question in ## Questions, or the asked stage of a task asked before
// q2adr8 (asked_* set, no section).
func HasOpenQuestion(t Task) bool {
	if t.State != "todo" && t.State != "doing" {
		return false
	}
	if len(OpenQuestions(t)) > 0 {
		return true
	}
	_, ok := legacyQuestion(t)
	return ok
}

// openQuestionRefusal is done's refusal of t while a question on it is
// open (72n8m4): the answer is still owed, and done would withdraw it as
// a side effect. The fix withdraws it on purpose (hug task resume).
func openQuestionRefusal(t Task) error {
	if !HasOpenQuestion(t) {
		return nil
	}
	q, _ := legacyQuestion(t)
	if open := OpenQuestions(t); len(open) > 0 {
		q = open[0]
	}
	which := "a question"
	if q.N > 0 && len(t.Questions) > 0 {
		which = "question " + strconv.Itoa(q.N)
	}
	if n := len(OpenQuestions(t)); n > 1 {
		which += " and " + strconv.Itoa(n-1) + " more"
	}
	to := strings.TrimSpace(q.To)
	if to == "" {
		to = "someone"
	}
	return &Error{Kind: "open-question", ID: t.ID,
		Value: t.ID + " has " + which + " open to " + to + " (" + strconv.Quote(oneLineFix(q.Text)) + "): done waits for the answer; to close it without one, withdraw the question first",
		Fix:   "hug task resume " + t.ID}
}

// nextQuestion is the number d's next question gets: never a reused one.
func nextQuestion(d *Document) int {
	n := 1
	for _, q := range d.Task.Questions {
		if q.N >= n {
			n = q.N + 1
		}
	}
	return n
}

// questionBlock is a question's block in the Questions section.
func questionBlock(n int, by, to string, check int, show, text, nl string) string {
	return questionHeadAt(stamp(), n, by, to, check, show) + nl + indent(text, "  ", nl)
}

// questionHeadAt is a question's head line, asked at at.
func questionHeadAt(at string, n int, by, to string, check int, show string) string {
	var meta []string
	if check > 0 {
		meta = append(meta, "check "+strconv.Itoa(check))
	}
	if show != "" {
		meta = append(meta, "show "+show)
	}
	head := "- " + at + " question " + strconv.Itoa(n) + " by " + by + " to " + to
	if len(meta) > 0 {
		head += " (" + strings.Join(meta, ", ") + ")"
	}
	return head + ":"
}

// mirrorOn is mirror for a task in state: a todo task's open questions
// (yphbra: asked without taking it) show in its design_* fields, which
// have no check (the check is in ## Questions); a doing task's in asked_*.
func mirrorOn(state string, q AskedQuestion) (set map[string]string, clear []string) {
	set, clear = mirror(q)
	if state != "todo" {
		return set, clear
	}
	design := map[string]string{"asked_what": "design_question", "asked_for": "design_for", "asked_since": "design_since", "asked_by": "design_asked_by", "asked_show": "design_show"}
	dset := map[string]string{}
	for k, v := range set {
		if d, ok := design[k]; ok {
			dset[d] = v
		}
	}
	var dclear []string
	for _, k := range clear {
		if d, ok := design[k]; ok {
			dclear = append(dclear, d)
		}
	}
	return dset, dclear
}

// askStage is the stage a question puts a task in state in: asked on a
// doing task, design on a todo one (yphbra).
func askStage(state string) string {
	if state == "todo" {
		return "design"
	}
	return "asked"
}

// mirror is the asked_* fields for q, the oldest open question (crab
// 19:49): what the stage shows. since is when q was asked.
func mirror(q AskedQuestion) (set map[string]string, clear []string) {
	set = map[string]string{"asked_what": q.Text, "asked_for": q.To, "asked_since": q.At}
	if q.By != "" {
		set["asked_by"] = q.By
	}
	for k, v := range map[string]string{"asked_check": checkText(q.Check), "asked_show": q.Show} {
		if v != "" {
			set[k] = v
		} else {
			clear = append(clear, k)
		}
	}
	return set, clear
}

func checkText(n int) string {
	if n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// legacyQuestion is a task asked before q2adr8: asked_* set, no Questions
// section. It is adopted as question 1 when a second question is asked,
// so both are numbered.
func legacyQuestion(t Task) (AskedQuestion, bool) {
	f := t.Fields
	// A todo task's hug task design question is adopted the same way when
	// someone asks on it (yphbra).
	if len(t.Questions) == 0 && t.State == "todo" && ownHolds(t, "design") {
		return AskedQuestion{N: 1, By: f["design_asked_by"], To: f["design_for"], At: f["design_since"], Show: f["design_show"], Text: f["design_question"]}, true
	}
	if len(t.Questions) > 0 || t.State != "doing" || !ownHolds(t, "asked") {
		return AskedQuestion{}, false
	}
	c, _ := strconv.Atoi(strings.TrimSpace(f["asked_check"]))
	return AskedQuestion{N: 1, By: f["asked_by"], To: f["asked_for"], At: f["asked_since"], Check: c, Show: f["asked_show"], Text: f["asked_what"]}, true
}

// pickQuestion is the open question an answer is for: n if n > 0, the only
// open one otherwise. ok false (and the error) when n names no open
// question, or n is 0 and several are open.
func pickQuestion(t Task, n int, text string) (AskedQuestion, error) {
	open := OpenQuestions(t)
	var openNs, all []string
	for _, q := range open {
		openNs = append(openNs, strconv.Itoa(q.N))
	}
	if n == 0 {
		if len(open) == 1 {
			return open[0], nil
		}
		return AskedQuestion{}, usageErr(t.ID, fmt.Sprintf("%d questions are open on %s (%s): say which one this answers", len(open), t.ID, strings.Join(openNs, ", ")),
			"hug task answer "+t.ID+" "+openNs[0]+" "+strconv.Quote(oneLineFix(text)))
	}
	for _, q := range t.Questions {
		all = append(all, strconv.Itoa(q.N))
		if q.N != n {
			continue
		}
		switch {
		case q.Answer != nil:
			return AskedQuestion{}, usageErr(t.ID, "question "+strconv.Itoa(n)+" on "+t.ID+" is already answered, by "+q.Answer.By+" at "+q.Answer.At, "hug task show "+t.ID)
		case q.Withdrawn != nil:
			// 5vehza: a withdrawn question has no answer to read (it
			// panicked here); it waits for nothing, so ask it again.
			return AskedQuestion{}, usageErr(t.ID, "question "+strconv.Itoa(n)+" on "+t.ID+" was withdrawn at "+strings.Replace(q.Withdrawn.At, "T", " ", 1)+" ("+q.Withdrawn.Text+")", PersonWait{For: q.To, What: q.Text, Check: q.Check}.AskCommand(t.ID))
		}
		return q, nil
	}
	problem := t.ID + " has no questions"
	if len(all) > 0 {
		problem = t.ID + " has no question " + strconv.Itoa(n) + "; its questions are " + strings.Join(all, ", ") + " (open: " + orNone(openNs) + ")"
	}
	return AskedQuestion{}, usageErr(t.ID, problem, "hug task show "+t.ID)
}

// tickAnswers is the question a tick of check n by actor answers: the
// open question about check n asked of actor (qsqhpe; q2adr8: only that
// one, the others stay open). q is its number, 0 for a task asked before
// q2adr8.
func tickAnswers(t Task, n int, actor string) (q int, ok bool) {
	actor = strings.TrimSpace(actor)
	stage, _, _ := question(t)
	if (stage != "asked" && stage != "design") || actor == "" {
		return 0, false
	}
	// A design question names no check; one asked on a todo task (yphbra)
	// is in ## Questions with its check.
	if stage == "design" && len(t.Questions) == 0 {
		return 0, false
	}
	if len(t.Questions) == 0 {
		f := t.Fields
		return 0, strings.EqualFold(strings.TrimSpace(f["asked_for"]), actor) && strings.TrimSpace(f["asked_check"]) == strconv.Itoa(n)
	}
	for _, aq := range OpenQuestions(t) {
		if aq.Check == n && strings.EqualFold(strings.TrimSpace(aq.To), actor) {
			return aq.N, true
		}
	}
	return 0, false
}

// rewriteQuestionHead points open question q at who, check and show, its
// number, asker, stamp and text kept: a correction, not a new question.
func (d *Document) rewriteQuestionHead(q AskedQuestion, who string, check int, show string) error {
	sec, ok := d.sections[QuestionsSection]
	if !ok {
		return &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint " + d.Task.ID}
	}
	lines := append([]string(nil), d.lines...)
	for i := sec[0] + 1; i < sec[1]; i++ {
		m := questionHead.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if m == nil || m[2] != strconv.Itoa(q.N) {
			continue
		}
		lines[i] = questionHeadAt(q.At, q.N, q.By, who, check, show) + d.newline
		d.replace(lines)
		return nil
	}
	return &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint " + d.Task.ID}
}

// writeAnswerUnder puts the answer block under question n in the
// Questions section.
func (d *Document) writeAnswerUnder(n int, text, by string) error {
	lines := append([]string(nil), d.lines...)
	sec, ok := d.sections[QuestionsSection]
	if !ok {
		return &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint " + d.Task.ID}
	}
	at := -1
	for i := sec[0] + 1; i < sec[1]; i++ {
		m := questionHead.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if m != nil && m[2] == strconv.Itoa(n) {
			at = i + 1
			for at < sec[1] && strings.HasPrefix(lines[at], "  ") {
				at++
			}
			break
		}
	}
	if at < 0 {
		return &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint " + d.Task.ID}
	}
	block := "  answered " + stamp() + " by " + by + ":" + d.newline + indent(text, "    ", d.newline)
	lines = append(lines[:at], append([]string{block}, lines[at:]...)...)
	d.replace(lines)
	return nil
}

// withdrawOpen closes d's open questions when the task has left asked
// (resume, wait, done, drop): nobody is waiting on them, so a later ask
// must not bring them back. moved is the move, "doing/asked ->
// doing/active".
func withdrawOpen(d *Document, moved string) {
	if ownHolds(d.Task, "asked") && d.Task.State == "doing" {
		return
	}
	// A todo task asked without taking it (yphbra) keeps its questions
	// open while design_* mirror them, whatever stage outranks design.
	if ownHolds(d.Task, "design") && d.Task.State == "todo" {
		return
	}
	for _, q := range OpenQuestions(d.Task) {
		lines := append([]string(nil), d.lines...)
		sec, ok := d.sections[QuestionsSection]
		if !ok {
			return
		}
		for i := sec[0] + 1; i < sec[1]; i++ {
			m := questionHead.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
			if m == nil || m[2] != strconv.Itoa(q.N) {
				continue
			}
			at := i + 1
			for at < sec[1] && strings.HasPrefix(lines[at], "  ") {
				at++
			}
			lines = append(lines[:at], append([]string{"  withdrawn " + stamp() + " (" + moved + ")" + d.newline}, lines[at:]...)...)
			d.replace(lines)
			break
		}
	}
}
