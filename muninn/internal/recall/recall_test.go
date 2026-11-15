package recall_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/testutil"
)

// failing embeds nothing: models Ollama going away after indexing.
type failing struct{}

func (failing) Model() string { return "fake-embed" }
func (failing) Embed(context.Context, []string) ([][]float32, error) {
	return nil, errors.New("connection refused")
}

func setup(t *testing.T) *store.DB {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	native := filepath.Join(dir, "n.json")
	ingest.Remember(native, `Trigger "banana mode": reply in Norwegian.`, "preference", "global", nil)
	ingest.Remember(native, "Deploys go through staging first.", "fact", "global", nil)
	if _, err := (&index.Indexer{DB: db, Embedder: &testutil.FakeEmbedder{}}).Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	return db
}

func engine(db *store.DB, e interface {
	Model() string
	Embed(context.Context, []string) ([][]float32, error)
}) *recall.Engine {
	o := search.DefaultOptions()
	o.Limit, o.MinScore = 5, 6
	return &recall.Engine{DB: db, Searcher: &search.Searcher{DB: db, Embedder: e}, Options: o, RelativeFloor: 0.62, MinSemantic: 0.99}
}

func TestRecallDegradesWhenEmbedderFails(t *testing.T) {
	db := setup(t)
	// MinSemantic 0.99 would reject everything if the gate ran on zero sims.
	resp, err := engine(db, failing{}).Recall(context.Background(), recall.Request{Message: "banana mode"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Block, "banana mode") {
		t.Fatalf("keyword-only fallback lost the match: %+v", resp)
	}
}

func TestRecallGuardsAndDedupe(t *testing.T) {
	db := setup(t)
	e := engine(db, failing{})
	ctx := context.Background()
	if r, _ := e.Recall(ctx, recall.Request{Message: recall.Marker + " - banana mode"}); r.Block != "" || r.Skipped != "own injection" {
		t.Fatalf("loop guard: %+v", r)
	}
	if r, _ := e.Recall(ctx, recall.Request{Message: "   "}); r.Block != "" {
		t.Fatalf("empty message: %+v", r)
	}
	first, _ := e.Recall(ctx, recall.Request{Session: "s1", Message: "banana mode"})
	second, _ := e.Recall(ctx, recall.Request{Session: "s1", Message: "banana mode"})
	other, _ := e.Recall(ctx, recall.Request{Session: "s2", Message: "banana mode"})
	if first.Block == "" || second.Block != "" || other.Block == "" {
		t.Fatalf("dedupe: first=%q second=%q other=%q", first.Block, second.Block, other.Block)
	}
	if !strings.HasPrefix(first.Block, recall.Marker) {
		t.Fatalf("block not marked: %q", first.Block)
	}
}

// Harness notices carry log text that matches memories by vocabulary only
// (2026-11-06: a failed Xcode build notice in a WebKit session recalled Brovs
// and Frogs memories). They are not the user asking, so recall stays quiet.
func TestRecallSkipsHarnessNotices(t *testing.T) {
	db := setup(t)
	e := engine(db, failing{})
	ctx := context.Background()
	for _, msg := range []string{
		"**Background task** `496022ps4i` · `Run build` (`bash`) · ✗ failed · banana mode",
		"🐝 **Swarm await finished**\n\nAll members done. banana mode",
		"[Scheduled task]\nA scheduled task for this session is now due: banana mode",
		"[auto] Your session was interrupted by a server reload. banana mode",
		"You have 3 incomplete todos. Continue working. banana mode",
	} {
		if r, _ := e.Recall(ctx, recall.Request{Message: msg}); r.Block != "" || r.Skipped != "harness notice" {
			t.Fatalf("notice recalled: %q -> %+v", msg, r)
		}
	}
	if r, _ := e.Recall(ctx, recall.Request{Message: "What about banana mode? The background task failed."}); r.Block == "" {
		t.Fatalf("a real question mentioning a background task must still recall: %+v", r)
	}
}

func TestTruncateRuneSafe(t *testing.T) {
	s := strings.Repeat("ø", 10) // 2 bytes each
	if got := recall.Truncate(s, 5); got != "øø" {
		t.Fatalf("got %q", got)
	}
}

// Records (what happened) are left out of automatic recall for long
// messages when SkipRecordsAbove is set; short messages still get them.
func TestRecallSkipsRecordsOnLongMessages(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	native := filepath.Join(dir, "n.json")
	ingest.RememberTyped(native, "Release banana 1.4 shipped on 3 November after the banana gate passed.", "fact", "record", "global", nil, "")
	if _, err := (&index.Indexer{DB: db, Embedder: &testutil.FakeEmbedder{}}).Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	e := engine(db, failing{})
	e.SkipRecordsAbove = 60
	short, _ := e.Recall(context.Background(), recall.Request{Message: "banana release gate"})
	long, _ := e.Recall(context.Background(), recall.Request{Message: "banana release gate " + strings.Repeat("and some more context about the work ", 3)})
	if !strings.Contains(short.Block, "banana 1.4") {
		t.Errorf("short message lost the record: %q", short.Block)
	}
	if strings.Contains(long.Block, "banana 1.4") {
		t.Errorf("long message still recalls the record: %q", long.Block)
	}
}
