// Package agents finds text that names an ephemeral agent (ez50rb).
// Agents are the swarm sessions of this machine, named after animals and
// plants (session_<name>_<ts>_<hex> in a jcode home's sessions folder); they
// come and go within hours, so a memory that names one ("report to
// coordinator orangutan") stops meaning anything once it stops. A name
// counts only in agent context, so "tiger" the animal and "Dragon 1" the
// design packet do not (hand-checked 50 of 50 on a new sample of the local
// store, 2026-11-16, after tuning on two earlier samples).
package agents

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var sessionFile = regexp.MustCompile(`^session_([a-z]+)_(\d+)_[0-9a-f]+\.json$`)

// Session is one agent session file: when it started (the millisecond
// stamp in its name) and when it last wrote (the file's mtime: a session
// file is rewritten every turn, and a crashed session can still say
// status Active inside the file, but its mtime does not lie).
type Session struct {
	Path        string
	Start, Last time.Time
}

// Names reads agent names from the sessions folders of jcode homes.
func Names(homes []string) []string {
	ss := Sessions(homes)
	out := make([]string, 0, len(ss))
	for n := range ss {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// tailBytes is how much of a session file's end NamesIn reads for its
// working_dir: jcode writes it after the messages.
const tailBytes = 256 << 10

var workingDir = regexp.MustCompile(`"working_dir":"((?:[^"\\]|\\.)*)"`)

// NamesIn is the agents whose sessions worked in one of roots: the
// session's working_dir is a root or inside one (rnpdnm attempt 2: the
// team of this repository, not every agent of the machine). Only the end
// of each .json session file is read.
func NamesIn(homes, roots []string) []string {
	var clean []string
	for _, r := range roots {
		if r != "" {
			clean = append(clean, canonical(r))
		}
	}
	set := map[string]bool{}
	for name, ss := range Sessions(homes) {
		for _, s := range ss {
			if strings.HasSuffix(s.Path, ".json") && inRoots(sessionDir(s.Path), clean) {
				set[name] = true
				break
			}
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// sessionDir is the last working_dir in the end of a session file, "".
func sessionDir(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil && fi.Size() > tailBytes {
		f.Seek(fi.Size()-tailBytes, 0)
	}
	b := make([]byte, tailBytes)
	n, _ := io.ReadFull(f, b)
	ms := workingDir.FindAllSubmatch(b[:n], -1)
	if len(ms) == 0 {
		return ""
	}
	var wd string
	if err := json.Unmarshal([]byte(`"`+string(ms[len(ms)-1][1])+`"`), &wd); err != nil {
		return ""
	}
	return wd
}

func inRoots(wd string, roots []string) bool {
	if wd == "" {
		return false
	}
	wd = canonical(wd)
	for _, r := range roots {
		if wd == r || strings.HasPrefix(wd, r+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// canonical is p cleaned, with symlinks resolved when it exists (macOS
// /var is /private/var; Windows 8.3 short names), folded to lower case on
// Windows, as huginn's owner check reads a session's folder.
func canonical(p string) string {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

// Sessions maps each agent name to its sessions, oldest first. Names are
// reused: "orangutan" today is not the orangutan of September.
func Sessions(homes []string) map[string][]Session {
	out := map[string][]Session{}
	for _, h := range homes {
		ents, err := os.ReadDir(filepath.Join(h, "sessions"))
		if err != nil {
			continue
		}
		for _, e := range ents {
			m := sessionFile.FindStringSubmatch(e.Name())
			if m == nil {
				continue
			}
			var ms int64
			fmt.Sscan(m[2], &ms)
			s := Session{Path: filepath.Join(h, "sessions", e.Name()), Start: time.UnixMilli(ms)}
			if fi, err := e.Info(); err == nil {
				s.Last = fi.ModTime()
			}
			out[m[1]] = append(out[m[1]], s)
		}
	}
	for _, ss := range out {
		sort.Slice(ss, func(i, j int) bool { return ss[i].Start.Before(ss[j].Start) })
	}
	return out
}

// Meant returns the sessions a memory written at t can mean: those alive
// then (started before t, last wrote after it), or failing that the
// newest one started before t.
func Meant(ss []Session, t time.Time) []Session {
	var out []Session
	var prev *Session
	for i := range ss {
		if ss[i].Start.After(t) {
			break
		}
		prev = &ss[i]
		if !ss[i].Last.Before(t) {
			out = append(out, ss[i])
		}
	}
	if len(out) == 0 && prev != nil {
		out = []Session{*prev}
	}
	return out
}

// Common nouns need explicit agent context; a capital letter (start of a
// clause, a design name) is not enough for them.
var common = map[string]bool{"bug": true, "cat": true, "ant": true, "bird": true, "bee": true, "fox": true, "bat": true, "pig": true,
	"snail": true, "dragon": true, "spider": true, "bear": true, "lion": true, "monkey": true, "hippo": true, "shark": true,
	"tulip": true, "rose": true, "frog": true, "goat": true, "sheep": true, "duck": true, "badger": true}

const ctx = `(?:agent|worker|lane|session|coordinator|foreman|reviewer|re-reviewer|scout|engineer|architect|DM|DMs|DM'd|spawned|swarm)`

// A name followed by a tag is a thing, not an agent: "Dragon 1", "Dragon-A",
// and so is a compound word: "dragon-task".
var tag = regexp.MustCompile(`^(?:[\s-]?(?:\d|[A-Z](?:[^A-Za-z]|$))|-[a-z])`)

// A name that mostly heads a compound noun ("scout bug hunt", "bug fix")
// is an agent only when no plain word follows it: "reviewer bug." and
// "DM bug with" are agents.
var compound = map[string]bool{"bug": true}

var nounNext = regexp.MustCompile(`^\s+[a-z]`)
var wordNext = regexp.MustCompile(`^\s+(?:and|or|to|for|with|within|before|after|when|if|is|was|has|had|will|should|must|may|can|on|in|at|about|that|which|who|rather|then|now|reports|owns|said|claimed|holds|stops?)\b`)

type pats struct {
	strong, capital *regexp.Regexp
	noun            bool
}

// Matcher finds agent names in text.
type Matcher struct {
	names []string
	p     map[string]pats
	// anyName prefilters: one pass over the text for every name, so a store
	// of 30,000 memories costs one regex each, not one per agent name.
	anyName *regexp.Regexp
}

// New builds a matcher for names.
func New(names []string) *Matcher {
	m := &Matcher{names: names, p: map[string]pats{}}
	quoted := make([]string, 0, len(names))
	for _, n := range names {
		quoted = append(quoted, regexp.QuoteMeta(n))
	}
	if len(quoted) > 0 {
		m.anyName = regexp.MustCompile(`(?i)\b(?:` + strings.Join(quoted, "|") + `)\b`)
	}
	for _, n := range names {
		q, c := regexp.QuoteMeta(n), regexp.QuoteMeta(strings.ToUpper(n[:1])+n[1:])
		p := pats{
			// The name is captured so the tag check sees what follows it.
			strong: regexp.MustCompile(`(?i)session_(` + q + `)_|` + ctx + `\W{1,3}(?:\w+\W{1,2})?\b(` + q + `)\b|\b(` + q + `)\b\W{1,3}(?:\w+\W{1,2})?` + ctx + `\b`),
		}
		if common[n] {
			// Common nouns: no sentence break between the context word and
			// the name ("No bug. The architect" names no one).
			p.strong = regexp.MustCompile(`(?i)session_(` + q + `)_|` + ctx + `[^\w.!?]{1,3}(?:\w+[^\w.!?]{1,2})?\b(` + q + `)\b|\b(` + q + `)\b[^\w.!?]{1,3}(?:\w+[^\w.!?]{1,2})?` + ctx + `\b`)
			p.noun = compound[n]
			p.capital = regexp.MustCompile(`\b(` + c + `)(?:(?:'s|’s)\s|\s(?:should|is|was|reports|claimed|said|stops?|owns|coordinat))`)
		} else {
			p.capital = regexp.MustCompile(`[a-z,;:] (` + c + `)\b`)
		}
		m.p[n] = p
	}
	return m
}

// Find returns the agent names text uses in agent context.
func (m *Matcher) Find(text string) []string {
	if m == nil || m.anyName == nil {
		return nil
	}
	// Only the names that occur at all get the context checks.
	var out []string
	seen := map[string]bool{}
	for _, w := range m.anyName.FindAllString(text, -1) {
		n := strings.ToLower(w)
		if seen[n] {
			continue
		}
		seen[n] = true
		p, ok := m.p[n]
		if ok && (used(text, p.strong, p.noun) || used(text, p.capital, false)) {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// determiner before a common noun, or one word before it, makes it the
// thing: "the spider", "a bug", "a house spider", "a compiler bug".
var determiner = regexp.MustCompile(`(?i)\b(?:a|an|the|this|that|these|those|each|every|any|no|some|one|its|their|our|my|your)\s+(?:[a-z]+\s+)?$`)

// FindIn returns the names text uses as agents, where context is the text
// it came from (a capture's transcript chunk). A name the context uses as
// an agent counts in text without agent context of its own: from a
// transcript where "coordinator spider" spoke, "spider pushes" and "via
// fox" name the agent (typq1e; CAPTURE4 left 16 such memories after the
// agent-context check). It is still not followed by a tag ("Spider-2"),
// and a common noun after a determiner ("the spider") is the thing.
func (m *Matcher) FindIn(text, context string) []string {
	out := m.Find(text)
	if m == nil || m.anyName == nil || context == "" {
		return out
	}
	have := map[string]bool{}
	for _, n := range out {
		have[n] = true
	}
	for _, n := range m.Find(context) {
		if have[n] {
			continue
		}
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(n) + `\b`)
		for _, loc := range re.FindAllStringIndex(text, -1) {
			if tag.MatchString(text[loc[1]:]) || common[n] && determiner.MatchString(text[:loc[0]]) {
				continue
			}
			out, have[n] = append(out, n), true
			break
		}
	}
	sort.Strings(out)
	return out
}

// used: some match whose captured name is not followed by a tag (or, for a
// common noun, by another word).
func used(text string, re *regexp.Regexp, noun bool) bool {
	for _, sub := range re.FindAllStringSubmatchIndex(text, -1) {
		for g := 2; g+1 < len(sub); g += 2 {
			if sub[g] < 0 {
				continue
			}
			rest := text[sub[g+1]:]
			if !tag.MatchString(rest) && !(noun && nounNext.MatchString(rest) && !wordNext.MatchString(rest)) {
				return true
			}
		}
	}
	return false
}
