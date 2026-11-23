package mcp

import (
	"context"
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// fthher: task_list takes asked and to, and runs hug task ls --asked
// [--to NAME], so an agent lists the questions waiting for someone as the
// shell does.
func TestListTakesAsked(t *testing.T) {
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
		in   map[string]any
		want []string
	}{
		{map[string]any{"asked": true}, []string{"ls", "--asked"}},
		{map[string]any{"asked": true, "to": "jhf"}, []string{"ls", "--asked", "--to", "jhf"}},
		{map[string]any{"asked": true, "state": "todo"}, []string{"ls", "--asked", "--state", "todo"}},
	} {
		r.args = nil
		out, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: "list", Arguments: tc.in})
		if e != nil {
			t.Fatal(e)
		}
		if out.IsError || !reflect.DeepEqual(r.args, tc.want) {
			t.Errorf("list %v: error %v, ran %v, want %v", tc.in, out.IsError, r.args, tc.want)
		}
	}
}
