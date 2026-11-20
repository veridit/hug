package cli

import (
	"strings"
	"testing"
)

// yf39ya check 8 (x8zzzn c3, ynh84t c4 on 2026-11-20: shipped, doing/active,
// the open check crab's own, left as shipped-not-done after the release):
// the release step hands a shipped task whose open check is not a
// person's to its owner as active work, said in the report and in a note,
// never a broken finding. The owner's tick of that check then runs the
// after-release step: it asks the person check that is left, or closes
// the task when every check holds.
func TestReleaseHandsAgentCheckToOwner(t *testing.T) {
	t.Parallel()
	a, person, agent := releaseBoard(t)
	mixed := execute(t, a, "new", "Mixed", "--goal", "g", "--purpose", "p", "--principle", "q",
		"--accept", "built", "--accept", "a tmux test drives the picker", "--accept", "Jørgen confirms on his board after release")["task"].(Reply)["id"].(string)
	execute(t, a, "take", mixed, "--as", "crab", "--model", "m")
	execute(t, a, "check", mixed, "1")
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Ship mixed ("+mixed+")")
	git(t, a.Cwd, "tag", "v0.1.2")
	got := strings.Join(deduced(t, a), "\n")
	for _, want := range []string{
		"marked: shipped in v0.1.1, handed on: check 2 open, workerb's active work",
		"marked: shipped in v0.1.2, handed on: check 2 open, crab's active work",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("deduce report lacks %q:\n%s", want, got)
		}
	}
	codes := questionCodes(t, a)
	for _, id := range []string{agent, mixed} {
		if codes[id] != "" {
			t.Fatalf("%s handed on, but lint says %s (person %s)\n%s", id, codes[id], person, taskFile(t, a, id))
		}
		if tk := stateOf(t, a, id); tk.State != "doing" || tk.Stage != "active" {
			t.Fatalf("%s: %s/%s", id, tk.State, tk.Stage)
		}
	}
	// The owner ticks its check: mixed asks jhf about check 3.
	execute(t, a, "check", mixed, "2", "--as", "crab")
	if tk := stateOf(t, a, mixed); tk.Stage != "asked" || tk.Fields["asked_for"] != "jhf" || tk.Fields["asked_check"] != "3" {
		t.Fatalf("mixed after crab's tick: %s %v\n%s", tk.Stage, tk.Fields, taskFile(t, a, mixed))
	}
	// agent's last check: it closes, with the release in History.
	execute(t, a, "check", agent, "2", "--as", "workerb")
	tk := stateOf(t, a, agent)
	if tk.State != "done" || tk.Fields["done_release"] != "v0.1.1" {
		t.Fatalf("agent after the tick: %s %q\n%s", tk.State, tk.Fields["done_release"], taskFile(t, a, agent))
	}
	if f := taskFile(t, a, agent); !strings.Contains(f, "(done after workerb ticked check 2 (release v0.1.1))") {
		t.Fatalf("History:\n%s", f)
	}
}

// yf39ya check 9 (0gztsy on kg07bt, 2026-11-20: 0gztsy in depends on
// kg07bt, which shipped and waited on review, stayed shipped-not-done): a
// shipped task in depends is not shipped-not-done, its dependency says
// why; once the dependency is done, its person check is asked.
func TestShippedDependentAskedWhenDependencyHolds(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no", "--name", "Jørgen H. Fjeld")
	execute(t, a, "coordinator", "crab")
	dep := newTicket(t, a, "Calm heading")
	id := execute(t, a, "new", "Detail focus", "--goal", "g", "--purpose", "p", "--principle", "q",
		"--accept", "built", "--accept", "Jørgen confirms the detail reads clearly")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "check", id, "1")
	execute(t, a, "take", dep, "--as", "macaque", "--model", "m")
	execute(t, a, "wait", dep, "crab review", "--as", "macaque")
	execute(t, a, "dep", id, "--add", dep)
	if tk := stateOf(t, a, id); tk.Stage != "depends" {
		t.Fatalf("dependent: %s", tk.Stage)
	}
	git(t, a.Cwd, "tag", "v0.1.0")
	deduced(t, a)
	git(t, a.Cwd, "commit", "--allow-empty", "-m", "Ship ("+id+", "+dep+")")
	git(t, a.Cwd, "tag", "v0.1.1")
	deduced(t, a)
	if codes := questionCodes(t, a); codes[id] != "" {
		t.Fatalf("%s waits on %s, but lint says %s\n%s", id, dep, codes[id], taskFile(t, a, id))
	}
	// The dependency holds: the person check is asked.
	execute(t, a, "check", dep, "1", "--as", "crab")
	execute(t, a, "done", dep)
	tk := stateOf(t, a, id)
	if tk.Stage != "asked" || tk.Fields["asked_for"] != "jhf" || tk.Fields["asked_check"] != "2" {
		t.Fatalf("dependent after %s is done: %s %v\n%s", dep, tk.Stage, tk.Fields, taskFile(t, a, id))
	}
}
