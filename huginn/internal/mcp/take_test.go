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

	failed, m := call("take", map[string]any{"id": id, "cwd": dir})
	if !failed || !strings.Contains(m["error"].(string), "no model given") || m["fix"] != "hug task take "+id+" --model MODEL" {
		t.Fatalf("take without model: %v", m)
	}
	failed, m = call("take", map[string]any{"id": id, "agent": "poodle", "model": "claude-opus-5-5", "cwd": dir})
	if failed {
		t.Fatal(m)
	}
	tk := m["task"].(map[string]any)
	// tzc9hb check 3: the MCP result carries the same reminder as the CLI.
	if g, _ := m["guide"].([]any); len(g) != 2 || !strings.Contains(g[0].(string), "todos: your own next steps on "+id) || m["next"] != "hug task show "+id {
		t.Fatalf("guide %v next %v", m["guide"], m["next"])
	}
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

// x1t1t1: task_take always requires cwd (one habit for agents). Without it,
// or with a relative path or a folder outside git, it fails with a fix line;
// with it, the branch and worktree of that folder are recorded.
func TestTakeRequiresCwdAndRecordsBranch(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	git := func(at string, args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", at}, args...)...)
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, e := c.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, out, e)
		}
	}
	git(dir, "init", "-q", "-b", "main")
	git(dir, "config", "user.name", "t")
	git(dir, "config", "user.email", "t@t")
	git(dir, "commit", "-q", "--allow-empty", "-m", "init")
	wt := filepath.Join(t.TempDir(), "feat-x")
	git(dir, "worktree", "add", "-q", "-b", "feat/x", wt)
	wt, _ = filepath.EvalSymlinks(wt)
	t.Setenv("HUG_USER", "")
	t.Setenv("HUG_MODEL", "")
	t.Setenv("JCODE_SESSION_ID", "")
	ctx := context.Background()
	// The server runs in the main checkout; the agent works in the worktree.
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
		json.Unmarshal([]byte(out.Content[0].(*sdk.TextContent).Text), &m)
		return out.IsError, m
	}
	_, created := call("new", map[string]any{"title": "Over MCP", "goal": "G", "accept": []string{"C"}})
	id := created["task"].(map[string]any)["id"].(string)
	for _, bad := range []map[string]any{
		{"id": id, "model": "claude-opus-5-5"},
		{"id": id, "model": "claude-opus-5-5", "cwd": "relative/dir"},
		{"id": id, "model": "claude-opus-5-5", "cwd": t.TempDir()},
	} {
		failed, m := call("take", bad)
		fix, _ := m["fix"].(string)
		if !failed || fix == "" || !strings.Contains(fix, "cwd") {
			t.Fatalf("take %v must fail with a cwd fix: %v", bad, m)
		}
	}
	failed, m := call("take", map[string]any{"id": id, "agent": "poodle", "model": "claude-opus-5-5", "cwd": filepath.Join(wt)})
	if failed {
		t.Fatal(m)
	}
	tk := m["task"].(map[string]any)
	if tk["branch"] != "feat/x" || tk["worktree"] != wt || tk["model"] != "claude-opus-5-5" {
		t.Fatalf("branch %v worktree %v model %v", tk["branch"], tk["worktree"], tk["model"])
	}
}
