package index

import "sort"

// Graph is the concept co-occurrence graph with Louvain communities, updated
// incrementally with DF-Louvain (arXiv:2404.19634, Algorithm 1) exactly as in
// seid search: apply the batch edge delta, seed a frontier from structural
// changes, and run a frontier-restricted move loop. Only the neighbourhood of
// what changed is reconsidered; nothing is recomputed from scratch.
//
// Every weight is a count (each unit adds or removes 1 per pair), so K, M
// and SumTot are exact integers in float64 and their sums do not depend on
// order. Vertices and candidate communities are visited in id order, so one
// graph and one batch always give the same communities (4vd1mz: Go map order
// made two identical updates label communities differently).
type Graph struct {
	// Adj holds the adjacency of the vertices loaded so far; adj(v) loads a
	// stored vertex's on first use (all of them, for a graph built in memory).
	Adj       map[int]map[int]float64 // symmetric weights
	K         map[int]float64         // weighted degree
	Community map[int]int
	SumTot    map[int]float64
	M         float64 // total edge weight

	load    func(v int) (map[int]float64, error) // nil: Adj is complete
	err     error                                // first load error
	dirty   map[int]bool                         // K, community or adjacency changed
	comm0   map[int]int                          // communities as loaded
	sumTot0 map[int]float64                      // community totals as loaded
	// Loads counts adjacencies read from storage (4vd1mz's counting test).
	Loads int
}

// NewGraph returns an empty graph.
func NewGraph() *Graph {
	return &Graph{Adj: map[int]map[int]float64{}, K: map[int]float64{}, Community: map[int]int{},
		SumTot: map[int]float64{}, dirty: map[int]bool{}}
}

// adj is v's adjacency, loaded from storage the first time it is needed.
func (g *Graph) adj(v int) map[int]float64 {
	if m, ok := g.Adj[v]; ok {
		return m
	}
	var m map[int]float64
	if _, stored := g.Community[v]; stored && g.load != nil {
		g.Loads++
		var err error
		if m, err = g.load(v); err != nil && g.err == nil {
			g.err = err
		}
	}
	if m == nil {
		m = map[int]float64{}
	}
	g.Adj[v] = m
	return m
}

// sortedKeys returns m's keys in increasing order.
func sortedKeys[V any](m map[int]V) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// UnitDelta is one unit's concept set before and after a change.
// A new unit is (nil, new); a removed unit is (old, nil).
type UnitDelta struct{ Old, New []int }

type edge struct{ a, b int }

func pair(a, b int) edge {
	if a > b {
		a, b = b, a
	}
	return edge{a, b}
}

// cliqueEdges returns every pair within a concept set (a unit is a clique).
func cliqueEdges(cs []int) map[edge]bool {
	out := map[edge]bool{}
	for i := 0; i < len(cs); i++ {
		for j := i + 1; j < len(cs); j++ {
			if cs[i] != cs[j] {
				out[pair(cs[i], cs[j])] = true
			}
		}
	}
	return out
}

// MaxConceptsPerUnit caps clique size so one huge unit cannot add O(n^2)
// edges (seid search skips Louvain past a size threshold for the same reason).
const MaxConceptsPerUnit = 48

func capped(cs []int) []int {
	if len(cs) > MaxConceptsPerUnit {
		return cs[:MaxConceptsPerUnit]
	}
	return cs
}

func (g *Graph) ensure(v int) (isNew bool) {
	if _, ok := g.Community[v]; ok {
		return false
	}
	g.Community[v] = v
	if g.Adj[v] == nil {
		g.Adj[v] = map[int]float64{}
	}
	g.SumTot[v] += g.K[v]
	g.dirty[v] = true
	return true
}

// Update applies unit deltas and re-optimises communities on the frontier.
// Returns (waves, moves).
func (g *Graph) Update(deltas []UnitDelta) (int, int) {
	// 7. Batch edge delta: each unit contributes +-1 to every pair in its clique.
	ed := map[edge]float64{}
	newVerts := map[int]bool{}
	for _, d := range deltas {
		oldE, newE := cliqueEdges(capped(d.Old)), cliqueEdges(capped(d.New))
		for e := range newE {
			if !oldE[e] {
				ed[e]++
			}
		}
		for e := range oldE {
			if !newE[e] {
				ed[e]--
			}
		}
		for _, v := range d.New {
			if g.ensure(v) {
				newVerts[v] = true
			}
		}
	}
	// 8. Apply delta; track structural insertions and deletions.
	var added, removed []edge
	for e, dw := range ed {
		if dw > -1e-12 && dw < 1e-12 {
			continue
		}
		g.ensure(e.a)
		g.ensure(e.b)
		adjA, adjB := g.adj(e.a), g.adj(e.b)
		g.dirty[e.a], g.dirty[e.b] = true, true
		oldW := adjA[e.b]
		newW := oldW + dw
		g.K[e.a] += dw
		g.K[e.b] += dw
		g.M += dw
		ca, cb := g.Community[e.a], g.Community[e.b]
		g.SumTot[ca] += dw
		g.SumTot[cb] += dw
		if newW <= 1e-9 {
			delete(adjA, e.b)
			delete(adjB, e.a)
			if oldW > 1e-9 {
				removed = append(removed, e)
			}
		} else {
			adjA[e.b] = newW
			adjB[e.a] = newW
			if oldW <= 1e-9 {
				added = append(added, e)
			}
		}
	}
	// 9. Seed frontier: within-community deletion or cross-community insertion
	// marks the source; brand-new vertices are always visited.
	frontier := map[int]bool{}
	for _, e := range removed {
		if g.Community[e.a] == g.Community[e.b] {
			frontier[e.a] = true
		}
	}
	for _, e := range added {
		if g.Community[e.a] != g.Community[e.b] {
			frontier[e.a] = true
		}
	}
	for v := range newVerts {
		frontier[v] = true
	}
	return g.moveLoop(frontier)
}

// moveLoop is the frontier-restricted Louvain local-moving phase: at most 20
// waves, stop on a zero-move wave or when a wave's total gain drops below
// tolerance. Note: seid search compares the gain against 1e-2*m, but gains
// are modularity deltas (already normalised by 2m), so that floor grows with
// total weight and ends the loop after one wave on real graphs. muninn
// compares against tau directly, as in the paper.
func (g *Graph) moveLoop(frontier map[int]bool) (int, int) {
	const maxWaves, tolerance = 20, 1e-2
	if g.M < 1e-9 || len(frontier) == 0 {
		return 0, 0
	}
	m2 := 2 * g.M
	floor := tolerance
	waves, moves := 0, 0
	for len(frontier) > 0 && waves < maxWaves {
		waves++
		current := sortedKeys(frontier)
		frontier = map[int]bool{}
		waveGain, waveMoves := 0.0, 0
		for _, i := range current {
			adjI := g.adj(i) // loads K[i] too
			ci := g.Community[i]
			ki := g.K[i]
			// Weight from i to each neighbouring community (ci included).
			ncomms := map[int]float64{ci: 0}
			for j, w := range adjI {
				if j != i {
					ncomms[g.Community[j]] += w
				}
			}
			// Take i out of ci, then evaluate every candidate with the standard
			// Blondel gain dQ(c) = k_i,c/m - SumTot_c*k_i/(2m^2). seid search
			// used k_i,c/(2m) for the first term, which halves the pull of a
			// community and leaves new concepts stranded alone.
			g.SumTot[ci] -= ki
			gainOf := func(c int) float64 { return ncomms[c]/g.M - g.SumTot[c]*ki/(m2*g.M) }
			stay := gainOf(ci)
			bestC, bestGain := ci, 0.0
			for _, c := range sortedKeys(ncomms) {
				if c == ci {
					continue
				}
				if d := gainOf(c) - stay; d > bestGain+1e-15 || (d > 0 && d >= bestGain-1e-15 && c < bestC && bestC != ci) {
					bestGain, bestC = d, c
				}
			}
			g.SumTot[bestC] += ki
			g.Community[i] = bestC
			if bestC != ci {
				g.dirty[i] = true
				moves++
				waveMoves++
				waveGain += bestGain
				for j := range adjI { // onChange(i): expand to neighbours
					if j != i {
						frontier[j] = true
					}
				}
			}
		}
		if waveMoves == 0 || waveGain < floor {
			break
		}
	}
	return waves, moves
}

// CommunitySize returns the number of concepts in each community.
func (g *Graph) CommunitySize() map[int]int {
	out := map[int]int{}
	for _, c := range g.Community {
		out[c]++
	}
	return out
}
