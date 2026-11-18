package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// 41hhgn check 13 (crab 02:05): v0.7.52's upgrade moved s6ff5k's park
// ("... parked per crab ruling, no fourth arm", waiting_for crab) to
// asked_for crab. A wait moves to asked only when it reads as a question:
// a '?', or a check of the task named. Any other stays a wait; its
// waiting_for goes (no node field any more, so lint would ask for the
// upgrade again), History says so, and the preview lists it with the ask
// that would make it a question.
func TestUpgradeLeavesAParkAsWait(t *testing.T) {
	t.Parallel()
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	execute(t, a, "init")
	ids := map[string]string{}
	for _, c := range [][2]string{{"Park", "hatchling"}, {"Ask", "crab"}, {"Check", "calf"}} {
		ids[c[0]] = execute(t, a, "new", c[0], "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
		execute(t, a, "take", ids[c[0]], "--as", c[1], "--model", "m")
	}
	// The three kinds on hug's board at v0.7.52 (8a18427b), as v0.7.50
	// wrote them: a park, a question, a request naming a check.
	park := "H3 failed Part 2 (10 wrong vs bar 6); parked per crab ruling, no fourth arm"
	waits := map[string][2]string{
		"Park":  {park, "crab"},
		"Ask":   {"a week of use: has he had to remind an agent? look again 2026-11-24", "jhf"},
		"Check": {"approve the design (attached: d.md), check 1, and answer Q1-Q6 at its end", "jhf"},
	}
	board := filepath.Join(a.Cwd, TaskDir)
	for name, w := range waits {
		execute(t, a, "wait", ids[name], w[0])
		m, _ := filepath.Glob(filepath.Join(board, "*"+ids[name]+"*.md"))
		raw, _ := os.ReadFile(m[0])
		old := strings.Replace(string(raw), "\nwaiting_on:", "\nwaiting_for: \""+w[1]+"\"\nwaiting_on:", 1)
		if e := os.WriteFile(m[0], []byte(old), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	git(t, board, "-c", "core.hooksPath=/dev/null", "commit", "-qam", "as v0.7.50 wrote them")
	code, preview := runText(a, "upgrade")
	for _, want := range []string{
		"2 questions to a person moved from waiting to asked",
		"1 wait on a person left waiting (it asks nothing: no '?', no check named, no request such as approve)",
		ids["Park"] + " waiting_for crab: " + strconv.Quote(park),
		"hug task ask " + ids["Park"] + " --to crab ",
	} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview (exit %d) lacks %q:\n%s", code, want, preview)
		}
	}
	execute(t, a, "upgrade", "--apply")
	f := taskFile(t, a, ids["Park"])
	if !strings.Contains(f, `stage: "waiting"`) || !strings.Contains(f, `waiting_on: "`+park+`"`) || strings.Contains(f, "\nasked_") || strings.Contains(f, "waiting_for:") {
		t.Fatalf("the park moved or kept waiting_for:\n%s", f)
	}
	if !strings.Contains(f, "waiting_for crab removed by hug upgrade (it asks nothing: no '?', no check named, no request such as approve; the wait stays, 41hhgn)") {
		t.Fatalf("History does not say waiting_for went:\n%s", f)
	}
	for _, name := range []string{"Ask", "Check"} {
		if g := taskFile(t, a, ids[name]); !strings.Contains(g, `stage: "asked"`) || !strings.Contains(g, `asked_for: "jhf"`) {
			t.Fatalf("%s did not move to asked:\n%s", name, g)
		}
	}
	if _, out := runText(a, "lint", ids["Park"]); strings.Contains(out, "old layout") {
		t.Fatalf("lint after upgrade:\n%s", out)
	}
	if _, out := runText(a, "upgrade"); strings.Contains(out, "left waiting") || strings.Contains(out, "to asked") {
		t.Fatalf("second preview:\n%s", out)
	}
}
