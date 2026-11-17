package task

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// The one place hug still reads the word "status" in a task file (6ephqk
// step 4, crab 11:10 and 13:07): hug upgrade renames the old key to state,
// once, in every file, and writes old zoneless times (*_since, *_at, local
// time) as UTC with Z (crab 13:33). Nothing else reads or writes the old
// key; lint names a file that still has it (fix: hug upgrade). The source
// test that greps for "status" exempts this file only.

const legacyStateKey = "status"

// UpgradeFrontmatter rewrites a task file's frontmatter to the current
// form: "status:" becomes "state:" (value and line kept), and each
// *_since/*_at value without a zone, read as local time in loc, becomes UTC
// with Z. Line endings are kept. The key is left when the file already has
// a state: line (lint reports both: one must go by hand, since hug cannot
// tell which is right). Running it again changes nothing.
func UpgradeFrontmatter(data []byte, loc *time.Location) ([]byte, bool) {
	lines := strings.SplitAfter(string(data), "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return data, false
	}
	old, has, end := -1, false, -1
	for n := 1; n < len(lines); n++ {
		line := strings.TrimRight(lines[n], "\r\n")
		if line == "---" {
			end = n
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
	if end < 0 {
		return data, false
	}
	changed := false
	if old >= 0 && !has {
		lines[old] = "state:" + strings.TrimPrefix(lines[old], legacyStateKey+":")
		changed = true
	}
	for n := 1; n < end; n++ {
		body := strings.TrimRight(lines[n], "\r\n")
		eol := lines[n][len(body):]
		k, v, ok := strings.Cut(body, ":")
		if !ok || !TimeField(k) || strings.ContainsAny(k, " \t") {
			continue
		}
		raw := strings.TrimSpace(v)
		if u, e := strconv.Unquote(raw); e == nil {
			raw = u
		}
		if _, e := time.Parse(time.RFC3339, raw); e == nil {
			continue
		}
		at, e := time.ParseInLocation(zonelessLayout, raw, loc)
		if e != nil {
			continue // not a time hug wrote: lint names it
		}
		lines[n] = k + ": " + strconv.Quote(at.UTC().Format(StampLayout)) + eol
		changed = true
	}
	if !changed {
		return data, false
	}
	return []byte(strings.Join(lines, "")), true
}

// HasLegacyStateKey says whether a file still carries the old key and no
// state:.
func HasLegacyStateKey(data []byte) bool {
	lines := strings.SplitAfter(string(data), "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return false
	}
	old, has := false, false
	for _, l := range lines[1:] {
		l = strings.TrimRight(l, "\r\n")
		if l == "---" {
			break
		}
		old = old || strings.HasPrefix(l, legacyStateKey+":")
		has = has || strings.HasPrefix(l, "state:")
	}
	return old && !has
}

// UpgradeFiles brings every task file on the board to the current
// frontmatter and returns the changed paths, for hug upgrade to commit as
// one change. Old times are read in loc (the machine running upgrade; the
// boards so far were all written in one zone).
func (s *Store) UpgradeFiles(loc *time.Location) ([]string, error) {
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
		out, did := UpgradeFrontmatter(b, loc)
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
