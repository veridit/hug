package run

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// Tool-call recall (n7jbmx): with tool_recall on, jcode's post_tool_feedback
// hook prints the rules and facts that name a path or command of the call,
// once per session, from this binding's scopes only. Off (the default), the
// hook does what it did before: nothing when nothing is parked.
func TestJcodeToolRecall(t *testing.T) {
	f := scopetest.New(t)
	t.Chdir(f.Cwd)
	j := newJcodeTurns(t, f)
	prev := toolSimilar
	toolSimilar = func(*app.App) func(context.Context, string, []int64) (map[int64]float64, error) {
		return func(_ context.Context, _ string, ids []int64) (map[int64]float64, error) {
			out := map[int64]float64{}
			for _, id := range ids {
				out[id] = 0.9
			}
			return out, nil
		}
	}
	t.Cleanup(func() { toolSimilar = prev })
	for _, m := range [][2]string{
		{"Edits to huginn/internal/task/questions.go go through hatchling only.", "project:matrix-project"},
		{"In the other project huginn/internal/task/questions.go is vendored.", "project:other-project"},
	} {
		if _, _, err := ingest.RememberTyped(app.NativePath(), m[0], "fact", "rule", m[1], nil, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	call := func() string {
		t.Helper()
		t.Setenv("JCODE_HOOK_EVENT", "post_tool_feedback")
		t.Setenv("JCODE_HOOK_TOOL_NAME", "edit")
		t.Setenv("JCODE_HOOK_TOOL_INPUT", `{"file_path":"huginn/internal/task/questions.go"}`)
		t.Setenv("JCODE_HOOK_TOUCHED_PATHS", filepath.Join(f.Cwd, "huginn", "internal", "task", "questions.go"))
		out, err := captureSurface(t, func() error { return runHook("jcode", 5*time.Second) })
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	logPath := filepath.Join(config.DataDir(), "hook.log")
	before, _ := os.ReadFile(logPath)
	if out := call(); out != "" {
		t.Fatalf("tool_recall off, yet the hook printed:\n%s", out)
	}
	if after, _ := os.ReadFile(logPath); len(after) != len(before) {
		t.Fatalf("tool_recall off, yet the hook logged:\n%s", after[len(before):])
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	on := true
	cfg.ToolRecall = &on
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	out := call()
	if !strings.Contains(out, recallMarker) || !strings.Contains(out, "hatchling only") {
		t.Fatalf("tool_recall on: the rule naming the edited file is missing:\n%s", out)
	}
	if strings.Contains(out, "vendored") {
		t.Fatalf("another project's memory reached this binding:\n%s", out)
	}
	if again := call(); again != "" {
		t.Fatalf("shown twice in one session:\n%s", again)
	}
	if b, _ := os.ReadFile(logPath); !strings.Contains(string(b), "tool=edit hits=1") {
		t.Fatalf("hook.log lacks the tool recall line:\n%s", b)
	}
	j.noTranscript()
}
