package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/veridit/hug/huginn/internal/cli"
)

// et85h1: task_take takes agent and model and records both; without a model
// it fails with a fix that names the model.
func TestTakeRecordsAgentAndModel(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.name", "Git Person"}, {"config", "user.email", "t@example.invalid"}, {"commit", "-q", "--allow-empty", "-m", "init"}} {
		if out, e := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, out, e)
		}
	}
	t.Setenv("HUG_USER", "")
	t.Setenv("HUG_MODEL", "")
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
	call := func(name string, input map[string]any) (bool, map[string]any) {
		t.Helper()
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: input})
		if e != nil {
			t.Fatal(e)
		}
		var m map[string]any
		if e := json.Unmarshal([]byte(out.Content[0].(*sdk.TextContent).Text), &m); e != nil {
			t.Fatal(e)
		}
		return out.IsError, m
	}
	failed, created := call("new", map[string]any{"title": "Over MCP", "goal": "G", "accept": []string{"C"}})
	if failed {
		t.Fatal(created)
	}
	id := created["task"].(map[string]any)["id"].(string)

	failed, m := call("take", map[string]any{"id": id})
	if !failed || !strings.Contains(m["error"].(string), "no model given") || m["fix"] != "hug task take "+id+" --model MODEL" {
		t.Fatalf("take without model: %v", m)
	}
	failed, m = call("take", map[string]any{"id": id, "agent": "poodle", "model": "claude-opus-5-5"})
	if failed {
		t.Fatal(m)
	}
	tk := m["task"].(map[string]any)
	if as := tk["assignee"].([]any); len(as) != 1 || as[0] != "poodle" || tk["model"] != "claude-opus-5-5" {
		t.Fatalf("assignee %v model %v", tk["assignee"], tk["model"])
	}
	_, m = call("show", map[string]any{"id": id})
	if tk := m["task"].(map[string]any); tk["model"] != "claude-opus-5-5" {
		t.Fatal(m)
	}
	files, _ := filepath.Glob(filepath.Join(dir, cli.TaskDir, "*"+id+"*.md"))
	b, _ := os.ReadFile(files[0])
	if !strings.Contains(string(b), `assignee: ["poodle"]`) || !strings.Contains(string(b), `model: "claude-opus-5-5"`) || strings.Contains(string(b), "Git Person") {
		t.Fatalf("task file:\n%s", b)
	}
}
