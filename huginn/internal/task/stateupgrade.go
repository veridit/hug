package task

import (
	"os"
	"strings"
)

// The one place hug still reads the word "status" in a task file (6ephqk
// step 4, crab 11:10 and 13:07): hug upgrade renames the old key to state,
// once, in every file. Nothing else reads or writes it; lint names a file
// that still has it (fix: hug upgrade). The source test that greps for
// "status" exempts this file only.

const legacyStateKey = "status"

// RenameStateKey rewrites a task file's frontmatter key "status:" to
// "state:", keeping the value, its line, and the file's line endings. It
// changes nothing when the file has no frontmatter, has no old key, or
// already has a state: line (lint reports both keys: one must go by hand,
// since hug cannot tell which is right).
func RenameStateKey(data []byte) ([]byte, bool) {
	lines := strings.SplitAfter(string(data), "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return data, false
	}
	old, has := -1, false
	for n := 1; n < len(lines); n++ {
		line := strings.TrimRight(lines[n], "\r\n")
		if line == "---" {
			break
		}
		switch {
		case strings.HasPrefix(line, legacyStateKey+":"):
			if old < 0 {
				old = n
			}
		case strings.HasPrefix(line, "state:"):
			has = true
		}
	}
	if old < 0 || has {
		return data, false
	}
	lines[old] = "state:" + strings.TrimPrefix(lines[old], legacyStateKey+":")
	return []byte(strings.Join(lines, "")), true
}

// HasLegacyStateKey says whether a file still carries the old key and no
// state: (lint's legacy-state finding).
func HasLegacyStateKey(data []byte) bool {
	_, would := RenameStateKey(data)
	return would
}

// UpgradeStateKey renames the key in every task file on the board and
// returns the changed paths, for hug upgrade to commit as one change.
func (s *Store) UpgradeStateKey() ([]string, error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, e
	}
	var changed []string
	for _, t := range ts {
		path, e := s.diskPath(t.Path)
		if e != nil {
			return changed, e
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return changed, e
		}
		out, did := RenameStateKey(b)
		if !did {
			continue
		}
		s.Invalidate()
		if e := AtomicWrite(path, out); e != nil {
			return changed, e
		}
		changed = append(changed, t.Path)
	}
	return changed, nil
}
