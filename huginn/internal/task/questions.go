package task

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Questions hug cannot settle (at3qep, epic 47sxn2): states only the
// ticket's owner can resolve become one question on the ticket, naming what
// hug observed, from what, and one fix. Each is recomputed from the board
// and git, nothing is stored, so fixing the state clears the question. One
// question per ticket at a time, the most important first:
//
//	landed-unchecked  (a) a commit naming it reached main, no check settled
//	shipped-not-done  (c) a release contains it, it is not done
//	checks-all-hold   (f) waiting, every check and child done (ChecksHoldFindings)
//	branch-gone       (e) doing, its branch merged or deleted (BranchFindings)
//	todo-has-commit   (d) todo, but a commit naming it reached main
//	doing-idle        (b) doing and active, nothing on it for IdleAfter
//
// (a), (c) and (d) are read from the ticket's own notes (g9dg32 writes
// "landed SHA on BRANCH ..." and "shipped in TAG (SHA)"), so the parse
// finds them and the cache keeps them. (e) and (b) read git, on lint only.

// IdleAfter is how long a doing, active ticket may go without a commit
// touching it before hug asks its owner whether it is still being worked.
var IdleAfter = 2 * time.Hour

var (
	landedLine  = regexp.MustCompile(`(?m)^landed ([0-9a-f]{7,40}) on (\S+)`)
	shippedLine = regexp.MustCompile(`(?m)^shipped in (\S+) \(([0-9a-f]{7,40})\)`)
)

// questionRank orders the question codes: lower asks first.
var questionRank = map[string]int{"landed-unchecked": 0, "shipped-not-done": 1, "checks-all-hold": 2, "branch-gone": 3, "todo-has-commit": 4, "doing-idle": 5}

// QuestionCode reports whether code is one of the question codes.
func QuestionCode(code string) bool {
	_, ok := questionRank[code]
	return ok
}

// noteQuestion is the question a ticket's notes and state raise, if any:
// (a), (c) or (d), the first that holds.
func noteQuestion(t Task, notes string) *Finding {
	if t.State == "done" || t.State == "dropped" {
		return nil
	}
	landed := landedLine.FindStringSubmatch(notes)
	shipped := shippedLine.FindStringSubmatch(notes)
	settled, firstOpen := 0, 0
	var open []string
	for i, a := range t.Acceptance {
		if a.Settled() {
			settled++
		} else {
			if firstOpen == 0 {
				firstOpen = i + 1
			}
			open = append(open, itoa(i+1))
		}
	}
	who := strings.Join(t.Assignee, ", ")
	if who == "" {
		who = "nobody"
	}
	switch {
	case landed != nil && t.State == "doing" && settled == 0 && len(t.Acceptance) > 0:
		return &Finding{Path: t.Path, Line: 1, Code: "landed-unchecked",
			Problem: "landed, but no check is ticked: commit " + landed[1][:7] + " naming " + t.ID + " reached " + landed[2] + " (its note says so) and 0 of " + itoa(len(t.Acceptance)) + " checks are ticked or struck; " + who + ": which checks hold?",
			Fix:     "hug task check " + t.ID + " 1"}
	case shipped != nil && (stageSaysWhy(t) || HasOpenQuestion(t)):
		// The stage already says why it is not done (kestrel's ggydj2
		// finding, crab 00:49): waiting, asked, review. No question, so
		// asking settles it (stageSettled, gpyqq9). So does a question
		// open to a person on a todo task (yf39ya check 7: txjkrm, todo
		// with jhf's question open since 11-18, stayed shipped-not-done).
		return nil
	case shipped != nil && len(open) > 0:
		// A fix hug takes (wdkqzx, Jørgen 23:48): done refuses open
		// checks, so the question is the first open one.
		which, does := "check "+open[0]+" is open", "does it hold"
		if len(open) > 1 {
			which, does = "checks "+andList(open)+" are open", "do they hold"
		}
		return &Finding{Path: t.Path, Line: 1, Code: "shipped-not-done",
			Problem: "shipped, but " + t.State + ": release " + shipped[1] + " contains commit " + shipped[2][:7] + " naming " + t.ID + " (its note says so) and " + which + "; " + who + ": " + does + ", or strike it?",
			Fix:     "hug task check " + t.ID + " " + itoa(firstOpen)}
	case shipped != nil:
		return &Finding{Path: t.Path, Line: 1, Code: "shipped-not-done",
			Problem: "shipped, but " + t.State + ": release " + shipped[1] + " contains commit " + shipped[2][:7] + " naming " + t.ID + " (its note says so); " + who + ": is it done?",
			Fix:     "hug task done " + t.ID}
	case landed != nil && t.State == "todo":
		return &Finding{Path: t.Path, Line: 1, Code: "todo-has-commit",
			Problem: "todo, but commit " + landed[1][:7] + " naming " + t.ID + " reached " + landed[2] + " (its note says so); who is on it?",
			Fix:     "hug task take " + t.ID + " --model MODEL"}
	}
	return nil
}

// IdleFindings asks the owner of each doing, active ticket that nothing
// has touched for IdleAfter (b): no commit on the task branch changed its
// file and no commit on main named it. Every agent commits under one git
// author, so any commit on the ticket counts, not only its taker's. A
// ticket that already has a question (file or git) is skipped: one at a
// time. last maps a ticket id to the newest commit touching it; a nil map
// means the history is not readable here, and nothing is said.
func IdleFindings(ts []Task, last map[string]time.Time, asked map[string]bool, now time.Time) []Finding {
	if last == nil {
		return nil
	}
	var out []Finding
	for _, t := range ts {
		if t.State != "doing" || t.Stage != "active" || asked[t.ID] {
			continue
		}
		since, err := time.ParseInLocation("2006-01-02T15:04", t.Fields["doing_since"], time.Local)
		if err != nil || now.Sub(since) < IdleAfter {
			continue
		}
		seen := since
		if l, ok := last[t.ID]; ok && l.After(seen) {
			seen = l
		}
		if now.Sub(seen) < IdleAfter {
			continue
		}
		who := strings.Join(t.Assignee, ", ")
		if who == "" {
			who = "nobody"
		}
		out = append(out, Finding{Path: t.Path, Line: 1, Code: "doing-idle",
			Problem: "doing, but nothing on it for " + hours(now.Sub(seen)) + ": no note, check or commit naming " + t.ID + " since " + seen.Format("2006-01-02 15:04") + " (task branch and main history); " + who + ": still on it?",
			Fix:     "hug task note " + t.ID + ` "where it stands"`})
	}
	return out
}

// ChecksHoldFindings asks the owner of each doing/waiting task whose every
// check holds (ticked or struck) and whose every child is done or dropped
// whether it is done (f, bg1rd8; Jørgen on 44w1ev: all 5 checks and 3
// children done, a day in waiting on a long-landed "landing slot", and
// nothing told anyone). The wait says what the task waited on, not that
// work is left, so a stale one hides a finished task. Only when hug task
// done would go through (no open dependency or board condition), so the
// fix is one hug takes. Not doing/active (the session's own board line
// says close it, and the check that ticks the last item says so too), not
// asked or review (a person decides), not a wait that asks a person later
// (hug task ask --after), and not before the wait is IdleAfter old: parking
// a finished task for a review or CI run is normal, a wait nobody ends is
// not. A wait without a readable waiting_since counts as old.
func ChecksHoldFindings(ts []Task, now time.Time) []Finding {
	var kids map[string][]Task
	var out []Finding
	for _, t := range ts {
		if t.State != "doing" || t.Stage != "waiting" || len(t.Acceptance) == 0 || strings.TrimSpace(t.Fields["waiting_ask_what"]) != "" {
			continue
		}
		since, err := time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(t.Fields["waiting_since"]), time.Local)
		if err == nil && now.Sub(since) < IdleAfter {
			continue
		}
		holds := true
		for _, a := range t.Acceptance {
			holds = holds && a.Settled()
		}
		if !holds || len(Blocked(t, ts)) > 0 || len(Unmet(t, ts, true)) > 0 {
			continue
		}
		if kids == nil {
			kids = ChildIndex(ts)
		}
		children := kids[strings.ToLower(t.ID)]
		for _, c := range children {
			holds = holds && (c.State == "done" || c.State == "dropped")
		}
		if !holds {
			continue
		}
		who := strings.Join(t.Assignee, ", ")
		if who == "" {
			who = "nobody"
		}
		waited := ""
		if err == nil {
			waited = " for " + hours(now.Sub(since))
		}
		n := itoa(len(t.Acceptance))
		what := n + " of " + n + " checks are ticked or struck"
		if len(children) > 0 {
			c := itoa(len(children))
			what += " and " + c + " of " + c + " children are done"
		}
		out = append(out, Finding{Path: t.Path, Line: 1, Code: "checks-all-hold",
			Problem: "every check holds, but doing/waiting on " + oneLineFix(t.Fields["waiting_on"]) + waited + ": " + what + "; " + who + ": is it done?",
			Fix:     "hug task done " + t.ID})
	}
	return out
}

// stageSettled are the question codes a stage that says why (waiting,
// asked, review) settles: noteQuestion raises none of them on such a task.
// So asking the owner is how such a question is answered, and hug task ask
// is not held back by it (gpyqq9, crab on 4vd1mz: ask refused by the
// shipped-not-done finding that asking settles, and wait pointed back to
// ask).
var stageSettled = map[string]bool{"shipped-not-done": true}

// StageSettles reports whether moving a task to a stage that says why it
// is not done (hug task ask: doing/asked) settles the finding code.
func StageSettles(code string) bool { return stageSettled[code] }

// stageSaysWhy reports whether t's stage already says why it is not done:
// doing/waiting, doing/asked (41hhgn) or review. doing/active and todo do
// not.
func stageSaysWhy(t Task) bool {
	switch t.Stage {
	case "waiting", "asked", "review":
		return true
	}
	return ownHolds(t, "asked") || ownHolds(t, "waiting") || ownHolds(t, "review")
}

// andList joins "7", "8", "9" as "7, 8 and 9".
func andList(xs []string) string {
	if len(xs) < 2 {
		return strings.Join(xs, "")
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " and " + xs[len(xs)-1]
}

func hours(d time.Duration) string {
	h := int(d.Hours())
	if h < 1 {
		return "under an hour"
	}
	if h == 1 {
		return "1 hour"
	}
	return strconv.Itoa(h) + " hours"
}

// TopQuestions keeps, among the question findings in fs, only the highest
// ranked per ticket, and every other finding as it is.
func TopQuestions(fs []Finding) []Finding {
	best := map[string]int{}
	for _, f := range fs {
		code := f.Code
		if code == "" {
			code = FindingCode(f.Problem)
		}
		if r, ok := questionRank[code]; ok {
			if b, seen := best[f.Path]; !seen || r < b {
				best[f.Path] = r
			}
		}
	}
	out := fs[:0:0]
	for _, f := range fs {
		code := f.Code
		if code == "" {
			code = FindingCode(f.Problem)
		}
		if r, ok := questionRank[code]; ok && r != best[f.Path] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// TouchedSince maps each ticket id to the newest commit time touching it,
// from git log lines "UNIXTIME<TAB>SUBJECT<TAB>FILE..." of the task branch
// (a file named PREFIX.ID.SLUG.md) and of main (a subject naming the id,
// SubjectIDs).
func TouchedSince(lines []string) map[string]time.Time {
	out := map[string]time.Time{}
	mark := func(id string, at time.Time) {
		if at.After(out[id]) {
			out[id] = at
		}
	}
	for _, l := range lines {
		parts := strings.Split(l, "\t")
		if len(parts) < 2 {
			continue
		}
		sec, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || sec <= 0 {
			continue
		}
		at := time.Unix(sec, 0)
		for _, id := range SubjectIDs(parts[1]) {
			mark(id, at)
		}
		// hug's own commits on the task branch: "hug task VERB ID".
		if f := strings.Fields(parts[1]); len(f) >= 4 && f[0] == "hug" && f[1] == "task" && idWord.MatchString(f[3]) {
			mark(f[3], at)
		}
		for _, file := range parts[2:] {
			if id, ok := IDOf(file); ok {
				mark(id, at)
			}
		}
	}
	return out
}
