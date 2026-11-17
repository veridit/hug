package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// ez50rb check 3: capture never stores a memory that names a swarm agent;
// the same word as an animal, and memories naming tickets, are kept.
func TestCaptureRejectsAgentNames(t *testing.T) {
	f := scopetest.New(t)
	sessions := filepath.Join(f.Root, "alpha", ".jcode", "sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessions, "session_orangutan_1787738929412_b29823251e479ae4.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b := f.Bind(t, "alpha")
	reply := strings.Join([]string{
		line("fact", "medium", "Quartz deployment results are reported to coordinator orangutan per stage.", ""),
		line("fact", "medium", "An orangutan is a great ape that lives in Borneo and Sumatra.", ""),
		line("decision", "high", "TASK-2052 storage lanes run in their own worktree on branch lane/2052-storage.", ""),
	}, "\n")
	fakeCapture(t, f, reply)
	r, err := b.Capture(context.Background(), f.App, captureOpts("agents"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Actions) != 2 || r.Rejected != 1 {
		t.Fatalf("actions %+v rejected %d", r.Actions, r.Rejected)
	}
	for _, a := range r.Actions {
		if strings.Contains(a.Content, "coordinator orangutan") {
			t.Fatalf("agent-naming memory kept: %+v", a)
		}
	}
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mems {
		if strings.Contains(m.Content, "coordinator orangutan") {
			t.Fatal("stored:", m.Content)
		}
	}
	var reason string
	if err := f.App.DB.QueryRow(`SELECT record FROM capture_runs ORDER BY id DESC LIMIT 1`).Scan(&reason); err != nil || !strings.Contains(reason, "names an agent: orangutan") {
		t.Fatalf("audit lacks the rejection reason (%v)", err)
	}
}
