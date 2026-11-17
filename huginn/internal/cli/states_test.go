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
	code, out = runText(a, "states", "--json")
	if code != 0 {
		t.Fatalf("--json exit %d: %s", code, out)
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
