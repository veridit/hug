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

// Active is the first task other than except that name has in doing with
// the stage active: the work name has in hand. ts is the board.
func Active(name, except string, ts []Task) (Task, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Task{}, false
	}
	for _, t := range ts {
		if t.ID == except || t.State != "doing" || !contains(t.Assignee, name) {
			continue
		}
		if Stage(t, ts) == "active" {
			return t, true
		}
	}
	return Task{}, false
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

// DetourFix is the fourth way out of a refused take of ref (4tycwd): go
// to ref as a detour; active waits on it as a dependency, so it is not
// active work, and is active again when ref is done.
func DetourFix(active, ref string) string {
	return "hug task dep " + active + " --add " + ref + " && hug task note " + active + ` "where I was"`
}

// busy is the refusal of taking ref while name works on active. detour:
// ref is a task to take (not one to resume), so the detour is a way out.
func busy(ref Task, name string, active Task, detour bool) *Error {
	fixes := BusyFixes(active.ID)
	if detour {
		fixes = append(fixes, DetourFix(active.ID, ref.ID))
	}
	return &Error{Kind: "busy", ID: ref.ID, Field: "doing", Value: name + " is on " + active.ID + " (" + active.Title + "): one ticket in doing at a time",
		Fix: fixes[0], Fixes: fixes}
}
