package task

import (
	"os"
	"strconv"
	"strings"

	"github.com/veridit/hug/internal/memshare"
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

// SetMemoryShare writes hug.toml [memory] share = true (share) or removes
// the key, and the [memory] section with it once empty (34r1ft: absent
// means local), keeping every other line. It reports whether the file
// changed.
func SetMemoryShare(root string, share bool) (bool, error) {
	path := configPath(root)
	b, e := os.ReadFile(path)
	if e != nil && !os.IsNotExist(e) {
		return false, e
	}
	var lines []string
	if len(b) > 0 {
		lines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
	var out []string
	in, section := false, -1
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			in = strings.TrimSpace(strings.Trim(t, "[]")) == "memory"
			if in {
				section = len(out)
			}
		} else if key, _, ok := strings.Cut(t, "="); in && ok && strings.TrimSpace(key) == "share" {
			continue
		}
		out = append(out, l)
	}
	if share {
		if section >= 0 {
			out = append(out[:section+1], append([]string{"share = true"}, out[section+1:]...)...)
		} else {
			if len(out) > 0 {
				out = append(out, "")
			}
			out = append(out, "[memory]", "share = true")
		}
	} else if section >= 0 {
		// Drop a [memory] header left with nothing under it.
		end := section + 1
		for end < len(out) && strings.TrimSpace(out[end]) == "" {
			end++
		}
		if end == len(out) || strings.HasPrefix(strings.TrimSpace(out[end]), "[") {
			out = append(out[:section], out[end:]...)
			for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
				out = out[:len(out)-1]
			}
		}
	}
	next := strings.Join(out, "\n") + "\n"
	if len(out) == 0 {
		next = ""
	}
	if next == string(b) {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(next), 0o644)
}

// MemoryShare is hug.toml [memory] share as this checkout has it (34r1ft);
// absent is false. memshare.Shares reads the committed one.
func MemoryShare(root string) (bool, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	return memshare.Parse(b)
}
