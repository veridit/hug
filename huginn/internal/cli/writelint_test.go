package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wdkqzx check 4 (Jørgen 23:51: "the lint should have been returned to the
// agent that tried to modify the task"): every task write ends with that
// task's own findings and fix: line, the block show and take print; on a
// clean task it prints nothing extra.
func TestEveryWriteEndsWithTheTasksLint(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	dirty := execute(t, a, "new", "Dirty", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one", "--accept", "two", "--accept", "three")["task"].(Reply)["id"].(string)
	clean := execute(t, a, "new", "Clean", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one", "--accept", "two", "--accept", "three")["task"].(Reply)["id"].(string)
	useEditor(t, a, "## Principle\nQ\n", "")
	execute(t, a, "edit", dirty)
	fix := "hug task set " + dirty + ` --principle "..."`
	block := "lint: 1 finding\n  line "
	file := filepath.Join(t.TempDir(), "evidence.txt")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Each family, as an agent runs it; ID is the task under test.
	steps := [][]string{
		{"take", "ID", "--model", "test-model", "--as", "AGENT"},
		{"note", "ID", "where it stands"},
		{"check", "ID", "1"},
		{"strike", "ID", "2", "--reason", "not relevant", "--as", "AGENT"},
		{"strike", "ID", "2", "--undo", "--as", "AGENT"},
		{"attach", "ID", file},
		{"detach", "ID", "evidence.txt"},
		{"dep", "ID", "--add", "OTHER"},
		{"dep", "ID", "--remove", "OTHER"},
		{"set", "ID", "--priority", "high"},
		{"wait", "ID", "CI run 1", "--as", "AGENT"},
		{"resume", "ID"},
		{"ask", "ID", "does check 3 hold?", "--to", "crab", "--check", "3", "--as", "AGENT"},
		{"answer", "ID", "yes", "--as", "crab"},
		{"review", "ID", "--by", "crab"},
		{"answer", "ID", "looks good", "--as", "crab"},
	}
	for _, id := range []string{dirty, clean} {
		agent, other := "kestrel", clean
		if id == clean {
			agent, other = "wyvern", dirty
		}
		for _, st := range steps {
			args := make([]string, len(st))
			for i, x := range st {
				args[i] = strings.NewReplacer("ID", id, "OTHER", other, "AGENT", agent).Replace(x)
			}
			code, out := runText(a, args...)
			if code != 0 {
				t.Fatalf("%v: exit %d\n%s", args, code, out)
			}
			if id == dirty && (!strings.Contains(out, block) || !strings.Contains(out, "[missing-section] missing section Principle") || !strings.Contains(out, "fix: "+fix+"\n")) {
				t.Fatalf("%v on a task with a finding does not end with its lint:\n%s", args, out)
			}
			if id == clean && strings.Contains(out, "lint:") {
				t.Fatalf("%v on a clean task prints a lint block:\n%s", args, out)
			}
		}
	}
	// --json carries the block too, as show's.
	r := execute(t, a, "note", dirty, "json")
	if l, ok := r["lint"].(*TaskLint); !ok || l.Fix != fix || len(l.Findings) != 1 {
		t.Fatalf("note --json lint: %+v", r["lint"])
	}
	// todo/design, on a todo task with a long title.
	d := execute(t, a, "new", "Design", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	execute(t, a, "set", d, "--title", strings.Repeat("Long title ", 9))
	if code, out := runText(a, "design", d, "which layout?", "--for", "jhf"); code != 0 || !strings.Contains(out, "lint: 1 finding") || !strings.Contains(out, "[long-title] title is 99 characters") {
		t.Fatalf("design on a task with a finding: exit %d\n%s", code, out)
	}
}
