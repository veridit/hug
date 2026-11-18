package task

import (
	"os"
	"strconv"
	"strings"
)

// The board's coordinator (hbgybn, crab 00:58): the session that hears what
// no owner hears, an unowned request or the outcome of an answer (41hhgn
// check 11). One name in hug.toml:
//
//	[board]
//	coordinator = "crab"
//
// written by hug task coordinator NAME. With none set, every session on the
// board hears an unowned request once: a request is never lost quietly.

// Coordinator is hug.toml [board] coordinator, or "" when unset.
func Coordinator(root string) (string, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	in := false
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			in = strings.TrimSpace(strings.Trim(line, "[]")) == "board"
			continue
		}
		if key, value, ok := strings.Cut(line, "="); in && ok && strings.TrimSpace(key) == "coordinator" {
			return scalar(value), nil
		}
	}
	return "", nil
}

// SetCoordinator writes hug.toml [board] coordinator = name, keeping every
// other line; name "" removes the key.
func SetCoordinator(root, name string) error {
	path := configPath(root)
	b, e := os.ReadFile(path)
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	var lines []string
	if len(b) > 0 {
		lines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
	entry := "coordinator = " + strconv.Quote(name)
	var out []string
	in, board, done := false, -1, false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			in = strings.TrimSpace(strings.Trim(t, "[]")) == "board"
			if in {
				board = len(out)
			}
		} else if key, _, ok := strings.Cut(t, "="); in && ok && strings.TrimSpace(key) == "coordinator" {
			if name != "" && !done {
				out = append(out, entry)
			}
			done = true
			continue
		}
		out = append(out, l)
	}
	if name != "" && !done {
		if board >= 0 {
			out = append(out[:board+1], append([]string{entry}, out[board+1:]...)...)
		} else {
			if len(out) > 0 {
				out = append(out, "")
			}
			out = append(out, "[board]", entry)
		}
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

// RequestFor says whether an open request on t reaches who: the task's
// assignee; on a task nobody holds, the coordinator; with no coordinator,
// everyone (crab 00:58).
func RequestFor(t Task, who, coordinator string) bool {
	who = strings.TrimSpace(who)
	if who == "" || len(OpenRequests(t)) == 0 {
		return false
	}
	for _, a := range t.Assignee {
		if strings.EqualFold(strings.TrimSpace(a), who) {
			return true
		}
	}
	if len(t.Assignee) > 0 && t.State == "doing" {
		return false
	}
	return coordinator == "" || strings.EqualFold(coordinator, who)
}
