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

// fkpyg3 check 4: task_new and task_update return every problem as a list
// plus the one combined fix.
func TestNewAndUpdateReturnFindingsAndOneFix(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.name", "Git Person"}, {"config", "user.email", "t@example.invalid"}, {"commit", "-q", "--allow-empty", "-m", "init"}} {
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
	// Only a title: every missing part comes back at once.
	failed, m := call("new", map[string]any{"title": "x"})
	fs, _ := m["findings"].([]any)
	if !failed || len(fs) != 4 || m["fix"] != `hug task new "x" --goal "..." --purpose "..." --principle "..." --accept "observable check"` {
		t.Fatalf("new: %v", m)
	}
	var codes []string
	for _, f := range fs {
		codes = append(codes, f.(map[string]any)["code"].(string))
	}
	if strings.Join(codes, ",") != "missing-goal,missing-section,missing-section,no-acceptance" {
		t.Fatal(codes)
	}
	// A complete task, then a set that leaves one problem: the reply lists
	// it with its fix; a set that fixes it clears the list.
	failed, m = call("new", map[string]any{"title": "Ok", "goal": "G", "purpose": "P", "principle": "Q", "accept": []string{"C"}})
	if failed || m["lint"] != nil {
		t.Fatal(m)
	}
	id := m["task"].(map[string]any)["id"].(string)
	failed, m = call("update", map[string]any{"id": id, "action": "set", "title": strings.Repeat("Long title ", 9)})
	l, _ := m["lint"].(map[string]any)
	if failed || l == nil || l["fix"] != "hug task set "+id+` --title "..."` || len(l["findings"].([]any)) != 1 {
		t.Fatalf("update: %v", m)
	}
	// wdkqzx check 4: a note (and every other write) returns the same block.
	failed, m = call("update", map[string]any{"id": id, "action": "note", "text": "where it stands"})
	if l, _ = m["lint"].(map[string]any); failed || l == nil || l["fix"] != "hug task set "+id+` --title "..."` {
		t.Fatalf("update note: %v", m)
	}
	failed, m = call("update", map[string]any{"id": id, "action": "set", "title": "Short", "purpose": "Why now", "accept": []string{"D"}})
	if failed || m["lint"] != nil {
		t.Fatalf("update fix: %v", m)
	}
	if p := m["task"].(map[string]any)["purpose"]; p != "Why now" {
		t.Fatalf("purpose %v", m["task"])
	}
	// zdgw60 check 3: task_update set says what it wrote, as the CLI does.
	w, _ := m["wrote"].([]any)
	var wrote []string
	for _, x := range w {
		wrote = append(wrote, x.(string))
	}
	if got := strings.Join(wrote, "\n"); !strings.Contains(got, "title: ") || !strings.Contains(got, "purpose: Why now") || !strings.Contains(got, "added check 2: D") || len(wrote) != 3 {
		t.Fatalf("update set wrote: %v", m["wrote"])
	}
	// 0pr03h check 1: task_new and task_update set refuse a Goal of more
	// than one sentence or 140 characters, with the same fix as the CLI.
	failed, m = call("new", map[string]any{"title": "Two", "goal": "It works. Then it ships.", "purpose": "P", "principle": "Q", "accept": []string{"C"}})
	if fs, _ := m["findings"].([]any); !failed || len(fs) != 1 || fs[0].(map[string]any)["code"] != "goal-too-long" || m["fix"] != `hug task new "Two" --goal "one sentence" --purpose "P" --principle "Q" --details "..." --accept "C"` {
		t.Fatalf("new long goal: %v", m)
	}
	failed, m = call("update", map[string]any{"id": id, "action": "set", "goal": strings.Repeat("x", 141)})
	if !failed || m["fix"] != "hug task set "+id+` --goal "one sentence" --details "..."` {
		t.Fatalf("update long goal: %v", m)
	}
}
