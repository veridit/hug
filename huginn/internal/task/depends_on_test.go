package task

import (
	"encoding/json"
	"strings"
	"testing"
)

// jorgenEdit is the exact text Jørgen added to task qawmnb on 2026-11-16
// (rx74bp): a question in the Depends on section, not a dependency.
const jorgenEdit = "Is there a task:asdfasdf or ticket:asdlfkjasldkfj?"

func withDeps(lines string) []byte {
	return []byte(strings.Replace(sample, "## Depends on\n", "## Depends on\n"+lines, 1))
}

func TestDependencyLineForms(t *testing.T) {
	for _, c := range []struct{ line, ref, note string }{
		{"- p3rt9a", "p3rt9a", ""},
		{"p3rt9a", "p3rt9a", ""},
		{"* p3rt9a", "p3rt9a", ""},
		{"- task:p3rt9a waits for the gate", "p3rt9a", "waits for the gate"},
		{"- P3RT9A", "P3RT9A", ""},
		{"TASK-123 (note)", "TASK-123", "(note)"},
		{"- GH-12", "GH-12", ""},
		{"- TASK-7@drafts", "TASK-7@drafts", ""},
	} {
		ref, note, ok := parseDependencyLine(c.line)
		if !ok || ref != c.ref || note != c.note {
			t.Errorf("%q: got %q %q %v, want %q %q", c.line, ref, note, ok, c.ref, c.note)
		}
	}
	for _, line := range []string{jorgenEdit, "- " + jorgenEdit, "Is", "- missing", "- task:asdfasdf", "see the notes", "- [ ] p3rt9a", "TODO: later"} {
		if ref, _, ok := parseDependencyLine(line); ok {
			t.Errorf("%q read as dependency %q", line, ref)
		}
	}
}

// Prose in Depends on: its own finding, no dependency, not blocked.
func TestProseInDependsOnIsNotADependency(t *testing.T) {
	d := Parse("zzzzzz.k7m2qx.recall-gate.md", withDeps(jorgenEdit+"\n"))
	if len(d.Task.Dependencies) != 0 {
		t.Fatalf("prose became dependencies: %+v", d.Task.Dependencies)
	}
	if got := Blocked(d.Task, []Task{d.Task}); len(got) != 0 {
		t.Fatalf("prose blocked the task: %+v", got)
	}
	if got := DependencyDetails(d.Task, []Task{d.Task}); len(got) != 0 {
		t.Fatalf("prose shows as dependency rows: %+v", got)
	}
	fs := Lint([]*Document{d})
	if len(fs) != 1 {
		t.Fatalf("want one finding, got %v", fs)
	}
	f := fs[0]
	want := `not a dependency: "Is there a task:asdfasdf or ticket:asdlfkjasldkfj?" (for no dependencies, leave the section empty or remove it; a dependency line is "- <task id>", or run hug task dep k7m2qx --add REF; prose belongs in Notes)`
	if f.Problem != want || f.Line != 15 || FindingCode(f.Problem) != "not-a-dependency" || f.FixLine() != "hug task edit k7m2qx" {
		t.Fatalf("finding:\n%s\nfix: %s", f, f.FixLine())
	}
	if strings.Contains(f.Problem, "dangling") {
		t.Fatal("jargon in finding: " + f.Problem)
	}
	t.Log(f.String() + "\nfix: " + f.FixLine())
}

// A dependency line naming no task: says which id and that no task has it,
// fix removes it, code stays dangling-dependency.
func TestMissingDependencyNamesIDAndFix(t *testing.T) {
	d := Parse("zzzzzz.k7m2qx.recall-gate.md", withDeps("- zz9zz9\n"))
	fs := Lint([]*Document{d})
	if len(fs) != 1 {
		t.Fatalf("want one finding, got %v", fs)
	}
	f := fs[0]
	if f.Problem != "depends on zz9zz9, but no task has that id or alias; remove the line, or write an existing task's id" ||
		FindingCode(f.Problem) != "dangling-dependency" || f.FixLine() != "hug task dep k7m2qx --remove zz9zz9" || f.Line != 15 {
		t.Fatalf("finding:\n%s\nfix: %s", f, f.FixLine())
	}
	if strings.Contains(f.Problem, "dangling") {
		t.Fatal("jargon in finding: " + f.Problem)
	}
	// The missing dependency still blocks: that edge was meant.
	if len(Blocked(d.Task, []Task{d.Task})) != 1 {
		t.Fatal("a real missing dependency must still block")
	}
}

// Prose next to a real dependency: only the real line counts.
func TestProseBesideRealDependency(t *testing.T) {
	a := Parse("a.k7m2qx.a.md", withDeps("- p3rt9a\n"+jorgenEdit+"\n"))
	b := Parse("b.p3rt9a.b.md", []byte(strings.Replace(strings.Replace(sample, "id: k7m2qx", "id: p3rt9a", 1), "status: todo", "status: done", 1)))
	if len(a.Task.Dependencies) != 1 || a.Task.Dependencies[0].Ref != "p3rt9a" {
		t.Fatalf("deps: %+v", a.Task.Dependencies)
	}
	if got := Blocked(a.Task, []Task{a.Task, b.Task}); len(got) != 0 {
		t.Fatalf("blocked by %+v", got)
	}
}

// Every finding carries a fix, also in JSON.
func TestEveryFindingHasFix(t *testing.T) {
	docs := []*Document{
		Parse("garbage.md", []byte("garbage")),
		Parse("a.k7m2qx.a.md", withDeps("- zz9zz9\n"+jorgenEdit+"\n")),
		Parse("b.p3rt9a.b.md", []byte(strings.Replace(strings.Replace(sample, "id: k7m2qx", "id: p3rt9a", 1), "status: todo", "status: done", 1))),
	}
	fs := Lint(docs)
	if len(fs) < 4 {
		t.Fatalf("too few findings: %v", fs)
	}
	for _, f := range fs {
		if !strings.HasPrefix(f.FixLine(), "hug task ") {
			t.Errorf("no runnable fix: %s -> %q", f, f.FixLine())
		}
		var m map[string]any
		b, _ := json.Marshal(f)
		if json.Unmarshal(b, &m) != nil || m["fix"] != f.FixLine() || m["code"] == "" {
			t.Errorf("json: %s", b)
		}
	}
}

// "Nothing" under Depends on (Jørgen, 2026-11-16): the message leads with
// how to say "no dependencies" (6neax1).
func TestNothingInDependsOnSaysLeaveItEmpty(t *testing.T) {
	d := Parse("zzzzzz.qawmnb.recall-nudges-keep-task.md", []byte(strings.Replace(strings.Replace(sample, "id: k7m2qx", "id: qawmnb", 1), "## Depends on\n", "## Depends on\nNothing\n", 1)))
	fs := Lint([]*Document{d})
	want := `not a dependency: "Nothing" (for no dependencies, leave the section empty or remove it; a dependency line is "- <task id>", or run hug task dep qawmnb --add REF; prose belongs in Notes)`
	if len(fs) != 1 || fs[0].Problem != want || len(d.Task.Dependencies) != 0 {
		t.Fatalf("%v", fs)
	}
}
