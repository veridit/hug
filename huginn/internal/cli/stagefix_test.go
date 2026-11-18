package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 3nhp67 check 3: s6ff5k as it is on hug's board (5b8733b9): stage
// waiting, asked_* and waiting_* fields. hug task lint names a fix; running
// that fix as an agent would makes the finding go, and the file ends as
// doing/waiting with waiting_* only, the asked_* fields kept in History.
func TestStageDisagreesFixRuns(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Park", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "hatchling", "--model", "m")
	execute(t, a, "wait", id, "parked: no fourth arm", "--as", "hatchling")
	board := filepath.Join(a.Cwd, TaskDir)
	m, _ := filepath.Glob(filepath.Join(board, "*"+id+"*.md"))
	raw, _ := os.ReadFile(m[0])
	stale := "asked_by: \"hatchling\"\nasked_for: \"crab\"\nasked_since: \"2026-11-17T20:20\"\nasked_what: \"H3 failed Part 2; parked per crab ruling\"\n"
	old := strings.Replace(string(raw), "\nwaiting_on:", "\n"+stale+"waiting_on:", 1)
	if e := os.WriteFile(m[0], []byte(old), 0o644); e != nil {
		t.Fatal(e)
	}
	git(t, board, "-c", "core.hooksPath=/dev/null", "commit", "-qam", "as 5b8733b9 left s6ff5k")
	_, out := runText(a, "lint", id)
	if !strings.Contains(out, "stage-disagrees") || !strings.Contains(out, "holds the fields of 2 stages (asked, waiting)") {
		t.Fatalf("lint:\n%s", out)
	}
	fix := ""
	for _, l := range strings.Split(out, "\n") {
		if f, ok := strings.CutPrefix(strings.TrimSpace(l), "next: "); ok {
			fix, _, _ = strings.Cut(f, "; then ")
			break
		}
	}
	if want := "hug task set " + id + " --state doing"; fix != want {
		t.Fatalf("first fix %q, want %q, in:\n%s", fix, want, out)
	}
	args := strings.Fields(strings.TrimPrefix(fix, "hug task "))
	execute(t, a, args...)
	if _, out := runText(a, "lint", id); strings.Contains(out, "stage-disagrees") || strings.Contains(out, "stray-node-field") || strings.Contains(out, "unknown-person") {
		t.Fatalf("after %q, lint:\n%s", fix, out)
	}
	f := taskFile(t, a, id)
	if !strings.Contains(f, `stage: "waiting"`) || strings.Contains(f, "\nasked_") || !strings.Contains(f, `waiting_on: "parked: no fourth arm"`) {
		t.Fatalf("after %q:\n%s", fix, f)
	}
	if !strings.Contains(f, "asked_what: H3 failed Part 2; parked per crab ruling") {
		t.Fatalf("History lost asked_what:\n%s", f)
	}
}
