package project

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/veridit/hug/muninn/internal/ingest"
)

// Gate is the model-free placement check at sync (rnpdnm B, gav9nw: Jørgen
// chose A+ and B together, 2026-11-18). Beside LooksSecret it holds back a
// project memory whose text names a home folder, another project or client,
// or the person, or that is a preference or correction: those belong to
// the person or to another place, and a pushed line can't be taken back
// (forget keeps its text, git history keeps every commit, other clones keep
// their copy). On hug's 868 public memories this rule, seeded from the
// local store's places, measured 13 leaks in 705 pushed (1.8%) and held
// 118 of the 810 that belong (14.6%): a held line is the cheap mistake.
//
// Terms come from the local config (muninn config.json sync_gate.terms),
// never from the repository: a list naming clients is itself not for a
// public repo. A held line is released by moving it (hug mem rescope) or
// by a local exception (sync_gate.release), never by loosening the rule.
type Gate struct {
	// Terms are the names of other projects, clients and the person,
	// matched case-insensitively as whole words, with Nordic and accented
	// letters folded (Jørgen also matches Jorgen).
	Terms []string
	// Release holds memory ids (full id, short id or the file's line id)
	// that push even though the rule holds them.
	Release []string
	// Own are this project's own names (its folder label and client):
	// never a reason to hold its memories back.
	Own []string
	// Team holds a line that names the team or this machine (rnpdnm
	// attempt 2, PREREG2 (a)): the board coordinator, the agents whose
	// sessions worked in this repository, and TeamPhrases. Read at each
	// sync from hug.toml and the session list, never stored, never
	// printed. Only with extract.wide_places on; empty, nothing changes.
	Team []string
}

// TeamPhrases are the fixed phrases of the team set (PREREG2 (a)).
var TeamPhrases = []string{"this machine", "this environment", "agents should", "the agent gate"}

// The reasons a line is held, one sync output line each.
const (
	HeldHome       = "names a home folder"
	HeldPreference = "is a preference or correction"
	HeldName       = "names another project, client or person"
	// HeldTeam: a working rule for this team or a fact about this machine
	// (attempt 1's 8 hand leaks: 7 team rules, 1 machine fact).
	HeldTeam = "names the team or this machine"
)

// HeldReasons is the order sync reports them in.
var HeldReasons = []string{HeldHome, HeldName, HeldTeam, HeldPreference}

// homeRe: a home folder on macOS, Linux or Windows, or ~/.
// /home/ counts only where a path starts ("internal/home/x.go" is a repo path).
var homeRe = regexp.MustCompile(`(?i)/Users/|(^|[\s"'(=:,])/home/[a-z_]|(^|[^\w.])~/|[a-z]:\\Users\\`)

// Hold names why m stays local, or "" when the rule lets it push. term is
// the configured term that matched (HeldName only).
func (g *Gate) Hold(m ingest.NativeMemory, lineID string) (reason, term string) {
	if g == nil {
		return "", ""
	}
	for _, r := range g.Release {
		if r != "" && (r == m.ID || r == m.SID || r == lineID) {
			return "", ""
		}
	}
	text := m.Title + "\n" + m.Content
	if homeRe.MatchString(text) {
		return HeldHome, ""
	}
	if t := g.names(text); t != "" {
		return HeldName, t
	}
	if t := g.match(text, g.Team); t != "" {
		return HeldTeam, t
	}
	switch strings.ToLower(m.Category) {
	case "preference", "correction":
		return HeldPreference, ""
	}
	return "", ""
}

// names returns the first configured term text names as a whole word.
func (g *Gate) names(text string) string { return g.match(text, g.Terms) }

// match returns the first of terms text names as a whole word, the
// project's own names excepted.
func (g *Gate) match(text string, terms []string) string {
	hay := fold(text)
	own := map[string]bool{}
	for _, o := range g.Own {
		own[fold(o)] = true
	}
	for _, t := range terms {
		ft := fold(strings.TrimSpace(t))
		if len([]rune(ft)) < 2 || own[ft] {
			continue
		}
		if wholeWord(hay, ft) {
			return t
		}
	}
	return ""
}

// wholeWord: needle occurs in hay with no letter or digit right before or
// after it. "Jørgen's", "jhf:" and "(seid)" match; "jhfx" and "research"
// (for "search") do not.
func wholeWord(hay, needle string) bool {
	for from := 0; ; {
		i := strings.Index(hay[from:], needle)
		if i < 0 {
			return false
		}
		i += from
		j := i + len(needle)
		before, after := rune(0), rune(0)
		if i > 0 {
			before = lastRune(hay[:i])
		}
		if j < len(hay) {
			after = []rune(hay[j:])[0]
		}
		if !wordRune(before) && !wordRune(after) {
			return true
		}
		from = i + 1
		for from < len(hay) && !utf8Start(hay[from]) {
			from++
		}
	}
}

func lastRune(s string) rune { r := []rune(s); return r[len(r)-1] }
func wordRune(r rune) bool   { return r != 0 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') }
func utf8Start(b byte) bool  { return b&0xC0 != 0x80 }

// fold lowercases and folds the letters people type both ways: æ ø å and
// accents, so a term written Jørgen matches text written Jorgen.
func fold(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'ø', 'ö', 'ó', 'ò', 'ô', 'õ':
			b.WriteRune('o')
		case 'æ', 'ä':
			b.WriteString("ae")
		case 'å', 'á', 'à', 'â', 'ã':
			b.WriteRune('a')
		case 'é', 'è', 'ê', 'ë':
			b.WriteRune('e')
		case 'í', 'ì', 'î', 'ï':
			b.WriteRune('i')
		case 'ú', 'ù', 'û', 'ü':
			b.WriteRune('u')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Held is one memory the gate keeps local.
type Held struct {
	ID, SID, Reason, Term, Text string
	// Pushed: the line is in the branch's file now, so the remote's
	// history has it; holding it drops it from the tip only.
	Pushed bool
}

// HeldLines lists the project memories the gate keeps local, read-only:
// nothing is written or fetched.
func HeldLines(p Project, nativePath string, g *Gate) ([]Held, error) {
	mems, err := ingest.LoadNative(nativePath)
	if err != nil {
		return nil, err
	}
	inFile := map[string]bool{}
	if lines, err := ReadFile(p.File()); err == nil {
		for _, l := range lines {
			inFile[l.ID] = true
		}
	}
	var out []Held
	for _, m := range mems {
		if m.Scope != p.Scope() || LooksSecret(m.Title) || LooksSecret(m.Content) {
			continue
		}
		id := lineID(p, m)
		if reason, term := g.Hold(m, id); reason != "" {
			// OriginHash is set when a sync wrote the line (MarkSynced): it
			// reached the branch once, so its history keeps it.
			out = append(out, Held{ID: m.ID, SID: m.SID, Reason: reason, Term: term, Text: m.Content, Pushed: inFile[id] || m.OriginHash != ""})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Reason != out[j].Reason {
			return out[i].Reason < out[j].Reason
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// lineID is the id a memory has in the project's file.
func lineID(p Project, m ingest.NativeMemory) string {
	if strings.HasPrefix(m.Origin, originFor(p, "")) {
		return strings.TrimPrefix(m.Origin, originFor(p, ""))
	}
	return m.ID
}

// PeopleTerms reads the names hug.toml [people] gives the board's people:
// the short name, the full name, its first word, and every alias. It is
// used once, to seed the local term list.
func PeopleTerms(hugToml []byte) []string {
	var out []string
	section := ""
	for _, line := range strings.Split(string(hugToml), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			if short, ok := strings.CutPrefix(section, "people."); ok {
				out = append(out, unquote(short))
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		switch {
		case section == "people":
			out = append(out, unquote(key))
		case strings.HasPrefix(section, "people.") && key == "name":
			name := unquote(value)
			out = append(out, name)
			if first, _, ok := strings.Cut(name, " "); ok {
				out = append(out, first)
			}
		case strings.HasPrefix(section, "people.") && key == "alias":
			for _, a := range strings.Split(strings.Trim(strings.TrimSpace(value), "[]"), ",") {
				if a = unquote(a); a != "" {
					out = append(out, a)
				}
			}
		}
	}
	return out
}

// CoordinatorTerm reads hug.toml [board] coordinator, "" when unset.
func CoordinatorTerm(hugToml []byte) string {
	section := ""
	for _, line := range strings.Split(string(hugToml), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok && section == "board" && strings.TrimSpace(key) == "coordinator" {
			value = strings.TrimSpace(value)
			// A quoted value may carry a comment after it.
			if q := value[:min(1, len(value))]; q == `"` || q == "'" {
				if end := strings.Index(value[1:], q); end >= 0 {
					return value[1 : end+1]
				}
			}
			return unquote(value)
		}
	}
	return ""
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, " #"); i >= 0 && !strings.HasPrefix(s, `"`) {
		s = s[:i]
	}
	return strings.Trim(strings.TrimSpace(s), `"'`)
}
