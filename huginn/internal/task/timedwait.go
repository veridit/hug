package task

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Waiting for a date (z2vmte, Jørgen 2026-11-18 02:28: "waiting for time
// to pass is a sensible state and when the time has passed the system
// brings it back"). Time is a thing a task waits on, like CI, so there is
// no new stage: hug task wait ID --until DATE "why" is a wait with
// waiting_until, and hug task ask ID --to NAME --after DATE "question" is
// that wait holding the question (waiting_ask_*), in nobody's ASKED until
// the date. On the first hug read on or after DATE (local), Wake ends it:
// a plain wait goes back to doing/active, first in its owner's next; a
// held question becomes doing/asked, the question verbatim. One History
// line says the date came; the wait's fields go to History (3nhp67).

// dayLayout is how a date is written: YYYY-MM-DD, the person's local day.
const dayLayout = "2006-01-02"

// askFields are the parked question's fields and where they go at the date.
var askFields = [][2]string{{"waiting_ask_what", "asked_what"}, {"waiting_ask_for", "asked_for"}, {"waiting_ask_check", "asked_check"}, {"waiting_ask_show", "asked_show"}}

// today is the local day Now is in, at midnight.
func today() time.Time {
	n := Now().Local()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local)
}

// parseDay reads a YYYY-MM-DD date as a local day.
func parseDay(s string) (time.Time, error) {
	return time.ParseInLocation(dayLayout, strings.TrimSpace(s), time.Local)
}

// dayArg checks a --until or --after date: YYYY-MM-DD and after today (a
// date that has come is no wait). fix gets the date to suggest (%s).
func dayArg(id, flag, v, fix string) (string, error) {
	tomorrow := today().AddDate(0, 0, 1).Format(dayLayout)
	d, err := parseDay(v)
	if err != nil {
		return "", &Error{Kind: "usage", ID: id, Value: fmt.Sprintf("%s takes a date YYYY-MM-DD, not %q", flag, v), Fix: fmt.Sprintf(fix, tomorrow)}
	}
	if !d.After(today()) {
		return "", &Error{Kind: "usage", ID: id, Value: fmt.Sprintf("%s %s has come already (today is %s): a wait for a date ends the day it comes", flag, d.Format(dayLayout), today().Format(dayLayout)), Fix: fmt.Sprintf(fix, tomorrow)}
	}
	return d.Format(dayLayout), nil
}

// DatedWait is a wait for a date as its stage line says it (atex3d,
// Jørgen 2026-11-20 13:05): the date first, then what follows it. until is
// waiting_until; rest is "then ask NAME" for a question held for the date
// (waiting_ask_for), else "on WHY" (waiting_on), "" when it names
// nothing: "waiting until 2026-11-24 · then ask jhf". ok is false for a
// wait with no date. The stored waiting_on of ask --after, "the date
// DATE, then ask NAME", is never shown: it is for an older hug.
func DatedWait(f map[string]string) (until, rest string, ok bool) {
	until = strings.TrimSpace(f["waiting_until"])
	if until == "" {
		return "", "", false
	}
	if who := strings.TrimSpace(f["waiting_ask_for"]); who != "" {
		return until, "then ask " + strings.Join(strings.Fields(who), " "), true
	}
	on := strings.Join(strings.Fields(f["waiting_on"]), " ")
	if r, cut := strings.CutPrefix(on, "the date "+until+", "); cut {
		return until, r, true // its ask fields lost: "then ask NAME" as written
	}
	if on == "" {
		return until, "", true
	}
	return until, "on " + on, true
}

// DatedWaitLine is DatedWait in one line: "waiting until DATE · REST".
func DatedWaitLine(until, rest string) string {
	if rest == "" {
		return "waiting until " + until
	}
	return "waiting until " + until + " · " + rest
}

// dueDay says whether a waiting_until date has come.
func dueDay(v string) bool {
	if strings.TrimSpace(v) == "" {
		return false
	}
	d, err := parseDay(v)
	return err == nil && !today().Before(d)
}

// Due are the tasks a read wakes now: doing, waiting, waiting_until come.
func Due(ts []Task) []string {
	var out []string
	for _, t := range ts {
		if t.State == "doing" && ownHolds(t, "waiting") && dueDay(t.Fields["waiting_until"]) {
			out = append(out, t.ID)
		}
	}
	return out
}

// WaitUntil is Wait with a date: until "" is a plain wait; otherwise the
// task wakes on its own that day (z2vmte). A wait replaces a question held
// for a date; its fields go to History.
func (s *Store) WaitUntil(ref, on, until, by string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	id := d.Task.ID
	on = strings.TrimSpace(on)
	if on == "" {
		fix := "hug task wait " + id + ` "CI run 123"`
		if until != "" {
			fix = "hug task wait " + id + " --until " + until + ` "why"`
		}
		return Result{}, usageErr(id, "wait needs what it waits on", fix)
	}
	// A question to a person is asked, never waited on (agxqax): "owner:
	// check 4 ..." as a wait is in nobody's ASKED.
	if people, _ := People(s.Root); len(people) > 0 {
		if w, ok := AddressesPerson(people, on, len(d.Task.Acceptance)); ok {
			return Result{}, usageErr(id, "wait is for things with no person to answer (CI, a release); a question to a person is hug task ask: "+strconv.Quote(oneLineFix(on))+" is for "+w.For, w.AskCommand(id))
		}
	}
	if d.Task.State != "doing" {
		return Result{}, refuse(d.Task, "doing/waiting", "hug task take "+id+" --model MODEL")
	}
	if until != "" {
		if until, e = dayArg(id, "--until", until, "hug task wait "+id+" --until %s "+strconv.Quote(oneLineFix(on))); e != nil {
			return Result{}, e
		}
	}
	f := d.Task.Fields
	if f["waiting_on"] == on && f["waiting_until"] == until && f["waiting_ask_what"] == "" && ownHolds(d.Task, "waiting") && len(heldStages(d.Task)) == 1 {
		return s.finish(d, false)
	}
	set := map[string]string{"waiting_on": on, "waiting_since": stamp()}
	if by = strings.TrimSpace(by); by != "" {
		set["waiting_asked_by"] = by
	}
	clear := []string{}
	for _, k := range askFields {
		clear = append(clear, k[0])
	}
	why := "wait"
	if until != "" {
		set["waiting_until"] = until
		why = "wait until " + until
	} else {
		clear = append(clear, "waiting_until")
	}
	if e = applyMove(d, ts, move{state: "doing", stage: "waiting", set: set, clear: clear, why: why}); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// AskAfter is Ask with a date: after "" asks now; otherwise the task waits
// until the date, holding the question, and asks it then (z2vmte). Nothing
// shows in q.For's ASKED before the date.
func (s *Store) AskAfter(ref string, q Question, after, by string) (Result, error) {
	if strings.TrimSpace(after) == "" {
		return s.Ask(ref, q, by)
	}
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	id := d.Task.ID
	what, who, check, show, e := s.askable(d, q)
	if e != nil {
		return Result{}, e
	}
	// --after parks the asker's own doing task until the day; a todo task
	// has nobody to park (yphbra): ask it now, or take it first.
	if d.Task.State != "doing" {
		return Result{}, refuse(d.Task, "doing/waiting", "hug task ask "+id+" --to "+who+" "+strconv.Quote(oneLineFix(what)))
	}
	fix := "hug task ask " + id + " --to " + who + " --after %s"
	if check != "" {
		fix += " --check " + check
	}
	if after, e = dayArg(id, "--after", after, fix+" "+strconv.Quote(oneLineFix(what))); e != nil {
		return Result{}, e
	}
	f := d.Task.Fields
	if f["waiting_until"] == after && f["waiting_ask_what"] == what && f["waiting_ask_for"] == who && f["waiting_ask_check"] == check && f["waiting_ask_show"] == show && len(heldStages(d.Task)) == 1 {
		return s.finish(d, false)
	}
	// No "NAME:" in waiting_on: an old reader takes that for a question.
	set := map[string]string{"waiting_on": "the date " + after + ", then ask " + who, "waiting_since": stamp(), "waiting_until": after, "waiting_ask_what": what, "waiting_ask_for": who}
	clear := []string{}
	if by = strings.TrimSpace(by); by != "" {
		set["waiting_asked_by"] = by
	}
	for k, v := range map[string]string{"waiting_ask_check": check, "waiting_ask_show": show} {
		if v != "" {
			set[k] = v
		} else {
			clear = append(clear, k)
		}
	}
	if e = applyMove(d, ts, move{state: "doing", stage: "waiting", set: set, clear: clear, why: "ask " + who + " after " + after}); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}

// Wake ends ref's wait when its date has come: a held question becomes
// doing/asked (asked_by is who waited), else the task is doing/active
// again. Unchanged when ref waits for no date, or not yet.
func (s *Store) Wake(ref string) (Result, error) {
	d, ts, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	f := d.Task.Fields
	until := f["waiting_until"]
	if d.Task.State != "doing" || !ownHolds(d.Task, "waiting") || !dueDay(until) {
		return s.finish(d, false)
	}
	why := wokeWhy + until + " has come"
	m := move{state: "doing", clear: stageFields("doing", "waiting"), why: why}
	if strings.TrimSpace(f["waiting_ask_what"]) != "" {
		set := map[string]string{"asked_since": stamp()}
		for _, k := range askFields {
			if v := f[k[0]]; v != "" {
				set[k[1]] = v
			}
		}
		if v := strings.TrimSpace(f["waiting_asked_by"]); v != "" {
			set["asked_by"] = v
		}
		m = move{state: "doing", stage: "asked", set: set, why: why + ", asked " + f["waiting_ask_for"]}
	}
	if e = applyMove(d, ts, m); e != nil {
		return Result{}, e
	}
	s.waking = true
	defer func() { s.waking = false }()
	return s.finish(d, true)
}

// backLine is written when the owner first writes on a task its date woke:
// it is in hand again, no longer first in next --mine.
const backLine = " back on it after the wait"

// markBack appends "NAME back on it after the wait" when actor, an owner
// of d, writes on it while it is woken (z2vmte).
func markBack(d *Document, actor string) {
	if actor = strings.TrimSpace(actor); actor == "" || !d.Task.Woken || !contains(d.Task.Assignee, actor) {
		return
	}
	d.History(stamp() + " " + actor + backLine)
	d.Task.Woken = false
}

// wokeWhy starts the History reason of a wake.
const wokeWhy = "woke: the date "

// wokeLine is a wake back to active, the History line Woken looks for.
var wokeLine = regexp.MustCompile(`^- \S+ doing/waiting -> doing/active \(` + regexp.QuoteMeta(wokeWhy))

// woken says whether the newest History line is a wake back to active:
// nothing has moved the task since, so next --mine puts it first.
func woken(history []string) bool {
	for i := len(history) - 1; i >= 0; i-- {
		l := strings.TrimRight(history[i], "\r\n")
		if strings.TrimSpace(l) == "" || strings.HasPrefix(l, "  ") {
			continue
		}
		return wokeLine.MatchString(l)
	}
	return false
}

// untilFindings flags a waiting_until that is no date: such a wait would
// never end on its own.
func untilFindings(t Task) []Finding {
	v := strings.TrimSpace(t.Fields["waiting_until"])
	if v == "" {
		return nil
	}
	if _, err := parseDay(v); err == nil {
		return nil
	}
	return []Finding{{Path: t.Path, Line: 1, Problem: fmt.Sprintf("invalid waiting_until %q: a date YYYY-MM-DD, or the wait never ends on its own", v),
		Fix: "hug task wait " + t.ID + " --until " + today().AddDate(0, 0, 1).Format(dayLayout) + " " + strconv.Quote(oneLineFix(t.Fields["waiting_on"]))}}
}
