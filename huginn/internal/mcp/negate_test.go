package mcp

import (
	"context"
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// 7zg9mf check 2 (the MCP half): task_list, task_search and task_next take
// a negated state or type, as the CLI does ("!done": every task but the
// done ones); the schema names the negated values, so a typo is still
// refused before it runs.
func TestListTakesNegatedFilters(t *testing.T) {
	ctx := context.Background()
	r := &recorder{}
	ct, st := sdk.NewInMemoryTransports()
	ss, e := New(r).Connect(ctx, st, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer ss.Close()
	cs, e := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer cs.Close()
	for _, tc := range []struct {
		tool string
		in   map[string]any
		want []string
	}{
		{"list", map[string]any{"state": "!done"}, []string{"ls", "--state", "!done"}},
		{"list", map[string]any{"type": "!bug", "state": "!dropped"}, []string{"ls", "--type", "!bug", "--state", "!dropped"}},
		{"search", map[string]any{"query": "x", "state": "!done"}, []string{"search", "--state", "!done", "--", "x"}},
		{"next", map[string]any{"type": "!docs"}, []string{"next", "--type", "!docs"}},
	} {
		r.args = nil
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: tc.tool, Arguments: tc.in})
		if e != nil {
			t.Fatal(e)
		}
		if out.IsError || !reflect.DeepEqual(r.args, tc.want) {
			t.Errorf("%s %v: error %v, ran %v, want %v", tc.tool, tc.in, out.IsError, r.args, tc.want)
		}
	}
	for _, in := range []map[string]any{{"state": "!don"}, {"state": "!"}, {"type": "!!bug"}} {
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: "list", Arguments: in})
		if e != nil {
			t.Fatal(e)
		}
		if !out.IsError {
			t.Errorf("list %v accepted", in)
		}
	}
}
