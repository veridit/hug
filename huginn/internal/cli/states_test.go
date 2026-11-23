package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// 6ephqk check 1: hug task states prints the tree from task.States, each
// node with its meaning and setter and its fields; --json carries the
// table itself.
func TestStatesCommandPrintsTree(t *testing.T) {
	t.Parallel()
	a := app(t)
	code, out := runText(a, "states")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if !strings.HasPrefix(out, task.StatesText()) || !strings.HasSuffix(out, "next: hug task next\n") {
		t.Fatalf("states output is not the table's text:\n%s", out)
	}
	for _, want := range []string{"todo                 nobody is working on it", "  waiting            the owner waits on something outside the board", "      dropped_reason   why never (by agent, required)", "doing/*        -> done"} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %q:\n%s", want, out)
		}
	}
	code, out, serr := runJSON(a, "states", "--json")
	if code != 0 {
		t.Fatalf("--json exit %d: %s\nstderr:\n%s", code, out, serr)
	}
	var r struct {
		States      []task.Node       `json:"states"`
		Transitions []task.Transition `json:"transitions"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("--json: %v\n%s", err, out)
	}
	if len(r.States) != 4 || len(r.States[0].Stages) != 6 || len(r.Transitions) != len(task.Transitions) {
		t.Fatalf("--json tree: %d states, %d todo stages, %d transitions", len(r.States), len(r.States[0].Stages), len(r.Transitions))
	}
	if code, out = runText(a, "help", "states"); code != 0 || !strings.Contains(out, "state and stage tree") {
		t.Fatalf("help states: %d %s", code, out)
	}
}

// 6ephqk check 3, CLI: wait, resume, design and drop take their text;
// each refusal ends with a fix: line that runs without an editor.
func TestMoveCommands(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Moves", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	for _, c := range []struct {
		args []string
		code int
		fix  string
	}{
		{[]string{"done", id}, 2, "hug task take " + id + " --model MODEL"},
		{[]string{"drop", id}, 2, "hug task drop " + id + ` "duplicate of OTHER"`},
		{[]string{"wait", id}, 2, "hug task wait " + id + ` "CI run 123"`},
		{[]string{"wait", id, "CI"}, 2, "hug task take " + id + " --model MODEL"},
		{[]string{"design", id}, 2, "hug task design " + id + ` "the question" --for NAME`},
		{[]string{"design", id, "type on rows?", "--as", "crab"}, 2, "hug task design " + id + ` "type on rows?" --for NAME`},
		{[]string{"design", id, "type on rows?", "--as", "crab", "--for", "jhf"}, 0, ""},
		{[]string{"design", id, "--clear"}, 0, ""},
		{[]string{"take", id}, 0, ""},
		{[]string{"resume", id}, 2, "hug task show " + id},
		{[]string{"wait", id, "CI run 7", "--as", "nautilus"}, 0, ""},
		{[]string{"wait", id, "--on", "jhf", "approve the look", "--as", "nautilus"}, 2, "hug task ask " + id + ` --to jhf "approve the look"`},
		{[]string{"resume", id}, 0, ""},
		{[]string{"ask", id, "--to", "jhf", "approve the look", "--as", "nautilus"}, 0, ""},
		{[]string{"resume", id}, 0, ""},
		{[]string{"drop", id, "superseded"}, 0, ""},
		{[]string{"set", id, "--state", "done"}, 2, "hug task show " + id},
		{[]string{"set", id, "--state", "todo"}, 0, ""},
	} {
		code, out := runText(a, c.args...)
		if code != c.code || (c.fix != "" && !strings.Contains(out, "fix: "+c.fix+"\n")) || strings.Contains(out, "fix: hug task edit") {
			t.Fatalf("%v: exit %d, want %d with fix %q:\n%s", c.args, code, c.code, c.fix, out)
		}
	}
	file := taskFile(t, a, id)
	for _, want := range []string{"(design); ", "design_asked_by: crab; design_for: jhf", "(take)", "(wait)", "(wait)", "(ask)", "asked_by: nautilus; asked_for: jhf; asked_since: ", "asked_what: approve the look", "(drop)", "(reopen); was dropped_at: ", "dropped_reason: superseded"} {
		if !strings.Contains(file, want) && !strings.Contains(file, strings.TrimSuffix(want, "; ")) {
			t.Errorf("History lacks %q:\n%s", want, file)
		}
	}
	if code, out := runText(a, "lint"); code != 0 || !strings.Contains(out, "no findings") {
		t.Fatalf("lint after the moves: %d\n%s", code, out)
	}
}

// 3jtfw9 check 2 (CLI): hug task answer answers the stage's question; the
// asker sees it first in show and next --mine until it writes on the task.
func TestAnswerCommand(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	if code, out := runText(a, "answer", id, "drop it"); code != 2 || !strings.Contains(out, "asks nobody") || !strings.Contains(out, "fix: hug task show "+id+"\n") {
		t.Fatalf("nothing asked: %d\n%s", code, out)
	}
	execute(t, a, "design", id, "should type leave the rows?", "--as", "nautilus", "--for", "jhf")
	if code, out := runText(a, "answer", id); code != 2 || !strings.Contains(out, "fix: hug task answer "+id+` "the answer"`) {
		t.Fatalf("no text: %d\n%s", code, out)
	}
	execute(t, a, "answer", id, "drop it", "--as", "jhf")
	file := taskFile(t, a, id)
	if !strings.Contains(file, "answered design, by jhf, for nautilus (todo/design -> todo/ready):\n  > should type leave the rows?\n  drop it\n") || strings.Contains(file, "design_question") {
		t.Fatalf("file:\n%s", file)
	}
	// The asker: show leads with it, next --mine offers it first.
	_, out := runText(a, "show", id)
	if !strings.Contains(out, "answer for nautilus: jhf answered on "+id+": drop it (design: should type leave the rows?)\n") {
		t.Fatalf("show:\n%s", out)
	}
	if r := execute(t, a, "next", "--mine", "--as", "nautilus"); r["task"].(Reply)["id"] != id || r["task"].(Reply)["answer"].(map[string]any)["text"] != "drop it" {
		t.Fatalf("next --mine: %v", r)
	}
	// The asker writes: read. Show no longer leads with it.
	a.Env["HUG_USER"] = "crab" // someone else's write: still unseen
	execute(t, a, "note", id, "crab looked")
	if file = taskFile(t, a, id); strings.Contains(file, "read the answer") {
		t.Fatalf("crab marked it read:\n%s", file)
	}
	a.Env["HUG_USER"] = "nautilus"
	execute(t, a, "note", id, "dropping type")
	if file = taskFile(t, a, id); !strings.Contains(file, "nautilus read the answer") {
		t.Fatalf("not read:\n%s", file)
	}
	if _, out = runText(a, "show", id); strings.Contains(out, "answer for") {
		t.Fatalf("still shown:\n%s", out)
	}
	if code, out := runText(a, "lint"); code != 0 || !strings.Contains(out, "no findings") {
		t.Fatalf("lint: %d\n%s", code, out)
	}
}
