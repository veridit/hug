package task

import (
	"fmt"
	"github.com/veridit/hug/internal/sharefile"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Reflection is an agent's answer, at done, to "did the instructions guide
// you well?" (tb0dra). It lives in the task's Notes as one line:
//
//	Reflection (open, 2026-11-17, guppy): the AGENTS.md line "..." made me ...
//
// and is resolved by rewriting open to applied or dismissed. A done that
// came without an answer leaves "Reflection (unanswered, DATE, WHO)" with
// no text (2k68eg), so the answer rate shows; it is not a reflection to
// resolve, and a later answer takes its place.
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

// Unanswered is the state of a done that came without an answer.
const Unanswered = "unanswered"

var reflectionLine = regexp.MustCompile(`^Reflection \((open|applied|dismissed|unanswered), (\d{4}-\d{2}-\d{2})(?:, ([^)]*))?\)(?:: (.*))?$`)

// quoteMarks pairs each opening mark with its closing one, and the shortest
// span that counts as quoting an instruction.
var quoteMarks = []struct {
	open, close rune
	min         int
}{{'"', '"', 3}, {'“', '”', 3}, {'\'', '\'', 3}}

// FirstQuote is the first span of text quoted on purpose: the opening mark
// does not follow a letter or digit and the closing mark is not followed by
// one, so the apostrophe in AGENTS.md's or Jørgen's never opens a quote and
// the one in don't never closes it (2k68eg: 10 of 28 instruction fields on
// hug's board were fragments cut at such an apostrophe).
func FirstQuote(text string) string {
	best, bestAt := "", len(text)+1
	for _, q := range quoteMarks {
		for i := 0; i < len(text) && i < bestAt; {
			r, w := utf8.DecodeRuneInString(text[i:])
			if r != q.open || wordRune(lastRune(text[:i])) {
				i += w
				continue
			}
			start := i + w
			end := closeAt(text, start, q.close)
			if end < 0 {
				break
			}
			if span := text[start:end]; utf8.RuneCountInString(span) >= q.min && strings.TrimSpace(span) == span {
				best, bestAt = span, i
				break
			}
			i = start
		}
	}
	return best
}

// closeAt is the offset of the first close mark at or after from that is
// not followed by a letter or digit, or -1.
func closeAt(text string, from int, close rune) int {
	for j := from; j < len(text); {
		r, w := utf8.DecodeRuneInString(text[j:])
		if r == close {
			next, _ := utf8.DecodeRuneInString(text[j+w:])
			if j+w >= len(text) || !wordRune(next) {
				return j
			}
		}
		j += w
	}
	return -1
}

func lastRune(s string) rune {
	r, _ := utf8.DecodeLastRuneInString(s)
	return r
}
func wordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// NothingToReflect: the agent answered that the instructions served it;
// that answer is welcome and needs no note.
func NothingToReflect(text string) bool {
	t := strings.ToLower(strings.Trim(strings.TrimSpace(text), ".!\"'"))
	return t == "nothing" || t == "none" || t == "no" || t == "nothing to report" || t == "-"
}

// ReflectionNote is the Notes line for text, written today by who.
func ReflectionNote(text, who string, now time.Time) string {
	text = strings.Join(strings.Fields(text), " ")
	return reflectionHead("open", who, now) + ": " + text
}

// UnansweredNote is the Notes line a done without an answer leaves.
func UnansweredNote(who string, now time.Time) string {
	return reflectionHead(Unanswered, who, now)
}

func reflectionHead(state, who string, now time.Time) string {
	head := "Reflection (" + state + ", " + now.Format("2006-01-02")
	if who = strings.Join(strings.Fields(strings.ReplaceAll(who, ")", "")), " "); who != "" {
		head += ", " + who
	}
	return head + ")"
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
		r.Instruction = FirstQuote(r.Text)
		out = append(out, r)
	}
	return out
}

// Reflections lists the reflections on the board in board order, open ones
// only unless all. Unanswered dones are never listed; they come back as
// the ids of the tasks that hold one.
func (s *Store) Reflections(all bool) ([]Reflection, []string, error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, nil, e
	}
	var out []Reflection
	var unanswered []string
	for _, t := range ts {
		b, e := sharefile.ReadFile(filepath.Join(s.Root, filepath.FromSlash(t.Path)))
		if e != nil || !strings.Contains(string(b), "Reflection (") {
			continue
		}
		for _, r := range ParseReflections(t, string(b)) {
			if r.State == Unanswered {
				unanswered = append(unanswered, r.ID)
			} else if all || r.State == "open" {
				out = append(out, r)
			}
		}
	}
	return out, unanswered, nil
}

// ReflectionGroup is the reflections that name one instruction (Key), or
// none (Key "").
type ReflectionGroup struct {
	Key         string       `json:"key"`
	Reflections []Reflection `json:"reflections"`
}

var hugCommand = regexp.MustCompile(`\bhug (task|mem) [a-z]+\b`)

// ReflectionKey is what a reflection is about, for grouping: the hug
// command inside its quoted instruction, else the quote, else the first hug
// command in the text, else "". No model: text only (2k68eg principle).
func ReflectionKey(r Reflection) string {
	if r.Instruction != "" {
		if c := hugCommand.FindString(r.Instruction); c != "" {
			return c
		}
		return r.Instruction
	}
	return hugCommand.FindString(r.Text)
}

// GroupReflections groups rs by ReflectionKey: the largest group first
// (ties in first-seen order), the reflections in board order within one,
// and those that name nothing last.
func GroupReflections(rs []Reflection) []ReflectionGroup {
	var gs []ReflectionGroup
	at := map[string]int{}
	for _, r := range rs {
		k := ReflectionKey(r)
		i, ok := at[k]
		if !ok {
			i = len(gs)
			at[k] = i
			gs = append(gs, ReflectionGroup{Key: k})
		}
		gs[i].Reflections = append(gs[i].Reflections, r)
	}
	sort.SliceStable(gs, func(i, j int) bool {
		if (gs[i].Key == "") != (gs[j].Key == "") {
			return gs[j].Key == ""
		}
		return len(gs[i].Reflections) > len(gs[j].Reflections)
	})
	return gs
}

// SetReflection records done's answer on ref: text as an open reflection,
// or, when text is empty, that done came without one. A task holds at most
// one unanswered line: a later answer, or "nothing", takes its place. A
// plain done writes nothing when the task already has its line or an
// answer (a repeated done).
func (s *Store) SetReflection(ref, text, who string, now time.Time) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	lines := append([]string(nil), d.lines...)
	drop, answered := -1, false
	for i, l := range lines {
		m := reflectionLine.FindStringSubmatch(strings.TrimSpace(strings.TrimRight(l, "\r\n")))
		if m != nil && m[1] == Unanswered && drop < 0 {
			drop = i
		} else if m != nil {
			answered = true
		}
	}
	empty := strings.TrimSpace(text) == ""
	answer := !empty && !NothingToReflect(text)
	unchanged := Result{Task: d.Task, Next: hint(d.Task)}
	if empty && (drop >= 0 || answered) || !answer && !empty && drop < 0 {
		return unchanged, nil
	}
	if drop >= 0 {
		d.replace(append(lines[:drop], lines[drop+1:]...))
	}
	if answer {
		d.Note(ReflectionNote(text, who, now))
	} else if empty {
		d.Note(UnansweredNote(who, now))
	}
	return s.finish(d, true)
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
		if m[1] == Unanswered {
			return Result{}, &Error{Kind: "unanswered-reflection", ID: d.Task.ID, Value: fmt.Sprintf("reflection %d of %s is a done without an answer, not a reflection to resolve", n, d.Task.ID), Fix: "hug task done " + d.Task.ID + " --reflect \"...\""}
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
