package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// tb0dra: done asks one question; --reflect stores the answer as a
// Reflection note; reflections lists open ones; resolve closes one.
func TestDoneReflectAndReflections(t *testing.T) {
	today := pinClock(t, lastTick) // the note's date is the board clock's (jsyksd)
	a, id := takeFixture(t)
	execute(t, a, "take", id, "--as", "poodle", "--model", "claude-opus-5-5")
	// A blocked done stores nothing.
	if code, out := runText(a, "done", id, "--reflect", "early"); code != 3 || strings.Contains(taskFile(t, a, id), "Reflection (") {
		t.Fatalf("blocked done: exit %d\n%s", code, out)
	}
	execute(t, a, "check", id, "1")
	// Plain done asks the question, in one line, and stores only that it
	// went unanswered (2k68eg), never an open reflection.
	code, out := runText(a, "done", id)
	if code != 0 || strings.Count(out, "now: reflect: ") != 1 || !strings.Contains(out, "hug task done "+id+" --reflect") || strings.Contains(taskFile(t, a, id), "Reflection (open") {
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

// doneWith makes a task, takes it as who, ticks it and closes it with args.
func doneWith(t *testing.T, a *App, who string, args ...string) string {
	t.Helper()
	r := execute(t, a, "new", "Reflect "+who, "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")
	id := r["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", who, "--model", "claude-opus-5-5")
	execute(t, a, "check", id, "1")
	execute(t, a, append([]string{"done", id}, args...)...)
	return id
}

// 2k68eg check 2: --by-instruction puts the three reflections about
// 'hug task check ID N' (at3qep, 49n9kw, x1ra0t on hug's board) in one
// group, the largest first, with the ungrouped rest last.
func TestReflectionsByInstruction(t *testing.T) {
	t.Parallel()
	a := app(t)
	a.Env["HUG_USER"], a.Env["HUG_MODEL"] = "", ""
	check := []string{
		"One gap: hug task check takes no evidence text ('hug task check ID N'), but AGENTS.md says 'hug task check ID N with evidence'; I wrote the evidence as a separate note each time.",
		"nothing misled me. One gap: 'hug task check ID N' takes no evidence, so each tick needs a note first.",
		"hug task check's help says 'hug task check ID N' with no evidence argument, while the board instructions say 'hug task check ID N with evidence'.",
	}
	var ids []string
	for i, text := range check {
		ids = append(ids, doneWith(t, a, fmt.Sprintf("agent%d", i), "--reflect", text))
	}
	other := doneWith(t, a, "agent9", "--reflect", "the flag in \"hug task note ID --as NAME\" was refused")
	loose := doneWith(t, a, "agent8", "--reflect", "A ticket held work for two repos.")
	r := execute(t, a, "reflections", "--by-instruction")
	gs := r["groups"].([]task.ReflectionGroup)
	if len(gs) != 3 || gs[0].Key != "hug task check" || len(gs[0].Reflections) != 3 || gs[1].Key != "hug task note" || gs[2].Key != "" || gs[2].Reflections[0].ID != loose {
		t.Fatalf("groups %+v", gs)
	}
	for i, x := range gs[0].Reflections {
		if x.ID != ids[i] {
			t.Fatalf("group order %+v, want %v", gs[0].Reflections, ids)
		}
	}
	_, out := runText(a, "reflections", "--by-instruction")
	for _, want := range []string{"hug task check  (3 open)\n  " + ids[0] + " 1  open  ", "hug task note  (1 open)\n  " + other + " 1", "no instruction named  (1 open)\n  " + loose + " 1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("text lacks %q:\n%s", want, out)
		}
	}
	if code, out := runText(a, "reflections", "--by-instruction", "--all"); code != 0 || !strings.Contains(out, "hug task check  (3 open)") {
		t.Fatalf("--all: exit %d\n%s", code, out)
	}
}

// 2k68eg check 3: a done with no answer leaves one unanswered line, so the
// answer rate shows; a later answer replaces it, "nothing" removes it.
// The line is dated by the board clock, pinned a tenth of a second before
// midnight: CI 44424953831 dated it after midnight and compared it with a
// day read before (jsyksd).
func TestDoneWithoutAnswerIsCounted(t *testing.T) {
	today := pinClock(t, lastTick)
	a := app(t)
	a.Env["HUG_USER"], a.Env["HUG_MODEL"] = "", ""
	plain := doneWith(t, a, "rabbit")
	if f := taskFile(t, a, plain); strings.Count(f, "Reflection (unanswered, "+today+", rabbit)") != 1 {
		t.Fatalf("plain done:\n%s", f)
	}
	// A repeated plain done asks again and writes no second line.
	if code, out := runText(a, "done", plain); code != 0 || !strings.Contains(out, "now: reflect: ") || strings.Count(taskFile(t, a, plain), "Reflection (") != 1 {
		t.Fatalf("repeat: exit %d\n%s\n%s", code, out, taskFile(t, a, plain))
	}
	answered := doneWith(t, a, "hibiscus", "--reflect", "the 'Never edit .hug/tasks/ files by hand' rule held")
	silent := doneWith(t, a, "otter")
	r := execute(t, a, "reflections")
	if rs := r["reflections"].([]task.Reflection); len(rs) != 1 || rs[0].ID != answered {
		t.Fatalf("open list %+v", rs)
	}
	if u := r["unanswered"].([]string); len(u) != 2 || u[0] != plain && u[1] != plain {
		t.Fatalf("unanswered %v", r["unanswered"])
	}
	_, out := runText(a, "reflections")
	if !strings.Contains(out, "unanswered: 2 done without an answer: ") || !strings.Contains(out, plain) || !strings.Contains(out, silent) {
		t.Fatalf("text:\n%s", out)
	}
	if r := execute(t, a, "reflections", "--all"); len(r["reflections"].([]task.Reflection)) != 1 {
		t.Fatalf("--all lists unanswered: %v", r)
	}
	// The late answer takes the unanswered line's place.
	execute(t, a, "done", plain, "--reflect", "late: the 'hug task check ID N' help named no evidence")
	if f := taskFile(t, a, plain); strings.Contains(f, "unanswered") || strings.Count(f, "Reflection (open, "+today+", rabbit): late:") != 1 {
		t.Fatalf("late answer:\n%s", f)
	}
	execute(t, a, "done", silent, "--reflect", "nothing")
	if f := taskFile(t, a, silent); strings.Contains(f, "Reflection (") {
		t.Fatalf("nothing kept a line:\n%s", f)
	}
	if r := execute(t, a, "reflections"); len(r["unanswered"].([]string)) != 0 {
		t.Fatalf("still unanswered %v", r["unanswered"])
	}
	// An unanswered line is not a reflection to resolve.
	again := doneWith(t, a, "kestrel")
	if code, out := runText(a, "reflections", "resolve", again, "1", "applied"); code != 2 || !strings.Contains(out, "fix: hug task done "+again+" --reflect") {
		t.Fatalf("resolve unanswered: exit %d\n%s", code, out)
	}
}
