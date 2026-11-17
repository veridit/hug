package cli

import (
	"strings"
	"testing"
)

// 456vyt check 1: text show of a waiting task prints the wait as line 2,
// worded as the TUI row: who, what, and since when.
func TestShowPrintsWaitAsSecondLine(t *testing.T) {
	t.Parallel()
	a, id := takeFixture(t)
	execute(t, a, "take", id, "--model", "test-model")
	execute(t, a, "wait", id, "--on", "jhf", "approve the diff")
	code, out := runText(a, "show", id)
	lines := strings.Split(out, "\n")
	if code != 0 || len(lines) < 2 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if !strings.HasPrefix(lines[1], "stage: waiting on jhf: approve the diff since 20") {
		t.Fatalf("line 2 is %q, want the wait:\n%s", lines[1], out)
	}
	// A wait on a thing, not a person.
	execute(t, a, "wait", id, "CI run 123")
	if _, out = runText(a, "show", id); !strings.HasPrefix(strings.Split(out, "\n")[1], "stage: waiting on CI run 123 since 20") {
		t.Fatalf("wait on a thing:\n%s", out)
	}
}

// 456vyt check 2: a task waiting on nothing prints its stage word alone,
// no wait text; resume ends the wait line.
func TestShowStageWithoutWait(t *testing.T) {
	t.Parallel()
	a, id := takeFixture(t)
	execute(t, a, "take", id, "--model", "test-model")
	execute(t, a, "wait", id, "--on", "jhf", "approve")
	execute(t, a, "resume", id)
	_, out := runText(a, "show", id)
	if l := strings.Split(out, "\n")[1]; l != "stage: active" {
		t.Fatalf("line 2 is %q, want %q:\n%s", l, "stage: active", out)
	}
	if strings.Contains(out, "waiting on") {
		t.Fatalf("a resumed task still shows a wait:\n%s", out)
	}
}
