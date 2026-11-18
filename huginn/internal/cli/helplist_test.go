package cli

import (
	"strings"
	"testing"
)

// Every command hug task runs is listed in hug task help, and hug task help
// COMMAND gives it a usage line. Jørgen 2026-11-18 00:36: attach worked but
// help did not list it; 16 commands were missing, and attach and detach had
// no usage at all.
func TestHelpListsEveryCommand(t *testing.T) {
	listed := map[string]bool{}
	for _, line := range strings.Split(Help, "\n") {
		if f := strings.Fields(line); len(f) > 1 && strings.HasPrefix(line, "  ") && strings.HasSuffix(f[0], ":") {
			for _, c := range f[1:] {
				listed[c] = true
			}
		}
	}
	for _, c := range Commands {
		if !listed[c] {
			t.Errorf("hug task help does not list %s", c)
		}
		h := commandHelp(c)
		first := strings.SplitN(h, "\n", 2)[0]
		if strings.TrimSpace(strings.TrimPrefix(first, "hug task")) == "" {
			t.Errorf("hug task help %s has no usage line: %q", c, first)
		}
	}
}

// Help never names a flag the command refuses.
func TestHelpNamesNoRemovedFlag(t *testing.T) {
	if h := commandHelp("upgrade"); strings.Contains(h, "--dry-run") {
		t.Errorf("hug task help upgrade names --dry-run, which upgrade refuses:\n%s", h)
	}
}
