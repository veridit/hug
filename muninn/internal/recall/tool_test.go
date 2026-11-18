package recall_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/store"
)

// toolSetup indexes memories for tool-call recall (n7jbmx).
func toolSetup(t *testing.T, mems [][3]string) *store.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	native := filepath.Join(dir, "n.json")
	for _, m := range mems { // text, type, scope
		if _, _, err := ingest.RememberTyped(native, m[0], "fact", m[1], m[2], nil, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := (&index.Indexer{DB: db}).Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestToolRecallAnchors(t *testing.T) {
	db := toolSetup(t, [][3]string{
		{"Edits to huginn/internal/task/questions.go go through hatchling only.", "rule", "project:hug"},
		{"questions.go once held the lint table.", "record", "project:hug"},                    // a record: never on a tool call
		{"In brovs, huginn/internal/task/questions.go is vendored.", "fact", "project:brovs"},  // another project
		{"Run `go test` with GOMAXPROCS=2 on this machine.", "preference", "global"},           // common anchor
		{"`go test` of muninn/run/ needs -short for the tmux goldens.", "fact", "project:hug"}, // rarer anchors
		{"`go test` output goes to tmp/, never to the console.", "rule", "global"},
	})
	e := &recall.Engine{DB: db}
	ctx := context.Background()
	call := func(session, tool, cmd string, touched ...string) recall.Response {
		t.Helper()
		r, err := e.ToolRecall(ctx, recall.ToolRequest{Session: session, Harness: "jcode", Tool: tool, Command: cmd, Touched: touched,
			Root: "/w/hug", Home: "/w", Scopes: []string{"global", "project:hug"}})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	r := call("s1", "edit", "", "/w/hug/huginn/internal/task/questions.go")
	if len(r.Hits) != 1 || !strings.Contains(r.Hits[0].Text, "hatchling only") {
		t.Fatalf("edit questions.go: want the one rule in scope, got %+v", r.Hits)
	}
	if !strings.HasPrefix(r.Block, recall.Marker) {
		t.Fatalf("block not marked: %q", r.Block)
	}
	if r := call("s1", "edit", "", "/w/hug/huginn/internal/task/questions.go"); len(r.Hits) != 0 {
		t.Fatalf("once per session: shown again %+v", r.Hits)
	}
	if r := call("s2", "edit", "", "/w/hug/huginn/internal/task/questions.go"); len(r.Hits) != 1 {
		t.Fatalf("another session sees it: %+v", r.Hits)
	}
	// Three memories name `go test`; at most two are shown, the one that
	// also names the call's path first.
	r = call("s3", "bash", "cd /w/hug && go test -short ./muninn/run/")
	if len(r.Hits) != 2 || !strings.Contains(r.Hits[0].Text, "tmux goldens") {
		t.Fatalf("go test: want 2, rarest match first, got %+v", r.Hits)
	}
	if r := call("s4", "bash", "ls -la"); len(r.Hits) != 0 || r.Skipped != "no anchors" {
		t.Fatalf("no anchors: %+v", r)
	}
	if r := call("s5", "edit", "", "/w/hug/docs/other.md"); len(r.Hits) != 0 || r.Block != "" {
		t.Fatalf("no match: %+v", r)
	}
}

// An anchor that many memories share says nothing about any one of them:
// above MaxShared it never triggers on its own.
func TestToolRecallCommonAnchor(t *testing.T) {
	var mems [][3]string
	for i := 0; i < 5; i++ {
		mems = append(mems, [3]string{"Use `git push` only after a rebase, note " + string(rune('a'+i)) + ".", "rule", "global"})
	}
	db := toolSetup(t, mems)
	e := &recall.Engine{DB: db}
	r, err := e.ToolRecall(context.Background(), recall.ToolRequest{Session: "s", Tool: "bash", Command: "git push", MaxShared: 4, Scopes: []string{"global"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Hits) != 0 {
		t.Fatalf("an anchor 5 memories share triggered: %+v", r.Hits)
	}
	r, _ = e.ToolRecall(context.Background(), recall.ToolRequest{Session: "s", Tool: "bash", Command: "git push", MaxShared: 5, Scopes: []string{"global"}})
	if len(r.Hits) != 2 {
		t.Fatalf("at MaxShared 5: want 2 hits, got %+v", r.Hits)
	}
}

// The shipped arm: a path anchor ending at the call's path or folder, a
// command alone never triggers, and the cosine gate. Without an embedder the
// gate cannot pass, so nothing is shown.
func TestToolRecallShipped(t *testing.T) {
	db := toolSetup(t, [][3]string{
		{"Run `go test` with GOMAXPROCS=2 on this machine.", "preference", "global"},                      // command only
		{"Work in a fresh clone under Projects/Brovs/.jcode/scratch, never in DevOps.", "rule", "global"}, // far ancestor
		{"Edits to huginn/internal/task/questions.go go through hatchling only.", "rule", "global"},
		{"huginn/internal/task/questions.go has the lint table.", "fact", "global"},
	})
	e := &recall.Engine{DB: db}
	ctx := context.Background()
	near := map[string]float64{"hatchling": 0.8, "lint table": 0.5}
	similar := func(_ context.Context, _ string, ids []int64) (map[int64]float64, error) {
		out := map[int64]float64{}
		for _, id := range ids {
			u, _ := db.GetUnit(id)
			for k, v := range near {
				if strings.Contains(u.Body, k) {
					out[id] = v
				}
			}
		}
		return out, nil
	}
	req := recall.ToolRequest{Session: "s", Tool: "bash", Command: "cd /w/Projects/Brovs/.jcode/scratch/x/y && go test ./...", Root: "/w/hug", Home: "/w", Scopes: []string{"global"}, Similar: similar}.Shipped()
	if r, _ := e.ToolRecall(ctx, req); len(r.Hits) != 0 {
		t.Fatalf("a command alone or a far ancestor triggered: %+v", r.Hits)
	}
	req.Command, req.Touched = "", []string{"/w/hug/huginn/internal/task/questions.go"}
	r, _ := e.ToolRecall(ctx, req)
	if len(r.Hits) != 1 || !strings.Contains(r.Hits[0].Text, "hatchling") {
		t.Fatalf("want only the memory above the cosine gate, got %+v", r.Hits)
	}
	req.Session, req.Similar = "s2", nil
	if r, _ := e.ToolRecall(ctx, req); len(r.Hits) != 0 || r.Skipped != "no embedder" {
		t.Fatalf("no embedder: want nothing, got %+v", r)
	}
}
