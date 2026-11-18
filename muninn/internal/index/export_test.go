package index

import "github.com/veridit/hug/muninn/internal/store"

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
