package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// tb0dra: done asks one question; --reflect stores the answer as a
// Reflection note; reflections lists open ones; resolve closes one.
func TestDoneReflectAndReflections(t *testing.T) {
	t.Parallel()
	a, id := takeFixture(t)
	execute(t, a, "take", id, "--as", "poodle", "--model", "claude-opus-5-5")
	// A blocked done stores nothing.
	if code, out := runText(a, "done", id, "--reflect", "early"); code != 3 || strings.Contains(taskFile(t, a, id), "Reflection (") {
		t.Fatalf("blocked done: exit %d\n%s", code, out)
	}
	execute(t, a, "check", id, "1")
	// Plain done asks the question, in one line, and stores nothing.
	code, out := runText(a, "done", id)
	if code != 0 || strings.Count(out, "now: reflect: ") != 1 || !strings.Contains(out, "hug task done "+id+" --reflect") || strings.Contains(taskFile(t, a, id), "Reflection (") {
		t.Fatalf("plain done: exit %d\n%s", code, out)
	}
	// "nothing" is a fine answer and needs no note.
	if code, _ := runText(a, "done", id, "--reflect", "Nothing."); code != 0 || strings.Contains(taskFile(t, a, id), "Reflection (") {
		t.Fatal("nothing stored a note")
	}
	text := `the AGENTS.md line "Finish: hug task done ID when every acceptance check is ticked" did not say check 4 waits for a release`
	code, out = runText(a, "done", id, "--reflect", text)
	if code != 0 || strings.Contains(out, "now: reflect:") {
		t.Fatalf("done --reflect: exit %d\n%s", code, out)
	}
	today := time.Now().Format("2006-01-02")
	if f := taskFile(t, a, id); !strings.Contains(f, "Reflection (open, "+today+", poodle): "+text) {
		t.Fatalf("task file:\n%s", f)
	}
	r := execute(t, a, "reflections")
	rs := r["reflections"].([]task.Reflection)
	if len(rs) != 1 {
		t.Fatalf("reflections: %v", r)
	}
	one := rs[0]
	if one.ID != id || one.N != 1 || one.State != "open" || one.Date != today || one.Who != "poodle" || one.Instruction != "Finish: hug task done ID when every acceptance check is ticked" {
		t.Fatalf("reflection %v", one)
	}
	if !strings.HasPrefix(r["next"].(string), "hug task reflections resolve "+id+" 1 applied") {
		t.Fatal(r["next"])
	}
	_, out = runText(a, "reflections")
	if !strings.Contains(out, id+" 1  open  "+today+" poodle") || !strings.Contains(out, `instruction: "Finish: hug task done ID`) {
		t.Fatalf("text:\n%s", out)
	}
	if code, out := runText(a, "reflections", "resolve", id, "1", "maybe"); code != 2 || !strings.Contains(out, "fix: hug task reflections resolve "+id+" 1 applied") {
		t.Fatalf("bad state: exit %d\n%s", code, out)
	}
	if code, out := runText(a, "reflections", "resolve", id, "2", "applied"); code != 2 || !strings.Contains(out, "1 reflection(s), not 2") || !strings.Contains(out, "fix: hug task reflections --all") {
		t.Fatalf("bad n: exit %d\n%s", code, out)
	}
	execute(t, a, "reflections", "resolve", id, "1", "applied")
	if r := execute(t, a, "reflections"); len(r["reflections"].([]task.Reflection)) != 0 {
		t.Fatalf("still open: %v", r)
	}
	if _, out := runText(a, "reflections"); !strings.Contains(out, "no open reflections") {
		t.Fatal(out)
	}
	r = execute(t, a, "reflections", "--all")
	if rs := r["reflections"].([]task.Reflection); len(rs) != 1 || rs[0].State != "applied" {
		t.Fatalf("--all: %v", r)
	}
	// Resolve is idempotent.
	execute(t, a, "reflections", "resolve", id, "1", "applied")
}

// tb0dra check 4: the installed section asks for an honest answer.
func TestAgentSectionAsksReflection(t *testing.T) {
	t.Parallel()
	for _, want := range []string{"Done asks whether\nthe instructions guided you well; answer honestly with --reflect", `or "nothing".`} {
		if !strings.Contains(agentSection, want) {
			t.Errorf("agentSection lacks %q", want)
		}
	}
}

func TestReflectionParse(t *testing.T) {
	t.Parallel()
	tk := task.Task{ID: "abcdef", Title: "T"}
	rs := task.ParseReflections(tk, "## Notes\nplain note\nReflection (open, 2026-11-17): the hint 'hug task take ID' said nothing about cwd\nReflection (dismissed, 2026-11-16, crab): no quote here\n")
	if len(rs) != 2 || rs[0].Who != "" || rs[0].Instruction != "hug task take ID" || rs[1].State != "dismissed" || rs[1].Who != "crab" || rs[1].N != 2 {
		t.Fatalf("%+v", rs)
	}
}
