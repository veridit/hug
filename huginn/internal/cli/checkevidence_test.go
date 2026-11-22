package cli

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

func atoiTest(s string) int  { n, _ := strconv.Atoi(s); return n }
func itoaCheck(n int) string { return strconv.Itoa(n) }

// pw0t19 check 1 (at3qep, 49n9kw, x1ra0t: the instructions say "hug task
// check ID N with evidence", check took no evidence and refused it with
// 'expected 2 positional arguments, got 3' / 'fix: hug task help'): check
// ID N "evidence" ticks N and writes the evidence as one note, in one
// commit. Plain check ID N still works and writes no note.
func TestCheckTakesEvidence(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Evidence", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one", "--accept", "two")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--model", "test-model", "--as", "bonehound")
	commits := git(t, a.Cwd, "rev-list", "--count", TaskBranch)
	code, out := runText(a, "check", id, "1", "TestX passes: ok huginn/internal/cli 1.2s")
	if code != 0 {
		t.Fatalf("check ID 1 \"evidence\": exit %d, want 0\n%s", code, out)
	}
	f := taskFile(t, a, id)
	if !strings.Contains(f, "- [x] one") || strings.Count(f, "check 1: TestX passes: ok huginn/internal/cli 1.2s") != 1 {
		t.Fatalf("want check 1 ticked and the evidence as one note naming it:\n%s", f)
	}
	if log := git(t, a.Cwd, "log", "-1", "--format=%s", TaskBranch); log != "hug task check "+id {
		t.Fatalf("one commit for tick and note, got subject %q", log)
	}
	if after := git(t, a.Cwd, "rev-list", "--count", TaskBranch); after != itoaCheck(atoiTest(commits)+1) {
		t.Fatalf("tick and note: %s commits before, %s after, want one more", commits, after)
	}
	// Plain check: no note.
	execute(t, a, "check", id, "2")
	if f2 := taskFile(t, a, id); !strings.Contains(f2, "- [x] two") || strings.Contains(f2, "check 2:") {
		t.Fatalf("plain check 2 wrote a note or did not tick:\n%s", f2)
	}
	// Ticking an already ticked check with evidence still records it.
	execute(t, a, "check", id, "2", "seen again in CI run 9")
	if f3 := taskFile(t, a, id); !strings.Contains(f3, "check 2: seen again in CI run 9") {
		t.Fatalf("evidence on a ticked check is lost:\n%s", f3)
	}
	// An index out of range writes nothing, evidence or not.
	before := taskFile(t, a, id)
	if code, out := runText(a, "check", id, "9", "evidence"); code == 0 || taskFile(t, a, id) != before {
		t.Fatalf("check 9 of 2 with evidence: exit %d, file changed=%v\n%s", code, taskFile(t, a, id) != before, out)
	}
}

// pw0t19 check 3 (p29rbz: "a note signed by habit is lost"): note and check
// accept --as NAME like take and wait; the evidence or note is kept.
func TestNoteAndCheckAcceptAs(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Signed", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--model", "test-model", "--as", "bonehound")
	for _, args := range [][]string{
		{"note", id, "signed note", "--as", "bonehound"},
		{"check", id, "1", "signed evidence", "--as", "bonehound"},
	} {
		if code, out := runText(a, args...); code != 0 {
			t.Fatalf("%v: exit %d, want 0\n%s", args, code, out)
		}
	}
	f := taskFile(t, a, id)
	if !strings.Contains(f, "signed note") || !strings.Contains(f, "check 1: signed evidence") || !strings.Contains(f, "- [x] one") {
		t.Fatalf("note and check with --as:\n%s", f)
	}
}

// pw0t19 check 4: a wrong positional count on any task command names that
// command's usage line and ends with fix: hug task COMMAND --help, never
// the bare 'fix: hug task help'.
func TestWrongArgCountNamesTheCommand(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Arity", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	// Not probed: mcp runs only on its own transport, and the commands
	// whose extra words are input by design (search WORDS, rebalance IDS,
	// attach FILES) have no wrong count. Every other command checks its
	// count before it reads the outside world, runs an editor or a TUI.
	skip := []string{"mcp", "search", "rebalance", "attach"}
	for _, c := range Commands {
		if slices.Contains(skip, c) {
			continue
		}
		code, out := runText(a, c, id, "1", "x", "y", "z", "w", "v", "u")
		if code != 2 {
			t.Errorf("%s with 8 arguments: exit %d, want 2\n%s", c, code, out)
			continue
		}
		usageLine, _, _ := strings.Cut(commandHelp(c), "\n")
		if !strings.HasSuffix(out, "fix: hug task "+c+" --help\n") || !strings.Contains(out, "usage: "+usageLine) {
			t.Errorf("%s with 8 arguments: want its usage line and fix: hug task %s --help, got:\n%s", c, c, out)
		}
	}
}

// y8we58 check 1 (tb0dra r1, crab: 'hug task check tb0dra 5 x' ticked a
// check with evidence 'x'): evidence under 8 characters is refused with a
// fix: line and writes nothing; 8 characters pass.
func TestCheckRefusesShortEvidence(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Short", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "one")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--model", "test-model", "--as", "bonehound")
	before := taskFile(t, a, id)
	for _, ev := range []string{"x", "ok", "  done  ", "1234567"} {
		code, out := runText(a, "check", id, "1", ev)
		want := "fix: hug task check " + id + " 1 \"what shows it holds\"\n"
		if code == 0 || !strings.Contains(out, "at least 8 characters") || !strings.HasSuffix(out, want) || taskFile(t, a, id) != before {
			t.Fatalf("check with evidence %q: exit %d, file changed=%v, want refusal ending %q:\n%s", ev, code, taskFile(t, a, id) != before, want, out)
		}
	}
	if code, out := runText(a, "check", id, "1", "12345678"); code != 0 || !strings.Contains(taskFile(t, a, id), "check 1: 12345678") {
		t.Fatalf("8 characters: exit %d\n%s", code, out)
	}
}
