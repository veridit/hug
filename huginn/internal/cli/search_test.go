package cli

import (
	"strings"
	"testing"
)

// 49n9kw checks 1, 2, 4 through the command: ranked text with the matched
// field and snippet; an alias first, then a note citing it; a note added is
// found by the next search; filters narrow it.
func TestSearchCommand(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	oom := execute(t, a, "new", "Memory limit exit 137 in the worker", "--goal", "Stop the OOM kill", "--accept", "C")["task"].(Reply)["id"].(string)
	logs := execute(t, a, "new", "Logging", "--goal", "exit code 137 means the memory limit was hit", "--accept", "C", "--type", "bug")["task"].(Reply)["id"].(string)
	code, out := runText(a, "search", "memory limit exit 137")
	lines := strings.Split(out, "\n")
	if code != 0 || !strings.HasPrefix(lines[0], oom+"  todo     Memory limit exit 137 in the worker") || lines[1] != "        title: Memory limit exit 137 in the worker" ||
		!strings.HasPrefix(lines[2], logs+"  ") || !strings.HasPrefix(lines[3], "        goal: exit code 137 means the memory limit") || !strings.Contains(out, "next: hug task show "+oom+"\n") {
		t.Fatalf("search: %d\n%s", code, out)
	}
	if _, out := runText(a, "search", "memory", "--type", "bug"); strings.Contains(out, oom) || !strings.Contains(out, logs) {
		t.Fatalf("--type bug:\n%s", out)
	}
	// An alias ranks above a note that cites it.
	cited := execute(t, a, "new", "Follow-up", "--goal", "Later", "--accept", "C")["task"].(Reply)["id"].(string)
	execute(t, a, "note", cited, "continues STATBUS-402 after the import")
	r := execute(t, a, "search", "STATBUS-402")
	if rs := r["results"].([]Reply); len(rs) != 1 || rs[0]["id"] != cited || rs[0]["field"] != "notes" {
		t.Fatalf("note found by the next search: %v", rs)
	}
	if code, out := runText(a, "search", "zzzzqq"); code != 0 || !strings.HasPrefix(out, `no task matches "zzzzqq"`) {
		t.Fatalf("no match: %d %q", code, out)
	}
	if code, out := runText(a, "search"); code != 2 || !strings.Contains(out, "fix: hug task search") {
		t.Fatalf("no words: %d %q", code, out)
	}
}
