package cli

import (
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// releaseBoard is personBoard's board (jhf, erik, hhz) with a coordinator
// and a task whose checks 1-2 hold and whose checks 3-4 are Jørgen's
// after-release confirmations (check 4 with a verb no list names, as
// 5z4zt4's, ebben8), plus one whose open check is an agent's. A
// release tag contains both (deduced).
func releaseBoard(t *testing.T) (a *App, person, agent string) {
	t.Helper()
	a = app(t)
	execute(t, a, "init")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no", "--name", "Jørgen H. Fjeld")
	execute(t, a, "coordinator", "crab")
	person = execute(t, a, "new", "Lint view", "--goal", "g", "--purpose", "p", "--principle", "q",
		"--accept", "built", "--accept", "tested", "--accept", "Jørgen confirms on his board after release", "--accept", "Jørgen attaches a screenshot to it and the detail shows it inline")["task"].(Reply)["id"].(string)
	agent = execute(t, a, "new", "Colours", "--goal", "g", "--purpose", "p", "--principle", "q",
		"--accept", "built", "--accept", "a tmux test drives the Files section")["task"].(Reply)["id"].(string)
	for i, id := range []string{person, agent} {
		execute(t, a, "take", id, "--as", "worker"+string(rune('a'+i)), "--model", "m")
		execute(t, a, "check", id, "1")
	}
	execute(t, a, "check", person, "2")
	git(t, a.Cwd, "tag", "v0.1.0")
	deduced(t, a)
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Ship ("+person+", "+agent+")")
	git(t, a.Cwd, "tag", "v0.1.1")
	return a, person, agent
}

// yf39ya check 3: after the release, a shipped task whose only open checks
// are a person's is asked of that person (check 3 first, one at a time),
// shows in their ASKED and raises no shipped-not-done; one with an agent's
// open check is handed to its owner as active work (yf39ya check 8,
// TestReleaseHandsAgentCheckToOwner), no finding either. Their tick of check 3 asks check 4; their
// tick of check 4 closes it, with the release in History. A text answer
// closes nothing.
func TestShippedPersonCheckIsAsked(t *testing.T) {
	t.Parallel()
	a, person, agent := releaseBoard(t)
	got := deduced(t, a)
	if !strings.Contains(strings.Join(got, "\n"), "marked: shipped in v0.1.1, asked jhf about check 3") {
		t.Fatalf("deduce report: %q", got)
	}
	tk := stateOf(t, a, person)
	if tk.Stage != "asked" || tk.Fields["asked_for"] != "jhf" || tk.Fields["asked_check"] != "3" || tk.Fields["asked_by"] != "crab" {
		t.Fatalf("person task: stage %q fields %v\n%s", tk.Stage, tk.Fields, taskFile(t, a, person))
	}
	r := execute(t, a, "lint")
	codes := map[string]string{}
	for _, f := range r["findings"].([]task.Finding) {
		id, _ := task.IDOf(f.Path)
		if c := task.FindingCode(f.Problem); task.QuestionCode(c) {
			codes[id] = c
		}
	}
	if codes[person] != "" || codes[agent] != "" {
		t.Fatalf("question findings %v (person %s agent %s)", codes, person, agent)
	}
	// A text answer: nothing closes, nothing re-asked.
	execute(t, a, "answer", person, "no, the colour is broken", "--as", "jhf")
	if tk := stateOf(t, a, person); tk.State != "doing" || tk.Stage == "asked" {
		t.Fatalf("after a text answer: %s/%s", tk.State, tk.Stage)
	}
	// Asked again (as the release did), jhf ticks 3: check 4 is asked.
	execute(t, a, "ask", person, "--to", "jhf", "--check", "3", "Jørgen confirms on his board after release", "--as", "crab")
	r = execute(t, a, "check", person, "3", "--as", "jhf")
	if tk := stateOf(t, a, person); tk.Stage != "asked" || tk.Fields["asked_check"] != "4" {
		t.Fatalf("after ticking 3: %s fields %v warnings %v", tk.Stage, tk.Fields, r["warnings"])
	}
	r = execute(t, a, "check", person, "4", "--as", "jhf")
	tk = stateOf(t, a, person)
	if tk.State != "done" || tk.Fields["done_release"] != "v0.1.1" {
		t.Fatalf("after ticking 4: %s done_release %q\n%s", tk.State, tk.Fields["done_release"], taskFile(t, a, person))
	}
	if f := taskFile(t, a, person); !strings.Contains(f, "(done after jhf ticked check 4 (release v0.1.1))") {
		t.Fatalf("History:\n%s", f)
	}
	if w, _ := r["warnings"].([]string); len(w) == 0 || !strings.Contains(strings.Join(w, " "), "after the release: done") {
		t.Fatalf("check reply warnings %v", r["warnings"])
	}
}
