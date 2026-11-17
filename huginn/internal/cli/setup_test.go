package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	for _, want := range []string{`"step":"task branch","done":false`, `"do":"hug task init"`, `replace the BACKLOG.MD MCP GUIDELINES block`, `remove the \"backlog\" server`, `"step":"jcode MCP","done":false`} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup missing %s in %s", want, got)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".hug", "tasks")); err == nil {
		t.Fatal("setup must not change the repository")
	}
}

// TestSetupOffersForbidScratch (cvbpdy check 3): in a repo with .jcode/ and
// no forbid_cite, setup offers the line; running its fix writes and commits
// it, after which setup reports the step done and lint reports citations.
func TestSetupOffersForbidScratch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fix is a POSIX shell command")
	}
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	if err := os.MkdirAll(filepath.Join(a.Cwd, ".jcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	step := func() SetupStep {
		t.Helper()
		r := execute(t, a, "setup")
		for _, s := range r["setup"].([]SetupStep) {
			if s.Step == "scratch citations" {
				return s
			}
		}
		t.Fatalf("setup has no scratch citations step: %v", r["setup"])
		return SetupStep{}
	}
	s := step()
	if s.Done || !strings.Contains(s.Do, "forbid_cite = [") {
		t.Fatalf("step %+v; want not done with a fix that writes forbid_cite", s)
	}
	// Run the fix as an agent would, without the final lint (no hug on PATH).
	fix := strings.Split(s.Do, " && hug task lint")[0]
	c := exec.Command("sh", "-c", fix)
	c.Dir = a.Cwd
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("fix %q: %v %s", fix, err, out)
	}
	if s := step(); !s.Done {
		t.Fatalf("after the fix the step is %+v", s)
	}
	if st := git(t, filepath.Join(a.Cwd, ".hug", "tasks"), "status", "--porcelain"); st != "" {
		t.Fatalf("fix left the task branch dirty: %s", st)
	}
	r := execute(t, a, "new", "Cites scratch", "--goal", "see .jcode/scratch/T/x.md", "--purpose", "p", "--principle", "q", "--accept", "x")
	id := r["task"].(Reply)["id"].(string)
	b, _ := json.Marshal(execute(t, a, "lint", id, "--code", "forbidden-citation"))
	if !strings.Contains(string(b), "cites .jcode/scratch/T/x.md") {
		t.Fatalf("lint after the fix: %s", b)
	}
}
