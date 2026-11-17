package task

import (
	"sort"
	"strings"
)

// Every check reports every problem at once (fkpyg3, Jørgen 2026-11-17):
// findings are grouped per task, and the fixes of one task are merged into
// one command where one command can fix them. An agent fixes a task in one
// round instead of one finding per round.

// TaskFindings is every finding of one task, with the one combined fix.
type TaskFindings struct {
	ID       string    `json:"id"`
	Path     string    `json:"path"`
	Codes    []string  `json:"codes"`
	Findings []Finding `json:"findings"`
	// Fix is one command that fixes every finding it can, then, separated
	// by "; then ", each command no single one can replace (hug upgrade,
	// a dependency edit). Every part is runnable once "..." is filled in.
	Fix string `json:"fix"`
}

// ByTask groups findings by task file, in the order the files first
// appear. Findings without a task file (the board itself) come last, one
// group each.
func ByTask(fs []Finding) []TaskFindings {
	var out []TaskFindings
	at := map[string]int{}
	for _, f := range fs {
		key := f.Path
		_, id, _, err := ParseName(f.Path)
		if err != nil || id == "" {
			key = ""
		}
		i, ok := at[key]
		if !ok || key == "" {
			at[key] = len(out)
			i = len(out)
			out = append(out, TaskFindings{ID: id, Path: f.Path})
		}
		g := &out[i]
		g.Findings = append(g.Findings, f)
		code := FindingCode(f.Problem)
		if !contains(g.Codes, code) {
			g.Codes = append(g.Codes, code)
		}
	}
	for i := range out {
		out[i].Fix = CombinedFix(out[i].ID, out[i].Findings)
	}
	return out
}

// CombinedFix merges the fixes of one task's findings. Every
// "hug task set ID --flag value" joins one set command with all the flags
// (each flag once; --accept may repeat); every other command (hug upgrade,
// hug task dep ...) follows once, in first-seen order, after "; then ".
func CombinedFix(id string, fs []Finding) string {
	var flags []string
	seenFlag := map[string]bool{}
	var others []string
	seenOther := map[string]bool{}
	prefix := "hug task set " + id + " "
	for _, f := range fs {
		fix := f.FixLine()
		if id != "" && strings.HasPrefix(fix, prefix) {
			for _, fl := range splitFlags(strings.TrimPrefix(fix, prefix)) {
				name, _, _ := strings.Cut(fl, " ")
				if name != "--accept" && seenFlag[name] {
					continue
				}
				if seenFlag[fl] {
					continue
				}
				seenFlag[name], seenFlag[fl] = true, true
				flags = append(flags, fl)
			}
			continue
		}
		if !seenOther[fix] {
			seenOther[fix] = true
			others = append(others, fix)
		}
	}
	var parts []string
	if len(flags) > 0 {
		sort.SliceStable(flags, func(i, j int) bool { return flagRank(flags[i]) < flagRank(flags[j]) })
		parts = append(parts, prefix+strings.Join(flags, " "))
	}
	parts = append(parts, others...)
	return strings.Join(parts, "; then ")
}

// flagRank orders a combined set command as a task reads: title, then the
// description sections, then acceptance, then the rest.
func flagRank(fl string) int {
	name, _, _ := strings.Cut(fl, " ")
	for i, n := range []string{"--title", "--goal", "--purpose", "--principle", "--details", "--accept"} {
		if name == n {
			return i
		}
	}
	return 99
}

// splitFlags splits `--a "x y" --b z` into `--a "x y"` and `--b z`,
// keeping quoted values whole.
func splitFlags(s string) []string {
	var out []string
	var cur strings.Builder
	quoted := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' && (i == 0 || s[i-1] != '\\'):
			quoted = !quoted
		case c == ' ' && !quoted && i+2 < len(s) && s[i+1] == '-' && s[i+2] == '-':
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if t := strings.TrimSpace(cur.String()); t != "" {
		out = append(out, t)
	}
	return out
}
