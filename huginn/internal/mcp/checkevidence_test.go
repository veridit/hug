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

// pw0t19 check 2: task_update action check takes an optional text, the
// evidence, with the effect of hug task check ID N "evidence": the check is
// ticked and the evidence is one note naming it. Without text, no note.
func TestCheckEvidenceOverMCP(t *testing.T) {
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
	text, bad := call("new", map[string]any{"title": "Evidence", "goal": "G", "purpose": "P", "principle": "Q", "accept": []any{"one", "two"}})
	if bad {
		t.Fatal(text)
	}
	var created map[string]any
	json.Unmarshal([]byte(text), &created)
	id := created["task"].(map[string]any)["id"].(string)
	if text, bad := call("update", map[string]any{"id": id, "action": "check", "item": 1, "text": "CI run 7 green"}); bad {
		t.Fatalf("check with text: %s", text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "check", "item": 2}); bad {
		t.Fatalf("check without text: %s", text)
	}
	text, _ = call("show", map[string]any{"id": id})
	var shown map[string]any
	json.Unmarshal([]byte(text), &shown)
	tk := shown["task"].(map[string]any)
	notes, _ := tk["notes"].(string)
	if strings.Count(notes, "check 1: CI run 7 green") != 1 || strings.Contains(notes, "check 2:") {
		t.Fatalf("notes after check 1 with text and check 2 without: %q", notes)
	}
	for _, x := range tk["acceptance"].([]any) {
		if m := x.(map[string]any); m["state"] != "ticked" {
			t.Fatalf("not ticked: %v", m)
		}
	}
}

// y8we58 check 1 over MCP: task_update check with text under 8 characters
// is refused with the same fix: and ticks nothing; 8 characters pass.
func TestCheckShortEvidenceOverMCP(t *testing.T) {
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
	text, bad := call("new", map[string]any{"title": "Short", "goal": "G", "purpose": "P", "principle": "Q", "accept": []any{"one"}})
	if bad {
		t.Fatal(text)
	}
	var created map[string]any
	json.Unmarshal([]byte(text), &created)
	id := created["task"].(map[string]any)["id"].(string)
	text, bad = call("update", map[string]any{"id": id, "action": "check", "item": 1, "text": "x"})
	var r map[string]any
	json.Unmarshal([]byte(text), &r)
	if !bad || r["fix"] != "hug task check "+id+" 1 \"what shows it holds\"" || !strings.Contains(text, "at least 8 characters") {
		t.Fatalf("check with text 'x' over MCP: bad=%v\n%s", bad, text)
	}
	text, _ = call("show", map[string]any{"id": id})
	if strings.Contains(text, `"state":"ticked"`) || strings.Contains(text, "check 1: x") {
		t.Fatalf("refused check wrote: %s", text)
	}
	if text, bad = call("update", map[string]any{"id": id, "action": "check", "item": 1, "text": "12345678"}); bad {
		t.Fatalf("8 characters over MCP: %s", text)
	}
}
