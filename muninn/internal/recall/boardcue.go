package recall

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Board nudges (qawmnb, 2026-11-16). The board is how the user and other
// agents see progress, so the recall hook says one short line when the
// session's own actions and the board disagree, with the exact command that
// makes them agree. Never a timer, never doctrine: a line only on a concrete
// mismatch, at most once per kind per BoardEvery, at most one line per turn.
//
//   - take:  the session changed code in its repository and no task is
//            doing for it (M1). With a task it named or one whose title
//            matches the work, the line names it; otherwise it waits for a
//            second turn of edits and points at hug task next / new.
//   - work:  the session checked or noted a task that is still todo (M3).
//   - done:  a task doing for the session has every check ticked (M2).
//   - request: a person's open request on a task (hbgybn), to its
//            assignee, else the board's coordinator, else every session;
//            said once per request, never held back by the hourly rate.
//   - answered: the board's coordinator hears, once, what an asker did
//            after a person's answer (41hhgn check 11): the asker's first
//            write on the task. Only with a coordinator set, never to the
//            asker itself.
//   - lint:  at a break only, one open finding with its fix, at most once per
//            LintEvery (the dream hand-out rule), preferring tasks the
//            session touched, then doing, then todo.
//
// "For the session" is: its name is an assignee, or it took the task itself
// (remembered here), or, in a linked worktree or on a non-main branch, the
// task was taken there (x1t1t1). Never the model alone: several sessions run
// the same model.

// BoardEvery limits each kind of board line per session.
var BoardEvery = time.Hour

// LintEvery limits lint lines per session (as CueEvery for dream items).
var LintEvery = 6 * time.Hour

// BoardTask is one task as the nudge sees it.
type BoardTask struct {
	ID, Title, State string
	Assignee         []string
	Branch, Worktree string
	Checks, Open     int
	// Answer is a person's answer to a question on the task that its
	// asker has not read yet (3jtfw9); nil when there is none.
	Answer *BoardAnswer
	// Requests are the task's open requests (hbgybn), oldest first.
	Requests []BoardRequest
	// Outcome is what the asker did first after the newest answer (41hhgn
	// check 11); nil before it acts.
	Outcome *BoardOutcome
	// Asked: a question on the task is open to someone (72n8m4). hug task
	// done refuses it, so the done line is not offered.
	Asked bool
}

// BoardOutcome is an answer's outcome: Stage and By of the answer, For the
// asker, At and Text its first write after reading it.
type BoardOutcome struct {
	Stage, By, For, At, Text string
}

// BoardRequest is one open request a person wrote on a task: N its number
// on the task, By who asked, At when, Text the request verbatim.
type BoardRequest struct {
	N            int
	By, At, Text string
}

// BoardAnswer is the newest unread answer on a task: Stage is design,
// waiting or review; For is the asker's name; At the History stamp.
type BoardAnswer struct {
	Stage, Question, Text, By, For, At string
}

// BoardFinding is one open lint finding with its repair command.
type BoardFinding struct {
	TaskID, TaskState, Problem, Fix string
}

// BoardSelf is who and where the session is.
type BoardSelf struct {
	Name, Model      string
	Branch, Worktree string
	Linked           bool
}

// BoardAction is one tool call of the session, reduced.
type BoardAction struct {
	Edits   bool   // changed a file in the worktree (or committed)
	Subject string // commit subject, if it committed
	Hug     []BoardHug
}

// BoardHug is one hug task command the session ran.
type BoardHug struct {
	Verb, ID string
	N        int
}

// BoardInput is the board and the session's recent actions.
type BoardInput struct {
	Tasks []BoardTask
	// Coordinator is the board's coordinator (hbgybn): who hears a request
	// on a task nobody holds; "" means every session.
	Coordinator string
	Findings    func() []BoardFinding
	Self        BoardSelf
	Actions     []BoardAction
	// Gone reports whether a harness session name has gone at now, by the
	// rule hug task lint's stale-owner uses (9ahvgt: its newest session
	// closed over an hour ago, or silent for a day, also under a "You are
	// NAME" alias), for the coordinator's checks-all-hold line (agh2h7);
	// known is false for a name the harness does not know (a person, a
	// model). Nil: no harness list, nobody counts as gone.
	Gone func(name string, now time.Time) (gone, known bool)
	// Reflections returns the open reflections' dates (YYYY-MM-DD), for
	// the coordinator's cue on old ones (rk5q7r). Read only at a break, for
	// the coordinator; nil: none.
	Reflections func() []string
}

// BoardSource yields the input for actions after since. It returns nil when
// there is no board (or nothing to read); then nothing is said. active
// reports whether the session did anything since; the source may skip
// reading the board when it did not and lint is not wanted.
type BoardSource func(since time.Time, wantLint bool) *BoardInput

// NoteTurn records a user turn and returns its break signal, as Recall does
// (dream.go). Offline replays (cmd/board4) use it with a past clock.
func (e *Engine) NoteTurn(session, msg string, now time.Time) string {
	return e.noteTurn(session, msg, now)
}

func (e *Engine) boardTables() {
	e.DB.Exec(`CREATE TABLE IF NOT EXISTS board_cues(session TEXT NOT NULL, kind TEXT NOT NULL, at_ns INTEGER NOT NULL, PRIMARY KEY(session, kind))`)
	e.DB.Exec(`CREATE TABLE IF NOT EXISTS board_seen(session TEXT PRIMARY KEY, seen_ns INTEGER NOT NULL, edit_turns INTEGER NOT NULL DEFAULT 0)`)
	e.DB.Exec(`CREATE TABLE IF NOT EXISTS board_taken(session TEXT NOT NULL, id TEXT NOT NULL, PRIMARY KEY(session, id))`)
	// Lint findings handed out, across sessions: one finding goes to one
	// agent, then the next (as dream items are handed out once).
	e.DB.Exec(`CREATE TABLE IF NOT EXISTS board_lint(key TEXT PRIMARY KEY, at_ns INTEGER NOT NULL)`)
}

// BoardCue returns the board line for this turn ("" for none) and its kind.
// brk is the break signal of the turn (start, return, wrap-up or "").
func (e *Engine) BoardCue(session, msg, brk string, now time.Time, src BoardSource) (line, kind string) {
	if session == "" || src == nil {
		return "", ""
	}
	// A turn that tells the agent to stop, or not to touch anything, gets
	// no board line at all (x325pp; held-out replay: hog was handed a lint
	// fix in the turn that said "STOP NOW ... do not run any command").
	if stopOrder.MatchString(msg) {
		return "", ""
	}
	e.boardTables()
	var seen, editTurns int64
	if e.DB.QueryRow(`SELECT seen_ns, edit_turns FROM board_seen WHERE session=?`, session).Scan(&seen, &editTurns) != nil {
		seen = now.Add(-30 * time.Minute).UnixNano()
	}
	wantLint := brk != "" && e.boardDue(session, "lint", LintEvery, now)
	// Old reflections have their own daily rate (rk5q7r): the board is read
	// at a break where either is due. Only the coordinator ever hears of
	// them; any other session marks the rate on reading, so it reads the
	// board at a break at most once a day more than lint asks.
	wantReflect := brk != "" && e.boardDue(session, "reflect", ReflectEvery, now)
	in := src(time.Unix(0, seen), wantLint || wantReflect)
	if in == nil {
		e.DB.Exec(`INSERT INTO board_seen(session, seen_ns, edit_turns) VALUES(?,?,0) ON CONFLICT(session) DO UPDATE SET seen_ns=excluded.seen_ns`, session, now.UnixNano())
		return "", ""
	}
	for _, a := range in.Actions {
		for _, h := range a.Hug {
			if h.Verb == "take" && h.ID != "" {
				e.DB.Exec(`INSERT OR IGNORE INTO board_taken(session, id) VALUES(?,?)`, session, h.ID)
			}
		}
	}
	d := e.detect(session, msg, in)
	if d.edits && !d.hasMine {
		editTurns++
	} else {
		editTurns = 0
	}
	e.DB.Exec(`INSERT INTO board_seen(session, seen_ns, edit_turns) VALUES(?,?,?) ON CONFLICT(session) DO UPDATE SET seen_ns=excluded.seen_ns, edit_turns=excluded.edit_turns`, session, now.UnixNano(), editTurns)
	for _, c := range d.candidates(editTurns, in.Self.Model) {
		// take and work ask for the same thing (hug task take): one hour
		// between them, not one each (dev replay: crab 13 Nov, 22:02 work
		// then 22:12 take on the same stretch).
		rate := c.kind
		if rate == "work" {
			rate = "take"
		}
		// An answer is said once (its key names the answer), never held
		// back by the hourly rate: a person is waiting on the agent.
		if (c.kind == "answer" || c.kind == "request" || c.kind == "answered") && e.boardDue(session, c.key(), RepeatEvery, now) {
			e.boardMark(session, c.key(), now)
			return c.line, c.kind
		}
		if c.kind != "answer" && c.kind != "request" && c.kind != "answered" && e.boardDue(session, rate, BoardEvery, now) && e.boardDue(session, c.key(), RepeatEvery, now) {
			e.boardMark(session, rate, now)
			e.boardMark(session, c.key(), now)
			return c.line, c.kind
		}
	}
	if wantLint && in.Findings != nil {
		// At the start of a session only findings on tasks it named: a
		// fresh assignment is not the moment for unrelated housekeeping.
		// An unrelated finding also waits while the session's own doing
		// task has one: that one is the session's to fix first, and a
		// missing section only ever goes to the task's session (ownerOnly,
		// tpnd3n).
		// (x325pp tried holding unrelated findings until the session had no
		// task of its own or had just closed one: held-out false lines 5 -> 0
		// but fresh lint lines 14 -> 8 and fresh2's 3 judged false;
		// muninn/docs/BOARD4.md. Not kept.)
		fs := in.Findings()
		unrelated := brk != "start" && !ownHasFinding(fs, d.own)
		if f, ok := e.pickFinding(fs, d.touched, d.mention, unrelated, now); ok {
			e.boardMark(session, "lint", now)
			e.DB.Exec(`INSERT INTO board_lint(key, at_ns) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET at_ns=excluded.at_ns`, f.key(), now.UnixNano())
			return fmt.Sprintf("Task board (%s): %s: %s. fix: %s", brk, f.TaskID, Truncate(oneLine(f.Problem), 160), f.Fix), "lint"
		} else if f, who, ok := e.pickGone(fs, in, unrelated, now); ok {
			// The owner of a finished task is gone: the coordinator closes it
			// (agh2h7; 44w1ev sat a day waiting after wyvern's session ended).
			e.boardMark(session, "lint", now)
			e.DB.Exec(`INSERT INTO board_lint(key, at_ns) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET at_ns=excluded.at_ns`, f.key(), now.UnixNano())
			return fmt.Sprintf("Task board (%s): %s: %s. Its owner %s has no live session, so it is yours as coordinator. fix: %s", brk, f.TaskID, Truncate(oneLine(f.Problem), 160), who, f.Fix), "lint"
		}
	}
	// Old open reflections reach the coordinator at a break (rk5q7r: 0 of
	// 56 reflections on three boards were resolved by 2026-11-18): their
	// count and the command that reads them, once per ReflectEvery, at a
	// break no other line took, so no other line moves.
	if wantReflect {
		// Checked once per ReflectEvery whatever it finds: a break that
		// reads nothing old does not read the board again until tomorrow.
		e.boardMark(session, "reflect", now)
		if isCoordinator(in) && in.Reflections != nil {
			if n := oldReflections(in.Reflections(), now); n > 0 {
				its := "reflections are"
				if n == 1 {
					its = "reflection is"
				}
				return fmt.Sprintf("Task board (%s): %d open %s older than %d days; read them grouped by the instruction they name: hug task reflections --by-instruction", brk, n, its, int(ReflectAfter.Hours()/24)), "reflect"
			}
		}
	}
	return "", ""
}

// ReflectAfter is how old an open reflection is before the coordinator
// hears of it; ReflectEvery how often it hears (rk5q7r).
var (
	ReflectAfter = 3 * 24 * time.Hour
	ReflectEvery = 24 * time.Hour
)

// isCoordinator reports whether the session is the board's coordinator.
func isCoordinator(in *BoardInput) bool {
	self := strings.TrimSpace(in.Self.Name)
	return self != "" && in.Coordinator != "" && strings.EqualFold(strings.TrimSpace(in.Coordinator), self)
}

// oldReflections counts the dates (YYYY-MM-DD) more than ReflectAfter
// whole days before now's day: on 11-21 a reflection of 11-17 is old, one
// of 11-18 (3 days) not yet.
func oldReflections(dates []string, now time.Time) int {
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	n := 0
	for _, s := range dates {
		day, err := time.Parse("2006-01-02", s)
		if err == nil && today.Sub(day) > ReflectAfter {
			n++
		}
	}
	return n
}

func (e *Engine) boardDue(session, kind string, every time.Duration, now time.Time) bool {
	var at int64
	e.DB.QueryRow(`SELECT at_ns FROM board_cues WHERE session=? AND kind=?`, session, kind).Scan(&at)
	return at == 0 || now.Sub(time.Unix(0, at)) >= every
}

func (e *Engine) boardMark(session, kind string, now time.Time) {
	e.DB.Exec(`INSERT INTO board_cues(session, kind, at_ns) VALUES(?,?,?) ON CONFLICT(session, kind) DO UPDATE SET at_ns=excluded.at_ns`, session, kind, now.UnixNano())
}

type detection struct {
	edits   bool
	hasMine bool
	mineAll []BoardTask // doing for the session with every check ticked
	owned   []BoardTask // doing, worked by the session, assigned to another name (v3k6cp)
	self    string      // the session's name, from the harness
	working []BoardTask // todo tasks the session checked or noted
	named   []BoardTask // todo tasks named by the message, commands or commits
	matched *BoardTask  // todo task whose title matches the work
	touched map[string]bool
	// mention holds touched tasks the session neither owns, acted on, nor
	// was given as work: named only as context (quoted, a dependency, a
	// task not to start) or another session's doing task it only read
	// (4vxqp4). They count as touched for every finding but an ownerOnly
	// one (a missing section, an at3qep question).
	mention map[string]bool
	own     map[string]bool // the session's own doing tasks (tpnd3n)
	answers []BoardTask     // tasks with an unread answer for this session (3jtfw9)
	asks    []BoardTask     // tasks with an open request routed to this session (hbgybn)
	acted   []BoardTask     // tasks whose answer was acted on, for the coordinator (41hhgn)
}

type candidate struct{ kind, id, line string }

// key names the line for RepeatEvery: the kind and the task it is about.
func (c candidate) key() string { return c.kind + ":" + c.id }

// RepeatEvery limits the same line about the same task per session: an
// ignored line said again every hour is noise (held-out replay: one seid
// session was told 'hug task done mtzydx' 57 times in three days).
var RepeatEvery = 24 * time.Hour

func (d detection) candidates(editTurns int64, model string) []candidate {
	if model == "" {
		model = "MODEL"
	}
	var out []candidate
	// A person's answer to this session's question comes first: it is the
	// newest thing anyone said to it (3jtfw9). The id carries the answer's
	// time, so a later answer on the same task is not held back.
	for _, t := range d.answers {
		a := t.Answer
		out = append(out, candidate{"answer", t.ID + "@" + a.At, fmt.Sprintf("%s answered your question on %s: %s (%s: %s). hug task show %s", a.By, t.ID, Truncate(oneLine(a.Text), 300), a.Stage, short(a.Question), t.ID)})
	}
	// The coordinator hears what an asker did with an answer (41hhgn check
	// 11), once per outcome.
	for _, t := range d.acted {
		o := t.Outcome
		out = append(out, candidate{"answered", t.ID + "@" + o.At, fmt.Sprintf("Task board: %s acted on %s's answer on %s (%s): %s. hug task show %s", o.For, o.By, t.ID, o.Stage, Truncate(oneLine(o.Text), 200), t.ID)})
	}
	// A person's request (hbgybn): a person waits on it too. Once per
	// request (the id carries its number), oldest first.
	for _, t := range d.asks {
		for _, r := range t.Requests {
			out = append(out, candidate{"request", fmt.Sprintf("%s#%d", t.ID, r.N), fmt.Sprintf("Task board: %s asks on %s: \"%s\" (request %d). Act on it, then: hug task resolve %s %d \"what changed\"", r.By, t.ID, Truncate(oneLine(r.Text), 300), r.N, t.ID, r.N)})
		}
	}
	for _, t := range d.owned {
		out = append(out, candidate{"owner", t.ID, fmt.Sprintf("Task board: you are working on %s but the board says %s. If it is yours now: hug task set %s --assignee %s", t.ID, strings.Join(t.Assignee, ", "), t.ID, d.self)})
	}
	for _, t := range d.mineAll {
		out = append(out, candidate{"done", t.ID, fmt.Sprintf("Task board: every check of %s holds (%s). Close it: hug task done %s", t.ID, short(t.Title), t.ID)})
	}
	for _, t := range d.working {
		out = append(out, candidate{"work", t.ID, fmt.Sprintf("Task board: you are working on %s but it is %s. hug task take %s --model %s", t.ID, t.State, t.ID, model)})
	}
	if d.edits && !d.hasMine {
		switch {
		case len(d.named) > 0:
			t := d.named[0]
			out = append(out, candidate{"take", t.ID, fmt.Sprintf("Task board: you are changing code and no task is doing for you; this looks like %s (%s). hug task take %s --model %s", t.ID, short(t.Title), t.ID, model)})
		case d.matched != nil:
			t := *d.matched
			out = append(out, candidate{"take", t.ID, fmt.Sprintf("Task board: you are changing code and no task is doing for you; this looks like %s (%s). hug task take %s --model %s", t.ID, short(t.Title), t.ID, model)})
		case editTurns >= 2:
			out = append(out, candidate{"take", "", "Task board: you are changing code and no task is doing for you. Take the task this is (hug task next), or hug task new \"title\" --goal \"...\" if none fits, then hug task take ID --model " + model})
		}
	}
	return out
}

var taskRef = regexp.MustCompile(`\b[a-z0-9]{6}\b`)

// stopOrder: a message that tells the agent to stop or to do nothing.
var stopOrder = regexp.MustCompile(`(?i)\bstop (?:now|immediately|all work|everything)\b|\bdo not (?:run any|run anything|edit any|edit anything|touch any)\b|\bdon'?t (?:run any|edit any|touch any)\b|\bend (?:your|the) session\b|\bstand down\b`)

func (e *Engine) detect(session, msg string, in *BoardInput) detection {
	d := detection{touched: map[string]bool{}, own: map[string]bool{}, mention: map[string]bool{}}
	byID := map[string]BoardTask{}
	for _, t := range in.Tasks {
		byID[t.ID] = t
	}
	taken := map[string]bool{}
	if rows, err := e.DB.Query(`SELECT id FROM board_taken WHERE session=?`, session); err == nil {
		for rows.Next() {
			var id string
			rows.Scan(&id)
			taken[id] = true
		}
		rows.Close()
	}
	self := in.Self
	mine := func(t BoardTask) bool {
		// A task the session took itself is its own even while the board
		// still says todo (the take commit lands after the turn starts).
		if taken[t.ID] && t.State != "done" && t.State != "dropped" {
			return true
		}
		if t.State != "doing" {
			return false
		}
		if taken[t.ID] {
			return true
		}
		for _, a := range t.Assignee {
			if self.Name != "" && strings.EqualFold(strings.TrimSpace(a), self.Name) {
				return true
			}
		}
		// The worktree or branch says whose a task is only off the main
		// line: every session in the main checkout shares its folder.
		if self.Linked && t.Worktree != "" && samePlace(t.Worktree, self.Worktree) {
			return true
		}
		offMain := self.Branch != "" && self.Branch != "main" && self.Branch != "master"
		return offMain && t.Branch == self.Branch
	}
	// acted: tasks the session owns or acted on (took, checked, noted, set,
	// edited ...). Reading another session's doing task with show is not
	// acting on it (4vxqp4; replay: palmtree read hibiscus's n8zmjw and was
	// handed its missing Purpose); reading an open todo still counts
	// (nautilus read e4w9ya, judged right).
	acted := map[string]bool{}
	others := func(id string) bool {
		t, ok := byID[id]
		if !ok || t.State != "doing" || len(t.Assignee) == 0 {
			return false
		}
		for _, a := range t.Assignee {
			if self.Name != "" && strings.EqualFold(strings.TrimSpace(a), self.Name) {
				return false
			}
		}
		return true
	}
	for _, t := range in.Tasks {
		if mine(t) {
			d.hasMine = true
			d.touched[t.ID] = true
			acted[t.ID] = true
			if t.State == "doing" {
				d.own[t.ID] = true
			}
			// Not while a question on it is open (72n8m4: the cue offered
			// done on ddpxs8, 4cn9vv and gvkpzt as they waited on jhf).
			if t.Checks > 0 && t.Open == 0 && !t.Asked {
				d.mineAll = append(d.mineAll, t)
			}
		}
	}
	d.self = self.Name
	for _, t := range in.Tasks {
		if t.Answer != nil && self.Name != "" && strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(t.Answer.For, "@")), self.Name) {
			d.answers = append(d.answers, t)
		}
	}
	for _, t := range in.Tasks {
		if o := t.Outcome; o != nil && in.Coordinator != "" && strings.EqualFold(in.Coordinator, self.Name) && !strings.EqualFold(o.For, self.Name) {
			d.acted = append(d.acted, t)
		}
		if len(t.Requests) > 0 && requestFor(t, self.Name, in.Coordinator) {
			d.asks = append(d.asks, t)
		}
	}
	// owner (v3k6cp): the session works a doing task (took it, works in its
	// recorded worktree or branch, ticks its checks, or notes it while
	// editing code) but the assignee is another name. Only with the
	// session's own name from the harness: never guessed.
	worked := map[string]bool{}
	notedDoing := map[string]bool{}
	anyEdits := false
	for _, a := range in.Actions {
		anyEdits = anyEdits || a.Edits
		for _, h := range a.Hug {
			switch h.Verb {
			case "check":
				worked[h.ID] = true
			case "note":
				notedDoing[h.ID] = true
			}
		}
	}
	if self.Name != "" {
		for _, t := range in.Tasks {
			if t.State != "doing" || len(t.Assignee) == 0 {
				continue
			}
			assigned := false
			for _, a := range t.Assignee {
				assigned = assigned || strings.EqualFold(strings.TrimSpace(a), self.Name)
			}
			if assigned {
				continue
			}
			if mine(t) || worked[t.ID] || notedDoing[t.ID] && anyEdits {
				d.owned = append(d.owned, t)
			}
		}
	}
	var words []string
	seenNamed := map[string]bool{}
	addNamed := func(id string) {
		if t, ok := byID[id]; ok && t.State == "todo" && !seenNamed[id] && !taken[id] {
			seenNamed[id] = true
			d.named = append(d.named, t)
		}
	}
	seenWork := map[string]bool{}
	noted := map[string]bool{}
	for _, a := range in.Actions {
		d.edits = d.edits || a.Edits
		for _, h := range a.Hug {
			if h.ID == "" {
				continue
			}
			d.touched[h.ID] = true
			if h.Verb != "show" || !others(h.ID) {
				acted[h.ID] = true
			}
			// Ticking a check is doing the task. A note alone is not: a
			// coordinator records design talk on todo tasks it is not working
			// on (dev replay, BOARD4.md). A note counts with code edits in the
			// same window.
			if t, ok := byID[h.ID]; ok && t.State == "todo" && !seenWork[h.ID] && !taken[h.ID] {
				if h.Verb == "check" {
					seenWork[h.ID] = true
					d.working = append(d.working, t)
				} else if h.Verb == "note" {
					noted[h.ID] = true
				}
			}
			// Reading a task is not working on it: guppy read jrj1vq (a
			// dependency of its queue) and was told 'this looks like jrj1vq ...
			// hug task take jrj1vq' (rzfrgp, 6fqhg9 replay; judged wrong twice).
			if h.Verb == "check" || h.Verb == "note" {
				addNamed(h.ID)
			}
		}
		if a.Subject != "" {
			for _, id := range taskRef.FindAllString(a.Subject, -1) {
				addNamed(id)
			}
			words = append(words, a.Subject)
		}
	}
	plain := assigningRefs(msg)
	for _, id := range taskRef.FindAllString(strings.ToLower(msg), -1) {
		if _, ok := byID[id]; ok {
			d.touched[id] = true
		}
		addNamed(id)
	}
	// A touched task the session neither owns, acted on, nor was given as
	// work by the message is only mentioned (4vxqp4).
	for id := range d.touched {
		if !acted[id] && !plain[id] {
			d.mention[id] = true
		}
	}
	if d.edits {
		for _, t := range in.Tasks {
			if noted[t.ID] && !seenWork[t.ID] {
				seenWork[t.ID] = true
				d.working = append(d.working, t)
			}
		}
	}
	if d.edits && !d.hasMine && len(d.named) == 0 {
		d.matched = matchTitle(in.Tasks, msg+" "+strings.Join(words, " "))
	}
	return d
}

// matchTitle returns the todo task whose title shares the most rare words
// with the text, when it clearly stands out; nil otherwise.
func matchTitle(ts []BoardTask, text string) *BoardTask {
	q := map[string]bool{}
	for _, w := range titleWords(text) {
		q[w] = true
	}
	if len(q) == 0 || len(ts) == 0 {
		return nil
	}
	df := map[string]int{}
	words := make([][]string, len(ts))
	for i, t := range ts {
		words[i] = uniq(titleWords(t.Title))
		for _, w := range words[i] {
			df[w]++
		}
	}
	n := float64(len(ts))
	best, second, bi := 0.0, 0.0, -1
	for i, t := range ts {
		if t.State != "todo" {
			continue
		}
		s, hits := 0.0, 0
		for _, w := range words[i] {
			if q[w] {
				s += math.Log(1 + n/float64(df[w]))
				hits++
			}
		}
		if hits < 2 {
			continue
		}
		if s > best {
			best, second, bi = s, best, i
		} else if s > second {
			second = s
		}
	}
	if bi < 0 || best < MatchMin || best < MatchMargin*second {
		return nil
	}
	return &ts[bi]
}

// MatchMin and MatchMargin gate the title match (tuned on the dev replay,
// muninn/docs/BOARD4.md).
var (
	MatchMin    = 6.0
	MatchMargin = 1.5
)

var stop = map[string]bool{"the": true, "and": true, "for": true, "with": true, "not": true, "are": true, "from": true, "into": true, "that": true, "this": true, "its": true, "when": true, "one": true, "all": true, "hug": true, "task": true, "tasks": true, "every": true, "new": true, "can": true, "you": true, "use": true, "has": true, "but": true, "was": true, "out": true, "any": true, "how": true, "what": true, "show": true, "says": true}

func titleWords(s string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127)
	}) {
		if len(f) >= 3 && !stop[f] {
			out = append(out, f)
		}
	}
	return out
}

func uniq(xs []string) []string {
	sort.Strings(xs)
	out := xs[:0]
	for i, x := range xs {
		if i == 0 || x != xs[i-1] {
			out = append(out, x)
		}
	}
	return out
}

// LintRepeat is how long a handed-out finding rests before any session is
// handed it again (held-out replay: one title finding went to 17 seid
// sessions; the board has thousands of others).
var LintRepeat = 7 * 24 * time.Hour

func (f BoardFinding) key() string { return f.TaskID + "\x00" + f.Problem }

// runnable reports whether f's fix is one this agent can run on this one
// task: "hug upgrade" (section order) rewrites the whole board and
// upgrades the binary.
func runnable(f BoardFinding) bool {
	return f.TaskID != "" && strings.HasPrefix(f.Fix, "hug task ") && strings.Contains(f.Fix, f.TaskID)
}

// ownerOnly reports whether f is one only the task's own session should
// fill: a missing Purpose or Principle says why the work is done and what
// it does not do, which a passer-by cannot know (tpnd3n; replay after
// fkpyg3: all 20 false lint lines were missing-section findings, 18 on
// tasks the session had not touched, e.g. s6ff5k named to hog and
// dromedary while the user forbade every other task; BOARD4.md). A
// question hug cannot settle (at3qep: landed but unchecked, shipped but
// open, todo with a commit, idle) is asked of the ticket's owner only.
func ownerOnly(f BoardFinding) bool {
	for _, p := range ownerOnlyPrefixes {
		if strings.HasPrefix(f.Problem, p) {
			return true
		}
	}
	return false
}

// ownerOnlyPrefixes start the problems of ownerOnly findings (hug task
// lint's missing-section and the at3qep question codes).
var ownerOnlyPrefixes = []string{"missing section", "landed, but no check", "shipped, but ", "todo, but commit ", "doing, but nothing on it", checksHold}

// checksHold starts the problem of a checks-all-hold finding (bg1rd8): a
// waiting task with every check and child done. Its owner closes it, or,
// with the owner gone, the board's coordinator (agh2h7).
const checksHold = "every check holds, but "

// ownerGone returns the assignees of task id when every one of them is a
// harness session name that has gone (BoardInput.Gone, the stale-owner
// rule), or ["nobody"] for a task with no assignee. Nil when someone may
// still be there: a live session, a name the harness does not know, no
// harness list.
func ownerGone(in *BoardInput, id string, now time.Time) []string {
	if in.Gone == nil {
		return nil
	}
	for _, t := range in.Tasks {
		if t.ID != id {
			continue
		}
		if len(t.Assignee) == 0 {
			return []string{"nobody"}
		}
		var out []string
		for _, a := range t.Assignee {
			a = strings.TrimSpace(strings.TrimPrefix(a, "@"))
			if gone, known := in.Gone(strings.ToLower(a), now); !known || !gone {
				return nil
			}
			out = append(out, a)
		}
		return out
	}
	return nil
}

// pickGone returns a checks-all-hold finding whose owner is gone, for the
// board's coordinator only (closing another's task is the coordinator's
// call; with no coordinator set nobody is told), at a break where an
// unrelated finding may be said.
func (e *Engine) pickGone(fs []BoardFinding, in *BoardInput, any bool, now time.Time) (BoardFinding, string, bool) {
	if !any || !isCoordinator(in) {
		return BoardFinding{}, "", false
	}
	for _, f := range fs {
		if !runnable(f) || !strings.HasPrefix(f.Problem, checksHold) {
			continue
		}
		var at int64
		e.DB.QueryRow(`SELECT at_ns FROM board_lint WHERE key=?`, f.key()).Scan(&at)
		if at > 0 && now.Sub(time.Unix(0, at)) < LintRepeat {
			continue
		}
		if who := ownerGone(in, f.TaskID, now); who != nil {
			return f, strings.Join(who, ", "), true
		}
	}
	return BoardFinding{}, "", false
}

// ownHasFinding reports whether one of the session's own doing tasks has
// a finding with a fix it can run (tpnd3n).
func ownHasFinding(fs []BoardFinding, own map[string]bool) bool {
	for _, f := range fs {
		if own[f.TaskID] && runnable(f) {
			return true
		}
	}
	return false
}

// quotedSpan is text the message quotes: a fenced block, backticks, or
// double quotes within one line.
var quotedSpan = regexp.MustCompile("(?s)```.*?```|`[^`\n]*`|\"[^\"\n]{0,300}\"|“[^”\n]{0,300}”")

// bareRef is a quoted span that is only a task id or a hug command on one:
// an instruction ("`hug task take abc123`", "Top task: `abc123`"), not a
// quote of output or of a prompt.
var bareRef = regexp.MustCompile(`^(?:task:)?[a-z0-9]{6}$|^hug task [a-z]+ (?:task:)?[a-z0-9]{6}\b`)

// contextBefore ends right before a task id that the message names as a
// dependency or as a task not to start.
var contextBefore = regexp.MustCompile(`(?:\b(?:depends? on|builds? on|blocked by|waits? (?:on|for)|parent)\s*:?\s*(?:task:)?|\b(?:do not|don['’]?t|never)\s+(?:start|take|touch|work on|begin|pick up)\b[^.!?\n]{0,80})$`)

// assigningRefs returns the task ids a message names at least once as
// work, not only as context: outside quoted text (or quoted bare), and not
// right after "depends on" or "do not start" (4vxqp4; replay: s6ff5k
// reached hog and dromedary quoted as "Confirm edit s6ff5k Title?" in a
// brief that forbade every other task, vk9td0 reached kitten and poodle
// quoted as `next: hug task take vk9td0`, n8zmjw reached evergreen as
// "p39w4a depends on n8zmjw").
func assigningRefs(msg string) map[string]bool {
	low := strings.ToLower(msg)
	var spans [][]int
	for _, s := range quotedSpan.FindAllStringIndex(low, -1) {
		in := strings.TrimSpace(strings.Trim(low[s[0]:s[1]], "`\"“”"))
		if !bareRef.MatchString(in) {
			spans = append(spans, s)
		}
	}
	out := map[string]bool{}
	for _, m := range taskRef.FindAllStringIndex(low, -1) {
		quoted := false
		for _, s := range spans {
			quoted = quoted || s[0] <= m[0] && m[1] <= s[1]
		}
		if quoted || contextBefore.MatchString(low[max(0, m[0]-120):m[0]]) {
			continue
		}
		out[low[m[0]:m[1]]] = true
	}
	return out
}

func (e *Engine) pickFinding(fs []BoardFinding, touched, mention map[string]bool, any bool, now time.Time) (BoardFinding, bool) {
	rank := func(f BoardFinding) int {
		switch {
		case touched[f.TaskID]:
			return 0
		case f.TaskState == "doing":
			return 1
		case f.TaskState == "todo":
			return 2
		}
		return 3
	}
	best, bi := 4, -1
	for i, f := range fs {
		// A missing section (any ownerOnly finding) is the task's own
		// session's to fill: a task it only mentions is not that (4vxqp4).
		if !runnable(f) || ownerOnly(f) && mention[f.TaskID] {
			continue
		}
		r := rank(f)
		if r >= best || !(r == 0 || any && !ownerOnly(f)) {
			continue
		}
		var at int64
		e.DB.QueryRow(`SELECT at_ns FROM board_lint WHERE key=?`, f.key()).Scan(&at)
		if at > 0 && now.Sub(time.Unix(0, at)) < LintRepeat {
			continue
		}
		best, bi = r, i
	}
	if bi < 0 {
		return BoardFinding{}, false
	}
	return fs[bi], true
}

func short(s string) string {
	s = oneLine(s)
	if r := []rune(s); len(r) > 60 {
		return string(r[:59]) + "…"
	}
	return s
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func samePlace(a, b string) bool {
	return strings.TrimRight(a, `/\`) == strings.TrimRight(b, `/\`)
}

// requestFor says whether an open request on t reaches the session named
// who, as hug task next routes it (task.RequestFor): the assignee of a
// doing task; on a task nobody holds, the coordinator; with no
// coordinator, every session. Without the session's name, nothing.
func requestFor(t BoardTask, who, coordinator string) bool {
	who = strings.TrimSpace(who)
	if who == "" {
		return false
	}
	for _, a := range t.Assignee {
		if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(a, "@")), who) {
			return true
		}
	}
	if len(t.Assignee) > 0 && t.State == "doing" {
		return false
	}
	return coordinator == "" || strings.EqualFold(coordinator, who)
}
