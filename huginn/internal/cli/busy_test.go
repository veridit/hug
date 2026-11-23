package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// n2hq8p check 1 (CLI): the repro from 2026-11-17 13:10, a second
// hug task take by the same agent while the first ticket is active, is
// refused with exit 3, a line naming the active ticket and four fix:
// lines (the fourth, the detour, since 4tycwd); --json gives fix (the
// first) and fixes (all four).
func TestSecondTakeRefusedWithThreeFixLines(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	first := execute(t, a, "new", "First", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	second := execute(t, a, "new", "Second", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	execute(t, a, "take", first, "--model", "claude-opus-5-5", "--as", "kestrel")
	code, out := runText(a, "take", second, "--model", "claude-opus-5-5", "--as", "kestrel")
	want := "not taking " + second + ": kestrel is on " + first + " (First): one ticket in doing at a time; if they ship together (one change or release), take it with the active one; if it is a detour, the active one waits on it\n" +
		"fix: hug task done " + first + "\n" +
		"fix: hug task wait " + first + " \"on what\"\n" +
		"fix: hug task set " + first + " --state todo && hug task note " + first + " \"why it goes back\"\n" +
		"fix: hug task dep " + first + " --add " + second + " && hug task note " + first + " \"where I was\"\n" +
		"fix: hug task take " + second + " --with " + first + " --model claude-opus-5-5 --as kestrel\n"
	if code != 3 || out != want {
		t.Fatalf("exit %d, want 3 and\n%s\ngot\n%s", code, want, out)
	}
	code, out, serr := runJSON(a, "take", second, "--model", "claude-opus-5-5", "--as", "kestrel", "--json")
	var r map[string]any
	if e := json.Unmarshal([]byte(out), &r); e != nil || code != 3 {
		t.Fatalf("exit %d: %v\n%s\nstderr:\n%s", code, e, out, serr)
	}
	if fixes, _ := r["fixes"].([]any); len(fixes) != 5 || r["fix"] != "hug task done "+first || r["next"] != r["fix"] {
		t.Fatalf("json: %s", out)
	}
	if f := taskFile(t, a, second); !strings.Contains(f, "state: todo") && !strings.Contains(f, `state: "todo"`) || strings.Contains(f, "kestrel") {
		t.Fatalf("refused take changed the task:\n%s", f)
	}
	// Parking the first (the second fix line) lets the take through.
	execute(t, a, "wait", first, "CI run 1")
	execute(t, a, "take", second, "--model", "claude-opus-5-5", "--as", "kestrel")
}
