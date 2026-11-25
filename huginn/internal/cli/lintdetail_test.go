package cli

import (
	"strings"
	"testing"
)

// fkpyg3 checks 9 and 10: hug task lint is the overview (totals per code,
// one line per task); hug task lint ID gives every finding of the task and
// the one combined fix as next; show and take print the same block; ls rows
// carry no finding text.
func TestLintOverviewDetailAndSameBlock(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Two problems", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	useEditor(t, a, "## Principle\nQ\n", "")
	execute(t, a, "edit", id)
	// The title's words are not a finding code: a layout-3 slug is the
	// title's (w40cmk), and ls shows it.
	execute(t, a, "set", id, "--title", strings.Repeat("Wide title ", 9))

	_, over := runText(a, "lint")
	wantOver := "2 findings (0 broken) on 1 task, in 2 codes:\n"
	line := "  " + id + "  missing-section, long-title  -> hug task lint " + id + "\n"
	fix := "hug task set " + id + ` --title "..." --principle "..."`
	if !strings.HasPrefix(over, wantOver) || !strings.Contains(over, line) || !strings.HasSuffix(over, "next: "+fix+"\n") || strings.Contains(over, "fix: ") {
		t.Fatalf("overview:\n%s\nwant %q and %q", over, line, fix)
	}

	_, detail := runText(a, "lint", id)
	if strings.Count(detail, "] ") != 2 || !strings.Contains(detail, "[long-title] title is 99 characters") ||
		!strings.Contains(detail, "[missing-section] missing section Principle") || !strings.HasSuffix(detail, "next: "+fix+"\n") {
		t.Fatalf("detail:\n%s", detail)
	}

	block := "lint: 2 findings\n"
	_, show := runText(a, "show", id)
	if !strings.Contains(show, block) || !strings.Contains(show, "fix: "+fix+"\n") || strings.Count(show, "  line 1: [") != 2 {
		t.Fatalf("show:\n%s", show)
	}
	_, take := runText(a, "take", id, "--model", "test-model")
	if !strings.Contains(take, block) || !strings.Contains(take, "fix: "+fix+"\n") {
		t.Fatalf("take:\n%s", take)
	}
	// The JSON block is the same as lint ID's.
	r := execute(t, a, "show", id)
	if l, ok := r["lint"].(*TaskLint); !ok || l.Fix != fix || len(l.Findings) != 2 {
		t.Fatalf("show json: %+v", r["lint"])
	}
	// Check 10: rows stay as they were, no finding text, no counts.
	_, ls := runText(a, "ls")
	if strings.Contains(ls, "missing") || strings.Contains(ls, "long-title") || strings.Contains(ls, "finding") {
		t.Fatalf("ls:\n%s", ls)
	}
	// The one fix runs and clears the task.
	execute(t, a, "set", id, "--title", "Short", "--principle", "How")
	if _, out := runText(a, "lint"); !strings.HasPrefix(out, "no findings") {
		t.Fatalf("after fix:\n%s", out)
	}
	if r := execute(t, a, "show", id); r["lint"] != nil {
		t.Fatalf("clean show still has a block: %v", r["lint"])
	}
}

// fkpyg3 check 1 at the CLI: new with only a title names every missing part
// and gives one fix: line.
func TestNewNamesEveryMissingFlag(t *testing.T) {
	a := app(t)
	execute(t, a, "init")
	code, out := runText(a, "new", "x")
	want := "incomplete-task: missing Goal; missing section Purpose (why); missing section Principle (how: what we do and do not do); Acceptance requires at least one item\n" +
		`fix: hug task new "x" --goal "..." --purpose "..." --principle "..." --accept "observable check"` + "\n"
	if code != 2 || out != want {
		t.Fatalf("%d:\n%s\nwant:\n%s", code, out, want)
	}
}
