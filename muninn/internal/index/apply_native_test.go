package index_test

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/testutil"
)

// 4vd1mz check 2: after N targeted forgets (ApplyNative, no index pass) the
// index answers like the one an index pass over the same store keeps: the
// same units and active flags, unit concepts, neighbour lists (ids and
// sims), concept graph (edge weights, degrees, total) and, for a fixed
// query set under an identical injected visitation schedule, the same ordered
// results with every Unit and score field equal, including community-only hits
// and unrounded scores. No result order or candidate is normalized away.
// A separate actual-production-map-order arm checks structural invariants and
// per-unit community counts, not equality of independently random communities.
// Controlled scheduling does not prove equality of all production random outputs.
// A next pass on the targeted index finds nothing to do. Twins (5sc5rw)
// forget together.
func TestApplyNativeForgetsMatchIndexPass(t *testing.T) {
	t.Run("controlled_full_results", func(t *testing.T) { applyNativeForgetsMatchIndexPass(t, true) })
	t.Run("production_map_invariants", func(t *testing.T) { applyNativeForgetsMatchIndexPass(t, false) })
}

func applyNativeForgetsMatchIndexPass(t *testing.T, controlled bool) {
	ctx := context.Background()
	dir := t.TempDir()
	native := filepath.Join(dir, "native.json")
	r := rand.New(rand.NewSource(7))
	now := time.Now().UTC()
	var mems []ingest.NativeMemory
	for i := 0; i < 60; i++ {
		mems = append(mems, ingest.NativeMemory{ID: fmt.Sprintf("mem_%03d", i), Content: synthetic(r, i), Active: true, Scope: "global", Created: now, Updated: now})
	}
	// A twin: an imp_ copy a sync made of mem_000, which forget retires too.
	mems = append(mems, ingest.NativeMemory{ID: "imp_twin", Origin: "repo:demo/mem_000", Content: mems[0].Content, Active: true, Scope: "global", Created: now, Updated: now})
	if err := ingest.SaveNative(native, mems); err != nil {
		t.Fatal(err)
	}
	open := func(name string) (*store.DB, *index.Indexer) {
		db, err := store.Open(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		ix := &index.Indexer{DB: db, Embedder: &testutil.FakeEmbedder{Dim: 64}}
		if controlled {
			index.ControlledGraphOrder(ix)
		} else if !index.ProductionGraphOrder(ix) {
			t.Fatal("production arm did not load a nil-order graph")
		}
		if _, err := ix.Run(ctx, index.Options{NativePath: native}); err != nil {
			t.Fatal(err)
		}
		return db, ix
	}
	tdb, tix := open("targeted.db")
	pdb, pix := open("pass.db")
	forget := []string{"mem_000", "mem_017", "mem_031", "mem_044", "mem_052", "mem_005", "mem_029"}
	for round, id := range forget {
		f, err := ingest.ForgetManyChanged(native, []string{id})
		if err != nil || !f.Done[id] {
			t.Fatalf("forget %s: %+v %v", id, f, err)
		}
		if err := tix.ApplyNative(ctx, native, f.Before, f.After, f.Changed); err != nil {
			t.Fatalf("round %d: ApplyNative: %v", round, err)
		}
		if _, err := pix.Run(ctx, index.Options{NativePath: native}); err != nil {
			t.Fatal(err)
		}
		if id == "mem_000" && len(f.Changed) != 2 {
			t.Fatalf("forgetting mem_000 changed %d memories, want it and its twin", len(f.Changed))
		}
		compareIndexes(t, round, tdb, pdb, controlled)
	}
	if tix.Passes != 1 {
		t.Fatalf("targeted index ran %d passes, want only its first", tix.Passes)
	}
	st, err := tix.Run(ctx, index.Options{NativePath: native})
	if err != nil {
		t.Fatal(err)
	}
	if st.SourcesScanned != 0 || st.Changed+st.Added+st.Removed != 0 {
		t.Fatalf("a pass after the targeted forgets still found work: %s", st)
	}
	compareIndexes(t, len(forget), tdb, pdb, controlled)
	// A file another process changed since the index saw it is not the
	// targeted update's to fix: it refuses and changes nothing.
	f, err := ingest.ForgetManyChanged(native, []string{"mem_010"})
	if err != nil {
		t.Fatal(err)
	}
	stale := f.Before
	stale.Size++
	if err := tix.ApplyNative(ctx, native, stale, f.After, f.Changed); err != index.ErrNotCurrent {
		t.Fatalf("stale Before: %v, want ErrNotCurrent", err)
	}
	var active int
	tdb.QueryRow(`SELECT active FROM units WHERE key='mem_010'`).Scan(&active)
	if active != 1 {
		t.Fatalf("a refused ApplyNative changed the unit")
	}
}

// compareIndexes fails when the two indexes answer differently.
func compareIndexes(t *testing.T, round int, a, b *store.DB, controlled bool) {
	t.Helper()
	for _, q := range []struct{ name, sql string }{
		{"units", `SELECT u.key || ' ' || u.active || ' ' || u.ident FROM units u ORDER BY u.key`},
		{"concepts", `SELECT u.key || ' ' || c.term FROM unit_concepts uc JOIN units u ON u.id=uc.unit_id JOIN concepts c ON c.id=uc.concept_id ORDER BY 1`},
		{"neighbors", `SELECT u.key || ' ' || v.key || ' ' || printf('%.5f', n.sim) FROM neighbors n JOIN units u ON u.id=n.unit_id JOIN units v ON v.id=n.nbr_id ORDER BY 1`},
	} {
		if ga, gb := dumpRows(t, a, q.sql), dumpRows(t, b, q.sql); ga != gb {
			t.Fatalf("round %d: %s differ\ntargeted:\n%s\npass:\n%s", round, q.name, diffLines(ga, gb), diffLines(gb, ga))
		}
	}
	adjA, kA, mA, err := index.GraphEdges(a)
	if err != nil {
		t.Fatal(err)
	}
	adjB, kB, mB, err := index.GraphEdges(b)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(adjA) != fmt.Sprint(adjB) || fmt.Sprint(kA) != fmt.Sprint(kB) || mA != mB {
		t.Fatalf("round %d: concept graphs differ (M %v vs %v)", round, mA, mB)
	}
	checkUnitCommunities(t, a, round)
	checkUnitCommunities(t, b, round)
	if !controlled {
		return // actual nil-order production arm: structural invariants only
	}
	sa := &search.Searcher{DB: a, Embedder: &testutil.FakeEmbedder{Dim: 64}}
	sb := &search.Searcher{DB: b, Embedder: &testutil.FakeEmbedder{Dim: 64}}
	o := search.DefaultOptions()
	o.Limit, o.MinScore = 100, 0
	for _, q := range []string{"quartz policy", "deploy staging", "banana cake", "note", "synthetic release window"} {
		ra, err := sa.Search(context.Background(), q, o)
		if err != nil {
			t.Fatal(err)
		}
		rb, err := sb.Search(context.Background(), q, o)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ra, rb) {
			t.Fatalf("round %d: search %q differs\ntargeted %#v\npass     %#v", round, q, ra, rb)
		}
	}
}

func dumpRows(t *testing.T, db *store.DB, q string) string {
	t.Helper()
	rows, err := db.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return strings.Join(out, "\n")
}

// diffLines is the lines of a that b lacks.
func diffLines(a, b string) string {
	have := map[string]bool{}
	for _, l := range strings.Split(b, "\n") {
		have[l] = true
	}
	var out []string
	for _, l := range strings.Split(a, "\n") {
		if !have[l] {
			out = append(out, l)
		}
	}
	sort.Strings(out)
	return strings.Join(out, "\n")
}
