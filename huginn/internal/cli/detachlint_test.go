package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bqfrda check 1, end to end: hug task attach then hug task detach leaves
// no missing-file in hug task lint ID (on main: the same missing file
// twice, from the attach and the detach notes).
func TestDetachLeavesNoLintFinding(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Files", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	file := filepath.Join(t.TempDir(), "ev.txt")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	execute(t, a, "attach", id, file)
	execute(t, a, "detach", id, "ev.txt")
	if _, out := runText(a, "lint", id); strings.Contains(out, "missing-file") {
		t.Fatalf("after attach and detach, lint reports missing-file:\n%s", out)
	}
}

// 1phmre, end to end (w40cmk's note): a quoted .hug/tasks path of a
// Statbus task is no finding in hug task lint ID; a task: link to no task
// is, and its fix line runs as given, without an editor, and clears it.
func TestQuotedOtherBoardPathAndTaskLinkFix(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Layout", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	execute(t, a, "note", id, "Proposal (crab, pending his yes): one folder per task holding everything: .hug/tasks/37bryg/README.md is the task, its files sit beside it")
	if _, out := runText(a, "lint", id); strings.Contains(out, "missing-task-ref") || strings.Contains(out, "37bryg") {
		t.Fatalf("a quoted Statbus path is reported:\n%s", out)
	}
	execute(t, a, "note", id, "log in task:zz9zz9/run.log")
	_, out := runText(a, "lint", id)
	want := `hug task note ` + id + ` "task:zz9zz9/run.log (lost)"`
	if !strings.Contains(out, "[missing-task-ref] task:zz9zz9/run.log: no task zz9zz9") || !strings.Contains(out, want) || strings.Contains(out, "hug task edit") {
		t.Fatalf("a task: link to no task, want missing-task-ref with fix %q:\n%s", want, out)
	}
	if code, out := runText(a, "note", id, "task:zz9zz9/run.log (lost)"); code != 0 {
		t.Fatalf("the fix: exit %d\n%s", code, out)
	}
	if _, out := runText(a, "lint", id); strings.Contains(out, "missing-task-ref") {
		t.Fatalf("after the fix, lint still reports it:\n%s", out)
	}
}
