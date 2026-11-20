package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// 0pr03h check 1 at the CLI: new and set refuse a Goal of more than one
// sentence or over 140 characters, exit 2, with a fix: line that runs.
func TestNewAndSetRefuseLongGoal(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	code, out := runText(a, "new", "T", "--goal", "It works. Then it ships.", "--purpose", "P", "--principle", "Q", "--accept", "C")
	want := "incomplete-task: Goal is 2 sentences: the Goal is one sentence of at most 140 characters; detail goes in Details\n" +
		`fix: hug task new "T" --goal "one sentence" --purpose "P" --principle "Q" --details "..." --accept "C"` + "\n"
	if code != 2 || out != want {
		t.Fatalf("%d:\n%s\nwant:\n%s", code, out, want)
	}
	id := execute(t, a, "new", "T", "--goal", "Ship it, e.g. as v0.7.85 in main.go.", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	code, out = runText(a, "set", id, "--goal", strings.Repeat("x", 141))
	if code != 2 || !strings.Contains(out, "Goal is 141 characters") || !strings.HasSuffix(out, "fix: hug task set "+id+` --goal "one sentence" --details "..."`+"\n") {
		t.Fatalf("%d:\n%s", code, out)
	}
	// The fix, filled in, runs.
	execute(t, a, "set", id, "--goal", "It ships.", "--details", "It works first. Then it ships.")
}

// 0pr03h check 2 at the CLI: lint lists goal-too-long on a task whose long
// Goal is already on disk, with the same fix, and changes nothing.
func TestLintGoalTooLongCLI(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "T", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	long := "It works. Then it ships. " + strings.Repeat("More words here ", 10)
	useEditor(t, a, "## Goal\nG\n", "## Goal\n"+long+"\n")
	execute(t, a, "edit", id)
	fix := "hug task set " + id + ` --goal "one sentence" --details "..."`
	_, detail := runText(a, "lint", id)
	if !strings.Contains(detail, "advice [goal-too-long] Goal is 184 characters in 3 sentences: the Goal is one sentence of at most 140 characters; detail goes in Details") || !strings.HasSuffix(detail, "next: "+fix+"\n") {
		t.Fatalf("lint:\n%s", detail)
	}
	_, show := runText(a, "show", id)
	if !strings.Contains(show, strings.TrimSpace(long)) || !strings.Contains(show, "fix: "+fix+"\n") {
		t.Fatalf("show:\n%s", show)
	}
	if _, codes := runText(a, "lint", "--code"); !strings.Contains(codes, "goal-too-long") {
		t.Fatalf("codes:\n%s", codes)
	}
}

// 0pr03h check 3: help new, help set, the AGENTS.md section hug setup
// writes, and this repository's AGENTS.md state the one-sentence Goal in
// the same words.
func TestGoalRuleWording(t *testing.T) {
	const rule = "The Goal is one sentence of at most 140 characters; detail goes in Details."
	if task.GoalRule != rule {
		t.Fatalf("GoalRule %q", task.GoalRule)
	}
	flat := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	repo, err := os.ReadFile(filepath.Join("..", "..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ where, text string }{
		{"help new", commandHelp("new")},
		{"AGENTS.md section (hug setup)", agentSection},
		{"repository AGENTS.md", string(repo)},
	} {
		if !strings.Contains(flat(c.text), rule) {
			t.Errorf("%s does not say %q:\n%s", c.where, rule, c.text)
		}
	}
	if !strings.Contains(flat(commandHelp("set")), "the Goal is one sentence of at most 140 characters; detail goes in Details") {
		t.Errorf("help set:\n%s", commandHelp("set"))
	}
}
