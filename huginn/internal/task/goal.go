package task

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// The Goal is one short sentence: where we aim (0pr03h, Jørgen 2026-11-19
// 11:26Z: "AI tends to bloat, and that loses focus. We have to fight for
// focus."). Everything else goes to Details. hug task new and set
// refuse a Goal that breaks it; lint warns on one already on disk and never
// rewrites it.

// MaxGoal is the longest Goal, in characters, new and set accept.
const MaxGoal = 140

// GoalRule says the rule as help, AGENTS.md and hug setup's template state
// it; a test pins it in each.
const GoalRule = "The Goal is one sentence of at most 140 characters; detail goes in Details."

// SectionsRule names each section's job in the order a task is written
// and its page reads (32ym01, Jørgen 2026-11-24 18:24Z: "What, Why, How,
// Notes (to explain those over)"; a trial, judged on clarity and
// shortness). AGENTS.md, hug setup's template, help new and MCP task_new
// say it in these words; a test pins each.
const SectionsRule = "Describe a task with Goal (" + GoalJob + "), Purpose (" + PurposeJob + "), Principle (" + PrincipleJob + "), Details (notes that explain those), then Acceptance."

// The jobs SectionsRule gives the sections; lint names a missing section
// with its job in the same words.
const (
	GoalJob      = "what: where we aim"
	PurposeJob   = "why"
	PrincipleJob = "how: what we do and do not do"
)

// OldSectionsRule is the sentence before SectionsRule (v0.7.103 to
// v0.7.111); hug setup lists replacing it as an open step.
const OldSectionsRule = "Describe a task with Goal (where we aim), Purpose (why), Principle (what we do and do not do), Details, then Acceptance."

// goalFlag is the fix for a Goal that breaks the rule.
const goalFlag = `--goal "one sentence" --details "..."`

// GoalProblem names how goal breaks the rule; ok is false when it holds
// (an empty Goal is missing-goal's, not this).
func GoalProblem(goal string) (Problem, bool) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return Problem{}, false
	}
	n, s := len([]rune(goal)), Sentences(goal)
	var what string
	switch {
	case n > MaxGoal && s > 1:
		what = fmt.Sprintf("%d characters in %d sentences", n, s)
	case n > MaxGoal:
		what = fmt.Sprintf("%d characters", n)
	case s > 1:
		what = fmt.Sprintf("%d sentences", s)
	default:
		return Problem{}, false
	}
	return Problem{"goal-too-long", "Goal is " + what + ": the Goal is one sentence of at most 140 characters; detail goes in Details", goalFlag}, true
}

// Sentences counts the sentences in text, conservatively: when unsure it
// is one sentence. A boundary is
//   - a blank line (a new paragraph), or
//   - '.', '!' or '?' (the last of a run, e.g. "?!" or "..."), optionally
//     followed by closing ) ] " ' ” ’, then whitespace, then an
//     uppercase letter (optionally after an opening ( [ " ' “ ‘ * _ `),
//
// unless the word ending in '.' is an abbreviation (e.g., i.e., etc., vs.,
// cf., no., Dr. ...), a dotted short form (U.S., e.g), one capital letter
// (an initial) or a number (a list marker). So "e.g. Foo", "v0.7.85",
// "main.go", "task:abc123" and "1. Do" never split; "it works. Then"
// does, "it works. then" does not.
func Sentences(text string) int {
	n := 0
	for _, p := range paragraph.Split(strings.TrimSpace(text), -1) {
		if p = strings.TrimSpace(p); p != "" {
			n += 1 + boundaries([]rune(p))
		}
	}
	return n
}

var paragraph = regexp.MustCompile(`\n[ \t\r]*\n`)

var dotted = regexp.MustCompile(`^([A-Za-z]{1,2}\.)+[A-Za-z]{1,2}$`)

var abbreviations = map[string]bool{"e.g": true, "i.e": true, "etc": true, "vs": true, "cf": true, "approx": true, "incl": true, "excl": true, "no": true, "nr": true, "mr": true, "mrs": true, "ms": true, "dr": true, "prof": true, "st": true, "jr": true, "sr": true, "fig": true, "eq": true, "ca": true, "al": true, "resp": true, "ex": true, "viz": true, "esp": true}

func boundaries(rs []rune) int {
	n := 0
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		if i+1 < len(rs) && strings.ContainsRune(".!?", rs[i+1]) {
			continue // only the last of a run ends a sentence
		}
		j := i + 1
		for j < len(rs) && strings.ContainsRune(`)]"'”’»`, rs[j]) {
			j++
		}
		if j >= len(rs) || !unicode.IsSpace(rs[j]) {
			continue
		}
		for j < len(rs) && unicode.IsSpace(rs[j]) {
			j++
		}
		for j < len(rs) && strings.ContainsRune("([\"'“‘«*_`", rs[j]) {
			j++
		}
		if j >= len(rs) || !unicode.IsUpper(rs[j]) {
			continue
		}
		if r == '.' && abbreviated(rs, i) {
			continue
		}
		n++
	}
	return n
}

// abbreviated reports whether the word ending at the '.' at i is an
// abbreviation, an initial or a number rather than a sentence's end.
func abbreviated(rs []rune, i int) bool {
	k := i
	for k > 0 && !unicode.IsSpace(rs[k-1]) {
		k--
	}
	word := strings.TrimLeft(string(rs[k:i]), "([\"'“‘«*_`")
	if word == "" {
		return true
	}
	if abbreviations[strings.ToLower(word)] || dotted.MatchString(word) {
		return true
	}
	w := []rune(word)
	if len(w) == 1 && unicode.IsUpper(w[0]) {
		return true
	}
	for _, c := range w {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
