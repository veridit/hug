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

// txjkrm check 2 (MCP): task_update set takes tag (NAME, OLD=NEW, OLD=),
// and the reply's lint block gives an unlisted tag's fixes, each one a
// command, so an MCP agent can run the one it means.
func TestTagsOverMCP(t *testing.T) {
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
	for _, args := range [][]string{{"init"}, {"tags", "add", "compiler"}} {
		if _, e := a.Execute(ctx, args); e != nil {
			t.Fatal(e)
		}
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
	text, bad = call("update", map[string]any{"id": id, "action": "set", "tag": []any{"Compiler", "one-off"}})
	if bad {
		t.Fatal(text)
	}
	var reply struct {
		Task struct {
			Labels []string `json:"labels"`
		} `json:"task"`
		Lint struct {
			Findings []struct {
				Code  string   `json:"code"`
				Fix   string   `json:"fix"`
				Fixes []string `json:"fixes"`
			} `json:"findings"`
		} `json:"lint"`
	}
	if e := json.Unmarshal([]byte(text), &reply); e != nil {
		t.Fatalf("%v\n%s", e, text)
	}
	if strings.Join(reply.Task.Labels, ",") != "Compiler,one-off" {
		t.Fatalf("labels: %v", reply.Task.Labels)
	}
	got := map[string]string{}
	for _, f := range reply.Lint.Findings {
		if f.Code == "unlisted-tag" {
			got[f.Fix] = strings.Join(f.Fixes, " | ")
		}
	}
	if got["hug task set "+id+" --tag Compiler=compiler"] != "hug task set "+id+" --tag Compiler=compiler" ||
		got["hug task tags add one-off"] != "hug task tags add one-off | hug task set "+id+" --tag one-off=" {
		t.Fatalf("lint fixes: %v\n%s", got, text)
	}
	if text, bad = call("update", map[string]any{"id": id, "action": "set", "tag": []any{"Compiler=compiler", "one-off="}}); bad || !strings.Contains(text, `"labels":["compiler"]`) {
		t.Fatalf("fixes over MCP: %v\n%s", bad, text)
	}
}
