package index_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/testutil"
)

// graphFixture indexes n synthetic memories into dir/name and returns the
// store file and the index.
func graphFixture(t *testing.T, dir, name string, n int) (string, *store.DB, *index.Indexer) {
	t.Helper()
	native := filepath.Join(dir, "native.json")
	if _, err := os.Stat(native); err != nil {
		r := rand.New(rand.NewSource(11))
		now := time.Now().UTC()
		var mems []ingest.NativeMemory
		for i := 0; i < n; i++ {
			mems = append(mems, ingest.NativeMemory{ID: fmt.Sprintf("mem_%03d", i), Content: synthetic(r, i), Active: true, Scope: "global", Created: now, Updated: now})
		}
		if err := ingest.SaveNative(native, mems); err != nil {
			t.Fatal(err)
		}
	}
	db, err := store.Open(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ix := &index.Indexer{DB: db, Embedder: &testutil.FakeEmbedder{Dim: 64}}
	if _, err := ix.Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	if !index.ProductionGraphOrder(ix) {
		t.Fatal("graph fixture must exercise production nil visitation order")
	}
	return native, db, ix
}

// asLegacy turns db's graph back into the gob store hug kept before 4vd1mz:
// a graph_state table holding the whole graph, no graph tables.
func asLegacy(t *testing.T, db *store.DB) index.LegacyGraph {
	t.Helper()
	adj, k, m, err := index.GraphEdges(db)
	if err != nil {
		t.Fatal(err)
	}
	comm, sumTot, err := index.GraphCommunities(db)
	if err != nil {
		t.Fatal(err)
	}
	old := index.LegacyGraph{Adj: adj, K: k, Community: comm, SumTot: sumTot, M: m}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(old); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`DROP VIEW graph_state`, `DELETE FROM graph_vertex`, `DELETE FROM graph_community`, `DELETE FROM meta WHERE key='graph_m'`,
		`CREATE TABLE graph_state (id INTEGER PRIMARY KEY CHECK (id = 1), state BLOB NOT NULL)`} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO graph_state(id, state) VALUES(1, ?)`, buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	return old
}

// 4vd1mz, crab 04:44: a store whose graph is still the gob moves to the
// tables on first use, unchanged: every edge, degree, community and total,
// and so every search answer. graph_state is gone (an empty view a hug from
// before refuses to write), one line goes to hook.log, and meta graph_moved
// says when.
func TestGobGraphMovesUnchanged(t *testing.T) {
	dir := t.TempDir()
	_, db, _ := graphFixture(t, dir, "m.db", 80)
	before := searchDump(t, db)
	old := asLegacy(t, db)
	ix := &index.Indexer{DB: db}
	if !ix.GraphNeedsMove() {
		t.Fatal("a store with graph_state as a table is not reported as needing the move")
	}
	moved, err := ix.MoveGraph()
	if err != nil || !moved {
		t.Fatalf("MoveGraph: moved %v, %v", moved, err)
	}
	if ix.GraphNeedsMove() {
		t.Fatal("still needs the move after MoveGraph")
	}
	adj, k, m, err := index.GraphEdges(db)
	if err != nil {
		t.Fatal(err)
	}
	comm, sumTot, err := index.GraphCommunities(db)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(adj) != fmt.Sprint(old.Adj) || fmt.Sprint(k) != fmt.Sprint(old.K) || m != old.M {
		t.Fatalf("edges, degrees or total weight changed in the move (M %v, was %v)", m, old.M)
	}
	if fmt.Sprint(comm) != fmt.Sprint(old.Community) || fmt.Sprint(sumTot) != fmt.Sprint(old.SumTot) {
		t.Fatal("communities or their totals changed in the move")
	}
	if after := searchDump(t, db); after != before {
		t.Fatalf("search answers changed in the move:\n%s", diffLines(after, before))
	}
	checkUnitCommunities(t, db, 0)
	if _, err := db.Exec(`INSERT INTO graph_state(id, state) VALUES(1, x'00')`); err == nil || !strings.Contains(err.Error(), "view") {
		t.Fatalf("an old hug could still write graph_state: %v", err)
	}
	logged, _ := os.ReadFile(filepath.Join(dir, "hook.log"))
	if want := fmt.Sprintf("concept graph moved to tables, %d vertices, ", len(old.Community)); !strings.Contains(string(logged), want) {
		t.Fatalf("hook.log has no %q line:\n%s", want, logged)
	}
	if !strings.Contains(db.GetMeta("graph_moved"), "concept graph moved to tables") {
		t.Fatalf("meta graph_moved = %q", db.GetMeta("graph_moved"))
	}
	if moved, err := ix.MoveGraph(); moved || err != nil {
		t.Fatalf("a second MoveGraph moved %v, %v", moved, err)
	}
}

// The first index pass on a gob store moves it inside the pass: no
// separate step, and the pass's own update then runs on the tables.
func TestIndexPassMovesGobGraph(t *testing.T) {
	dir := t.TempDir()
	native, db, _ := graphFixture(t, dir, "p.db", 40)
	asLegacy(t, db)
	f, err := ingest.ForgetManyChanged(native, []string{"mem_007"})
	if err != nil || !f.Done["mem_007"] {
		t.Fatalf("forget: %+v %v", f, err)
	}
	ix := &index.Indexer{DB: db, Embedder: &testutil.FakeEmbedder{Dim: 64}}
	if _, err := ix.Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	if ix.GraphNeedsMove() || !strings.HasPrefix(ix.GraphMoved, "concept graph moved to tables") {
		t.Fatalf("the pass did not move the graph (moved %q)", ix.GraphMoved)
	}
	_, ref, _ := graphFixture(t, dir, "ref.db", 40)
	adjA, kA, mA, _ := index.GraphEdges(db)
	adjB, kB, mB, _ := index.GraphEdges(ref)
	if fmt.Sprint(adjA) != fmt.Sprint(adjB) || fmt.Sprint(kA) != fmt.Sprint(kB) || mA != mB {
		t.Fatal("after the move and a forget, the graph differs from a fresh index of the same store")
	}
	checkUnitCommunities(t, db, 0)
}

// A later failure rolls the whole pass back, including the graph move. A
// receipt must not claim a move that did not commit, and a retry logs once.
func TestRolledBackGraphMoveHasNoReceipt(t *testing.T) {
	dir := t.TempDir()
	native, db, _ := graphFixture(t, dir, "rollback.db", 12)
	asLegacy(t, db)
	if err := db.SetMeta("source_recipe", "force-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_after_move BEFORE UPDATE ON sources BEGIN SELECT RAISE(ABORT, 'after graph move'); END`); err != nil {
		t.Fatal(err)
	}
	ix := &index.Indexer{DB: db, Embedder: &testutil.FakeEmbedder{Dim: 64}}
	if _, err := ix.Run(context.Background(), index.Options{NativePath: native}); err == nil || !strings.Contains(err.Error(), "after graph move") {
		t.Fatalf("pass did not fail after moving the graph: %v", err)
	}
	if !ix.GraphNeedsMove() || db.GetMeta("graph_moved") != "" || ix.GraphMoved != "" {
		t.Errorf("rolled-back move left a receipt: needs move %v, meta %q, indexer %q", ix.GraphNeedsMove(), db.GetMeta("graph_moved"), ix.GraphMoved)
	}
	logPath := filepath.Join(dir, "hook.log")
	if b, err := os.ReadFile(logPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	} else if len(b) != 0 {
		t.Errorf("rolled-back move was logged: %s", b)
	}
	if _, err := db.Exec(`DROP TRIGGER fail_after_move`); err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	if ix.GraphNeedsMove() || ix.GraphMoved == "" {
		t.Fatal("retry did not commit the graph move")
	}
	if b, err := os.ReadFile(logPath); err != nil || strings.Count(string(b), "concept graph moved to tables") != 1 {
		t.Fatalf("want exactly one committed move log: %s, %v", b, err)
	}
}

// A forget loads the adjacency of the concepts it touches and of the
// frontier Louvain visits, never the whole graph (4vd1mz: the gob decode
// was 0.5 s and ~500 MB per forget on Jørgen's store). Counted, not timed.
func TestForgetLoadsOnlyTouchedVertices(t *testing.T) {
	dir := t.TempDir()
	native := filepath.Join(dir, "native.json")
	// 2,000 made-up words, 10 a memory: a sparse graph like a real store's.
	r := rand.New(rand.NewSource(5))
	word := func() string {
		b := make([]byte, 7)
		for i := range b {
			b[i] = byte('a' + r.Intn(26))
		}
		return string(b)
	}
	pool := make([]string, 2000)
	for i := range pool {
		pool[i] = word()
	}
	now := time.Now().UTC()
	var mems []ingest.NativeMemory
	for i := 0; i < 300; i++ {
		var ws []string
		for j := 0; j < 10; j++ {
			ws = append(ws, pool[r.Intn(len(pool))])
		}
		mems = append(mems, ingest.NativeMemory{ID: fmt.Sprintf("mem_%03d", i), Content: strings.Join(ws, " ") + ".", Active: true, Scope: "global", Created: now, Updated: now})
	}
	if err := ingest.SaveNative(native, mems); err != nil {
		t.Fatal(err)
	}
	_, db, ix := graphFixture(t, dir, "l.db", 0)
	var vertices, touched int
	db.QueryRow(`SELECT COUNT(*) FROM graph_vertex`).Scan(&vertices)
	db.QueryRow(`SELECT COUNT(*) FROM unit_concepts uc JOIN units u ON u.id=uc.unit_id WHERE u.key='mem_123'`).Scan(&touched)
	f, err := ingest.ForgetManyChanged(native, []string{"mem_123"})
	if err != nil || !f.Done["mem_123"] {
		t.Fatalf("forget: %+v %v", f, err)
	}
	ix.GraphLoads = 0
	if err := ix.ApplyNative(context.Background(), native, f.Before, f.After, f.Changed); err != nil {
		t.Fatal(err)
	}
	// Its own concepts, and the neighbours of any that change community:
	// a tenth of the graph is far more than that.
	if ix.GraphLoads == 0 || ix.GraphLoads > vertices/10 {
		t.Fatalf("one forget loaded %d of %d vertices (the memory has %d concepts)", ix.GraphLoads, vertices, touched)
	}
	t.Logf("one forget loaded %d of %d vertices (the memory has %d concepts)", ix.GraphLoads, vertices, touched)
}

func TestAdjacencyCodecRoundTrips(t *testing.T) {
	m := map[int]float64{3: 1, 4: 7, 900: 2, 1 << 30: 1 << 40}
	b, err := index.PackAdj(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := index.UnpackAdj(b)
	if err != nil || fmt.Sprint(got) != fmt.Sprint(m) {
		t.Fatalf("round trip: %v %v", got, err)
	}
	if _, err := index.PackAdj(map[int]float64{1: 0.5}); err == nil {
		t.Fatal("a weight that is no count was packed")
	}
}

// searchDump is the answers to a fixed query set, scores included.
func searchDump(t *testing.T, db *store.DB) string {
	t.Helper()
	var b strings.Builder
	for _, q := range []string{"quartz policy", "deploy staging", "banana cake", "note", "synthetic release window"} {
		b.WriteString(q + ":\n" + searchLines(t, db, q))
	}
	return b.String()
}

// searchLines is one query's answers in order, every score included.
func searchLines(t *testing.T, db *store.DB, q string) string {
	t.Helper()
	s := &search.Searcher{DB: db, Embedder: &testutil.FakeEmbedder{Dim: 64}}
	o := search.DefaultOptions()
	o.Limit, o.MinScore = 100, 0
	rs, err := s.Search(context.Background(), q, o)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, r := range rs {
		fmt.Fprintf(&b, "%s %s\n", r.Unit.Key, r.Explain())
	}
	return b.String()
}

// oldHugIndexes runs what every hug from v0.7.2 to v0.7.58 does to the
// graph in an index pass (pipeline.go loadGraph, then saveGraph): it reads
// graph_state, then upserts it. It returns the first error, as the old
// code returns it to the CLI and to MCP memory_remember.
func oldHugIndexes(db *store.DB) error {
	var blob []byte
	err := db.QueryRow(`SELECT state FROM graph_state WHERE id=1`).Scan(&blob)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	_, err = db.Exec(`INSERT INTO graph_state(id, state) VALUES(1, ?) ON CONFLICT(id) DO UPDATE SET state=excluded.state`, []byte{1})
	return err
}

// mbdfzs check 3 (crab 19:52Z): an MCP server started before v0.7.54 never
// reloads, and from v0.7.59 its memory_remember failed with "cannot modify
// graph_state because it is a view", which names no fix. Its read comes
// first, so the view now fails that read with the fix: restart this server.
// Fresh stores, moved stores and stores with the plain view all say it.
func TestOldHugIsToldToRestart(t *testing.T) {
	const want = "hug: this hug mcp server is older than v0.7.59 and cannot write this memory store; fix: restart it"
	dir := t.TempDir()
	_, fresh, _ := graphFixture(t, dir, "fresh.db", 20)
	_, moved, _ := graphFixture(t, dir, "moved.db", 20)
	asLegacy(t, moved)
	if _, err := (&index.Indexer{DB: moved}).MoveGraph(); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(dir, "plain.db")
	_, p, _ := graphFixture(t, dir, "plain.db", 20)
	for _, q := range []string{`DROP VIEW graph_state`, `CREATE VIEW graph_state(id, state) AS SELECT 1, NULL WHERE 0`} {
		if _, err := p.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	p.Close()
	reopened, err := store.Open(plain)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	for name, db := range map[string]*store.DB{"fresh": fresh, "moved": moved, "plain view, reopened": reopened} {
		err := oldHugIndexes(db)
		if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "mcp action=reload server=hug") {
			t.Errorf("%s store: an old hug's index pass says %v, want the fix %q", name, err, want)
		}
		if (&index.Indexer{DB: db}).GraphNeedsMove() {
			t.Errorf("%s store: reported as needing the gob move", name)
		}
	}
}
