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

// hbgybn check 5: task_update action request and resolve behave like the
// CLI; task_requests lists the open ones with the resolve fix.
func TestRequestOverMCP(t *testing.T) {
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
	req := "split it:\n\"rows\" and the header"
	if text, bad := call("update", map[string]any{"id": id, "action": "request", "text": req, "agent": "jhf"}); bad {
		t.Fatal(text)
	}
	text, bad = call("requests", map[string]any{})
	var listed map[string]any
	json.Unmarshal([]byte(text), &listed)
	rows, _ := listed["requests"].([]any)
	if bad || len(rows) != 1 {
		t.Fatalf("task_requests: %v %s", bad, text)
	}
	row := rows[0].(map[string]any)
	r := row["request"].(map[string]any)
	if row["id"] != id || r["text"] != req || r["by"] != "jhf" || r["n"] != float64(1) || row["fix"] != "hug task resolve "+id+" 1 \"what changed\"" {
		t.Fatalf("task_requests row: %s", text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "resolve", "text": "x"}); !bad || !strings.Contains(text, "resolve needs item") {
		t.Fatalf("resolve without item: %v %s", bad, text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "resolve", "item": 2, "text": "x", "agent": "nautilus"}); !bad || !strings.Contains(text, id+" has no request 2") {
		t.Fatalf("resolve 2: %v %s", bad, text)
	}
	if text, bad := call("update", map[string]any{"id": id, "action": "resolve", "item": 1, "text": "split into two checks\nheader is its own task", "agent": "nautilus"}); bad {
		t.Fatal(text)
	}
	text, _ = call("show", map[string]any{"id": id})
	var shown map[string]any
	json.Unmarshal([]byte(text), &shown)
	rs, _ := shown["task"].(map[string]any)["requests"].([]any)
	if len(rs) != 1 {
		t.Fatalf("task_show requests: %s", text)
	}
	res, _ := rs[0].(map[string]any)["resolved"].(map[string]any)
	if res["by"] != "nautilus" || res["text"] != "split into two checks\nheader is its own task" {
		t.Fatalf("task_show resolved: %s", text)
	}
	if text, _ = call("requests", map[string]any{}); strings.Contains(text, `"id":"`+id+`"`) {
		t.Fatalf("a resolved request still listed: %s", text)
	}
}

// gmzd87 check 3 (MCP half): task_requests and task_update describe how
// requests and answers reach the coordinator in cli.DeliveryFlow's words
// (the behaviour behind each claim: cli TestDeliveryInstructionsMatchCode);
// task_update request replies with the receipt, and task_requests gives
// request.receipt.state.
func TestDeliveryFlowInMCP(t *testing.T) {
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
	tools, e := cs.ListTools(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	found := 0
	for _, tl := range tools.Tools {
		if tl.Name == "requests" || tl.Name == "update" {
			found++
			if !strings.Contains(tl.Description, cli.DeliveryFlow) {
				t.Errorf("%s's description lacks DeliveryFlow:\n%s", tl.Name, tl.Description)
			}
		}
	}
	if found != 2 {
		t.Fatalf("found %d of requests and update", found)
	}
	call := func(name string, input map[string]any) map[string]any {
		t.Helper()
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: input})
		if e != nil || out.IsError {
			t.Fatal(e, out)
		}
		var m map[string]any
		json.Unmarshal([]byte(out.Content[0].(*sdk.TextContent).Text), &m)
		return m
	}
	id := call("new", map[string]any{"title": "Rows", "goal": "G", "purpose": "P", "principle": "Q", "accept": []any{"one"}})["task"].(map[string]any)["id"].(string)
	r := call("update", map[string]any{"id": id, "action": "request", "text": "split it", "agent": "jhf"})
	if r["receipt"] != "pending, not delivered: no coordinator set (hug task coordinator NAME)" {
		t.Fatalf("task_update request receipt: %v", r["receipt"])
	}
	rows := call("requests", map[string]any{})["requests"].([]any)
	rc := rows[0].(map[string]any)["request"].(map[string]any)["receipt"].(map[string]any)
	if rc["state"] != "pending" || rc["not_delivered"] != "no coordinator set (hug task coordinator NAME)" {
		t.Fatalf("task_requests receipt: %v", rc)
	}
}

// ec6z6j (MCP half): task_update request with ids writes one request for
// the selection: the text once on id (the home), a link on each of ids;
// an unknown id writes nothing.
func TestRequestForSeveralOverMCP(t *testing.T) {
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
	for _, title := range []string{"Rows", "Header"} {
		m, text, bad := call("new", map[string]any{"title": title, "goal": "G", "purpose": "P", "principle": "Q", "accept": []any{"one"}})
		if bad {
			t.Fatal(text)
		}
		ids = append(ids, m["task"].(map[string]any)["id"].(string))
	}
	if _, text, bad := call("update", map[string]any{"id": ids[0], "action": "request", "ids": []any{ids[1], "zzzzzz"}, "text": "one fix", "agent": "jhf"}); !bad || !strings.Contains(text, "no task zzzzzz: nothing was written") {
		t.Fatalf("unknown id: %v %s", bad, text)
	}
	if _, text, bad := call("update", map[string]any{"id": ids[0], "action": "request", "ids": []any{ids[1]}, "text": "one fix", "agent": "jhf"}); bad {
		t.Fatal(text)
	}
	m, text, _ := call("show", map[string]any{"id": ids[1]})
	rs, _ := m["task"].(map[string]any)["requests"].([]any)
	if len(rs) != 1 {
		t.Fatalf("link: %s", text)
	}
	r := rs[0].(map[string]any)
	home, _ := r["home"].(map[string]any)
	targets, _ := r["targets"].([]any)
	if r["text"] != "one fix" || home["id"] != ids[0] || len(targets) != 2 {
		t.Fatalf("link reads as the request: %s", text)
	}
}
