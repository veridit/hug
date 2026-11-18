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

// gpyqq9 check 1 (crab on 4vd1mz, 2026-11-18 11:56Z): a shipped task with
// every check ticked has one question left, is it done, and only its owner
// decides. Asking that person is how it is settled: ask goes through, the
// stage is asked, and lint no longer reports shipped-not-done.
func TestAskSettlesShippedNotDone(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Shipped", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one", "--accept", "two")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--model", "test-model", "--as", "crab")
	execute(t, a, "note", id, "shipped in v0.7.59 (9349d00aaaa)")
	execute(t, a, "check", id, "1")
	execute(t, a, "check", id, "2")
	if _, out := runText(a, "show", id); !strings.Contains(out, "[shipped-not-done]") || !strings.Contains(out, "fix: hug task done "+id) {
		t.Fatalf("shipped with every check ticked, want shipped-not-done:\n%s", out)
	}
	code, out := runText(a, "ask", id, "is it done?", "--to", "jhf", "--as", "crab")
	if code != 0 || strings.Contains(out, "not ready to hand") {
		t.Fatalf("ask on a shipped, all-ticked task: exit %d, want 0\n%s", code, out)
	}
	if tk := execute(t, a, "show", id, "--json")["task"].(Reply); tk["stage"] != "asked" {
		t.Fatalf("after ask: stage %v, want asked", tk["stage"])
	}
	if _, out := runText(a, "lint", id); strings.Contains(out, "shipped-not-done") {
		t.Fatalf("asked whether it is done, lint still reports shipped-not-done:\n%s", out)
	}
}

// gpyqq9 check 2: ask is excused only the question it settles. Another
// broken finding on a shipped, all-ticked task (an unknown type, edited by
// hand) still holds the ask back, names only that finding, gives its fix,
// and writes nothing.
func TestAskOnShippedTaskStillRefusesOtherBrokenFindings(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Shipped", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--model", "test-model", "--as", "crab")
	execute(t, a, "note", id, "shipped in v0.7.59 (9349d00aaaa)")
	execute(t, a, "check", id, "1")
	useEditor(t, a, "type: feature", "type: featur")
	execute(t, a, "edit", id)
	before := taskFile(t, a, id)
	code, out := runText(a, "ask", id, "is it done?", "--to", "jhf", "--as", "crab")
	if code != 2 || !strings.Contains(out, "is not ready to hand to jhf: 1 broken finding: [unknown-type] unknown type featur") || strings.Contains(out, "[shipped-not-done]") || !strings.HasSuffix(out, "fix: hug task set "+id+" --type feature\n") {
		t.Fatalf("ask on a shipped task with an unknown type: exit %d, want 2 naming only unknown-type with its fix:\n%s", code, out)
	}
	if after := taskFile(t, a, id); after != before {
		t.Fatalf("ask wrote the task while refusing:\n%s", after)
	}
	// After the fix the same ask goes through.
	execute(t, a, "set", id, "--type", "feature")
	if code, out := runText(a, "ask", id, "is it done?", "--to", "jhf", "--as", "crab"); code != 0 {
		t.Fatalf("after the fix the ask is still refused: exit %d\n%s", code, out)
	}
}

// gpyqq9, crab's second case (5z4zt4, 2026-11-18): shipped, checks 4 and 5
// open, and both are the person's to confirm. Asking that person about an
// open check settles shipped-not-done as asking "is it done?" does: the
// ask goes through with --check 4 or without --check, the stage is asked,
// and lint has no shipped-not-done.
func TestAskAboutOpenChecksOfAShippedTask(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	args := []string{"new", "Shipped", "--goal", "G", "--purpose", "P", "--principle", "Q"}
	for i := 1; i <= 5; i++ {
		args = append(args, "--accept", "check")
	}
	for _, ask := range [][]string{
		{"does check 4 hold?", "--to", "jhf", "--check", "4"},
		{"do checks 4 and 5 hold?", "--to", "jhf"},
	} {
		id := execute(t, a, args...)["task"].(Reply)["id"].(string)
		execute(t, a, "take", id, "--model", "test-model", "--as", "crab")
		execute(t, a, "note", id, "shipped in v0.7.55 (5a4b3c2aaaa)")
		for n := 1; n <= 3; n++ {
			execute(t, a, "check", id, strconv.Itoa(n))
		}
		if _, out := runText(a, "show", id); !strings.Contains(out, "[shipped-not-done]") || !strings.Contains(out, "checks 4 and 5 are open") {
			t.Fatalf("shipped with checks 4 and 5 open, want shipped-not-done:\n%s", out)
		}
		code, out := runText(a, append([]string{"ask", id}, append(ask, "--as", "crab")...)...)
		if code != 0 || strings.Contains(out, "not ready to hand") {
			t.Fatalf("ask %v on a shipped task with 2 open checks: exit %d, want 0\n%s", ask, code, out)
		}
		if tk := execute(t, a, "show", id, "--json")["task"].(Reply); tk["stage"] != "asked" {
			t.Fatalf("ask %v: stage %v, want asked", ask, tk["stage"])
		}
		if _, out := runText(a, "lint", id); strings.Contains(out, "shipped-not-done") {
			t.Fatalf("ask %v: lint still reports shipped-not-done:\n%s", ask, out)
		}
	}
}
