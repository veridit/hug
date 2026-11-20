package task

import (
	"regexp"
	"strconv"
	"strings"
)

// Answering a question in place (3jtfw9, crab 13:26). A question to a person
// is a stage's field: design_question (todo/design), asked_what
// (doing/asked, 41hhgn) or review_by (review). Answering writes no new
// field: the stage's fields go to History in one block that holds the
// question, the answer, who answered and for whom, and the stage
// re-derives (design -> ready, asked -> active, review -> ready or active;
// done stays its own step). Both texts are kept verbatim, line for line
// (41hhgn check 8, Jørgen 23:50: answers can be several lines): the
// question's lines follow "> ", the answer's are indented two spaces.
//
//	- 2026-11-17T14:30 answered asked (check 9), by jhf, for nautilus (doing/asked -> doing/active):
//	  > type stays, or "drop"?
//	  drop it: the column says it already
//	  and the filter keeps it
//
// Before 41hhgn the block was one line with both texts quoted
// ('answered design: "q" -> "a", by ...'); it still reads.
//
// The asker sees the newest answer first (show, next --mine, the recall
// board cue) until it writes anything on the task after it; that write
// appends "NAME read the answer: WHAT", which counts as seen, WHAT being the
// write itself (41hhgn check 11: the answer's outcome, which the board's
// coordinator hears once). Older lines without ": WHAT" still read.

// Answer is the newest answer on a task its asker has not seen yet, read
// from History when the file is parsed. Derived, never written as a field.
type Answer struct {
	Stage    string `json:"stage"`
	Check    int    `json:"check,omitempty"`
	Question string `json:"question"`
	// QuestionN is the numbered question it answers (q2adr8), 0 for one
	// asked before questions were numbered.
	QuestionN int    `json:"question_n,omitempty"`
	Text      string `json:"text"`
	By        string `json:"by"`
	For       string `json:"for,omitempty"`
	At        string `json:"at"`
	// n is the block's number in History ("answer n").
	n int
}

var (
	answeredLine = regexp.MustCompile(`^- (\S+) answered (\w+): ("(?:[^"\\]|\\.)*") -> ("(?:[^"\\]|\\.)*"), by (.+?)(?:, for (.+?))? \(`)
	// answeredHead starts a block answer (41hhgn): the texts follow on
	// "  > " (question) and "  " (answer) lines.
	// q2adr8: "(question N, check C)", "(question N)" or "(check C)";
	// the question number is the last group, so the others keep theirs.
	answeredHead      = regexp.MustCompile(`^- (\S+) answered (\w+)(?: \((?:question \d+, )?check (\d+)\)| \(question \d+\))?, by (.+?)(?:, for (.+?))? \([^()]*\):$`)
	answeredQuestionN = regexp.MustCompile(`^- \S+ answered \w+ \(question (\d+)`)
	readLine          = regexp.MustCompile(`^- (\S+) (.+?) read the answer(?:: (.*))?$`)
)

// unseenAnswer is the newest answered line in history whose asker has not
// written on the task since, or nil. An answer the asker gave themselves is
// seen.
func unseenAnswer(history []string) *Answer {
	a, _ := readAnswers(history)
	return a
}

// Outcome is what the asker did first after reading an answer (41hhgn
// check 11): the answer's stage and who gave it (By) for whom (For), and
// the asker's write (Text, at At).
type Outcome struct {
	Stage string `json:"stage"`
	By    string `json:"by"`
	For   string `json:"for"`
	At    string `json:"at"`
	Text  string `json:"text"`
	// N is the answer's number ("answer N"), the item "outcome N" hug
	// hands to the coordinator (41hhgn check 11); Check the check the
	// question was about; Answer the answer's text.
	N      int    `json:"n,omitempty"`
	Check  int    `json:"check,omitempty"`
	Answer string `json:"answer,omitempty"`
	// Receipt is how far the outcome has reached the coordinator.
	Receipt *Receipt `json:"receipt,omitempty"`
}

// readAnswers is unseenAnswer, and the newest answer's outcome, if its
// asker has acted on it.
func readAnswers(history []string) (*Answer, *Outcome) {
	cur, outcome, _ := scanAnswers(history)
	return cur, outcome
}

// scanAnswers is readAnswers, and the newest answer block seen or not
// (gmzd87: what an answer's delivery says).
func scanAnswers(history []string) (cur *Answer, outcome *Outcome, last *Answer) {
	seen := func() {
		if cur != nil && (cur.For == "" || strings.EqualFold(cur.For, cur.By)) {
			cur = nil
		}
	}
	n := 0 // answer blocks so far, counted as answerCount does: cur is "answer n"
	for i := 0; i < len(history); i++ {
		line := strings.TrimRight(history[i], "\r\n")
		if answerHeadAny.MatchString(line) {
			n++
		}
		if m := answeredHead.FindStringSubmatch(line); m != nil {
			c, _ := strconv.Atoi(m[3])
			a := &Answer{At: m[1], Stage: m[2], Check: c, By: m[4], For: m[5], n: n}
			if qm := answeredQuestionN.FindStringSubmatch(line); qm != nil {
				a.QuestionN, _ = strconv.Atoi(qm[1])
			}
			var q, t []string
			for i+1 < len(history) && strings.HasPrefix(history[i+1], "  ") {
				i++
				l := strings.TrimRight(history[i], "\r\n")[2:]
				if rest, ok := strings.CutPrefix(l, "> "); ok && len(t) == 0 {
					q = append(q, rest)
				} else if l == ">" && len(t) == 0 {
					q = append(q, "")
				} else {
					t = append(t, l)
				}
			}
			a.Question, a.Text = strings.Join(q, "\n"), strings.Join(t, "\n")
			cur, last = a, a
			seen()
			continue
		}
		if m := answeredLine.FindStringSubmatch(line); m != nil {
			q, e1 := strconv.Unquote(m[3])
			a, e2 := strconv.Unquote(m[4])
			if e1 != nil || e2 != nil {
				continue
			}
			cur = &Answer{At: m[1], Stage: m[2], Question: q, Text: a, By: m[5], For: m[6], n: n}
			last = cur
			seen()
			continue
		}
		if m := readLine.FindStringSubmatch(line); m != nil && cur != nil && strings.EqualFold(m[2], cur.For) {
			outcome = nil
			if m[3] != "" {
				outcome = &Outcome{Stage: cur.Stage, By: cur.By, For: cur.For, At: m[1], Text: m[3], N: cur.n, Check: cur.Check, Answer: cur.Text}
			}
			cur = nil
		}
	}
	return cur, outcome, last
}

// answerBlock is the History block of an answer: the head (without the
// "- " and time), then the question's lines after "> " and the answer's
// indented, each line verbatim.
func answerBlock(head, question, text, nl string) string {
	var b strings.Builder
	b.WriteString(head)
	b.WriteString(":")
	for _, l := range strings.Split(question, "\n") {
		b.WriteString(nl + "  >")
		if l = strings.TrimRight(l, "\r"); l != "" {
			b.WriteString(" " + l)
		}
	}
	for _, l := range strings.Split(text, "\n") {
		b.WriteString(nl + "  " + strings.TrimRight(l, "\r"))
	}
	return b.String()
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
// design question, the asked question, or the review. A wait asks nobody
// (41hhgn): it ends with hug task resume.
//
// An open question in ## Questions asks too when its mirror is gone
// (m5pbb0): a hug before q2adr8 (v0.7.69, left running in a TUI) answered
// question 1 of p87ngv into History only and cleared design_*, so three
// questions stayed open with no stage to answer them by, and every answer
// was refused. The stage is the one an ask puts the task in (askStage);
// answering one puts the mirror back on the oldest still open.
func question(t Task) (stage, q, asker string) {
	switch {
	case t.State == "todo" && ownHolds(t, "design"):
		return "design", t.Fields["design_question"], t.Fields["design_asked_by"]
	case t.State == "doing" && ownHolds(t, "asked"):
		return "asked", t.Fields["asked_what"], t.Fields["asked_by"]
	case ownHolds(t, "review"):
		owner := strings.Join(t.Assignee, ", ")
		return "review", "review by " + t.Fields["review_by"], owner
	}
	if open := OpenQuestions(t); len(open) > 0 && (t.State == "todo" || t.State == "doing") {
		return askStage(t.State), open[0].Text, open[0].By
	}
	return "", "", ""
}

// Answer answers the question ref's stage asks (design, asked or review)
// with text, as by: the stage's fields go to History in one answer block,
// both texts verbatim (text may span lines); the stage re-derives.
func (s *Store) Answer(ref, text, by string) (Result, error) {
	return s.AnswerQuestion(ref, 0, text, by)
}

// AnswerQuestion is Answer for question n of an asked task (q2adr8): n 0
// is the only open question; with several open, n is required.
func (s *Store) AnswerQuestion(ref string, n int, text, by string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	text, by = strings.Trim(strings.ReplaceAll(text, "\r\n", "\n"), " \t\n"), strings.TrimSpace(by)
	stage, _, _ := question(d.Task)
	if stage == "" {
		fix := "hug task show " + d.Task.ID
		if d.Task.State == "doing" && ownHolds(d.Task, "waiting") {
			fix = "hug task resume " + d.Task.ID
		}
		return Result{}, usageErr(d.Task.ID, d.Task.ID+" asks nobody: no open question on "+d.Task.ID+", its stage is not design, asked or review", fix)
	}
	if text == "" {
		return Result{}, usageErr(d.Task.ID, "answer needs the answer", "hug task answer "+d.Task.ID+` "the answer"`)
	}
	if by == "" {
		return Result{}, usageErr(d.Task.ID, "answer needs who answers: --as NAME, or HUG_USER", "hug task answer "+d.Task.ID+" "+strconv.Quote(oneLineFix(text))+" --as NAME")
	}
	q := 0
	// A todo task asked without taking it (yphbra) numbers its questions
	// as a doing one does. A bare answer with no numbered question open
	// answers the stage's own question (khwzpt, crab: a design question
	// asked after the numbered ones were answered), as on a task with none.
	bare := n == 0 && len(OpenQuestions(d.Task)) == 0
	if (stage == "asked" || stage == "design") && len(d.Task.Questions) > 0 && !bare {
		picked, e := pickQuestion(d.Task, n, text)
		if e != nil {
			return Result{}, e
		}
		q = picked.N
	} else if n > 0 {
		return Result{}, usageErr(d.Task.ID, d.Task.ID+" has no numbered questions: answer it without a number", "hug task answer "+d.Task.ID+" "+strconv.Quote(oneLineFix(text)))
	}
	if e = answerOn(d, ts, q, text, by); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// answerOn writes the answer text by by to the question d's stage asks:
// one answer block in History, the stage's fields go, the stage
// re-derives. Answer and a tick by the person asked (qsqhpe) both use it.
// q is the numbered question it answers (q2adr8), 0 for none: its answer
// goes under it in ## Questions, and while another stays open the task
// stays asked, its fields on the oldest open one.
func answerOn(d *Document, ts []Task, q int, text, by string) error {
	stage, qtext, asker := question(d.Task)
	clear := stageFields(d.Task.State, stage)
	if stage == "review" {
		clear = []string{"review_by", "review_since", "review_show"}
	}
	check := strings.TrimSpace(d.Task.Fields["asked_check"])
	var set map[string]string
	if q > 0 {
		for _, aq := range d.Task.Questions {
			if aq.N == q {
				qtext, check = aq.Text, checkText(aq.Check)
				if aq.By != "" {
					asker = aq.By
				}
			}
		}
		if e := d.writeAnswerUnder(q, text, by); e != nil {
			return e
		}
		if open := OpenQuestions(d.Task); len(open) > 0 {
			set, clear = mirrorOn(d.Task.State, open[0])
		}
	}
	head := "answered " + stage
	var about []string
	// The number tells questions apart; a task asked once has nothing to
	// tell apart, and its History reads as before q2adr8.
	if q > 0 && len(d.Task.Questions) > 1 {
		about = append(about, "question "+strconv.Itoa(q))
	}
	if (stage == "asked" || stage == "design") && check != "" {
		about = append(about, "check "+check)
	}
	if len(about) > 0 {
		head += " (" + strings.Join(about, ", ") + ")"
	}
	head += ", by " + by
	if asker = strings.TrimSpace(asker); asker != "" {
		head += ", for " + asker
	}
	qtext = strings.ReplaceAll(qtext, "\r\n", "\n")
	return applyMove(d, ts, move{state: d.Task.State, set: set, clear: clear, block: func(moved string) string { return answerBlock(head+" ("+moved+")", qtext, text, d.newline) }})
}

// markRead appends "NAME read the answer" when actor, writing on d, is who
// its newest unseen answer is for (crab 13:26: any write by the asker after
// the answer counts as seen). It reports whether it wrote an outcome (a
// read line with the write in it, 41hhgn check 11).
func markRead(d *Document, actor, action string) bool {
	if actor = strings.TrimSpace(actor); actor == "" || !AnswerFor(d.Task, actor) {
		return false
	}
	line := stamp() + " " + actor + " read the answer"
	if action = strings.Join(strings.Fields(action), " "); action != "" {
		line += ": " + action
	}
	d.History(line)
	return action != ""
}
