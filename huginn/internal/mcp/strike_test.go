package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/veridit/hug/huginn/internal/cli"
)

// sfa8w4 check 3 (MCP): task_update action strike strikes a check out with
// a reason; task_show reports the three states.
func TestStrikeOverMCP(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.name", "T"}, {"config", "user.email", "t@example.invalid"}, {"commit", "-q", "--allow-empty", "-m", "init"}} {
		if out, e := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, out, e)
		}
	}
	t.Setenv("HUG_USER", "")
	t.Setenv("HUG_MODEL", "test-model")
	t.Setenv("JCODE_SESSION_ID", "")
	ctx := context.Background()
	a := &cli.App{Cwd: dir, Out: os.Stderr, Err: os.Stderr}
	if _, e := a.Execute(ctx, []string{"init"}); e != nil {
		t.Fatal(e)
	}
	ct, st := sdk.NewInMemoryTransports()
	ss, e := New(a).Connect(ctx, st, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer ss.Close()
	cs, e := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer cs.Close()
	call := func(name string, input map[string]any) (string, bool) {
		t.Helper()
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: input})
		if e != nil {
			t.Fatal(e)
		}
		return out.Content[0].(*sdk.TextContent).Text, out.IsError
	}
	text, bad := call("new", map[string]any{"title": "Three states", "goal": "G", "accept": []any{"ticked", "struck", "open"}})
	if bad {
		t.Fatal(text)
	}
	var created map[string]any
	json.Unmarshal([]byte(text), &created)
	id := created["task"].(map[string]any)["id"].(string)
	if text, bad := call("update", map[string]any{"id": id, "action": "check", "item": 1}); bad {
		t.Fatal(text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "strike", "item": 2}); !bad || !strings.Contains(text, "strike needs reason") {
		t.Fatalf("no reason: %v %s", bad, text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "strike", "item": 2, "reason": "covered by item 1", "agent": "kestrel"}); bad {
		t.Fatal(text)
	}
	text, _ = call("show", map[string]any{"id": id})
	var shown map[string]any
	json.Unmarshal([]byte(text), &shown)
	var states []string
	for _, x := range shown["task"].(map[string]any)["acceptance"].([]any) {
		m := x.(map[string]any)
		states = append(states, m["state"].(string))
		if m["state"] == "struck" && (m["reason"] != "covered by item 1" || m["struck_by"] != "kestrel" || m["text"] != "struck") {
			t.Fatalf("struck item: %v", m)
		}
	}
	if strings.Join(states, ",") != "ticked,struck,open" {
		t.Fatalf("states %v: %s", states, text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "strike", "item": 2, "undo": true}); bad {
		t.Fatal(text)
	}
	text, _ = call("show", map[string]any{"id": id})
	if strings.Contains(text, `"state":"struck"`) || !strings.Contains(text, `"state":"open","text":"struck"`) {
		t.Fatalf("undo left it struck: %s", text)
	}
}
