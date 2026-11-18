package mcp

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/veridit/hug/huginn/internal/cli"
)

// yf39ya check 3 (MCP site, crab 20:09): task_update action check by the
// person asked, on a shipped task asked about its last open check, reaches
// the same helper as hug task check and the TUI's Space: the task closes,
// with the release in History.
func TestShippedAnswerClosesOverMCP(t *testing.T) {
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
	run := func(args ...string) cli.Reply {
		t.Helper()
		r, e := a.Execute(ctx, args)
		if e != nil {
			t.Fatalf("%v: %v", args, e)
		}
		return r
	}
	run("init")
	run("people", "add", "jhf", "jorgen@veridit.no", "--name", "Jørgen H. Fjeld")
	id := run("new", "Shipped", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "built", "--accept", "Jørgen confirms on his board after release")["task"].(cli.Reply)["id"].(string)
	run("take", id, "--as", "worker", "--model", "m")
	run("check", id, "1")
	run("note", id, "shipped in v0.2.0 (abcdef0)")
	run("ask", id, "--to", "jhf", "--check", "2", "Jørgen confirms on his board after release", "--as", "crab")

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
	out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: "update", Arguments: map[string]any{"id": id, "action": "check", "item": 2, "agent": "jhf"}})
	if e != nil {
		t.Fatal(e)
	}
	text := out.Content[0].(*sdk.TextContent).Text
	if out.IsError || !strings.Contains(text, "after the release: done") {
		t.Fatalf("check over MCP: %s", text)
	}
	r := run("show", id)
	tk := r["task"].(cli.Reply)
	if tk["state"] != "done" {
		t.Fatalf("state %v", tk["state"])
	}
}
