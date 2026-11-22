package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// jnaspy check 2: hug task show prints Purpose, Principle and Details
// after the goal and before the checks.
func TestShowPrintsDescriptionBeforeChecks(t *testing.T) {
	t.Parallel()
	a := app(t)
	r := execute(t, a, "new", "Ordered", "--goal", "Where we aim", "--purpose", "Why we aim", "--principle", "How we go", "--details", "Facts that matter", "--accept", "Observable check")
	id := r["task"].(Reply)["id"].(string)
	execute(t, a, "note", id, "a note")
	code, out := runText(a, "show", id)
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	order := []string{"goal: Where we aim", "purpose: Why we aim", "principle: How we go", "details: Facts that matter", "1. [ ] Observable check", "notes: "}
	at := -1
	for _, want := range order {
		i := strings.Index(out, want)
		if i < 0 || i < at {
			t.Fatalf("%q missing or out of order in:\n%s", want, out)
		}
		at = i
	}
	j := execute(t, a, "show", id, "--json")["task"].(Reply)
	if j["purpose"] != "Why we aim" || j["principle"] != "How we go" || j["details"] != "Facts that matter" {
		t.Fatalf("json: purpose %q principle %q details %q", j["purpose"], j["principle"], j["details"])
	}
}

// jnaspy check 3 end to end: hug upgrade on a board with mkzcqr's order
// rewrites it to canonical order with identical text and commits; a second
// run changes nothing. Check 4: lint reports it first, with fix: hug upgrade.
func TestUpgradeReordersSections(t *testing.T) {
	t.Parallel()
	a := app(t)
	r := execute(t, a, "new", "Sort builder", "--goal", "Build a nested sort", "--purpose", "Jørgen sorts", "--principle", "Esc keeps the order", "--accept", "p t Enter")
	id := r["task"].(Reply)["id"].(string)
	m, _ := globID(filepath.Join(a.Cwd, TaskDir), id)
	b, _ := os.ReadFile(m[0])
	// mkzcqr's layout: Depends on moved up between Goal and Purpose.
	text := strings.Replace(string(b), "\n## Depends on\n", "\n", 1)
	text = strings.Replace(text, "\n## Purpose\n", "\n## Depends on\n\n## Purpose\n", 1)
	if err := os.WriteFile(m[0], []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, filepath.Join(a.Cwd, TaskDir), "commit", "-qam", "mkzcqr order")
	_, out := runText(a, "lint", id)
	if !strings.Contains(out, "[section-order] sections out of order: Goal, Depends on, Purpose") || !strings.Contains(out, "next: hug upgrade") {
		t.Fatalf("lint:\n%s", out)
	}
	did := func() string { return strings.Join(execute(t, a, "upgrade", "--apply")["upgrade"].([]string), "; ") }
	if got := did(); !strings.Contains(got, "1 tasks tidied") {
		t.Fatalf("upgrade: %s", got)
	}
	after, _ := os.ReadFile(m[0])
	var heads []string
	for _, l := range strings.Split(string(after), "\n") {
		if strings.HasPrefix(l, "## ") {
			heads = append(heads, l[3:])
		}
	}
	if got := strings.Join(heads, "|"); got != "Goal|Purpose|Principle|Depends on|Acceptance|Notes" {
		t.Fatalf("order %s:\n%s", got, after)
	}
	for _, s := range []string{"## Goal\nBuild a nested sort\n", "## Purpose\nJørgen sorts\n", "## Principle\nEsc keeps the order\n", "- [ ] p t Enter\n"} {
		if !strings.Contains(string(after), s) {
			t.Fatalf("lost %q:\n%s", s, after)
		}
	}
	if st := git(t, filepath.Join(a.Cwd, TaskDir), "status", "--porcelain"); st != "" {
		t.Fatalf("not committed: %s", st)
	}
	// A board from before layout = N gets it last, after the files move (dx9vw2).
	if log := git(t, filepath.Join(a.Cwd, TaskDir), "log", "-2", "--format=%s"); log != upgradedFrom(1)+"\nhug upgrade: backlog leftovers and section order in 1 tasks" {
		t.Fatalf("commits %q", log)
	}
	if got := did(); strings.Contains(got, "tidied") {
		t.Fatalf("second upgrade: %s", got)
	}
	if again, _ := os.ReadFile(m[0]); string(again) != string(after) {
		t.Fatal("second upgrade changed the file")
	}
	if _, out = runText(a, "lint"); strings.Contains(out, "section-order") {
		t.Fatalf("lint after upgrade:\n%s", out)
	}
}
