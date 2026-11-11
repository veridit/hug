package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupChecklistGuidesSwitchFromBacklog(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "root")
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("<!-- BACKLOG.MD MCP GUIDELINES START -->\nuse backlog\n<!-- BACKLOG.MD MCP GUIDELINES END -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"backlog":{"command":"backlog"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	a := &App{Cwd: dir}
	reply, err := a.Execute(context.Background(), []string{"setup", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(reply)
	got := string(b)
	for _, want := range []string{`"step":"task branch","done":false`, `"do":"huginn init"`, `replace the BACKLOG.MD MCP GUIDELINES block`, `remove the \"backlog\" server`, `"step":"jcode MCP","done":false`} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup missing %s in %s", want, got)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".huginn")); err == nil {
		t.Fatal("setup must not change the repository")
	}
}
