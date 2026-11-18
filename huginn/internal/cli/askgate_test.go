package cli

import (
	"strconv"
	"strings"
	"testing"
)

// wdkqzx check 5 (Jørgen 23:51, crab 00:05): a person is never handed a
// broken task. ask --to, review --by and design --for refuse while the
// task's own file has a broken finding, name it, and give a fix hug takes;
// after the fix the same command goes through. design --for is not held
// back by a missing Goal or acceptance: the answer is often what fills
// them. A wait on a thing (no person) is not held back.
func TestAskingAPersonRefusesABrokenTask(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Broken", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one", "--accept", "two")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--model", "test-model", "--as", "kestrel")
	if code, out := runText(a, "wait", id, "CI run 1", "--as", "kestrel"); code != 0 {
		t.Fatalf("a wait on a thing is held back: exit %d\n%s", code, out)
	}
	execute(t, a, "resume", id)
	// A broken finding the agent can fix: the Goal emptied by hand.
	useEditor(t, a, "## Goal\nG\n", "## Goal\n")
	execute(t, a, "edit", id)
	before := taskFile(t, a, id)
	for _, args := range [][]string{
		{"ask", id, "does check 2 hold?", "--to", "jhf", "--check", "2", "--as", "kestrel"},
		{"review", id, "--by", "crab", "--as", "kestrel"},
	} {
		code, out := runText(a, args...)
		person := args[4]
		if args[0] == "review" {
			person = "crab"
		}
		want := "hug task " + args[0] + ": " + id + " is not ready to hand to " + person + ": 1 broken finding: [missing-goal] missing Goal"
		if code != 2 || !strings.Contains(out, want) || !strings.HasSuffix(out, "fix: hug task set "+id+` --goal "..."`+"\n") {
			t.Fatalf("%v: exit %d, want 2 with %q and the dep fix:\n%s", args, code, want, out)
		}
		if after := taskFile(t, a, id); after != before {
			t.Fatalf("%v wrote the task while refusing:\n%s", args, after)
		}
	}
	execute(t, a, "set", id, "--goal", "G")
	if code, out := runText(a, "ask", id, "does check 2 hold?", "--to", "jhf", "--check", "2", "--as", "kestrel"); code != 0 {
		t.Fatalf("after the fix the ask is still refused: exit %d\n%s", code, out)
	}

	// design --for on a draft task: a missing Goal or acceptance is no reason
	// to hold the question back (crab 00:05); a broken finding of another
	// kind is.
	d := execute(t, a, "new", "Draft", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	useEditor(t, a, "## Goal\nG\n", "## Goal\n")
	execute(t, a, "edit", d)
	useEditor(t, a, "- [ ] one\n", "")
	execute(t, a, "edit", d)
	if code, out := runText(a, "design", d, "which layout?", "--for", "jhf"); code != 0 {
		t.Fatalf("design --for on a task without acceptance is refused: exit %d\n%s", code, out)
	}
	execute(t, a, "design", d, "--clear")
	useEditor(t, a, "type: feature", "type: featur")
	execute(t, a, "edit", d)
	if code, out := runText(a, "design", d, "which layout?", "--for", "jhf"); code != 2 || !strings.Contains(out, "is not ready to hand to jhf: 1 broken finding: [unknown-type] unknown type featur") || !strings.HasSuffix(out, "fix: hug task set "+d+" --type feature\n") {
		t.Fatalf("design --for on a task with an unknown type: exit %d\n%s", code, out)
	}
}

// wdkqzx check 6: 6ephqk's shape, 8 of 9 checks settled and shipped, asked
// about check 9: the ask holds the release question, so lint has none and
// the ask goes through.
func TestAskAboutTheOpenCheckOfAShippedTask(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	args := []string{"new", "Shipped", "--goal", "G", "--purpose", "P", "--principle", "Q"}
	for i := 1; i <= 9; i++ {
		args = append(args, "--accept", "check")
	}
	id := execute(t, a, args...)["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--model", "test-model", "--as", "nautilus")
	execute(t, a, "note", id, "shipped in v0.7.45 (f4ee48baaaa)")
	for n := 1; n <= 8; n++ {
		execute(t, a, "check", id, strconv.Itoa(n))
	}
	if _, out := runText(a, "show", id); !strings.Contains(out, "[shipped-not-done]") || !strings.Contains(out, "fix: hug task check "+id+" 9") {
		t.Fatalf("active and shipped with check 9 open:\n%s", out)
	}
	code, out := runText(a, "ask", id, "look at the rows after v0.7.45", "--to", "jhf", "--check", "9", "--as", "nautilus")
	if code != 0 || strings.Contains(out, "shipped-not-done") {
		t.Fatalf("ask about check 9: exit %d\n%s", code, out)
	}
	if _, out := runText(a, "lint", id); strings.Contains(out, "shipped-not-done") {
		t.Fatalf("asked about check 9, lint still asks:\n%s", out)
	}
}
