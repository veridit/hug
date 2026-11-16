package cli

import (
	"strings"
	"testing"
)

// t709he check 2: every commit hug makes on hug/tasks ends with one Model:
// trailer, from HUG_MODEL (or --model where the command has it), the
// identity take records.
func TestBoardCommitsCarryModelTrailer(t *testing.T) {
	a := app(t) // HUG_MODEL=test-model
	execute(t, a, "init")
	id := execute(t, a, "new", "Trailer", "--goal", "G", "--accept", "C")["task"].(Reply)["id"].(string)
	t.Setenv("HUG_MODEL", "gpt-6.1-sol")
	execute(t, a, "note", id, "a note")
	execute(t, a, "take", id, "--model", "claude-opus-5-5")
	board := a.Cwd + "/" + TaskDir
	got := git(t, board, "log", "-3", "--format=%s|%(trailers:key=Model,valueonly,separator=%x2C)")
	want := "hug task take " + id + "|claude-opus-5-5\nhug task note " + id + "|gpt-6.1-sol\nhug task new " + id + "|test-model"
	if got != want {
		t.Fatalf("board commits:\n%s\nwant:\n%s", got, want)
	}
	if n := strings.Count(git(t, board, "log", "-1", "--format=%B"), "Model:"); n != 1 {
		t.Fatalf("%d Model: lines in the last commit, want 1", n)
	}
}
