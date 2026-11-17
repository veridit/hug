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
	ID, Title, Status string
	Assignee          []string
	Branch, Worktree  string
	Checks, Open      int
}

// BoardFinding is one open lint finding with its repair command.
type BoardFinding struct {
	TaskID, TaskStatus, Problem, Fix string
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
	Tasks    []BoardTask
	Findings func() []BoardFinding
	Self     BoardSelf
	Actions  []BoardAction
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
	in := src(time.Unix(0, seen), wantLint)
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
		if e.boardDue(session, rate, BoardEvery, now) && e.boardDue(session, c.key(), RepeatEvery, now) {
			e.boardMark(session, rate, now)
			e.boardMark(session, c.key(), now)
			return c.line, c.kind
		}
	}
	if wantLint && in.Findings != nil {
		// At the start of a session only findings on tasks it named: a
		// fresh assignment is not the moment for unrelated housekeeping.
		// (x325pp tried holding unrelated findings until the session had no
		// task of its own or had just closed one: held-out false lines 5 -> 0
		// but fresh lint lines 14 -> 8 and fresh2's 3 judged false;
		// muninn/docs/BOARD4.md. Not kept.)
		if f, ok := e.pickFinding(in.Findings(), d.touched, brk != "start", now); ok {
			e.boardMark(session, "lint", now)
			e.DB.Exec(`INSERT INTO board_lint(key, at_ns) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET at_ns=excluded.at_ns`, f.key(), now.UnixNano())
			return fmt.Sprintf("Task board (%s): %s: %s. fix: %s", brk, f.TaskID, Truncate(oneLine(f.Problem), 160), f.Fix), "lint"
		}
	}
	return "", ""
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
	for _, t := range d.owned {
		out = append(out, candidate{"owner", t.ID, fmt.Sprintf("Task board: you are working on %s but the board says %s. If it is yours now: hug task set %s --assignee %s", t.ID, strings.Join(t.Assignee, ", "), t.ID, d.self)})
	}
	for _, t := range d.mineAll {
		out = append(out, candidate{"done", t.ID, fmt.Sprintf("Task board: every check of %s holds (%s). Close it: hug task done %s", t.ID, short(t.Title), t.ID)})
	}
	for _, t := range d.working {
		out = append(out, candidate{"work", t.ID, fmt.Sprintf("Task board: you are working on %s but it is %s. hug task take %s --model %s", t.ID, t.Status, t.ID, model)})
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
	d := detection{touched: map[string]bool{}}
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
		if taken[t.ID] && t.Status != "done" && t.Status != "dropped" {
			return true
		}
		if t.Status != "doing" {
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
	for _, t := range in.Tasks {
		if mine(t) {
			d.hasMine = true
			d.touched[t.ID] = true
			if t.Checks > 0 && t.Open == 0 {
				d.mineAll = append(d.mineAll, t)
			}
		}
	}
	d.self = self.Name
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
			if t.Status != "doing" || len(t.Assignee) == 0 {
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
		if t, ok := byID[id]; ok && t.Status == "todo" && !seenNamed[id] && !taken[id] {
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
			// Ticking a check is doing the task. A note alone is not: a
			// coordinator records design talk on todo tasks it is not working
			// on (dev replay, BOARD4.md). A note counts with code edits in the
			// same window.
			if t, ok := byID[h.ID]; ok && t.Status == "todo" && !seenWork[h.ID] && !taken[h.ID] {
				if h.Verb == "check" {
					seenWork[h.ID] = true
					d.working = append(d.working, t)
				} else if h.Verb == "note" {
					noted[h.ID] = true
				}
			}
			if h.Verb == "show" || h.Verb == "check" || h.Verb == "note" {
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
	for _, id := range taskRef.FindAllString(strings.ToLower(msg), -1) {
		if _, ok := byID[id]; ok {
			d.touched[id] = true
		}
		addNamed(id)
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
		if t.Status != "todo" {
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

func (e *Engine) pickFinding(fs []BoardFinding, touched map[string]bool, any bool, now time.Time) (BoardFinding, bool) {
	rank := func(f BoardFinding) int {
		switch {
		case touched[f.TaskID]:
			return 0
		case f.TaskStatus == "doing":
			return 1
		case f.TaskStatus == "todo":
			return 2
		}
		return 3
	}
	best, bi := 4, -1
	for i, f := range fs {
		// Only a fix this agent can run on this one task: "hug upgrade"
		// (section order) rewrites the whole board and upgrades the binary.
		if f.TaskID == "" || !strings.HasPrefix(f.Fix, "hug task ") || !strings.Contains(f.Fix, f.TaskID) {
			continue
		}
		r := rank(f)
		if r >= best || !(any || r == 0) {
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
