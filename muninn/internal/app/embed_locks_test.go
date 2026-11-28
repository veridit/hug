package app_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veridit/hug/internal/flock"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/testutil"
)

// lockProbe embeds like FakeEmbedder and, on every call, checks from the
// outside that nobody holds the write turn or the SQLite write lock: it
// tries the turn's flock and BEGIN IMMEDIATE on a second connection, both
// without waiting. An embed under either lock is counted.
type lockProbe struct {
	testutil.FakeEmbedder
	db             *store.DB // a second connection to the store
	calls, underTx atomic.Int64
	underTurn      atomic.Int64
	firstUnderTx   atomic.Value
	firstUnderTurn atomic.Value
}

func (p *lockProbe) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	p.calls.Add(1)
	what := strings.Join(texts, " | ")
	if len(what) > 80 {
		what = what[:80]
	}
	if release, ok, err := flock.TryAcquire(p.db.Path + ".write.lock"); err == nil && ok {
		release()
	} else {
		p.underTurn.Add(1)
		p.firstUnderTurn.CompareAndSwap(nil, what)
	}
	if _, err := p.db.Exec(`BEGIN IMMEDIATE`); err == nil {
		p.db.Exec(`ROLLBACK`)
	} else {
		p.underTx.Add(1)
		p.firstUnderTx.CompareAndSwap(nil, what+": "+err.Error())
	}
	return p.FakeEmbedder.Embed(ctx, texts)
}

func newLockProbe(t *testing.T, a *app.App) *lockProbe {
	t.Helper()
	db, err := store.Open(a.DB.Path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`PRAGMA busy_timeout=0`); err != nil {
		t.Fatal(err)
	}
	p := &lockProbe{db: db}
	a.Embedder, a.Indexer.Embedder, a.Searcher.Embedder = p, p, p
	return p
}

// t0qap9 check 5: no embed call runs inside a write transaction or under
// the write turn. Every flow that embeds goes through the probe: a full
// index pass, a remember (its duplicate check and its new memory's
// vector), a remember that reinforces, a forget, and a write flow's own
// pass under the turn (FreshFor under the forget's turn).
func TestNoEmbedUnderWriteLocks(t *testing.T) {
	f := scopetest.New(t)
	a := f.App
	p := newLockProbe(t, a)
	ctx := context.Background()

	// A full pass embeds every unit of the fixture under the new pipeline.
	st, err := a.Reindex(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.Embedded == 0 {
		t.Fatalf("the pass embedded nothing: %+v", st)
	}
	b := f.Bind(t, "global")
	// A new memory: the duplicate check's query and the memory's vector.
	before, passes := p.calls.Load(), a.Indexer.Passes
	u, _, err := b.Remember(ctx, a, "Synthetic lockprobe walrus memory about kelp forests.", "fact", "global", nil)
	if err != nil {
		t.Fatal(err)
	}
	// One embed: the text before the turn serves the duplicate check and
	// is the new memory's vector (t0qap9 check 6: no pass, no re-embed).
	if got := p.calls.Load() - before; got != 1 {
		t.Fatalf("remember made %d embed calls, want 1", got)
	}
	if n := a.Indexer.Passes - passes; n != 0 {
		t.Fatalf("remember ran %d full index passes, want 0", n)
	}
	if !hasVector(t, a, u) {
		t.Fatalf("remembered memory %s has no vector after the remember", u.SID)
	}
	// The same text again: the check matches it, and reinforces.
	if _, re, err := b.Remember(ctx, a, "Synthetic lockprobe walrus memory about kelp forests.", "fact", "global", nil); err != nil || !re {
		t.Fatalf("second remember: reinforced=%v err=%v", re, err)
	}
	// A forget, and a pass under a write flow's own turn (FreshFor).
	if _, err := b.Forget(ctx, a, u.SID); err != nil {
		t.Fatal(err)
	}
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	mems = append(mems, ingest.NativeMemory{ID: "mem_lockprobe_fresh", Content: "Synthetic lockprobe narwhal note, written beside hug.", Category: "fact", Scope: "global", Active: true, Created: time.Now(), Updated: time.Now()})
	if err := ingest.SaveNative(app.NativePath(), mems); err != nil {
		t.Fatal(err)
	}
	turnCtx, release, err := a.WriteTurn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.FreshFor(turnCtx, []string{"mem_lockprobe_fresh"}); err != nil {
		release()
		t.Fatal(err)
	}
	release() // pays the pass's owed embedding, after the turn is gone
	fresh, err := a.DB.Resolve("mem_lockprobe_fresh")
	if err != nil {
		t.Fatal(err)
	}
	if !hasVector(t, a, fresh) {
		t.Fatal("a pass under the turn left its memory without a vector after the turn's release")
	}

	if n := p.underTx.Load(); n != 0 {
		t.Errorf("%d of %d embed calls ran inside a write transaction; first: %v", n, p.calls.Load(), p.firstUnderTx.Load())
	}
	if n := p.underTurn.Load(); n != 0 {
		t.Errorf("%d of %d embed calls ran under the write turn; first: %v", n, p.calls.Load(), p.firstUnderTurn.Load())
	}
	t.Logf("CHECK5 embed calls=%d under_tx=%d under_turn=%d", p.calls.Load(), p.underTx.Load(), p.underTurn.Load())
}

// The probe itself sees both locks: an embed made while a test holds the
// turn, and one made inside a transaction, are each counted.
func TestLockProbeSeesBothLocks(t *testing.T) {
	f := scopetest.New(t)
	p := newLockProbe(t, f.App)
	ctx, release, err := f.App.DB.WriteTurn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p.Embed(ctx, []string{"under the turn"})
	release()
	if _, err := f.App.DB.Exec(`BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}
	p.Embed(context.Background(), []string{"inside a transaction"})
	f.App.DB.Exec(`ROLLBACK`)
	if p.underTurn.Load() != 1 || p.underTx.Load() != 1 {
		t.Fatalf("probe counted turn=%d tx=%d, want 1 and 1", p.underTurn.Load(), p.underTx.Load())
	}
}

func hasVector(t *testing.T, a *app.App, u store.Unit) bool {
	t.Helper()
	var n int
	err := a.DB.QueryRow(`SELECT COUNT(*) FROM embeddings WHERE content_hash=? AND pipeline_hash=?`,
		u.ContentHash, index.PipelineHash(a.Embedder.Model())).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n == 1
}
