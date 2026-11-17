package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 6w0gve: hug task review ID --by NAME [--show FILE] moves a doing task to
// doing/review (the owner keeps it) with review_by, review_since and
// review_show; --show of a file the task does not carry is refused with a
// fix; answer clears all three and the owner is active again.
func TestReviewMove(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	board := filepath.Join(a.Cwd, TaskDir)

	// Only a doing task goes to review; without --by it is refused.
	if _, e := a.Execute(context.Background(), []string{"review", id, "--by", "jhf"}); e == nil || !strings.Contains(e.Error(), "hug task take "+id+" --model MODEL") {
		t.Fatalf("review of a todo task: %v", e)
	}
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	code, out := runText(a, "review", id)
	if code != 2 || !strings.Contains(out, "fix: hug task review "+id+" --by NAME") {
		t.Fatalf("review without --by: exit %d\n%s", code, out)
	}

	dir := t.TempDir()
	ans := filepath.Join(dir, "rows.ans")
	if e := os.WriteFile(ans, []byte("\x1b[31mIN PROGRESS\x1b[0m\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	execute(t, a, "attach", id, ans)
	if _, e := a.Execute(context.Background(), []string{"review", id, "--by", "jhf", "--show", "nope.ans"}); e == nil || !strings.Contains(e.Error(), "hug task attach "+id+" FILE --name nope.ans") {
		t.Fatalf("--show of a missing file: %v", e)
	}
	head := git(t, board, "rev-parse", "HEAD")
	r := execute(t, a, "review", id, "--by", "jhf", "--show", "rows.ans", "--as", "nautilus")
	if tk := r["task"].(Reply); tk["state"] != "doing" || tk["stage"] != "review" {
		t.Fatalf("review: %v/%v", tk["state"], tk["stage"])
	}
	f := taskFile(t, a, id)
	for _, want := range []string{`review_by: "jhf"`, "review_since: ", `review_show: "rows.ans"`, "(review)"} {
		if !strings.Contains(f, want) {
			t.Fatalf("review lacks %q:\n%s", want, f)
		}
	}
	if n := git(t, board, "rev-list", "--count", head+"..HEAD"); n != "1" {
		t.Fatalf("review made %s commits, want 1", n)
	}
	if _, out = runText(a, "show", id); !strings.Contains(out, "stage: in review by jhf") {
		t.Fatalf("show:\n%s", out)
	}
	// A new review without --show names no picture.
	execute(t, a, "review", id, "--by", "crab")
	if f = taskFile(t, a, id); strings.Contains(f, "\nreview_show:") || !strings.Contains(f, `review_by: "crab"`) {
		t.Fatalf("plain review kept review_show:\n%s", f)
	}
	execute(t, a, "review", id, "--by", "jhf", "--show", "rows.ans")
	execute(t, a, "answer", id, "looks right", "--as", "jhf")
	f = taskFile(t, a, id)
	for _, gone := range []string{"\nreview_by:", "\nreview_since:", "\nreview_show:"} {
		if strings.Contains(f, gone) {
			t.Fatalf("answer kept %s:\n%s", gone, f)
		}
	}
	if r = execute(t, a, "show", id); r["task"].(Reply)["stage"] != "active" {
		t.Fatalf("after the answer: %v", r["task"].(Reply)["stage"])
	}
}
