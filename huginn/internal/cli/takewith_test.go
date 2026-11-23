package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// byj479 (Jørgen 20:37Z, sql_saga: sabertooth worked dee3fq and bqtzce
// for one release, take refused the second, and bqtzce sat in NEXT UP
// "ready" with 2/2 ticked). Two tasks that ship together are both doing:
// hug task take B --with A, A the taker's active task.
func takeWithBoard(t *testing.T) (*App, string, string) {
	t.Helper()
	a := app(t)
	execute(t, a, "init")
	first := execute(t, a, "new", "First", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C", "--accept", "D")["task"].(Reply)["id"].(string)
	second := execute(t, a, "new", "Second", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C", "--accept", "D")["task"].(Reply)["id"].(string)
	execute(t, a, "take", first, "--model", "claude-opus-5-5", "--as", "sabertooth")
	return a, first, second
}

// Check 1: take B --with A succeeds while A is the taker's active task;
// without it the refusal lists the take-with command, saying when.
func TestTakeWithActiveTask(t *testing.T) {
	t.Parallel()
	a, first, second := takeWithBoard(t)
	code, out := runText(a, "take", second, "--model", "claude-opus-5-5", "--as", "sabertooth")
	withFix := "fix: hug task take " + second + " --with " + first + " --model claude-opus-5-5 --as sabertooth"
	if code != 3 || !strings.Contains(out, withFix+"\n") {
		t.Fatalf("refusal lacks %q (exit %d):\n%s", withFix, code, out)
	}
	if !strings.Contains(out, "ship together") {
		t.Fatalf("refusal does not say when to take with (they ship together):\n%s", out)
	}
	code, out = runText(a, "take", second, "--with", first, "--model", "claude-opus-5-5", "--as", "sabertooth")
	if code != 0 {
		t.Fatalf("take --with: exit %d\n%s", code, out)
	}
	f := taskFile(t, a, second)
	if !strings.Contains(f, `state: "doing"`) || !strings.Contains(f, `doing_with: "`+first+`"`) {
		t.Fatalf("take --with did not hold both:\n%s", f)
	}
	if !strings.Contains(f, "todo -> doing/active (take with "+first+")") {
		t.Fatalf("History does not say with whom:\n%s", f)
	}
	if f := taskFile(t, a, first); !strings.Contains(f, `state: "doing"`) || strings.Contains(f, "doing_with") {
		t.Fatalf("the active task changed:\n%s", f)
	}
	// show says it, and --json carries it.
	if _, out = runText(a, "show", second); !strings.Contains(out, "with "+first+" (First)") {
		t.Fatalf("show does not say with whom:\n%s", out)
	}
	if w := execute(t, a, "show", second)["task"].(Reply)["doing_with"]; w != first {
		t.Fatalf("show --json doing_with = %v", w)
	}
	// Each closes on its own: B done leaves A doing; a third take is
	// still refused (A is active), and taking with B is refused: B is not
	// the active one.
	third := execute(t, a, "new", "Third", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	if code, out = runText(a, "take", third, "--model", "claude-opus-5-5", "--as", "sabertooth"); code != 3 {
		t.Fatalf("third take not refused: exit %d\n%s", code, out)
	}
	execute(t, a, "check", second, "1")
	execute(t, a, "check", second, "2")
	execute(t, a, "done", second)
	if f := taskFile(t, a, first); !strings.Contains(f, `state: "doing"`) {
		t.Fatalf("done on B moved A:\n%s", f)
	}
}

// Check 4: --with a task that is not the taker's active one is refused
// with a fix line naming the active one.
func TestTakeWithNotActiveRefused(t *testing.T) {
	t.Parallel()
	a, first, second := takeWithBoard(t)
	third := execute(t, a, "new", "Third", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	// third is todo: not the taker's active task.
	code, out := runText(a, "take", second, "--with", third, "--model", "claude-opus-5-5", "--as", "sabertooth")
	if code == 0 || !strings.Contains(out, third+" is not what sabertooth is on") || !strings.Contains(out, "fix: hug task take "+second+" --with "+first+" --model claude-opus-5-5 --as sabertooth\n") {
		t.Fatalf("--with a todo task: exit %d\n%s", code, out)
	}
	// Another taker's active task: refused too.
	execute(t, a, "take", third, "--model", "claude-opus-5-5", "--as", "kestrel")
	if code, out = runText(a, "take", second, "--with", third, "--model", "claude-opus-5-5", "--as", "sabertooth"); code == 0 || !strings.Contains(out, "fix: hug task take "+second+" --with "+first) {
		t.Fatalf("--with kestrel's task: exit %d\n%s", code, out)
	}
	// With nothing active, --with is refused with plain take as the fix.
	execute(t, a, "wait", first, "CI run 1", "--as", "sabertooth")
	if code, out = runText(a, "take", second, "--with", first, "--model", "claude-opus-5-5", "--as", "sabertooth"); code == 0 || !strings.Contains(out, "fix: hug task take "+second+" --model claude-opus-5-5 --as sabertooth\n") {
		t.Fatalf("--with while nothing active: exit %d\n%s", code, out)
	}
	if f := taskFile(t, a, second); !strings.Contains(f, `state: "todo"`) && !strings.Contains(f, "state: todo") {
		t.Fatalf("a refused take --with wrote:\n%s", f)
	}
}

// Check 3: ticking a check on a todo task the agent has not taken ticks
// it and replies with a fix line to take it with the active one.
func TestCheckOnUntakenTaskSaysTakeWith(t *testing.T) {
	t.Parallel()
	a, first, second := takeWithBoard(t)
	code, out := runText(a, "check", second, "1", "--as", "sabertooth", "--model", "claude-opus-5-5")
	want := "fix: hug task take " + second + " --with " + first + " --model claude-opus-5-5 --as sabertooth\n"
	if code != 0 || !strings.Contains(out, want) || !strings.Contains(out, second+" is not taken") {
		t.Fatalf("check on an untaken task: exit %d, want %q\n%s", code, want, out)
	}
	if f := taskFile(t, a, second); !strings.Contains(f, "- [x] C") {
		t.Fatalf("not ticked:\n%s", f)
	}
	var r map[string]any
	_, js, serr := runJSON(a, "check", second, "2", "--as", "sabertooth", "--model", "claude-opus-5-5", "--json")
	if e := json.Unmarshal([]byte(js), &r); e != nil || r["fix"] != strings.TrimSuffix(strings.TrimPrefix(want, "fix: "), "\n") {
		t.Fatalf("json fix: %v\n%s\nstderr:\n%s", e, js, serr)
	}
	// Nothing active: the fix is a plain take.
	execute(t, a, "wait", first, "CI run 1", "--as", "sabertooth")
	third := execute(t, a, "new", "Third", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	if _, out = runText(a, "check", third, "1", "--as", "sabertooth", "--model", "claude-opus-5-5"); !strings.Contains(out, "fix: hug task take "+third+" --model claude-opus-5-5 --as sabertooth\n") {
		t.Fatalf("check with nothing active:\n%s", out)
	}
	// The owner ticking their own doing task: no such line.
	if _, out = runText(a, "check", first, "1", "--as", "sabertooth"); strings.Contains(out, "is not taken") {
		t.Fatalf("own task:\n%s", out)
	}
}
