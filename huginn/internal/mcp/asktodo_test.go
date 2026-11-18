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

// yphbra check 3: task_update ask on a todo task blocked by another, by
// an agent that holds neither, records the question and leaves the task
// todo with its dependency; task_update answer answers it for the asker.
func TestAskTodoTaskOverMCP(t *testing.T) {
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
	call := func(name string, input map[string]any) (map[string]any, string, bool) {
		t.Helper()
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: input})
		if e != nil {
			t.Fatal(e)
		}
		text := out.Content[0].(*sdk.TextContent).Text
		var m map[string]any
		json.Unmarshal([]byte(text), &m)
		return m, text, out.IsError
	}
	var ids []string
	for _, title := range []string{"Marks", "One request for the marked"} {
		m, text, bad := call("new", map[string]any{"title": title, "goal": "G", "purpose": "P", "principle": "Q", "accept": []any{"one"}})
		if bad {
			t.Fatal(text)
		}
		ids = append(ids, m["task"].(map[string]any)["id"].(string))
	}
	if _, text, bad := call("update", map[string]any{"id": ids[1], "action": "dep", "add_deps": []any{ids[0]}}); bad {
		t.Fatal(text)
	}
	if _, text, bad := call("take", map[string]any{"id": ids[0], "agent": "macaque", "model": "m", "cwd": dir}); bad {
		t.Fatal(text)
	}
	if _, text, bad := call("update", map[string]any{"id": ids[1], "action": "ask", "to": "jhf", "check": 1, "text": "A or B?", "agent": "piglet"}); bad {
		t.Fatalf("ask on a blocked todo task: %s", text)
	}
	m, text, _ := call("show", map[string]any{"id": ids[1]})
	tk := m["task"].(map[string]any)
	qs, _ := tk["questions"].([]any)
	if tk["state"] != "todo" || tk["stage"] != "depends" || len(qs) != 1 || qs[0].(map[string]any)["by"] != "piglet" {
		t.Fatalf("ask over MCP: %s", text)
	}
	if !strings.Contains(text, ids[0]) {
		t.Fatalf("the dependency went: %s", text)
	}
	if _, text, bad := call("update", map[string]any{"id": ids[1], "action": "answer", "text": "A", "agent": "jhf"}); bad {
		t.Fatalf("answer: %s", text)
	}
	m, text, _ = call("show", map[string]any{"id": ids[1]})
	ans, _ := m["task"].(map[string]any)["answer"].(map[string]any)
	if ans["for"] != "piglet" || ans["by"] != "jhf" {
		t.Fatalf("answer not for the asker: %s", text)
	}
}
