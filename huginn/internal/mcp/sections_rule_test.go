package mcp

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// 32ym01 check 1: task_new's description says each section's job in the
// sentence AGENTS.md, hug setup's template and help new say
// (huginn/internal/cli TestSectionsRuleWording).
func TestTaskNewSaysSectionsRule(t *testing.T) {
	const rule = "Describe a task with Goal (what: where we aim), Purpose (why), Principle (how: what we do and do not do), Details (notes that explain those), then Acceptance."
	ctx := context.Background()
	server := New(&recorder{})
	ct, st := sdk.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "new" {
			if !strings.Contains(tool.Description, rule) {
				t.Fatalf("task_new description does not say %q:\n%s", rule, tool.Description)
			}
			return
		}
	}
	t.Fatal("no new tool")
}
