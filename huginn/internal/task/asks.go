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
	var meta []string
	if check > 0 {
		meta = append(meta, "check "+strconv.Itoa(check))
	}
	if show != "" {
		meta = append(meta, "show "+show)
	}
	head := "- " + stamp() + " question " + strconv.Itoa(n) + " by " + by + " to " + to
	if len(meta) > 0 {
		head += " (" + strings.Join(meta, ", ") + ")"
	}
	return head + ":" + nl + indent(text, "  ", nl)
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
	if len(t.Questions) > 0 || t.State != "doing" || !ownHolds(t, "asked") {
		return AskedQuestion{}, false
	}
	f := t.Fields
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
		if !q.Open() {
			return AskedQuestion{}, usageErr(t.ID, "question "+strconv.Itoa(n)+" on "+t.ID+" is already answered, by "+q.Answer.By+" at "+q.Answer.At, "hug task show "+t.ID)
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
	if stage, _, _ := question(t); stage != "asked" || actor == "" {
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
