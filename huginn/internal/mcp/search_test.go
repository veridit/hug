package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/veridit/hug/huginn/internal/cli"
)

// 49n9kw check 3: MCP search returns what hug task search --json returns.
func TestSearchToolMatchesCLI(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.name", "T"}, {"config", "user.email", "t@example.invalid"}, {"commit", "-q", "--allow-empty", "-m", "init"}} {
		if out, e := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, out, e)
		}
	}
	ctx := context.Background()
	a := &cli.App{Cwd: dir, Out: os.Stderr, Err: os.Stderr, Env: map[string]string{"HUG_USER": "tester", "HUG_MODEL": "test-model", "JCODE_SESSION_ID": ""}}
	for _, args := range [][]string{{"init"}, {"new", "Memory limit exit 137", "--goal", "G", "--accept", "C"}, {"new", "Logging", "--goal", "exit 137 means memory", "--accept", "C"}, {"new", "Other", "--goal", "nothing", "--accept", "C"}} {
		if _, e := a.Execute(ctx, args); e != nil {
			t.Fatal(args, e)
		}
	}
	want, e := a.Execute(ctx, []string{"search", "memory", "137", "--json"})
	if e != nil {
		t.Fatal(e)
	}
	wantJSON, _ := json.Marshal(want["results"])
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
	out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: "search", Arguments: map[string]any{"query": "memory 137"}})
	if e != nil || out.IsError {
		t.Fatalf("%v %+v", e, out)
	}
	var got map[string]any
	if e := json.Unmarshal([]byte(out.Content[0].(*sdk.TextContent).Text), &got); e != nil {
		t.Fatal(e)
	}
	gotJSON, _ := json.Marshal(got["results"])
	if string(gotJSON) != string(wantJSON) || len(got["results"].([]any)) != 2 {
		t.Fatalf("MCP\n%s\nCLI\n%s", gotJSON, wantJSON)
	}
}
