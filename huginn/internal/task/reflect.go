package task

import (
	"fmt"
	"github.com/veridit/hug/internal/sharefile"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Reflection is an agent's answer, at done, to "did the instructions guide
// you well?" (tb0dra). It lives in the task's Notes as one line:
//
//	Reflection (open, 2026-11-17, guppy): the AGENTS.md line "..." made me ...
//
// and is resolved by rewriting open to applied or dismissed.
type Reflection struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	N           int    `json:"n"` // 1-based among the task's reflections
	State       string `json:"state"`
	Date        string `json:"date"`
	Who         string `json:"who,omitempty"`
	Text        string `json:"text"`
	Instruction string `json:"instruction,omitempty"` // the first quoted span of Text
}

// ReflectQuestion is the one question done asks.
const ReflectQuestion = "did the instructions (AGENTS.md, tool descriptions, hug messages) guide you well, or did one mislead you or leave a gap? Quote it and say what happened; \"nothing\" is a fine answer"

var reflectionLine = regexp.MustCompile(`^Reflection \((open|applied|dismissed), (\d{4}-\d{2}-\d{2})(?:, ([^)]*))?\): (.*)$`)
var quoted = regexp.MustCompile(`"([^"]{3,})"|“([^”]{3,})”|'([^']{6,})'`)

// NothingToReflect: the agent answered that the instructions served it;
// that answer is welcome and needs no note.
func NothingToReflect(text string) bool {
	t := strings.ToLower(strings.Trim(strings.TrimSpace(text), ".!\"'"))
	return t == "nothing" || t == "none" || t == "no" || t == "nothing to report" || t == "-"
}

// ReflectionNote is the Notes line for text, written today by who.
func ReflectionNote(text, who string, now time.Time) string {
	text = strings.Join(strings.Fields(text), " ")
	head := "Reflection (open, " + now.Format("2006-01-02")
	if who = strings.Join(strings.Fields(strings.ReplaceAll(who, ")", "")), " "); who != "" {
		head += ", " + who
	}
	return head + "): " + text
}

// ParseReflections reads the reflections in a task file's text.
func ParseReflections(t Task, text string) []Reflection {
	var out []Reflection
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		m := reflectionLine.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		r := Reflection{ID: t.ID, Title: t.Title, N: len(out) + 1, State: m[1], Date: m[2], Who: m[3], Text: m[4]}
		if q := quoted.FindStringSubmatch(r.Text); q != nil {
			r.Instruction = q[1] + q[2] + q[3]
		}
		out = append(out, r)
	}
	return out
}

// Reflections lists the reflections on the board in board order, open ones
// only unless all. It reads each task file once.
func (s *Store) Reflections(all bool) ([]Reflection, error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, e
	}
	var out []Reflection
	for _, t := range ts {
		b, e := sharefile.ReadFile(filepath.Join(s.Root, filepath.FromSlash(t.Path)))
		if e != nil || !strings.Contains(string(b), "Reflection (") {
			continue
		}
		for _, r := range ParseReflections(t, string(b)) {
			if all || r.State == "open" {
				out = append(out, r)
			}
		}
	}
	return out, nil
}

// ResolveReflection marks reflection n of ref applied or dismissed.
func (s *Store) ResolveReflection(ref string, n int, state string) (Result, error) {
	if state != "applied" && state != "dismissed" {
		return Result{}, &Error{Kind: "invalid-value", ID: ref, Value: state, Fix: fmt.Sprintf("hug task reflections resolve %s %d applied (or dismissed)", ref, n)}
	}
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	lines := append([]string(nil), d.lines...)
	seen := 0
	for i, l := range lines {
		m := reflectionLine.FindStringSubmatch(strings.TrimSpace(strings.TrimRight(l, "\r\n")))
		if m == nil {
			continue
		}
		seen++
		if seen != n {
			continue
		}
		if m[1] == state {
			return Result{Task: d.Task, Next: "hug task reflections"}, nil
		}
		lines[i] = strings.Replace(l, "Reflection ("+m[1]+",", "Reflection ("+state+",", 1)
		d.replace(lines)
		r, e := s.finish(d, true)
		r.Next = "hug task reflections"
		return r, e
	}
	return Result{}, &Error{Kind: "no-reflection", ID: d.Task.ID, Value: fmt.Sprintf("task %s has %d reflection(s), not %d", d.Task.ID, seen, n), Fix: "hug task reflections --all"}
}
