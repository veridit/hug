package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// tzc9hb: take teaches the loop. Todos are your own next steps on the task;
// waits and dependencies go in the task; next: reads the task first.
func TestTakeRemindsTodosAndShowsTask(t *testing.T) {
	a, id := takeFixture(t)
	code, out := runText(a, "take", id, "--model", "claude-opus-5-5")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	var now []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "now: ") {
			now = append(now, l)
		}
	}
	all := strings.Join(now, "\n")
	if len(now) > 2 || !strings.Contains(all, "todos: your own next steps on "+id) || !strings.Contains(all, "is not a todo") ||
		!strings.Contains(all, "hug task note "+id) || !strings.Contains(all, "hug task dep "+id) || !strings.Contains(all, "hug task done "+id+", then hug task next") {
		t.Fatalf("reminder (%d lines):\n%s", len(now), out)
	}
	if !strings.HasSuffix(out, "next: hug task show "+id+"\n") {
		t.Fatalf("take must end with next: hug task show %s:\n%s", id, out)
	}
	// A failed take prints no reminder.
	code, out = runText(a, "take", id, "--as", "someone-else", "--model", "claude-opus-5-5")
	if code == 0 || strings.Contains(out, "now: todos") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}

// tzc9hb check 1: the installed section states the loop.
func TestAgentSectionStatesTodoLoop(t *testing.T) {
	for _, want := range []string{
		"Todos are your own next steps on the task you took",
		"waits and dependencies go in tasks (hug task note, hug task dep), never\nin todos",
		"another agent's work is never your todo",
		"run hug task next: an empty todo list means take the next\ntask, not blocked",
	} {
		if !strings.Contains(agentSection, want) {
			t.Errorf("agentSection lacks %q", want)
		}
	}
	for _, harness := range []string{"todowrite", "TodoWrite", "jcode", "swarm"} {
		if strings.Contains(agentSection, harness) {
			t.Errorf("agentSection names harness word %q", harness)
		}
	}
}

// tzc9hb check 4: setup notices a section from an older hug and says how
// to update it; the current section counts as done.
func TestSetupFlagsStaleTaskSection(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "root"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	t.Setenv("HOME", t.TempDir())
	step := func() map[string]any {
		t.Helper()
		r, err := (&App{Cwd: dir}).Execute(context.Background(), []string{"setup", "--json"})
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(r["setup"])
		var steps []map[string]any
		json.Unmarshal(b, &steps)
		for _, s := range steps {
			if s["step"] == "agent instructions" {
				return s
			}
		}
		t.Fatal("no agent instructions step")
		return nil
	}
	old := "# Repo\n\n## Tasks: hug\nTrack all work with hug tasks.\nStart: hug task next.\n\n## Other\nkept\n"
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(old), 0o644)
	s := step()
	if s["done"] != false || !strings.Contains(s["do"].(string), `in AGENTS.md, replace the "## Tasks: hug" section`) || !strings.Contains(s["do"].(string), "Todos are your own next steps") {
		t.Fatalf("stale section: %v", s)
	}
	cur := "# Repo\n\n" + agentSection + "\n\n## Other\nkept\n"
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(strings.ReplaceAll(cur, "\n", "\r\n")), 0o644)
	if s := step(); s["done"] != true {
		t.Fatalf("current section (CRLF) not done: %v", s)
	}
}
