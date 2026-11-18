package task

import (
	"os"
	"strings"
)

// ProjectName is hug.toml [board] project, or "" when unset (0sw0dd,
// Jørgen 2026-11-16): the name hug tui's header gives the board on every
// checkout. Unset, the header names the checkout's folder, so two
// checkouts of one repository (hug, hug-testing) read differently.
//
//	[board]
//	project = "Seid"
func ProjectName(root string) (string, error) {
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
		if key, value, ok := strings.Cut(line, "="); in && ok && strings.TrimSpace(key) == "project" {
			return strings.TrimSpace(scalar(value)), nil
		}
	}
	return "", nil
}
