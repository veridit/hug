package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// 4tycwd (crab 17:57): a detour from a doing task to another one, through
// the CLI in text and --json. The busy refusal offers the detour fix
// (check 2); next --mine never offers a waiting task as work and, with only
// waits left, lists them one line each, exit 4 (checks 3, 4); resume says
// what the task waited on (check 5).
func TestDetourThroughTheCLI(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := func(title string) string {
		return execute(t, a, "new", title, "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	}
	first, second := id("First"), id("Second")
	execute(t, a, "take", first, "--model", "claude-opus-5-5", "--as", "kestrel")

	// Check 2: a fourth fix: line, and the same in the JSON fixes array.
	code, out := runText(a, "take", second, "--model", "claude-opus-5-5", "--as", "kestrel")
	detour := "hug task dep " + first + " --add " + second + " && hug task note " + first + ` "where I was"`
	// byj479: the take-with is a fifth line after the detour.
	if code != 3 || !strings.Contains(out, "fix: "+detour+"\nfix: hug task take "+second+" --with "+first) || strings.Count(out, "\nfix: ") != 5 {
		t.Fatalf("exit %d, want 3 and five fix: lines, the detour %q fourth:\n%s", code, detour, out)
	}
	code, out, serr := runJSON(a, "take", second, "--model", "claude-opus-5-5", "--as", "kestrel", "--json")
	var r map[string]any
	if e := json.Unmarshal([]byte(out), &r); e != nil || code != 3 {
		t.Fatalf("exit %d: %v\n%s\nstderr:\n%s", code, e, out, serr)
	}
	if fixes, _ := r["fixes"].([]any); len(fixes) != 5 || fixes[3] != detour || r["fix"] != "hug task done "+first {
		t.Fatalf("json fixes: %s", out)
	}

	// Checks 3 and 4: park first on a wait, take second; next --mine is
	// second, not the waiting first (which ranks before it).
	execute(t, a, "wait", first, "detour to "+second+"; was at step 2", "--as", "kestrel")
	execute(t, a, "take", second, "--model", "claude-opus-5-5", "--as", "kestrel")
	if got := execute(t, a, "next", "--mine", "--as", "kestrel")["task"].(Reply)["id"]; got != second {
		t.Fatalf("next --mine during the detour: %v, want %s", got, second)
	}
	execute(t, a, "check", second, "1", "detour done", "--as", "kestrel")
	execute(t, a, "done", second, "--as", "kestrel", "--reflect", "nothing")
	code, out = runText(a, "next", "--mine", "--as", "kestrel")
	wantLine := first + " waiting on detour to " + second + "; was at step 2\n"
	if code != 4 || !strings.Contains(out, "\n"+wantLine) || !strings.Contains(out, "nothing for kestrel to work on now") {
		t.Fatalf("next --mine with only a wait: exit %d, want 4 listing %q:\n%s", code, wantLine, out)
	}
	code, out, serr = runJSON(a, "next", "--mine", "--as", "kestrel", "--json")
	if e := json.Unmarshal([]byte(out), &r); e != nil || code != 4 {
		t.Fatalf("exit %d: %v\n%s\nstderr:\n%s", code, e, out, serr)
	}
	if ws, _ := r["waiting"].([]any); len(ws) != 1 || ws[0] != strings.TrimSuffix(wantLine, "\n") {
		t.Fatalf("json waiting: %s", out)
	}

	// Check 5: resume says what it waited on, once.
	code, out = runText(a, "resume", first)
	if want := "was waiting on: detour to " + second + "; was at step 2\n"; code != 0 || strings.Count(out, want) != 1 {
		t.Fatalf("resume: exit %d, want one %q:\n%s", code, want, out)
	}
	if got := execute(t, a, "next", "--mine", "--as", "kestrel")["task"].(Reply)["id"]; got != first {
		t.Fatalf("next --mine after resume: %v, want %s", got, first)
	}
}
