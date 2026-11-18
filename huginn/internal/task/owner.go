package task

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// StaleOwnerAfter is how long an assignee may have no live harness session
// before lint says its doing task has lost its owner (v3k6cp).
const StaleOwnerAfter = 24 * time.Hour

// GoneOwnerAfter is how long a closed session (jcode says "status":
// "Closed": ended, or reaped after 30 min idle) stays an owner (9ahvgt): a
// worker that reports and stops is closed at once, and its coordinator
// lands the work within the hour, so a closed owner is gone after one.
const GoneOwnerAfter = time.Hour

// Session is what the harness says of one session name: its newest write,
// and whether its newest session is closed (9ahvgt).
type Session struct {
	Seen   time.Time
	Closed bool
}

// Liveness is the harness's sessions by name (lower case). A nil map means
// the harness's session list is not readable here: say nothing. Aliases
// are names a session was told to work under ("You are kestrel, a worker
// on hug" in its title: take --as kestrel by session calf), mapped to the
// session's own name (9ahvgt).
type Liveness struct {
	Names   map[string]Session
	Aliases map[string]string
	// Read is the session folders that were read; Unread those that exist
	// but could not be (9ahvgt: say so once rather than stay silent).
	Read, Unread []string
	// Sessions is each counted session, for At (offline replays, agh2h7).
	Sessions []SessionRecord
}

// SessionRecord is one counted session: its name, start stamp, newest
// write, final status and the name its title gave it ("" for none).
type SessionRecord struct {
	Name, Start, Status, Alias string
	Seen                       time.Time
}

// Gone reports whether name (a session name or a "You are NAME" alias)
// has gone by the rule stale-owner uses (9ahvgt): its newest session
// closed over GoneOwnerAfter ago, or no write for over StaleOwnerAfter.
// known is false when the session list does not know the name (a person,
// a model, an agent name no session took on): it is never gone here. The
// recall hook's coordinator line on a finished task uses it (agh2h7), so
// the hook and lint agree on who is gone.
func (l *Liveness) Gone(name string, now time.Time) (gone, known bool) {
	if !l.Known() {
		return false, false
	}
	s, _, ok := l.ownerSeen(name)
	if !ok {
		return false, false
	}
	return sessionGone(s, now), true
}

// sessionGone is the gone rule for a known session name (9ahvgt).
func sessionGone(s Session, now time.Time) bool {
	return s.Closed && now.Sub(s.Seen) > GoneOwnerAfter || now.Sub(s.Seen) > StaleOwnerAfter
}

// At is the session list as it stood at t, for offline replays (agh2h7):
// only sessions started by t count, a session still writing after t was
// live at t (seen at t, not closed), and a name is closed at t when its
// newest session by then had stopped writing and ended Closed. Nil when
// no session had started by t.
func (l *Liveness) At(t time.Time) *Liveness {
	if l == nil {
		return nil
	}
	stamp := strconv.FormatInt(t.UnixMilli(), 10)
	out := &Liveness{Names: map[string]Session{}, Aliases: map[string]string{}, Read: l.Read, Unread: l.Unread}
	newest := map[string]SessionRecord{}
	for _, s := range l.Sessions {
		if later(s.Start, stamp) {
			continue
		}
		seen := s.Seen
		if seen.After(t) {
			seen = t
		}
		n := out.Names[s.Name]
		if seen.After(n.Seen) {
			n.Seen = seen
		}
		out.Names[s.Name] = n
		if w, ok := newest[s.Name]; !ok || later(s.Start, w.Start) {
			newest[s.Name] = s
		}
		if s.Alias != "" && s.Alias != s.Name {
			if had, ok := out.Aliases[s.Alias]; !ok || seen.After(out.Names[had].Seen) {
				out.Aliases[s.Alias] = s.Name
			}
		}
	}
	for name, s := range newest {
		n := out.Names[name]
		n.Closed = s.Status == "Closed" && !s.Seen.After(t)
		out.Names[name] = n
	}
	if len(out.Names) == 0 {
		return nil
	}
	return out
}

// RepoRoots is a repository's worktree folders: the checkout at root, the
// main one (common's parent) and each linked worktree
// (<common>/worktrees/ID/gitdir names its .git). Sessions that worked in
// one of them are this board's (9ahvgt).
func RepoRoots(root, common string) []string {
	roots := []string{root}
	if common != "" {
		roots = append(roots, filepath.Dir(common))
		es, _ := os.ReadDir(filepath.Join(common, "worktrees"))
		for _, e := range es {
			if b, err := os.ReadFile(filepath.Join(common, "worktrees", e.Name(), "gitdir")); err == nil {
				roots = append(roots, filepath.Dir(strings.TrimSpace(string(b))))
			}
		}
	}
	return roots
}

// Known reports whether the session list was read at all.
func (l *Liveness) Known() bool { return l != nil && len(l.Names) > 0 }

// JcodeLiveness reads jcode's session list (JCODE_HOME or ~/.jcode), every
// repository's sessions.
func JcodeLiveness() *Liveness { return JcodeLivenessAt(os.Getenv("JCODE_HOME"), nil) }

// JcodeLivenessAt is JcodeLiveness for a given JCODE_HOME ("" is ~/.jcode)
// and further homes (other jcode instances: workers a coordinator spawned
// elsewhere, 9ahvgt), counting the sessions that worked in roots (the
// repository's worktrees; nil: all). A session silent for over a week is
// not opened: it is gone by its time alone.
func JcodeLivenessAt(home string, roots []string, more ...string) *Liveness {
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		home = filepath.Join(h, ".jcode")
	}
	var dirs []string
	seen := map[string]bool{}
	for _, h := range append([]string{home}, more...) {
		d := filepath.Join(filepath.Clean(h), "sessions")
		if !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	return LivenessIn(roots, time.Now().Add(-7*StaleOwnerAfter), dirs...)
}

// LivenessFrom reads every session in dirs, of any repository.
func LivenessFrom(dirs ...string) *Liveness { return LivenessIn(nil, time.Time{}, dirs...) }

// LivenessIn reads session_<name>_<start>_<hex>.json and .journal.jsonl
// files in dirs (9ahvgt). A session counts for this board when its
// working_dir is one of roots or inside one (the repository's worktrees;
// no roots: every session): session names repeat across jcode instances,
// and Statbus's live wyvern is not hug's gone one. A name's last sign of
// life is its newest counted file's modification time; it is closed when
// its newest counted session (by start) says "status":"Closed" in its
// newest record. Sessions with no write since since are counted unread
// (their folder, status and title are not read: they are stale anyway).
// A folder that does not exist is skipped; one that exists and cannot be
// read is listed in Unread. Nil when no session was found.
func LivenessIn(roots []string, since time.Time, dirs ...string) *Liveness {
	l := &Liveness{Names: map[string]Session{}, Aliases: map[string]string{}}
	type session struct {
		name, start string
		seen        time.Time
		files       []string
	}
	sessions := map[string]*session{} // by dir + name + start
	var order []string
	for _, dir := range dirs {
		es, err := os.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				l.Unread = append(l.Unread, dir)
			}
			continue
		}
		l.Read = append(l.Read, dir)
		for _, e := range es {
			n := e.Name()
			base, ok := strings.CutSuffix(n, ".journal.jsonl")
			if !ok {
				base, ok = strings.CutSuffix(n, ".json")
			}
			if e.IsDir() || !ok || !strings.HasPrefix(base, "session_") {
				continue
			}
			parts := strings.SplitN(strings.TrimPrefix(base, "session_"), "_", 3)
			if len(parts) < 2 || parts[0] == "" {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			key := dir + "\x00" + base
			s := sessions[key]
			if s == nil {
				s = &session{name: strings.ToLower(parts[0]), start: parts[1]}
				sessions[key] = s
				order = append(order, key)
			}
			s.files = append(s.files, filepath.Join(dir, n))
			if t := info.ModTime(); t.After(s.seen) {
				s.seen = t
			}
		}
	}
	newest := map[string]*session{} // by name: the newest counted session
	closed := map[string]bool{}
	for _, key := range order {
		s := sessions[key]
		read := !s.seen.Before(since)
		status, title, wd := "", "", ""
		if read {
			status, title, wd = sessionStatus(s.files)
			if !inRoots(wd, roots) {
				continue
			}
		}
		n := l.Names[s.name]
		if s.seen.After(n.Seen) {
			n.Seen = s.seen
		}
		l.Names[s.name] = n
		if w := newest[s.name]; w == nil || later(s.start, w.start) {
			newest[s.name] = s
			closed[s.name] = status == "Closed"
		}
		alias := titleName(title)
		l.Sessions = append(l.Sessions, SessionRecord{Name: s.name, Start: s.start, Status: status, Alias: alias, Seen: s.seen})
		// Several sessions told "You are kestrel": the one seen last.
		if alias != "" && alias != s.name {
			if had, ok := l.Aliases[alias]; !ok || s.seen.After(l.Names[had].Seen) {
				l.Aliases[alias] = s.name
			}
		}
	}
	for name, c := range closed {
		n := l.Names[name]
		n.Closed = c
		l.Names[name] = n
	}
	if len(l.Names) == 0 && len(l.Unread) == 0 {
		return nil
	}
	return l
}

// later compares session start stamps (milliseconds, as decimal text).
func later(a, b string) bool { return len(a) > len(b) || len(a) == len(b) && a > b }

// inRoots: no roots counts every folder; a session with no working_dir
// (an older jcode) counts too, as nothing says it is elsewhere. Both sides
// are compared with symlinks resolved (macOS /var is /private/var) and,
// on Windows, without case.
func inRoots(wd string, roots []string) bool {
	if len(roots) == 0 || wd == "" {
		return true
	}
	wd = canonical(wd)
	for _, r := range roots {
		r = canonical(r)
		if wd == r || strings.HasPrefix(wd, r+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// canonical is p cleaned, with symlinks resolved when it exists, folded
// to lower case on Windows.
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

// tailBytes is how much of a session file's end is read for its status:
// jcode writes it after the messages (.json, within 5 KB of the end on
// 400 sessions on 2026-11-18) or in each journal line's meta.
const tailBytes = 256 << 10

var (
	statusRe = regexp.MustCompile(`"short_name":"[^"]*","status":"([A-Za-z]+)"`)
	wdRe     = regexp.MustCompile(`"working_dir":"((?:[^"\\]|\\.)*)",(?:"original_working_dir":(?:null|"(?:[^"\\]|\\.)*"),)?"short_name":"[^"]*","status":"`)
	titleRe  = regexp.MustCompile(`"title":"((?:[^"\\]|\\.)*)"`)
	youAreRe = regexp.MustCompile(`^You are ([a-z][a-z0-9-]*)[,.]`)
)

// sessionStatus is the newest status, a title and the working folder among
// one session's files: the newest file wins (the journal is written after
// the .json).
func sessionStatus(files []string) (status, title, wd string) {
	var at time.Time
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		b := tail(f, info.Size())
		ms := statusRe.FindAllSubmatch(b, -1)
		if len(ms) > 0 && !info.ModTime().Before(at) {
			at, status = info.ModTime(), string(ms[len(ms)-1][1])
			if w := wdRe.FindAllSubmatch(b, -1); len(w) > 0 {
				wd = jsonString(w[len(w)-1][1])
			}
		}
		if title == "" {
			if m := titleRe.FindSubmatch(head(f)); m != nil {
				title = jsonString(m[1])
			}
		}
	}
	return status, title, wd
}

func tail(path string, size int64) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	if size > tailBytes {
		if _, err := f.Seek(size-tailBytes, io.SeekStart); err != nil {
			return nil
		}
	}
	b, _ := io.ReadAll(io.LimitReader(f, tailBytes))
	return b
}

// head is a session file's first 4 KB, where jcode writes the title.
func head(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	b := make([]byte, 4096)
	n, _ := io.ReadFull(f, b)
	b = b[:n]
	if i := bytes.Index(b, []byte(`"messages"`)); i > 0 {
		b = b[:i]
	}
	return b
}

// titleName is the name a coordinator gave the session in its first
// message ("You are kestrel, a worker on hug ..."), or "".
func titleName(title string) string {
	if m := youAreRe.FindStringSubmatch(strings.TrimSpace(title)); m != nil {
		return m[1]
	}
	return ""
}

// ownerSeen is when name was last seen working and whether it is gone:
// its own session, else the session it was named in. ok is false when the
// session list does not know the name.
func (l *Liveness) ownerSeen(name string) (s Session, via string, ok bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if s, ok := l.Names[name]; ok {
		return s, "", true
	}
	if real, ok := l.Aliases[name]; ok {
		return l.Names[real], real, true
	}
	return Session{}, "", false
}

// OwnerFindings flags doing tasks whose every assignee has gone (v3k6cp,
// 9ahvgt): its session closed over GoneOwnerAfter ago (jcode marks a
// session Closed when it ends or is reaped idle), or no write for over
// StaleOwnerAfter. An agent name no session has (a name given with --as
// that no session took on, a role like crab-engineer) is gone as well once
// the task has been doing for StaleOwnerAfter: the board says someone is
// working who hug cannot find. A person (Model: human, or a name in
// hug.toml [people]: people) and a model name as assignee are never
// judged. The fix hands the task back or to whoever works it.
func OwnerFindings(live *Liveness, ts []Task, now time.Time, people ...Person) []Finding {
	if !live.Known() {
		return nil
	}
	var out []Finding
	for _, t := range ts {
		if t.State != "doing" || len(t.Assignee) == 0 || t.Model == "human" {
			continue
		}
		gone, last, why := true, time.Time{}, ""
		for _, a := range t.Assignee {
			if _, isPerson := PersonByName(people, a); isPerson || isModelName(a, t) {
				gone = false
				break
			}
			s, via, ok := live.ownerSeen(a)
			switch {
			case !ok:
				since, err := time.ParseInLocation("2006-01-02T15:04", t.Fields["doing_since"], time.Local)
				if err != nil || now.Sub(since) <= StaleOwnerAfter {
					gone = false
				} else if why == "" {
					why = "no session is named " + a
				}
			case sessionGone(s, now):
				if s.Seen.After(last) {
					last = s.Seen
				}
				if via != "" && why == "" {
					why = a + " was session " + via
				}
			default:
				gone = false
			}
			if !gone {
				break
			}
		}
		if !gone {
			continue
		}
		who := strings.Join(t.Assignee, ", ")
		problem := "owner " + who + " of " + t.ID + " has had no live session"
		if !last.IsZero() {
			problem += " since " + last.Local().Format("2006-01-02 15:04")
		}
		if why != "" {
			problem += " (" + why + ")"
		}
		problem += "; whoever works it now takes the assignee"
		out = append(out, Finding{Path: t.Path, Line: 1, Problem: problem, Code: "stale-owner", Fix: "hug task set " + t.ID + " --assignee NAME"})
	}
	return out
}

// isModelName: a take without a name makes the model the assignee.
func isModelName(a string, t Task) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	return a != "" && (a == strings.ToLower(t.Model) || strings.Contains(a, "-") && strings.ContainsAny(a, "0123456789") && strings.Count(a, "-") >= 2)
}

// jsonString decodes a JSON string's inside (a Windows path's \\ is \).
func jsonString(raw []byte) string {
	var s string
	if json.Unmarshal(append(append([]byte{'"'}, raw...), '"'), &s) != nil {
		return string(raw)
	}
	return s
}
