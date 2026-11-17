package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// r5r8c1: Seid (2026-11-17, item 3c) saw hug task take refuse a task with an
// open dependency as "taken by  (id)": no name, no reason. A take refused
// by a dependency names each open one, id and title and status, with a fix.
func TestTakeBlockedNamesTheDependency(t *testing.T) {
	t.Parallel()
	for _, remote := range []bool{false, true} {
		a := app(t)
		if remote {
			a, _, _ = clones(t)
		} else {
			execute(t, a, "init")
		}
		blocker := execute(t, a, "new", "Scratch check reads hug", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
		// Seid's c1s5kc was doing, taken by another agent.
		other := &App{Cwd: a.Cwd, In: a.In, Out: a.Out, Err: a.Err, Env: map[string]string{}, Sync: a.Sync}
		for k, v := range a.Env {
			other.Env[k] = v
		}
		other.Env["HUG_USER"] = "orangutan"
		execute(t, other, "take", blocker, "--model", "test-model")
		id := execute(t, a, "new", "Retire the old board", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C", "--dep", blocker)["task"].(Reply)["id"].(string)
		code, out := runText(a, "take", id, "--model", "test-model")
		if code != 3 {
			t.Errorf("remote %v: exit %d, want 3:\n%s", remote, code, out)
		}
		if strings.Contains(out, "taken by  (") || strings.Contains(out, "taken by ") {
			t.Errorf("remote %v: says taken:\n%s", remote, out)
		}
		for _, want := range []string{blocker, "Scratch check reads hug", "doing", "fix: "} {
			if !strings.Contains(out, want) {
				t.Errorf("remote %v: missing %q:\n%s", remote, want, out)
			}
		}
	}
}

// r5r8c1, the shape Seid has: 13 imported tasks carry `assignee: [""]`
// (9htfkh among them). The empty name counted as an assignee, so take
// said "taken by  (id)" before it looked at the dependency.
func TestTakeEmptyAssigneeIsNotTaken(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	blocker := execute(t, a, "new", "Scratch check reads hug", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	id := execute(t, a, "new", "Retire the old board", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C", "--dep", blocker)["task"].(Reply)["id"].(string)
	path := taskPath(t, a, id)
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	body := strings.Replace(string(b), "assignee: []", `assignee: [""]`, 1)
	if body == string(b) {
		t.Fatalf("fixture has no assignee: []:\n%s", b)
	}
	if e := os.WriteFile(path, []byte(body), 0o644); e != nil {
		t.Fatal(e)
	}
	git(t, filepath.Dir(path), "commit", "-qam", "assignee [\"\"] as imported")
	code, out := runText(a, "take", id, "--model", "test-model")
	if code != 3 || strings.Contains(out, "taken by") {
		t.Fatalf("exit %d, want 3 and no 'taken by':\n%s", code, out)
	}
	for _, want := range []string{blocker, "Scratch check reads hug", "todo", "fix: "} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	// Without the dependency it is taken, not refused as someone's.
	execute(t, a, "dep", id, "--remove", blocker)
	if code, out := runText(a, "take", id, "--model", "test-model"); code != 0 {
		t.Fatalf("take after the dependency went: exit %d\n%s", code, out)
	}
}

func taskPath(t *testing.T, a *App, id string) string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(a.Cwd, TaskDir, "*"+id+"*.md"))
	if len(m) != 1 {
		t.Fatalf("task file for %s: %v", id, m)
	}
	return m[0]
}
