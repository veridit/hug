package cli

import (
	"strings"
	"testing"
)

// zdgw60 check 1 (reflection asr4t2): set --accept on a task with a lint
// finding says what it added before the lint block, so a reply that ends
// in a fix: never reads as a failed write.
func TestSetSaysWhatItWroteBeforeLint(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "short", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "set", id, "--title", strings.Repeat("a", 90))
	code, out := runText(a, "set", id, "--accept", "VALUE")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	added, lint := strings.Index(out, "added check 2: VALUE\n"), strings.Index(out, "lint: 1 finding")
	if added < 0 || lint < 0 || added > lint {
		t.Fatalf("want 'added check 2: VALUE' before the lint block:\n%s", out)
	}
	if r := execute(t, a, "set", id, "--accept", "W"); !strings.Contains(strings.Join(wroteOf(r), "\n"), "added check 3: W") {
		t.Fatalf("JSON wrote: %v", r["wrote"])
	}
}

// zdgw60 check 2: each field set changes gets its own line; a set that
// changes nothing says so.
func TestSetSaysEachFieldItWrote(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	epic := execute(t, a, "new", "Epic", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	id := execute(t, a, "new", "Old title", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "piglet", "--model", "m")
	r := execute(t, a, "set", id, "--title", "New title", "--goal", "new goal", "--purpose", "new purpose", "--principle", "new principle",
		"--details", "new details", "--tag", "parser", "--parent", epic, "--state", "todo", "--priority", "high", "--type", "bug")
	got := strings.Join(wroteOf(r), "\n")
	for _, want := range []string{
		"title: Old title -> New title",
		"goal: new goal",
		"purpose: new purpose",
		"principle: new principle",
		"details: new details",
		"tags: + parser",
		"parent: " + epic,
		"state: doing -> todo",
		"assignee: piglet -> (none)", // back to todo gives the task up: said too
		"priority: ",
		"type: ",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("wrote lacks %q:\n%s", want, got)
		}
	}
	if n := len(wroteOf(r)); n != 11 {
		t.Fatalf("%d lines for 11 changed fields:\n%s", n, got)
	}
	// The same set again changes nothing and says so.
	r = execute(t, a, "set", id, "--title", "New title")
	if w := wroteOf(r); len(w) != 1 || !strings.HasPrefix(w[0], "unchanged:") {
		t.Fatalf("repeat set: %v", w)
	}
}

func wroteOf(r Reply) []string {
	w, _ := r["wrote"].([]string)
	return w
}
