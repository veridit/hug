package task

import "strings"

// One ticket in doing per agent (n2hq8p): doing means someone is on it now,
// so a second take by the same name is refused while the first is active.
// A task waiting (on CI, a reply, a landing turn), in review or depending on
// another task is not work in hand and does not count. The refusal is the
// fix: it names the active ticket and the ways to free the taker; a take
// also offers the detour (4tycwd): the active ticket depends on the one
// being taken, with a note of where it was, and comes back by itself when
// that one is done.
//
// Tasks that ship together (byj479, Jørgen 20:37Z on sql_saga) are both
// doing: hug task take B --with A, A the taker's active task, records
// doing_with: A on B. B is on hand with A, not a second thing in hand, so
// it does not count as active; the link is stated, never guessed, and
// each closes on its own.

// Active is the first task other than except that name has in doing with
// the stage active: the work name has in hand. ts is the board. A task
// taken with another (doing_with) is held alongside it and is not the one
// while that other is name's active task; once it is done, parked or
// given back, the one taken with it is the work in hand again.
func Active(name, except string, ts []Task) (Task, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Task{}, false
	}
	for _, t := range ts {
		if t.ID == except || t.State != "doing" || !contains(t.Assignee, name) || HeldWith(t, ts) != "" {
			continue
		}
		if Stage(t, ts) == "active" {
			return t, true
		}
	}
	return Task{}, false
}

// HeldWith is the task t is held alongside now: its doing_with, while
// that task is doing, active and held by t's owner. "" otherwise.
func HeldWith(t Task, ts []Task) string {
	w := TakenWith(t)
	if w == "" {
		return ""
	}
	for _, o := range ts {
		if o.ID == w && o.State == "doing" && Stage(o, ts) == "active" {
			for _, a := range t.Assignee {
				if contains(o.Assignee, a) {
					return w
				}
			}
		}
	}
	return ""
}

// TakenWith is the task t was taken with (doing_with), "" for none.
func TakenWith(t Task) string {
	if t.State != "doing" {
		return ""
	}
	return strings.TrimSpace(t.Fields["doing_with"])
}

// BusyFixes are the three ways out of a refused take, for the active task
// id: finish it, park it, or give it back.
func BusyFixes(id string) []string {
	return []string{
		"hug task done " + id,
		"hug task wait " + id + ` "on what"`,
		"hug task set " + id + " --state todo && hug task note " + id + ` "why it goes back"`,
	}
}

// MergedFixes are the ways out when the active task id has every check
// settled (y8we58, cd63d1 r1): it is green and waits for its branch to
// land, so the wait on the merge comes first, in place of the generic
// wait "on what"; then done, and back to todo.
func MergedFixes(id string) []string {
	return []string{
		"hug task wait " + id + ` "branch ` + id + ` merged"`,
		"hug task done " + id,
		"hug task set " + id + " --state todo && hug task note " + id + ` "why it goes back"`,
	}
}

// AllSettled: t has acceptance checks and every one is ticked or struck.
func AllSettled(t Task) bool {
	for _, a := range t.Acceptance {
		if !a.Settled() {
			return false
		}
	}
	return len(t.Acceptance) > 0
}

// DetourFix is the fourth way out of a refused take of ref (4tycwd): go
// to ref as a detour; active waits on it as a dependency, so it is not
// active work, and is active again when ref is done.
func DetourFix(active, ref string) string {
	return "hug task dep " + active + " --add " + ref + " && hug task note " + active + ` "where I was"`
}

// WithFix is the fifth way out (byj479): take ref with active, when they
// ship together. who is appended as the caller writes it (" --model M
// --as NAME"), "" for none.
func WithFix(ref, active, who string) string {
	return "hug task take " + ref + " --with " + active + who
}

// WithWhen says when to take with rather than detour: the refusal says
// it, so the agent picks the right fix.
const WithWhen = "if " + "they ship together (one change or release), take it with the active one; if it is a detour, the active one waits on it"

// Taker is how a fix line names who takes: " --model MODEL --as NAME",
// each part only when known. The CLI fills it so the fix runs as is.
type Taker struct{ Name, Model string }

func (w Taker) args() string {
	out := ""
	if w.Model != "" {
		out += " --model " + w.Model
	} else {
		out += " --model MODEL"
	}
	if w.Name != "" && w.Name != w.Model {
		out += " --as " + w.Name
	}
	return out
}

// busy is the refusal of taking ref while name works on active. detour:
// ref is a task to take (not one to resume), so the detour and the take
// with are ways out.
func busy(ref Task, who Taker, active Task, detour bool) *Error {
	fixes := BusyFixes(active.ID)
	if AllSettled(active) {
		fixes = MergedFixes(active.ID)
	}
	value := who.Name + " is on " + active.ID + " (" + active.Title + "): one ticket in doing at a time"
	if detour {
		fixes = append(fixes, DetourFix(active.ID, ref.ID), WithFix(ref.ID, active.ID, who.args()))
		value += "; " + WithWhen
	}
	return &Error{Kind: "busy", ID: ref.ID, Field: "doing", Value: value, Fix: fixes[0], Fixes: fixes}
}

// withRefused is the refusal of taking ref --with with, which is not
// who's active task: the fix takes it with the active one, or plainly
// when nothing is active.
func withRefused(ref Task, who Taker, with string, active Task, ok bool) *Error {
	fix := "hug task take " + ref.ID + who.args()
	problem := with + " is not what " + who.Name + " is on"
	if ok {
		fix = WithFix(ref.ID, active.ID, who.args())
		problem += ": " + active.ID + " (" + active.Title + ") is"
	} else {
		problem += ": nothing is active, so take it on its own"
	}
	return &Error{Kind: "usage", ID: ref.ID, Value: problem, Fix: fix}
}

// TakeHint is the fix line for who ticking or writing on t, a todo task
// they have not taken (byj479 check 3): take it with their active task
// (they ship together), or take it when nothing is active. "" when t is
// not todo, someone else holds it, or who is unknown.
func TakeHint(t Task, who Taker, ts []Task) string {
	if t.State != "todo" || strings.TrimSpace(who.Name) == "" || (len(t.Assignee) > 0 && !contains(t.Assignee, who.Name)) {
		return ""
	}
	if a, ok := Active(who.Name, t.ID, ts); ok {
		return WithFix(t.ID, a.ID, who.args())
	}
	return "hug task take " + t.ID + who.args()
}
