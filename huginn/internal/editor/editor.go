// Package editor finds the user's editor the way git does.
package editor

import (
	"os"
	"os/exec"
	"strings"
)

// Command returns the editor command line: $VISUAL, then $EDITOR, then git's
// own choice (GIT_EDITOR, core.editor), and vi only when nothing is set.
// The result is split on spaces, so "zed --wait" works.
func Command() []string {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if f := strings.Fields(os.Getenv(name)); len(f) > 0 {
			return f
		}
	}
	if out, err := exec.Command("git", "var", "GIT_EDITOR").Output(); err == nil {
		if f := strings.Fields(strings.TrimSpace(string(out))); len(f) > 0 {
			return f
		}
	}
	return []string{"vi"}
}
