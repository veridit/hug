package task

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StaleOwnerAfter is how long an assignee may have no live harness session
// before lint says its doing task has lost its owner (v3k6cp).
const StaleOwnerAfter = 24 * time.Hour

// Liveness reports when each harness session name was last active. A nil
// map means the harness's session list is not readable here: say nothing.
type Liveness map[string]time.Time

// JcodeLiveness reads jcode's session list (JCODE_HOME or ~/.jcode,
// sessions/session_<name>_<start>_<hex>.json): a name is live as of its
// newest session file's modification time. Nil when there is none.
func JcodeLiveness() Liveness { return JcodeLivenessAt(os.Getenv("JCODE_HOME")) }

// JcodeLivenessAt is JcodeLiveness for a given JCODE_HOME ("" is ~/.jcode).
func JcodeLivenessAt(home string) Liveness {
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		home = filepath.Join(h, ".jcode")
	}
	return LivenessFrom(filepath.Join(home, "sessions"))
}

// LivenessFrom reads session_<name>_*.json files in dir.
func LivenessFrom(dir string) Liveness {
	es, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	live := Liveness{}
	for _, e := range es {
		n := e.Name()
		if e.IsDir() || !strings.HasPrefix(n, "session_") || !strings.HasSuffix(n, ".json") {
			continue
		}
		name, _, ok := strings.Cut(strings.TrimPrefix(n, "session_"), "_")
		if !ok || name == "" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if t := info.ModTime(); t.After(live[name]) {
			live[name] = t
		}
	}
	if len(live) == 0 {
		return nil
	}
	return live
}

// OwnerFindings flags doing tasks whose every assignee is a harness
// session name that has been inactive for over StaleOwnerAfter (v3k6cp):
// the board says someone is working who is not. An assignee the session
// list does not know (a person, a model name, another harness) is never
// judged. The fix hands the task back or to whoever works it.
func OwnerFindings(live Liveness, ts []Task, now time.Time) []Finding {
	if live == nil {
		return nil
	}
	var out []Finding
	for _, t := range ts {
		if t.Status != "doing" || len(t.Assignee) == 0 {
			continue
		}
		stale, last := true, time.Time{}
		for _, a := range t.Assignee {
			seen, ok := live[strings.ToLower(strings.TrimSpace(a))]
			if !ok || now.Sub(seen) <= StaleOwnerAfter {
				stale = false
				break
			}
			if seen.After(last) {
				last = seen
			}
		}
		if !stale {
			continue
		}
		who := strings.Join(t.Assignee, ", ")
		problem := "owner " + who + " of " + t.ID + " has had no live session since " + last.Local().Format("2006-01-02 15:04") + "; whoever works it now takes the assignee"
		out = append(out, Finding{Path: t.Path, Line: 1, Problem: problem, Code: "stale-owner", Fix: "hug task set " + t.ID + " --assignee NAME"})
	}
	return out
}
