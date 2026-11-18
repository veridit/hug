// Package trigger is the situation a memory is for (jrj1vq): the storer
// states it at write time, when it knows the situation. A when-then memory
// is its note (the memory text, the THEN) plus one optional trigger:
//
//   - case: hard context, exact filters (scopes, paths, tools)
//   - when: the situation where an agent needs the note, as an activity
//   - when_not: look-alike situations where it does not help
//
// Every part is optional and a plain memory has no trigger at all. This
// package only defines the format; recall does not read it yet (jrj1vq
// check 3), so storing a trigger changes no recall result.
package trigger

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Case is the hard context a memory is bound to. Order carries no meaning:
// the canonical form sorts each list.
type Case struct {
	Scope []string `json:"scope,omitempty"` // projects or memory scopes
	Paths []string `json:"paths,omitempty"` // file paths or globs
	Tools []string `json:"tools,omitempty"` // tools, commands or services
}

// Trigger is the situation part of a when-then memory. The zero value and
// nil both mean "no trigger": a plain memory.
type Trigger struct {
	Case    *Case  `json:"case,omitempty"`
	When    string `json:"when,omitempty"`
	WhenNot string `json:"when_not,omitempty"`
}

// Limits keep a trigger a situation, not a second note.
const (
	MaxText  = 500 // runes in when or when_not
	MaxItems = 16  // entries in one case list
	MaxItem  = 200 // runes in one case entry
)

// Fix is the line a malformed trigger's error ends with.
const Fix = `fix: hug mem remember --when "SITUATION" [--when-not "LOOK-ALIKE"] [--case-path PATH] [--case-tool TOOL] [--case-scope SCOPE] "TEXT"`

// Canonical returns t with whitespace collapsed, case lists trimmed, sorted
// and deduplicated, and empty parts dropped; nil when nothing is left. It
// checks the limits. Two triggers are the same exactly when their
// canonical encodings are equal.
func Canonical(t *Trigger) (*Trigger, error) {
	if t == nil {
		return nil, nil
	}
	out := Trigger{When: squash(t.When), WhenNot: squash(t.WhenNot)}
	for _, p := range [][2]string{{"when", out.When}, {"when_not", out.WhenNot}} {
		if n := utf8.RuneCountInString(p[1]); n > MaxText {
			return nil, fmt.Errorf("trigger %s is %d characters, at most %d: state the situation in one sentence, the note goes in the memory text\n%s", p[0], n, MaxText, Fix)
		}
	}
	if t.Case != nil {
		c := Case{}
		var err error
		if c.Scope, err = list("case scope", t.Case.Scope); err != nil {
			return nil, err
		}
		if c.Paths, err = list("case path", t.Case.Paths); err != nil {
			return nil, err
		}
		if c.Tools, err = list("case tool", t.Case.Tools); err != nil {
			return nil, err
		}
		if len(c.Scope)+len(c.Paths)+len(c.Tools) > 0 {
			out.Case = &c
		}
	}
	if out.Case == nil && out.When == "" && out.WhenNot == "" {
		return nil, nil
	}
	return &out, nil
}

func squash(s string) string { return strings.Join(strings.Fields(s), " ") }

func list(name string, in []string) ([]string, error) {
	var out []string
	for _, s := range in {
		if s = strings.TrimSpace(s); s == "" || slices.Contains(out, s) {
			continue
		}
		if n := utf8.RuneCountInString(s); n > MaxItem {
			return nil, fmt.Errorf("trigger %s %q... is %d characters, at most %d\n%s", name, string([]rune(s)[:20]), n, MaxItem, Fix)
		}
		out = append(out, s)
	}
	if len(out) > MaxItems {
		return nil, fmt.Errorf("trigger has %d of %s, at most %d\n%s", len(out), name, MaxItems, Fix)
	}
	slices.Sort(out)
	return out, nil
}

// Encode is the canonical JSON of t, "" for no trigger. It is what the
// index stores (units.mtrigger) and what equality compares.
func Encode(t *Trigger) string {
	c, err := Canonical(t)
	if err != nil || c == nil {
		// An over-long trigger never reaches storage (remember checks it
		// first); encoding it as-is keeps what a hand-edited file holds.
		if t == nil || err == nil {
			return ""
		}
		c = t
	}
	b, _ := json.Marshal(c)
	return string(b)
}

// Decode reads an encoding; "" is no trigger.
func Decode(s string) (*Trigger, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var t Trigger
	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return nil, fmt.Errorf("trigger %q: %w", s, err)
	}
	return &t, nil
}

// Equal reports whether a and b are the same trigger (nil equals empty).
func Equal(a, b *Trigger) bool { return Encode(a) == Encode(b) }

// Empty reports whether t states nothing.
func Empty(t *Trigger) bool { return Encode(t) == "" }

// None is the trigger of a memory whose trigger was cleared on purpose
// (a supersede with --no-trigger). It is stored and synced as
// "trigger":{} so the clear travels to other clones as "replace with
// none"; a missing key (a plain memory, or a line rewritten by a hug that
// does not know triggers) never clears one. The index sees no trigger.
func None() *Trigger { return &Trigger{} }

// Cleared reports whether t is None: present, stating nothing.
func Cleared(t *Trigger) bool { return t != nil && Empty(t) }

// Stored is t as kept on disk: canonical, None kept as None, nil as nil.
func Stored(t *Trigger) (*Trigger, error) {
	c, err := Canonical(t)
	if err != nil {
		return nil, err
	}
	if c == nil && t != nil {
		return None(), nil
	}
	return c, nil
}

// Clone copies t deeply.
func Clone(t *Trigger) *Trigger {
	if t == nil {
		return nil
	}
	out := *t
	if t.Case != nil {
		c := Case{Scope: slices.Clone(t.Case.Scope), Paths: slices.Clone(t.Case.Paths), Tools: slices.Clone(t.Case.Tools)}
		out.Case = &c
	}
	return &out
}

// Lines renders t for people, one "label: value" line per part, none for
// no trigger (hug mem show).
func Lines(t *Trigger) []string {
	c, _ := Canonical(t)
	if c == nil {
		return nil
	}
	var out []string
	if c.Case != nil {
		var parts []string
		for _, p := range []struct {
			name string
			vals []string
		}{{"scope", c.Case.Scope}, {"paths", c.Case.Paths}, {"tools", c.Case.Tools}} {
			if len(p.vals) > 0 {
				parts = append(parts, p.name+" "+strings.Join(p.vals, ", "))
			}
		}
		out = append(out, "case: "+strings.Join(parts, "; "))
	}
	if c.When != "" {
		out = append(out, "when: "+c.When)
	}
	if c.WhenNot != "" {
		out = append(out, "when not: "+c.WhenNot)
	}
	return out
}
