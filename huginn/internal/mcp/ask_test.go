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

// 41hhgn check 6 (and 8, MCP): task_update action ask behaves like hug
// task ask: stage asked with asked_what verbatim (line breaks kept),
// asked_for, asked_check; a bad check is refused with a fix; answer over
// MCP keeps a multi-line answer line for line; wait refuses "on".
func TestAskOverMCP(t *testing.T) {
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
	text, bad := call("new", map[string]any{"title": "Rows", "goal": "G", "purpose": "P", "principle": "Q", "accept": []any{"one", "two"}})
	if bad {
		t.Fatal(text)
	}
	var created map[string]any
	json.Unmarshal([]byte(text), &created)
	id := created["task"].(map[string]any)["id"].(string)
	if text, bad := call("take", map[string]any{"id": id, "model": "claude-opus-5-5", "agent": "nautilus", "cwd": dir}); bad {
		t.Fatal(text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "ask", "to": "jhf", "check": 9, "text": "which rows?"}); !bad || !strings.Contains(text, id+" has 2 checks") || !strings.Contains(text, "hug task ask "+id+" --to jhf --check 2") {
		t.Fatalf("check 9 of 2: %v %s", bad, text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "ask", "to": "jhf", "check": 2, "text": "which rows?\nand: in what order?", "agent": "nautilus"}); bad {
		t.Fatal(text)
	}
	file := func() string {
		t.Helper()
		m, _ := filepath.Glob(filepath.Join(dir, cli.TaskDir, "*"+id+"*.md"))
		if len(m) != 1 {
			t.Fatalf("task file of %s: %v", id, m)
		}
		b, e := os.ReadFile(m[0])
		if e != nil {
			t.Fatal(e)
		}
		return string(b)
	}
	text, _ = call("show", map[string]any{"id": id})
	var asked map[string]any
	json.Unmarshal([]byte(text), &asked)
	if o, _ := asked["task"].(map[string]any)["asked"].(map[string]any); o["what"] != "which rows?\nand: in what order?" || o["for"] != "jhf" || o["check"] != float64(2) || o["by"] != "nautilus" {
		t.Fatalf("task_show's asked object: %s", text)
	}
	f := file()
	for _, want := range []string{`stage: "asked"`, `asked_what: "which rows?\nand: in what order?"`, `asked_for: "jhf"`, `asked_check: "2"`, `asked_by: "nautilus"`} {
		if !strings.Contains(f, want) {
			t.Fatalf("ask over MCP lacks %s:\n%s", want, f)
		}
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "wait", "text": "x", "on": "jhf"}); !bad || !strings.Contains(text, "unknown argument on") {
		t.Fatalf("wait with on: %v %s", bad, text)
	}
	answer := "the open ones: \"doing\" first\nthen todo"
	if text, bad := call("update", map[string]any{"id": id, "action": "answer", "text": answer, "agent": "jhf"}); bad {
		t.Fatal(text)
	}
	want := "answered asked (check 2), by jhf, for nautilus (doing/asked -> doing/active):\n  > which rows?\n  > and: in what order?\n  the open ones: \"doing\" first\n  then todo\n"
	if f = file(); !strings.Contains(f, want) || strings.Contains(f, "\nasked_") {
		t.Fatalf("answer over MCP, want the block:\n%s\nfile:\n%s", want, f)
	}
	text, _ = call("show", map[string]any{"id": id})
	var shown map[string]any
	json.Unmarshal([]byte(text), &shown)
	if ans, _ := shown["task"].(map[string]any)["answer"].(map[string]any); ans["text"] != answer {
		t.Fatalf("show's answer: %s", text)
	}
	// z2vmte over MCP: wait takes until, ask takes ask_after.
	if text, bad := call("update", map[string]any{"id": id, "action": "wait", "text": "a week of use", "until": "2099-01-02"}); bad {
		t.Fatal(text)
	}
	if f = file(); !strings.Contains(f, `waiting_until: "2099-01-02"`) {
		t.Fatalf("wait until over MCP:\n%s", f)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "ask", "to": "jhf", "text": "look again?", "ask_after": "2099-01-03"}); bad {
		t.Fatal(text)
	}
	if f = file(); !strings.Contains(f, `waiting_until: "2099-01-03"`) || !strings.Contains(f, `waiting_ask_what: "look again?"`) || strings.Contains(f, "\nasked_") {
		t.Fatalf("ask after over MCP:\n%s", f)
	}
}
