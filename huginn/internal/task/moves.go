package task

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/internal/sharefile"
)

// Moves (6ephqk check 3): the commands that change a task's state or stage
// write the node fields of where it goes, move the fields of where it was
// to ## History, and set the stage to the first that holds (Stage). Only the
// moves in Transitions are made; any other is refused with a fix: line that
// runs without an editor (crab 11:32).

// Now is the clock the moves stamp *_since and *_at fields with; tests set
// it.
var Now = time.Now

func stamp() string { return Now().Format("2006-01-02T15:04") }

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

// retagLastHistory replaces old with new in the last History line that
// holds old (yf39ya: a release's done says which release).
func (d *Document) retagLastHistory(old, new string) {
	s, ok := d.sections["History"]
	if !ok {
		return
	}
	lines := append([]string(nil), d.lines...)
	for i := s[1] - 1; i > s[0]; i-- {
		if strings.Contains(lines[i], old) {
			lines[i] = strings.Replace(lines[i], old, new, 1)
			d.replace(lines)
			return
		}
	}
}

// move is one change of state or stage: the state to go to, the fields
// to write there, and which node fields to drop (to History) though their
// node holds still (resume drops waiting_on).
type move struct {
	state string
	// stage, when set, is the stage the command puts the task in (wait,
	// ask, review, design): every other stage of the state is left, so its
	// fields go to History (3nhp67: a wait on an asked task kept asked_*).
	stage string
	set   map[string]string
	clear []string
	why   string // History line suffix: "take", "wait", "reopen" ...
	// line, when set, is the History line's text in place of the move
	// (3jtfw9: 'answered design: "q" -> "a", by NAME, for ASKER'); the
	// move follows in parentheses.
	line string
	// block, when set, makes the History entry: given the move
	// ("doing/asked -> doing/active"), it returns the entry, which may span
	// lines (41hhgn: an answer block).
	block func(moved string) string
}

// applyMove writes m on d: fields of nodes that no longer hold move to
// History with the time, m's fields are written, then the stage is set to
// the first that holds. ts is the board (dependencies' states).
func applyMove(d *Document, ts []Task, m move) error {
	from := d.Task.State
	if d.Task.Stage != "" {
		from += "/" + d.Task.Stage
	}
	if m.state != d.Task.State {
		if e := d.SetField("state", m.state); e != nil {
			return e
		}
	}
	for k, v := range m.set {
		if e := d.SetField(k, v); e != nil {
			return e
		}
	}
	owners := NodeFields()
	if m.stage != "" {
		m.clear = append(append([]string(nil), m.clear...), otherStageFields(m.state, m.stage)...)
	}
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
	to := d.Task.State
	if d.Task.Stage != "" {
		to += "/" + d.Task.Stage
	}
	// Questions left open when the task leaves asked are withdrawn
	// (q2adr8): no later ask brings them back.
	withdrawOpen(d, from+" -> "+to)
	if m.block != nil {
		d.History(stamp() + " " + m.block(from+" -> "+to))
		return nil
	}
	if m.line != "" {
		// The line already holds the moved fields' content (the question).
		d.History(stamp() + " " + m.line + " (" + from + " -> " + to + ")")
		return nil
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
	from := t.State
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
	from := t.State
	if t.Stage != "" {
		from += "/" + t.Stage
	}
	return &Error{Kind: "not-allowed", ID: t.ID, Value: from + " -> " + to + " is not a move a command makes (hug task states)", Fix: fix}
}

// Wait moves ref, a doing task, to doing/waiting on what: a thing outside
// the board with no person to answer (CI, a release). by is who waits
// (waiting_asked_by). A question to a person is Ask (41hhgn).
func (s *Store) Wait(ref, on, by string) (Result, error) { return s.WaitUntil(ref, on, "", by) }

// Question is what Ask asks: the text (verbatim, may span lines), who must
// answer, the check it is about (0: none) and a task file to show.
type Question struct {
	What, For, Show string
	Check           int
}

// Ask moves ref, a doing task, to doing/asked (41hhgn, Jørgen 23:48): its
// owner asked q.For something and can't go on until they answer. by is the
// asker (asked_by), told when it is answered. A check outside 1..N and a
// show that is not a task file are refused.
func (s *Store) Ask(ref string, q Question, by string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	what, who, check, show, e := s.askable(d, q)
	if e != nil {
		return Result{}, e
	}
	f := d.Task.Fields
	open := OpenQuestions(d.Task)
	// The same question again, still open, is no change; the same text to
	// another person, check or picture corrects that question in place
	// (lint's unknown-person fix re-asks it to the right name).
	for _, q := range open {
		if q.Text == what && q.To == who && checkText(q.Check) == check && q.Show == show {
			// Unchanged, unless the stage's fields went stale (written by
			// hand or an older hug): then they show the oldest open again.
			set, clear := mirrorOn(d.Task.State, open[0])
			stale := false
			for k, v := range set {
				stale = stale || d.Task.Fields[k] != v
			}
			for _, k := range clear {
				stale = stale || d.Task.Fields[k] != ""
			}
			if !stale {
				return s.finish(d, false)
			}
			m := askMove(d.Task.State, set, clear)
			m.line = "asked question " + strconv.Itoa(q.N) + " of " + who + " again"
			if e = applyMove(d, ts, m); e != nil {
				return Result{}, e
			}
			return s.finish(d, true)
		}
	}
	for _, q := range open {
		if q.Text != what {
			continue
		}
		c, _ := strconv.Atoi(check)
		if e = d.rewriteQuestionHead(q, who, c, show); e != nil {
			return Result{}, e
		}
		set, clear := mirrorOn(d.Task.State, OpenQuestions(d.Task)[0])
		m := askMove(d.Task.State, set, clear)
		m.line = "asked question " + strconv.Itoa(q.N) + " of " + who + " instead"
		if e = applyMove(d, ts, m); e != nil {
			return Result{}, e
		}
		return s.finish(d, true)
	}
	if len(d.Task.Questions) == 0 && f["asked_what"] == what && f["asked_for"] == who && f["asked_check"] == check && f["asked_show"] == show {
		return s.finish(d, false)
	}
	if by = strings.TrimSpace(by); by == "" {
		by = strings.TrimSpace(f["asked_by"])
	}
	asker := by
	if asker == "" {
		asker = strings.Join(d.Task.Assignee, ", ")
	}
	// A question asked before q2adr8 (asked_* and no Questions) becomes
	// question 1, so both are numbered and each is answered on its own;
	// the same text again replaces it, as before.
	if lq, ok := legacyQuestion(d.Task); ok && lq.Text != what {
		d.appendToSection(QuestionsSection, questionHeadAt(orStamp(lq.At), 1, lq.By, lq.To, lq.Check, lq.Show)+d.newline+indent(lq.Text, "  ", d.newline))
	}
	n := nextQuestion(d)
	c, _ := strconv.Atoi(check)
	d.appendToSection(QuestionsSection, questionBlock(n, asker, who, c, show, what, d.newline))
	// The stage shows the oldest open question (crab 19:49).
	set, clear := mirrorOn(d.Task.State, OpenQuestions(d.Task)[0])
	// Every ask is a History line. A first one moves the stage and names
	// what it left ("; was waiting_on: ..."); a second moves nothing, so
	// it says what it adds.
	m := askMove(d.Task.State, set, clear)
	if now := len(OpenQuestions(d.Task)); now > 1 {
		m.line = "asked question " + strconv.Itoa(n) + " of " + who + ", " + strconv.Itoa(now) + " open"
	}
	if e = applyMove(d, ts, m); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// askMove is the move an ask makes on a task in state: a doing task goes
// to doing/asked; a todo task (yphbra: asked without taking it) stays
// todo, nobody takes it, its dependencies stay, and its design_* fields
// mirror the oldest open question. Its stage derives as before: design,
// or depends, deferred or draft when one of those outranks it.
func askMove(state string, set map[string]string, clear []string) move {
	if state == "todo" {
		return move{state: "todo", set: set, clear: clear, why: "ask"}
	}
	return move{state: "doing", stage: "asked", set: set, clear: clear, why: "ask"}
}

// orStamp is at, or now when at is empty.
func orStamp(at string) string {
	if strings.TrimSpace(at) == "" {
		return stamp()
	}
	return at
}

// askable checks q on d, a task to ask: the question, who answers, a doing
// task, a check in 1..N, a show that is a task file. It returns them as
// written (check "" for none).
func (s *Store) askable(d *Document, q Question) (what, who, check, show string, e error) {
	id := d.Task.ID
	what = strings.TrimSpace(q.What)
	if what == "" {
		return "", "", "", "", usageErr(id, "ask needs the question", "hug task ask "+id+` --to NAME "the question"`)
	}
	who = strings.TrimSpace(q.For)
	if qs := severalQuestions(what); qs != nil {
		to := who
		if to == "" {
			to = "NAME"
		}
		return "", "", "", "", usageErr(id, fmt.Sprintf("that is %d questions in one ask: each gets its own answer, so ask each on its own", len(qs)), "ask each question on its own: hug task ask "+id+" --to "+to+" "+strconv.Quote(oneLineFix(qs[0])))
	}
	if who == "" {
		return "", "", "", "", usageErr(id, "ask needs who must answer: --to NAME", "hug task ask "+id+` --to NAME "`+oneLineFix(what)+`"`)
	}
	if d.Task.State != "doing" && d.Task.State != "todo" {
		return "", "", "", "", refuse(d.Task, "doing/asked", "hug task set "+id+" --state todo")
	}
	if who, e = s.answerer(id, who, what); e != nil {
		return "", "", "", "", e
	}
	if n := len(d.Task.Acceptance); q.Check != 0 && (q.Check < 1 || q.Check > n) {
		return "", "", "", "", &Error{Kind: "bad-check", ID: id, Value: fmt.Sprintf("--check %d: %s has %d checks", q.Check, id, n), Fix: fmt.Sprintf("hug task ask %s --to %s --check %d %q", id, who, max(1, n), oneLineFix(what))}
	}
	// A tick is the answer (qsqhpe): a ticked check is not asked about.
	if q.Check > 0 && d.Task.Acceptance[q.Check-1].Checked {
		fix := fmt.Sprintf("hug task set %s --accept %q", id, oneLineFix(what))
		for i, a := range d.Task.Acceptance {
			if !a.Checked && !a.Struck {
				fix = fmt.Sprintf("hug task ask %s --to %s --check %d %q", id, who, i+1, oneLineFix(what))
				break
			}
		}
		return "", "", "", "", &Error{Kind: "ticked-check", ID: id, Value: fmt.Sprintf("check %d is already ticked: ask about an open check, or add one with hug task set %s --accept ...", q.Check, id), Fix: fix}
	}
	if show, e = s.showable(d.Task, q.Show); e != nil {
		return "", "", "", "", e
	}
	if q.Check > 0 {
		check = strconv.Itoa(q.Check)
	}
	return what, who, check, show, nil
}

// oneLineFix is text for a fix: line: its first line.
func oneLineFix(text string) string {
	first, _, _ := strings.Cut(text, "\n")
	return strings.TrimSpace(first)
}

// numberedItem starts an item of a numbered list: "1.", "2)", "(3)",
// "Q4:" at a line start or after a space.
var numberedItem = regexp.MustCompile(`(?:^|\s)(?:\(\d{1,2}\)|\d{1,2}[.)]|[Qq]\d{1,2}:)\s`)

// severalQuestions splits what into the questions of a numbered list (q2adr8
// check 4, Jørgen: "If you ask three questions, I should have three
// questions to answer"): two or more numbered items that are each a
// question, ending in "?" or offering their own alternative (" or ", as
// rxr4z8's "(1) ◆ as the mark, or another sign; (2) ..."). A list of
// options under one question ("(1) the open ones, (2) the newest, or (3) by
// rank?") or of facts is one question, and so returns nil.
func severalQuestions(what string) []string {
	locs := numberedItem.FindAllStringIndex(what, -1)
	if len(locs) < 2 {
		return nil
	}
	var qs []string
	for i, l := range locs {
		end := len(what)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		item := strings.TrimSpace(what[l[1]:end])
		// The word joining one option to the next is not the item's own.
		body := strings.TrimRight(strings.TrimSuffix(strings.TrimRight(item, " ,;"), " or"), " ,;")
		if strings.HasSuffix(item, "?") || strings.Contains(" "+body+" ", " or ") {
			qs = append(qs, item)
		}
	}
	if len(qs) < 2 {
		return nil
	}
	return qs
}

// showable checks that name is one of t's files, as a question's picture
// (5z4zt4); "" is fine and means none.
func (s *Store) showable(t Task, name string) (string, error) {
	if name = strings.TrimSpace(name); name == "" {
		return "", nil
	}
	clean, e := cleanName(name)
	if e != nil {
		return "", e
	}
	p, e := s.diskPath(FolderOf(t) + "/" + clean)
	if e != nil {
		return "", e
	}
	if st, e := os.Stat(p); e != nil || !st.Mode().IsRegular() {
		return "", &Error{Kind: "file-not-found", ID: t.ID, Value: clean + " is not a file of " + t.ID + ", so the question cannot show it", Fix: "hug task attach " + t.ID + " FILE --name " + clean}
	}
	return clean, nil
}

// Resume ends a wait, or withdraws a question: the stage's fields go to
// History.
func (s *Store) Resume(ref string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if d.Task.State != "doing" || !(ownHolds(d.Task, "waiting") || ownHolds(d.Task, "asked")) {
		return Result{}, refuse(d.Task, "doing/active", "hug task show "+d.Task.ID)
	}
	// Resuming makes it active again: one active ticket per agent (n2hq8p).
	for _, who := range d.Task.Assignee {
		if a, ok := Active(who, d.Task.ID, ts); ok {
			return Result{}, busy(d.Task, Taker{Name: who}, a, false)
		}
	}
	// Once only (4tycwd): after the move the text is in ## History alone.
	on := strings.Join(strings.Fields(d.Task.Fields["waiting_on"]), " ")
	if e = applyMove(d, ts, move{state: "doing", clear: append(stageFields("doing", "waiting"), stageFields("doing", "asked")...), why: "resume"}); e != nil {
		return Result{}, e
	}
	r, e := s.finish(d, true)
	if e == nil && on != "" {
		r.Said = "was waiting on: " + on
	}
	return r, e
}

// Review asks who (a person's or agent's name, review_by, required) to
// review ref, a doing task (6w0gve, crab 18:30): doing/active becomes
// doing/review and the owner keeps it, so hug task answer brings it back
// to doing/active. show names the task file to look at (review_show), ""
// none. A review does not count against one doing ticket per agent.
func (s *Store) Review(ref, who, show string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if d.Task.State != "doing" {
		return Result{}, refuse(d.Task, "doing/review", "hug task take "+d.Task.ID+" --model MODEL")
	}
	if ownHolds(d.Task, "waiting") || ownHolds(d.Task, "asked") {
		return Result{}, refuse(d.Task, "doing/review", "hug task resume "+d.Task.ID)
	}
	if who = strings.TrimSpace(who); who == "" {
		return Result{}, usageErr(d.Task.ID, "review needs who reviews: --by NAME", "hug task review "+d.Task.ID+" --by NAME")
	}
	if show, e = s.showable(d.Task, show); e != nil {
		return Result{}, e
	}
	if d.Task.Fields["review_by"] == who && d.Task.Fields["review_show"] == show {
		return s.finish(d, false)
	}
	set := map[string]string{"review_by": who, "review_since": stamp()}
	var clear []string
	if show != "" {
		set["review_show"] = show
	} else {
		clear = []string{"review_show"}
	}
	if e = applyMove(d, ts, move{state: "doing", stage: "review", set: set, clear: clear, why: "review"}); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// Design asks question on ref, a todo task (todo/design), of who (a
// person's name, design_for, required: crab 12:38), recording who asked;
// clear (question "") answers it: the design fields go to History.
func (s *Store) Design(ref, question, by, who string) (Result, error) {
	return s.DesignShowing(ref, question, by, who, "")
}

// DesignShowing is Design naming the task file to look at (design_show,
// 5z4zt4); show must be one of the task's files, "" names none.
func (s *Store) DesignShowing(ref, question, by, who, show string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	if d.Task.State != "todo" {
		fix := "hug task set " + d.Task.ID + " --state todo"
		return Result{}, refuse(d.Task, "todo/design", fix)
	}
	question = strings.TrimSpace(question)
	if question == "" {
		if !ownHolds(d.Task, "design") {
			return s.finish(d, false)
		}
		if e = applyMove(d, ts, move{state: "todo", clear: []string{"design_question", "design_since", "design_asked_by", "design_for", "design_show"}, why: "design answered"}); e != nil {
			return Result{}, e
		}
		return s.finish(d, true)
	}
	if who = strings.TrimSpace(who); who == "" {
		return Result{}, usageErr(d.Task.ID, "design needs who must answer: --for NAME", "hug task design "+d.Task.ID+` "`+question+`" --for NAME`)
	}
	if show, e = s.showable(d.Task, show); e != nil {
		return Result{}, e
	}
	if d.Task.Fields["design_question"] == question && d.Task.Fields["design_for"] == who && d.Task.Fields["design_show"] == show {
		return s.finish(d, false)
	}
	set := map[string]string{"design_question": question, "design_since": stamp(), "design_for": who}
	if by = strings.TrimSpace(by); by != "" {
		set["design_asked_by"] = by
	}
	var clear []string
	if show != "" {
		set["design_show"] = show
	} else {
		clear = []string{"design_show"}
	}
	if e = applyMove(d, ts, move{state: "todo", stage: "design", set: set, clear: clear, why: "design"}); e != nil {
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
		if d.Task.State != "doing" {
			set["doing_since"] = stamp()
			why = "take"
			// reason is the task it is taken with (byj479).
			if reason != "" {
				set["doing_with"] = reason
				why = "take with " + reason
			}
			if d.Task.State == "todo" && ownHolds(d.Task, "review") && user != "" {
				set["review_by"] = user
				set["review_since"] = stamp()
				why = "take for review"
			}
			// Questions asked on the todo task (yphbra) come along, open:
			// the task is doing/asked and the taker sees what he waits on
			// (crab 21:59: a question is never lost).
			if open := OpenQuestions(d.Task); d.Task.State == "todo" && len(open) > 0 {
				askedSet, askedClear := mirror(open[0])
				for k, v := range askedSet {
					set[k] = v
				}
				return applyMove(d, ts, move{state: state, stage: "asked", set: set, clear: append(askedClear, stageFields("todo", "design")...), why: why + ", " + strconv.Itoa(len(open)) + " open question(s) kept"})
			}
		}
	case "todo":
		why = "give back"
		if d.Task.State == "done" || d.Task.State == "dropped" {
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
	if state == "todo" && d.Task.State == "doing" {
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
		if seen[t.ID] || !Staged(t) || t.State == "done" || t.State == "dropped" {
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
		if e = applyMove(d, ts, move{state: t.State, why: "dependency " + id}); e != nil {
			return touched, e
		}
		if _, e = s.finish(d, true); e != nil {
			return touched, e
		}
		touched = append(touched, fmt.Sprintf("%s %s", t.ID, d.Task.Stage))
		// Its dependency held it after a release (yf39ya check 9: 0gztsy
		// on kg07bt): now that it holds, the release step runs: close,
		// ask the person, or hand the open check to its owner.
		if tag := releaseOf(d); tag != "" && t.Stage == "depends" && d.Task.Stage != "depends" {
			did, _, e := s.afterShipped(t.ID, tag)
			if e != nil {
				return touched, e
			}
			if did != "" {
				touched[len(touched)-1] += ", after " + tag + ": " + did
			}
		}
	}
	return touched, nil
}

// RenameStatusKey turns the old "status: x" front-matter line into
// "state: x" in place (6ephqk step 4), keeping its value, quoting and
// position. It reports whether the file changed; a file that already says
// state: is left alone, and a stray status: next to it is dropped.
func (d *Document) RenameStatusKey() (bool, error) {
	n, ok := d.fields["status"]
	if !ok {
		return false, nil
	}
	if d.frontEnd < 0 || !d.frontmatterValid {
		return false, &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint"}
	}
	if _, both := d.fields["state"]; both {
		return true, d.RemoveField("status")
	}
	lines := append([]string(nil), d.lines...)
	lines[n] = "state" + strings.TrimPrefix(lines[n], "status")
	d.replace(lines)
	return true, nil
}

// MigrateState is hug upgrade's step for 6ephqk: every task file's
// status: becomes state:. It returns the changed paths and one line per
// change ("ID: status: doing -> state: doing").
func (s *Store) MigrateState(dryRun bool) (changed, lines []string, err error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, nil, e
	}
	for _, t := range ts {
		path, e := s.diskPath(t.Path)
		if e != nil {
			return changed, lines, e
		}
		raw, e := sharefile.ReadFile(path)
		if e != nil {
			return changed, lines, e
		}
		d := Parse(t.Path, raw)
		did, e := d.RenameStatusKey()
		if e != nil {
			return changed, lines, e
		}
		if !did {
			continue
		}
		changed = append(changed, t.Path)
		lines = append(lines, t.ID+": status: "+t.State+" -> state: "+d.Task.State)
		if dryRun {
			continue
		}
		s.Invalidate()
		if e = AtomicWrite(path, d.Bytes()); e != nil {
			return changed, lines, e
		}
	}
	return changed, lines, nil
}
