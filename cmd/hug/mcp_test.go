package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPServesBothParts: one server lists memory_* and task_* tools, and a
// call to each part works against a throwaway home and repository.
func TestMCPServesBothParts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MUNINN_HOME", t.TempDir())
	t.Setenv("JCODE_HOME", "")
	t.Setenv("MUNINN_NO_EMBED", "1")
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	old, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, closeFn, err := newMCPServer(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	st, ct := sdk.NewInMemoryTransports()
	if _, err := s.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	if !strings.Contains(cs.InitializeResult().Instructions, "task_next") {
		t.Errorf("instructions do not mention the prefixed task tools:\n%s", cs.InitializeResult().Instructions)
	}
	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tl := range tools.Tools {
		names = append(names, tl.Name)
	}
	for _, want := range []string{"memory_search", "memory_remember", "task_list", "task_new", "task_next"} {
		if !slices.Contains(names, want) {
			t.Errorf("missing tool %s in %v", want, names)
		}
	}
	for _, n := range names {
		if !strings.HasPrefix(n, "memory_") && !strings.HasPrefix(n, "task_") {
			t.Errorf("tool %q has neither prefix", n)
		}
	}

	callText := func(name string, args map[string]any) (string, bool) {
		t.Helper()
		r, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		b, _ := json.Marshal(r.Content)
		return string(b), r.IsError
	}
	if out, isErr := callText("memory_stats", map[string]any{}); isErr {
		t.Errorf("memory_stats failed: %s", out)
	}
	if out, isErr := callText("task_types", map[string]any{}); isErr || !strings.Contains(out, "feature") {
		t.Errorf("task_types: err=%v %s", isErr, out)
	}
}
