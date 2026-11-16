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
		if !a.Checked {
			open++
		}
	}
	switch action {
	case "new":
		// Not into todos: hug task next picks up work when it is time.
		return nil
	case "take":
		g := []string{}
		// One todo for the task, not one per check: the board holds the
		// checks, and the task stays doing through any detour.
		g = append(g, fmt.Sprintf("one todo for %s (not one per check); it stays doing through detours: hug task show %s brings you back", t.ID, t.ID))
		g = append(g, fmt.Sprintf("as each check holds: hug task check %s N, with evidence in hug task note %s \"...\"", t.ID, t.ID))
		return g
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
				if c.Parent == t.Parent && c.ID != t.ID && c.Status != "done" && c.Status != "dropped" {
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
