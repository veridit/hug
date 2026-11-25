package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

// The description pattern, one check list for writes and lint (fkpyg3):
// what hug task new and set accept, lint never flags later. Each problem
// names the exact field and comes with its part of the one combined fix.

// Problem is one thing a task's description lacks.
type Problem struct {
	Code string `json:"code"`           // missing-goal, no-acceptance, missing-section, long-title
	Text string `json:"problem"`        // what is wrong, naming the field
	Flag string `json:"flag,omitempty"` // the set/new flag that fixes it, e.g. --purpose "..."
}

// DescriptionProblems checks a task's title, Goal, Purpose, Principle and
// Acceptance, all of them, never stopping at the first.
func DescriptionProblems(title, goal string, hasPurpose, hasPrinciple bool, acceptance int) []Problem {
	var out []Problem
	if strings.TrimSpace(title) == "" {
		out = append(out, Problem{"missing-field", "missing title", ""})
	} else if n := len([]rune(title)); n > MaxTitle {
		out = append(out, Problem{"long-title", fmt.Sprintf("title is %d characters (max %d): move detail into the Goal", n, MaxTitle), `--title "..."`})
	}
	if strings.TrimSpace(goal) == "" {
		out = append(out, Problem{"missing-goal", "missing Goal", `--goal "..."`})
	} else if p, ok := GoalProblem(goal); ok {
		out = append(out, p)
	}
	if !hasPurpose {
		out = append(out, Problem{"missing-section", "missing section Purpose (" + PurposeJob + ")", `--purpose "..."`})
	}
	if !hasPrinciple {
		out = append(out, Problem{"missing-section", "missing section Principle (" + PrincipleJob + ")", `--principle "..."`})
	}
	if acceptance == 0 {
		out = append(out, Problem{"no-acceptance", "Acceptance requires at least one item", `--accept "observable check"`})
	}
	return out
}

// Is reports whether p is about the given flag's field.
func (p Problem) Is(flag string) bool { return strings.HasPrefix(p.Flag, flag+" ") }

// newProblems runs the description check list on a new task: every
// problem at once, each naming its flag, with one fix: the same command
// with every missing part as a "..." slot. Purpose and Principle are
// required when o.Pattern is set (hug task new and MCP task_new); the TUI's
// one-line add and promote only need title, Goal and Acceptance, and lint
// then names what is left with its own one fix.
func newProblems(o NewOptions) error {
	filled := 0
	for _, a := range o.Acceptance {
		if strings.TrimSpace(a) != "" {
			filled++
		}
	}
	ps := DescriptionProblems(o.Title, o.Goal, !o.Pattern || strings.TrimSpace(o.Purpose) != "", !o.Pattern || strings.TrimSpace(o.Principle) != "", filled)
	if !o.Pattern {
		// The TUI's one-line add and promote keep any title and Goal; lint
		// names a long one with its fix.
		kept := ps[:0]
		for _, p := range ps {
			if p.Code != "long-title" && p.Code != "goal-too-long" {
				kept = append(kept, p)
			}
		}
		ps = kept
	}
	if filled != len(o.Acceptance) && filled > 0 {
		ps = append(ps, Problem{"empty-acceptance", "an --accept value is empty", `--accept "observable check"`})
	}
	if len(ps) == 0 {
		return nil
	}
	q := func(s string) string { b, _ := json.Marshal(strings.TrimSpace(s)); return string(b) }
	title := `"title"`
	if strings.TrimSpace(o.Title) != "" && !hasCode(ps, "long-title") {
		title = q(o.Title)
	} else if strings.TrimSpace(o.Title) != "" {
		title = `"short title"`
	}
	fix := "hug task new " + title
	longGoal := hasCode(ps, "goal-too-long")
	for _, part := range []struct{ flag, value string }{{"--goal", o.Goal}, {"--purpose", o.Purpose}, {"--principle", o.Principle}, {"--details", o.Details}} {
		if longGoal && part.flag == "--goal" {
			// One sentence here, the rest of it into Details.
			fix += ` --goal "one sentence"`
		} else if longGoal && part.flag == "--details" {
			fix += ` --details "..."`
		} else if strings.TrimSpace(part.value) != "" {
			fix += " " + part.flag + " " + q(part.value)
		} else if part.flag != "--details" && (o.Pattern || part.flag == "--goal") {
			fix += " " + part.flag + ` "..."`
		}
	}
	for _, a := range o.Acceptance {
		if strings.TrimSpace(a) != "" {
			fix += " --accept " + q(a)
		}
	}
	if filled == 0 {
		fix += ` --accept "observable check"`
	}
	var texts []string
	for _, p := range ps {
		texts = append(texts, p.Text)
	}
	return &Error{Kind: "incomplete-task", Value: strings.Join(texts, "; "), Problems: ps, Fix: fix}
}

func hasCode(ps []Problem, code string) bool {
	for _, p := range ps {
		if p.Code == code {
			return true
		}
	}
	return false
}

// Pattern reports whether a task is held to the description pattern: every
// open task (fkpyg3, Jørgen 2026-11-17: the old cutoff hid it on 1,215 of
// seid-lang's 1,236 open tasks). Done and dropped tasks are history.
func Pattern(t Task) bool { return t.State != "done" && t.State != "dropped" }
