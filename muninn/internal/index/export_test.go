package index

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/veridit/hug/muninn/internal/store"
)

// ControlledGraphOrder injects the same synthetic schedule for strict result
// comparisons. This wrapper exists only in test builds, not the public API.
func ControlledGraphOrder(ix *Indexer) { ix.graphVisitOrder = sort.Ints }

// ProductionGraphOrder checks the actual graph loaded by this indexer, not
// merely a newly constructed Graph's defaults.
func ProductionGraphOrder(ix *Indexer) bool {
	g, err := ix.loadGraph()
	return err == nil && ix.graphVisitOrder == nil && g.visitOrder == nil
}

func TestGraphVisitOrderIsPerInstance(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "graph.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	plain := &Indexer{DB: db}
	controlled := &Indexer{DB: db}
	calls := 0
	controlled.graphVisitOrder = func(ids []int) { calls++; sort.Ints(ids) }
	for _, ix := range []*Indexer{plain, controlled, plain} {
		g, err := ix.loadGraph()
		if err != nil {
			t.Fatal(err)
		}
		g.Update([]UnitDelta{{New: []int{1, 2, 3, 4}}})
		if ix == plain {
			if g.visitOrder != nil || g.mapVisits == 0 {
				t.Fatalf("production nil path not exercised: nil=%v map visits=%d", g.visitOrder == nil, g.mapVisits)
			}
		} else if calls == 0 || g.mapVisits != 0 {
			t.Fatalf("controlled schedule unused or leaked: calls=%d map visits=%d", calls, g.mapVisits)
		}
	}
}

// GraphEdges is the persisted concept graph's edge weights, degrees and
// total weight, for tests that compare two indexes (4vd1mz).
func GraphEdges(db *store.DB) (map[int]map[int]float64, map[int]float64, float64, error) {
	g, err := (&Indexer{DB: db}).loadGraph()
	if err != nil {
		return nil, nil, 0, err
	}
	if err := g.loadAll(); err != nil {
		return nil, nil, 0, err
	}
	return g.Adj, g.K, g.M, nil
}

// GraphCommunities is the stored graph's communities and their totals.
func GraphCommunities(db *store.DB) (map[int]int, map[int]float64, error) {
	g, err := (&Indexer{DB: db}).loadGraph()
	if err != nil {
		return nil, nil, err
	}
	return g.Community, g.SumTot, nil
}

// LegacyGraph is the gob graph_state held before 4vd1mz, for the move test.
type LegacyGraph = legacyGraph

// PackAdj and UnpackAdj are the adjacency codec.
var PackAdj, UnpackAdj = packAdj, unpackAdj

// AfterPlan sets the hook that runs between a source file's plan and its
// writes (nil clears it).
func AfterPlan(f func(path string)) { afterPlan = f }
