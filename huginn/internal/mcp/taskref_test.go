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

// xxvm6t: the MCP tools accept task:ID wherever they take a task ref, and
// reply with the bare id.
func TestToolsAcceptTaskColon(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.name", "T"}, {"config", "user.email", "t@example.invalid"}, {"commit", "-q", "--allow-empty", "-m", "init"}} {
		if out, e := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, out, e)
		}
	}
	t.Setenv("HUG_USER", "tester")
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
	call := func(name string, input map[string]any) map[string]any {
		t.Helper()
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: input})
		if e != nil {
			t.Fatal(e)
		}
		text := out.Content[0].(*sdk.TextContent).Text
		if out.IsError {
			t.Fatalf("%s %v: %s", name, input, text)
		}
		if strings.Contains(text, "\"next\":\"hug task show task:") || strings.Contains(text, "hug task take task:") {
			t.Errorf("%s replied with task:ID in a command: %s", name, text)
		}
		var m map[string]any
		if e := json.Unmarshal([]byte(text), &m); e != nil {
			t.Fatal(e)
		}
		return m
	}
	idOf := func(m map[string]any) string { return m["task"].(map[string]any)["id"].(string) }
	epic := idOf(call("new", map[string]any{"title": "Epic", "goal": "G", "purpose": "P", "principle": "Q", "accept": []string{"E"}}))
	beta := idOf(call("new", map[string]any{"title": "Beta", "goal": "G", "purpose": "P", "principle": "Q", "accept": []string{"B"}, "dep": []string{"task:" + epic}, "after": "task:" + epic}))
	if got := idOf(call("show", map[string]any{"id": "task:" + beta})); got != beta {
		t.Fatalf("show task:%s gave %s", beta, got)
	}
	call("update", map[string]any{"id": "task:" + beta, "action": "dep", "remove_deps": []string{"task:" + epic}})
	call("update", map[string]any{"id": "task:" + beta, "action": "set", "parent": "task:" + epic})
	call("update", map[string]any{"id": "task:" + beta, "action": "move", "before": "task:" + epic})
	call("update", map[string]any{"id": "task:" + beta, "action": "note", "text": "evidence"})
	call("list", map[string]any{"epic": "task:" + epic})
	call("take", map[string]any{"id": "task:" + beta, "cwd": dir})
	call("update", map[string]any{"id": "task:" + beta, "action": "check", "item": 1})
	m := call("done", map[string]any{"id": "task:" + beta})
	if tk := m["task"].(map[string]any); tk["state"] != "done" || tk["parent"] != epic {
		t.Fatalf("beta: %v", tk)
	}
}
