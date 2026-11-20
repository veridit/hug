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

// 5vehza (MCP): task_update action answer on a withdrawn question is the
// same refusal as hug task answer (was withdrawn, a fix: line to ask it
// again), a tool error, not a crash of the server.
func TestAnswerWithdrawnQuestionOverMCP(t *testing.T) {
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
	text, bad := call("new", map[string]any{"title": "Rows", "goal": "G", "purpose": "P", "principle": "Q", "accept": []any{"one"}})
	if bad {
		t.Fatal(text)
	}
	var created map[string]any
	json.Unmarshal([]byte(text), &created)
	id := created["task"].(map[string]any)["id"].(string)
	if text, bad := call("take", map[string]any{"id": id, "model": "claude-opus-5-5", "agent": "nautilus", "cwd": dir}); bad {
		t.Fatal(text)
	}
	for _, step := range []map[string]any{
		{"id": id, "action": "ask", "to": "jhf", "check": 1, "text": "which rows?", "agent": "nautilus"},
		{"id": id, "action": "resume"},
		{"id": id, "action": "ask", "to": "jhf", "check": 1, "text": "which rows, again?", "agent": "nautilus"},
	} {
		if text, bad := call("update", step); bad {
			t.Fatalf("%v: %s", step, text)
		}
	}
	text, bad = call("update", map[string]any{"id": id, "action": "answer", "item": 1, "text": "yes", "agent": "jhf"})
	if !bad || !strings.Contains(text, "question 1 on "+id+" was withdrawn at ") || !strings.Contains(text, "hug task ask "+id+" --to jhf --check 1 \\\"which rows?\\\"") && !strings.Contains(text, "hug task ask "+id+" --to jhf --check 1 \"which rows?\"") {
		t.Fatalf("answer to withdrawn question 1 over MCP: error %v\n%s", bad, text)
	}
	// The server still serves: question 2 answers.
	if text, bad := call("update", map[string]any{"id": id, "action": "answer", "item": 2, "text": "yes", "agent": "jhf"}); bad {
		t.Fatal(text)
	}
}
