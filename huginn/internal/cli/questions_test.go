package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// at3qep checks 1 and 2: on a real board, a landed ticket with no check
// ticked is a question in hug task lint and in hug task show, with the
// observation and one fix; running the fix clears it. A doing ticket that
// nothing touched for IdleAfter is asked about from the task branch's git
// history; a note clears it.
func TestQuestionsInLintAndShow(t *testing.T) {
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Landed work", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one", "--accept", "two")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--model", "test-model")
	execute(t, a, "note", id, task.LandedNote(task.Commit{Hash: "61bfb21aaaaaaa", Subject: "hug: work (" + id + ")", Date: "2026-11-17"}, "main"))

	_, lint := runText(a, "lint", "--code", "landed-unchecked")
	want := "landed, but no check is ticked: commit 61bfb21 naming " + id + " reached main (its note says so) and 0 of 2 checks are ticked or struck; tester: which checks hold?"
	if !strings.Contains(lint, want) || !strings.Contains(lint, "hug task check "+id+" 1") {
		t.Fatalf("lint misses the question:\n%s", lint)
	}
	_, show := runText(a, "show", id)
	if !strings.Contains(show, "[landed-unchecked] "+want) || !strings.Contains(show, "fix: hug task check "+id+" 1") {
		t.Fatalf("show misses the question:\n%s", show)
	}
	execute(t, a, "check", id, "1")
	if _, out := runText(a, "show", id); strings.Contains(out, "landed-unchecked") {
		t.Fatalf("question stays after the fix:\n%s", out)
	}

	// (b) idle: every commit on the task is older than IdleAfter once it is
	// a nanosecond, and doing_since (minute precision) is older too after a
	// minute boundary; make both hold by moving IdleAfter, not the clock.
	old := task.IdleAfter
	task.IdleAfter = -time.Hour
	defer func() { task.IdleAfter = old }()
	_, out := runText(a, "show", id)
	if !strings.Contains(out, "[doing-idle] doing, but nothing on it for") || !strings.Contains(out, `hug task note `+id+` "where it stands"`) {
		t.Fatalf("show misses the idle question:\n%s", out)
	}
	task.IdleAfter = time.Hour
	execute(t, a, "note", id, "still on it")
	if _, out := runText(a, "show", id); strings.Contains(out, "doing-idle") {
		t.Fatalf("idle question stays after a note:\n%s", out)
	}
}
