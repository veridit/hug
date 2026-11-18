package mcp

import (
	"context"
	"slices"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/veridit/hug/huginn/internal/tui"
)

// 05r3pg check 5 (parity): every task_update action runs a hug task command
// the TUI's : palette offers on the task, through the same CLI Execute. A new
// action without a palette row fails here.
func TestPaletteCoversEveryUpdateAction(t *testing.T) {
	ctx := context.Background()
	r := &recorder{}
	ct, st := sdk.NewInMemoryTransports()
	ss, e := New(r).Connect(ctx, st, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer ss.Close()
	cs, e := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer cs.Close()
	// The least input each action needs to build its command.
	inputs := map[string]map[string]any{
		"set": {"title": "T"}, "check": {"item": 1}, "strike": {"item": 1, "reason": "r"}, "promote": {"item": 1},
		"note": {"text": "n"}, "move": {"top": true}, "move_group": {"group": "g"}, "mv": {"group": "g"},
		"dep": {"add_deps": []string{"abc124"}}, "drop": {"text": "why"}, "wait": {"text": "CI"},
		"ask": {"text": "q", "to": "ann"}, "resume": {}, "review": {"by": "ann"}, "design": {"text": "q", "for": "ann"},
		"answer": {"text": "a"}, "request": {"text": "r"}, "resolve": {"item": 1, "text": "t"}, "rebalance": {},
		"attach": {"name": "f.md", "content": "c"}, "detach": {"name": "f.md"},
	}
	verbs := tui.PaletteVerbs()
	for action, in := range inputs {
		in["id"], in["action"] = "abc123", action
		r.args = nil
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: "update", Arguments: in})
		if e != nil || out.IsError || len(r.args) == 0 {
			t.Fatalf("task_update %s did not run: %v %+v", action, e, out)
		}
		if !slices.Contains(verbs, r.args[0]) {
			t.Errorf("task_update %s runs hug task %s; the : palette has no row for it (palettetree.go contextItemList)", action, r.args[0])
		}
	}
}
