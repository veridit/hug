package task

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// A question to a person is asked, never waited on (agxqax, Jørgen
// 2026-11-18 on Statbus): five waiting tasks there were questions for him
// ("owner: check 4 on Norway rc.02 ...", "the owner's decision on hug
// upgrade --apply"), so none was in his ASKED, and one was asked of
// "jorgen", a name hug did not know, so it was in nobody's. hug refuses
// those shapes when they are written and lint flags them on a board
// written before, each with the ask that says it.

// foldName is name as hug compares names: lower case, accents and the
// Nordic letters spelled the way they are typed without them (Jørgen ->
// jorgen, Søberg -> soberg, Åse -> ase), and only letters and digits.
func foldName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if s, ok := nameFold[r]; ok {
			b.WriteString(s)
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// nameFold spells letters people leave out of a typed name.
var nameFold = map[rune]string{
	'ø': "o", 'ö': "o", 'ó': "o", 'ò': "o", 'ô': "o", 'õ': "o",
	'æ': "ae", 'ä': "a", 'å': "a", 'á': "a", 'à': "a", 'â': "a", 'ã': "a",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
	'í': "i", 'ì': "i", 'î': "i", 'ï': "i",
	'ú': "u", 'ù': "u", 'û': "u", 'ü': "u",
	'ñ': "n", 'ç': "c", 'ß': "ss", 'ý': "y",
}

// PersonMatch is the person name stands for on this board: their short
// name, an alias, their long name, or its first word (Jørgen, jorgen),
// compared folded (foldName). A first word two people share stands for
// neither. ok is false when name is nobody here.
func PersonMatch(people []Person, name string) (Person, bool) {
	if p, ok := PersonByName(people, name); ok {
		return p, true
	}
	n := foldName(strings.TrimPrefix(strings.TrimSpace(name), "@"))
	if n == "" {
		return Person{}, false
	}
	for _, p := range people {
		if foldName(p.Short) == n || p.Name != "" && foldName(p.Name) == n {
			return p, true
		}
		for _, a := range p.Alias {
			if foldName(a) == n {
				return p, true
			}
		}
	}
	var found []Person
	for _, p := range people {
		if first, _, _ := strings.Cut(strings.TrimSpace(p.Name), " "); first != "" && foldName(first) == n {
			found = append(found, p)
		}
	}
	if len(found) == 1 {
		return found[0], true
	}
	return Person{}, false
}

// Owner is who "the owner" means on this board: the first person in
// hug.toml [people]. ok is false with no people.
func Owner(people []Person) (Person, bool) {
	if len(people) == 0 {
		return Person{}, false
	}
	return people[0], true
}

// ownerWords are what agents call the board's person in a wait. "user"
// is not one: "the user table" is a thing.
var ownerWords = map[string]bool{"owner": true, "the owner": true}

// addressee matches a wait that opens by naming who it is for: "owner:",
// "jhf: check 2: ...", "Jørgen - approve ...".
var addressee = regexp.MustCompile(`^\s*@?([\p{L}\p{N}][\p{L}\p{N} ._-]{0,40}?)\s*(?::|\s-\s)\s*(.*)$`)

// lead is what may open a wait before who it is for: "check N:".
const lead = `^\s*(?:check\s+#?\d+\s*[:,-]\s*)?`

// possessive is a wait that opens on "NAME's" (straight or curly
// apostrophe) and what a person gives: "the owner's decision", "Jørgen's
// answer on footer-mock.md". Further in, it describes what the wait is
// on, not who answers ("tje4sx first (Jørgen's ruling)" waits on a task),
// and a thing of theirs is no question ("on Jørgen's machine").
var possessive = regexp.MustCompile(`(?i)` + lead + `(?:the\s+)?([\p{L}][\p{L}\p{N}._-]*)['’]s\s+(?:decision|decisions|go-ahead|go ahead|answer|answers|approval|ok|okay|sign-off|signoff|verdict|choice|call|ruling|input|feedback|reply|confirmation|review|opinion|pick|say-so|word|look)\b`)

// subject is a wait that opens on who must act and what they do: "owner
// confirms time frames", "check #4: owner removes a test record" (Statbus
// waits of 2026-11-17), "Jørgen looks at the colours".
var subject = regexp.MustCompile(`(?i)` + lead + `(?:the\s+)?([\p{L}][\p{L}\p{N}._-]*)\s+(?:approve|answer|confirm|decide|choose|pick|review|look|try|verify|test|check|remove|open|say|run|sign|tick)s?\b`)

// theOwner finds the owner as the one who acts: "waiting for the owner
// to merge", "until the owner confirms", "an answer from the owner".
var theOwner = regexp.MustCompile(`(?i)\b(?:the owner to|(?:for|from) the owner|the owner (?:decides|confirms|answers|approves|says|replies|picks|chooses|reviews|tests|verifies|checks))\b`)

// PersonWait is a wait written to a person: who it is for (a [people]
// short name) and the question, as hug task ask takes it (a leading
// "check N:" of the task moved to Check).
type PersonWait struct {
	For, What string
	Check     int
}

// AddressesPerson reports whether a wait's text is a question to a person
// (agxqax): it opens with "NAME:" where NAME is the owner or a person
// here (short name, alias, long name, first name), opens on "NAME's"
// decision, go-ahead, answer or the like ("the owner's decision"), opens
// on NAME doing what is asked ("owner confirms ...", "check #4: owner
// removes ..."), or names the owner as the one who acts ("for the owner
// to merge"). A person named in passing ("PR approved by jhf, CI run 9",
// "on Jørgen's machine", "tje4sx first (Jørgen's ruling)") is not a
// question to them. checks is how many acceptance checks the task has,
// so "check N" can become --check N.
func AddressesPerson(people []Person, text string, checks int) (PersonWait, bool) {
	text = strings.TrimSpace(text)
	if text == "" || len(people) == 0 {
		return PersonWait{}, false
	}
	who := func(name string) (Person, bool) {
		if ownerWords[strings.ToLower(strings.TrimSpace(name))] {
			return Owner(people)
		}
		return PersonMatch(people, name)
	}
	w := PersonWait{What: text}
	if m := addressee.FindStringSubmatch(text); m != nil && strings.TrimSpace(m[2]) != "" {
		if p, ok := who(m[1]); ok {
			w.For = p.Short
			w.What = strings.TrimSpace(m[2])
		}
	}
	if w.For == "" {
		for _, re := range []*regexp.Regexp{possessive, subject} {
			if m := re.FindStringSubmatch(text); m != nil {
				if p, ok := who(m[1]); ok {
					w.For = p.Short
					break
				}
			}
		}
	}
	if w.For == "" && theOwner.MatchString(text) {
		p, _ := Owner(people)
		w.For = p.Short
	}
	if w.For == "" {
		return PersonWait{}, false
	}
	// "check N: what" (N a check of the task) is asked as --check N; a
	// question that only names its check keeps its words.
	if m := checkPrefix.FindStringSubmatch(w.What); m != nil {
		if n, _ := strconv.Atoi(m[1]); n >= 1 && n <= checks {
			w.Check, w.What = n, strings.TrimSpace(w.What[len(m[0]):])
		}
	} else if m := leadingCheck.FindStringSubmatch(w.What); m != nil {
		if n, _ := strconv.Atoi(m[1]); n >= 1 && n <= checks {
			w.Check = n
		}
	}
	return w, true
}

// leadingCheck is a question that opens on its check without a colon:
// "check 4 on Norway rc.02 (...)", "check 6 (plain paging ...)".
var leadingCheck = regexp.MustCompile(`^(?i)check\s+#?(\d+)\b`)

// AskCommand is the hug task ask that says w on task id.
func (w PersonWait) AskCommand(id string) string {
	cmd := "hug task ask " + id + " --to " + w.For
	if w.Check > 0 {
		cmd += " --check " + strconv.Itoa(w.Check)
	}
	return cmd + " " + strconv.Quote(oneLineFix(w.What))
}

// answerer is who --to names on this board (agxqax): a person by any of
// their names becomes their short name (jorgen, Jørgen -> jhf); an
// agent on the board, the coordinator or the asker is asked as written.
// On a board with [people], a name that is none of these is refused,
// naming who is here: nobody would see the question as theirs. A board
// without [people] keeps --to as written (lint asks to add the person).
func (s *Store) answerer(id, who, what string) (string, error) {
	people, _ := People(s.Root)
	if len(people) == 0 {
		return who, nil
	}
	if p, ok := PersonMatch(people, who); ok {
		return p.Short, nil
	}
	name := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(who), "@"))
	if ownerWords[name] {
		p, _ := Owner(people)
		return p.Short, nil
	}
	ts, e := s.Tasks()
	if e != nil {
		return "", e
	}
	known := KnownNames(people, ts)
	if c, _ := Coordinator(s.Root); c != "" {
		known[strings.ToLower(c)] = true
	}
	if a := strings.TrimSpace(s.Actor); a != "" {
		known[strings.ToLower(a)] = true
	}
	if known[name] {
		return who, nil
	}
	owner, _ := Owner(people)
	return "", usageErr(id, "nobody here is called "+strconv.Quote(who)+": --to takes a person in hug.toml [people] ("+PeopleList(people)+") or an agent working on this board; to call a person by another name, add alias = [\"NAME\"] to their [people] entry",
		"hug task ask "+id+" --to "+owner.Short+" "+strconv.Quote(oneLineFix(what)))
}

// PeopleList names the people for a message: "jhf (Jørgen H. Fjeld),
// erik (Erik Søberg)".
func PeopleList(people []Person) string {
	parts := make([]string, len(people))
	for i, p := range people {
		parts[i] = p.Short
		if p.Name != "" {
			parts[i] += " (" + p.Name + ")"
		}
	}
	return strings.Join(parts, ", ")
}

// PersonWaitFindings flags an open task waiting on a text that is a
// question to a person (AddressesPerson), with the ask that says it. A
// wait that holds a question for a date (ask --after) is the ask itself.
func PersonWaitFindings(people []Person, ts []Task) []Finding {
	var out []Finding
	for _, t := range ts {
		if t.State != "doing" || t.Stage != "waiting" || strings.TrimSpace(t.Fields["waiting_ask_what"]) != "" {
			continue
		}
		on := t.Fields["waiting_on"]
		w, ok := AddressesPerson(people, on, len(t.Acceptance))
		if !ok {
			continue
		}
		out = append(out, Finding{Path: t.Path, Line: 1, Code: "wait-on-person",
			Problem: "a question waits as a wait: waiting_on " + strconv.Quote(oneLineFix(on)) + " is for " + w.For + ", who never sees a wait as theirs; a question to a person is hug task ask",
			Fix:     w.AskCommand(t.ID)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
