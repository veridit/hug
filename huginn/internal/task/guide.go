package task

import (
	"fmt"
	"strings"
)

// Guide is what the agent should do right now, at this state change, for
// this task: short, concrete, with the exact commands. It replaces doctrine
// ("always keep the board current") with the right reminder at the right
// moment (2026-11-16, user). Empty when there is nothing to add.
func Guide(action string, t Task, ts []Task) []string {
	open := 0
	for _, a := range t.Acceptance {
		if !a.Settled() {
			open++
		}
	}
	switch action {
	case "new":
		// Not into todos: hug task next picks up work when it is time.
		return nil
	case "take":
		// Todos are the agent's own next steps on this ticket; the ticket
		// holds checks, details, waits and dependencies (tzc9hb, Jørgen
		// 2026-11-16: todo lists that copied the board became walls of
		// "blocked" items). Two lines at most.
		if len(t.Acceptance) == 0 {
			// No checks yet (erv680 check 1): they are its work items
			// (Jørgen 2026-11-26: one list), written before the work.
			return []string{
				fmt.Sprintf("%s has no acceptance checks: write them first, one observable check each; they are its work items: %s (again for each check)", t.ID, AcceptFix(t.ID)),
				fmt.Sprintf("then, as each check holds: hug task check %s N \"evidence\"", t.ID),
			}
		}
		return []string{
			fmt.Sprintf("todos: your own next steps on %s, each named with the id (\"%s: ...\"); a wait, a dependency or another agent's work is not a todo: hug task note %s \"...\", hug task dep %s --add OTHER", t.ID, t.ID, t.ID, t.ID),
			fmt.Sprintf("as each check holds: hug task check %s N \"evidence\"; todos done: hug task done %s, then hug task next", t.ID, t.ID),
		}
	case "check":
		// next: already names the following check or done; add only the
		// todo bookkeeping that next: cannot say.
		if open == 0 {
			return []string{"every check holds: close it, and tick its todo off"}
		}
		return nil
	case "done":
		var g []string
		if t.Parent != "" {
			sib := 0
			for _, c := range ts {
				if c.Parent == t.Parent && c.ID != t.ID && c.State != "done" && c.State != "dropped" {
					sib++
				}
			}
			if sib == 0 {
				g = append(g, "that was the last open task in epic "+t.Parent+": hug task done "+t.Parent)
			} else {
				g = append(g, fmt.Sprintf("epic %s has %d open task(s): hug task next --epic %s", t.Parent, sib, t.Parent))
			}
		}
		return g
	case "drop":
		return []string{"note why it was dropped: hug task note " + t.ID + " \"...\"", "tick its todo off"}
	case "promote":
		return []string{"the source item ticks itself when " + t.ID + " is done; hug task next picks it up in turn"}
	}
	return nil
}

func oneLineShort(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
