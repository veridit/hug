package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bqfrda check 1, end to end: hug task attach then hug task detach leaves
// no missing-file in hug task lint ID (on main: the same missing file
// twice, from the attach and the detach notes).
func TestDetachLeavesNoLintFinding(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Files", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	file := filepath.Join(t.TempDir(), "ev.txt")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	execute(t, a, "attach", id, file)
	execute(t, a, "detach", id, "ev.txt")
	if _, out := runText(a, "lint", id); strings.Contains(out, "missing-file") {
		t.Fatalf("after attach and detach, lint reports missing-file:\n%s", out)
	}
}
