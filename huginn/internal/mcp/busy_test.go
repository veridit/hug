package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/veridit/hug/huginn/internal/cli"
	"github.com/veridit/hug/huginn/internal/task"
)

// n2hq8p check 3: task_take gives the same refusal as the CLI when the
// agent already has an active ticket: the problem names it, fix is the
// first way out and fixes lists all three.
func TestSecondTakeOverMCPIsRefused(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.name", "T"}, {"config", "user.email", "t@example.invalid"}, {"commit", "-q", "--allow-empty", "-m", "init"}} {
		if out, e := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, out, e)
		}
	}
	t.Setenv("HUG_USER", "")
	t.Setenv("HUG_MODEL", "")
	t.Setenv("JCODE_SESSION_ID", "")
	t.Setenv("JCODE_HOME", dir+"/.no-jcode")
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
	call := func(name string, input map[string]any) (map[string]any, bool) {
		t.Helper()
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: input})
		if e != nil {
			t.Fatal(e)
		}
		var m map[string]any
		if e := json.Unmarshal([]byte(out.Content[0].(*sdk.TextContent).Text), &m); e != nil {
			t.Fatal(e)
		}
		return m, out.IsError
	}
	ids := []string{}
	for _, title := range []string{"First", "Second"} {
		m, bad := call("new", map[string]any{"title": title, "goal": "G", "purpose": "P", "principle": "Q", "accept": []any{"C"}})
		if bad {
			t.Fatal(m)
		}
		ids = append(ids, m["task"].(map[string]any)["id"].(string))
	}
	take := func(id string) (map[string]any, bool) {
		return call("take", map[string]any{"id": id, "model": "claude-opus-5-5", "agent": "kestrel", "cwd": dir})
	}
	if m, bad := take(ids[0]); bad {
		t.Fatal(m)
	}
	m, bad := take(ids[1])
	if !bad {
		t.Fatalf("second take over MCP succeeded: %v", m)
	}
	want := []any{
		"hug task done " + ids[0],
		"hug task wait " + ids[0] + ` "on what"`,
		"hug task set " + ids[0] + " --state todo && hug task note " + ids[0] + ` "why it goes back"`,
		"hug task dep " + ids[0] + " --add " + ids[1] + " && hug task note " + ids[0] + ` "where I was"`,
		"hug task take " + ids[1] + " --with " + ids[0] + " --model claude-opus-5-5 --as kestrel",
	}
	fixes, _ := m["fixes"].([]any)
	if len(fixes) != 5 || fixes[0] != want[0] || fixes[1] != want[1] || fixes[2] != want[2] || fixes[3] != want[3] || fixes[4] != want[4] || m["fix"] != want[0] {
		t.Fatalf("fixes %v fix %v, want %v", m["fixes"], m["fix"], want)
	}
	if p, _ := m["error"].(string); p != "not taking "+ids[1]+": kestrel is on "+ids[0]+" (First): one ticket in doing at a time; "+task.WithWhen {
		t.Fatalf("error %q", p)
	}
	if code, _ := m["exit_code"].(float64); code != 3 {
		t.Fatalf("exit_code %v, want 3", m["exit_code"])
	}
	// The CLI says the same, with one fix: line per way out.
	code, p, fix := cli.ErrorDetails(mustTakeErr(t, a, ids[1]))
	if code != 3 || p != m["error"] || fix != want[0] || len(cli.ErrorFixes(mustTakeErr(t, a, ids[1]))) != 5 {
		t.Fatalf("CLI: %d %q %q", code, p, fix)
	}
	// byj479 check 3 over MCP: ticking a check on the second (todo, not
	// taken) ticks it and the reply's fix takes it with the first.
	m, bad = call("update", map[string]any{"id": ids[1], "action": "check", "item": float64(1), "agent": "kestrel", "model": "claude-opus-5-5"})
	if bad || m["fix"] != want[4] {
		t.Fatalf("check on an untaken task over MCP: fix %v, want %v (%v)", m["fix"], want[4], m)
	}
	// byj479 check 1 over MCP: task_take with: both doing.
	if m, bad = call("take", map[string]any{"id": ids[1], "with": ids[0], "model": "claude-opus-5-5", "agent": "kestrel", "cwd": dir}); bad || m["task"].(map[string]any)["doing_with"] != ids[0] {
		t.Fatalf("task_take with: %v", m)
	}
}

func mustTakeErr(t *testing.T, a *cli.App, id string) error {
	t.Helper()
	_, e := a.Execute(context.Background(), []string{"take", id, "--model", "claude-opus-5-5", "--as", "kestrel"})
	if e == nil {
		t.Fatalf("CLI take %s succeeded", id)
	}
	return e
}
