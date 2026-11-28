package index

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/store"
)

// The concept graph lives in tables (4vd1mz): communities (concept ->
// community, which search reads too), graph_community (each community's
// total degree), graph_vertex (each concept's degree and packed adjacency)
// and meta graph_m (total weight). An update loads the communities and
// their totals, and the adjacency of only the vertices it touches; it
// writes back only what changed. It used to decode and re-encode one gob of
// the whole graph (30.7 MB, 5,027,834 directed edges on Jørgen's store) on
// every change: 0.5 s and ~500 MB per forget.

// loadGraph opens the stored graph; adjacency loads per vertex on use.
func (ix *Indexer) loadGraph() (*Graph, error) {
	if err := ix.moveGraph(); err != nil {
		return nil, err
	}
	g := NewGraph()
	g.visitOrder = ix.graphVisitOrder
	g.M, _ = strconv.ParseFloat(ix.DB.GetMeta("graph_m"), 64)
	rows, err := ix.DB.Query(`SELECT concept_id, community_id FROM communities`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var v, c int
		if err := rows.Scan(&v, &c); err != nil {
			rows.Close()
			return nil, err
		}
		g.Community[v] = c
	}
	rows.Close()
	if rows, err = ix.DB.Query(`SELECT community_id, sum_tot FROM graph_community`); err != nil {
		return nil, err
	}
	for rows.Next() {
		var c int
		var s float64
		if err := rows.Scan(&c, &s); err != nil {
			rows.Close()
			return nil, err
		}
		g.SumTot[c] = s
	}
	rows.Close()
	g.comm0 = make(map[int]int, len(g.Community))
	for v, c := range g.Community {
		g.comm0[v] = c
	}
	g.sumTot0 = make(map[int]float64, len(g.SumTot))
	for c, s := range g.SumTot {
		g.sumTot0[c] = s
	}
	g.load = func(v int) (map[int]float64, error) {
		var k float64
		var blob []byte
		err := ix.DB.QueryRow(`SELECT k, adj FROM graph_vertex WHERE concept_id=?`, v).Scan(&k, &blob)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		g.K[v] = k
		return unpackAdj(blob)
	}
	g.Loads = 0
	return g, nil
}

// loadAll loads every vertex's adjacency (tests compare whole graphs).
func (g *Graph) loadAll() error {
	for _, v := range sortedKeys(g.Community) {
		g.adj(v)
	}
	return g.err
}

// saveGraph writes back what the update changed: vertices whose degree or
// adjacency changed, communities whose total changed, concepts that moved
// community (with the per-community counts of their units, and of units).
func (ix *Indexer) saveGraph(g *Graph, units map[int64]bool) error {
	if g.err != nil {
		return g.err
	}
	for _, v := range sortedKeys(g.dirty) {
		blob, err := packAdj(g.Adj[v])
		if err != nil {
			return fmt.Errorf("concept %d: %w", v, err)
		}
		if _, err := ix.DB.Exec(`INSERT INTO graph_vertex(concept_id, k, adj) VALUES(?,?,?)
		  ON CONFLICT(concept_id) DO UPDATE SET k=excluded.k, adj=excluded.adj`, v, g.K[v], blob); err != nil {
			return err
		}
	}
	for _, c := range sortedKeys(g.SumTot) {
		if old, ok := g.sumTot0[c]; ok && old == g.SumTot[c] {
			continue
		}
		if _, err := ix.DB.Exec(`INSERT INTO graph_community(community_id, sum_tot) VALUES(?,?)
		  ON CONFLICT(community_id) DO UPDATE SET sum_tot=excluded.sum_tot`, c, g.SumTot[c]); err != nil {
			return err
		}
	}
	if err := ix.DB.SetMeta("graph_m", strconv.FormatFloat(g.M, 'g', -1, 64)); err != nil {
		return err
	}
	var moved []string
	for _, v := range sortedKeys(g.dirty) { // a move or a new vertex marks it dirty
		if old, ok := g.comm0[v]; ok && old == g.Community[v] {
			continue
		}
		if _, err := ix.DB.Exec(`INSERT INTO communities(concept_id, community_id) VALUES(?,?)
		  ON CONFLICT(concept_id) DO UPDATE SET community_id=excluded.community_id`, v, g.Community[v]); err != nil {
			return err
		}
		moved = append(moved, strconv.Itoa(v))
	}
	for start := 0; start < len(moved); start += 500 {
		rows, err := ix.DB.Query(`SELECT DISTINCT unit_id FROM unit_concepts WHERE concept_id IN (` +
			strings.Join(moved[start:min(start+500, len(moved))], ",") + `)`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var u int64
			rows.Scan(&u)
			units[u] = true
		}
		rows.Close()
	}
	return ix.refreshUnitCommunities(units)
}

// packAdj encodes an adjacency as uvarint pairs (id gap, weight), ids in
// increasing order. Weights are counts, so they are whole numbers.
func packAdj(m map[int]float64) ([]byte, error) {
	buf := make([]byte, 0, len(m)*3)
	prev := 0
	for _, j := range sortedKeys(m) {
		w := m[j]
		if w < 1 || w != math.Trunc(w) || w > 1<<52 {
			return nil, fmt.Errorf("edge weight %v to %d is not a count", w, j)
		}
		buf = binary.AppendUvarint(buf, uint64(j-prev))
		buf = binary.AppendUvarint(buf, uint64(w))
		prev = j
	}
	return buf, nil
}

func unpackAdj(b []byte) (map[int]float64, error) {
	m := map[int]float64{}
	prev := 0
	for len(b) > 0 {
		gap, n := binary.Uvarint(b)
		if n <= 0 {
			return nil, errors.New("corrupt adjacency")
		}
		b = b[n:]
		w, n := binary.Uvarint(b)
		if n <= 0 {
			return nil, errors.New("corrupt adjacency")
		}
		b = b[n:]
		prev += int(gap)
		m[prev] = float64(w)
	}
	return m, nil
}

// legacyGraph is the gob hug kept in graph_state before 4vd1mz.
type legacyGraph struct {
	Adj       map[int]map[int]float64
	K         map[int]float64
	Community map[int]int
	SumTot    map[int]float64
	M         float64
}

// GraphNeedsMove reports a store whose concept graph is still the gob.
func (ix *Indexer) GraphNeedsMove() bool {
	var typ string
	ix.DB.QueryRow(`SELECT type FROM sqlite_master WHERE name='graph_state'`).Scan(&typ)
	return typ == "table"
}

// moveGraph moves a gob graph into the tables, once: the first writer that
// finds one (an index pass, a forget, hug mem upgrade) does it inside its
// own transaction, so it holds the index lock. The graph is copied as it
// is, communities included, so recall does not change. graph_state becomes
// an empty view: a hug from before 4vd1mz that opens this store then fails
// to write it ("cannot modify graph_state because it is a view") instead of
// saving a graph built from nothing. One line goes to hook.log and to meta
// graph_moved (hug mem stats).
func (ix *Indexer) moveGraph() error {
	if !ix.GraphNeedsMove() {
		return nil
	}
	t0 := time.Now()
	var old legacyGraph
	var blob []byte
	err := ix.DB.QueryRow(`SELECT state FROM graph_state WHERE id=1`).Scan(&blob)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		old = legacyGraph{}
	case err != nil:
		return err
	default:
		if err := gob.NewDecoder(bytes.NewReader(blob)).Decode(&old); err != nil {
			// The old code started over from an empty graph here.
			old = legacyGraph{}
		}
	}
	for _, q := range []string{`DELETE FROM graph_vertex`, `DELETE FROM graph_community`} {
		if _, err := ix.DB.Exec(q); err != nil {
			return err
		}
	}
	for _, v := range sortedKeys(old.Adj) {
		b, err := packAdj(old.Adj[v])
		if err != nil {
			return fmt.Errorf("move concept graph: concept %d: %w", v, err)
		}
		if _, err := ix.DB.Exec(`INSERT INTO graph_vertex(concept_id, k, adj) VALUES(?,?,?)`, v, old.K[v], b); err != nil {
			return err
		}
	}
	for _, c := range sortedKeys(old.SumTot) {
		if _, err := ix.DB.Exec(`INSERT INTO graph_community(community_id, sum_tot) VALUES(?,?)`, c, old.SumTot[c]); err != nil {
			return err
		}
	}
	// The communities table is the graph's: concepts the gob placed
	// differently (none on a store the old code kept consistent) move, and
	// their units' counts follow.
	units := map[int64]bool{}
	var moved []string
	for _, v := range sortedKeys(old.Community) {
		var have int
		e := ix.DB.QueryRow(`SELECT community_id FROM communities WHERE concept_id=?`, v).Scan(&have)
		if e == nil && have == old.Community[v] {
			continue
		}
		if _, err := ix.DB.Exec(`INSERT INTO communities(concept_id, community_id) VALUES(?,?)
		  ON CONFLICT(concept_id) DO UPDATE SET community_id=excluded.community_id`, v, old.Community[v]); err != nil {
			return err
		}
		moved = append(moved, strconv.Itoa(v))
	}
	for start := 0; start < len(moved); start += 500 {
		rows, err := ix.DB.Query(`SELECT DISTINCT unit_id FROM unit_concepts WHERE concept_id IN (` +
			strings.Join(moved[start:min(start+500, len(moved))], ",") + `)`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var u int64
			rows.Scan(&u)
			units[u] = true
		}
		rows.Close()
	}
	if err := ix.refreshUnitCommunities(units); err != nil {
		return err
	}
	if err := ix.DB.SetMeta("graph_m", strconv.FormatFloat(old.M, 'g', -1, 64)); err != nil {
		return err
	}
	if _, err := ix.DB.Exec(`DROP TABLE graph_state`); err != nil {
		return err
	}
	if _, err := ix.DB.Exec(store.OldHugView); err != nil {
		return err
	}
	msg := fmt.Sprintf("concept graph moved to tables, %d vertices, %.1f s", len(old.Community), time.Since(t0).Seconds())
	if len(moved) > 0 {
		msg += fmt.Sprintf(" (%d concepts took the graph's community)", len(moved))
	}
	ix.graphMovePending = msg
	if err := ix.DB.SetMeta("graph_moved", time.Now().UTC().Format(time.RFC3339)+" "+msg); err != nil {
		return err
	}
	return nil
}

// publishGraphMove publishes only a committed move. The transaction's meta
// receipt rolls back with the graph, and failures never claim success in logs.
func (ix *Indexer) publishGraphMove(msg string) {
	ix.mu.Lock()
	ix.GraphMoved = msg
	ix.mu.Unlock()
	if msg != "" && ix.DB.Path != "" {
		if f, err := os.OpenFile(filepath.Join(filepath.Dir(ix.DB.Path), "hook.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
			fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), msg)
			f.Close()
		}
	}
}

// MoveGraph moves a gob graph into the tables in its own transaction
// (hug mem upgrade --apply). It reports whether there was one to move.
func (ix *Indexer) MoveGraph() (bool, error) {
	if !ix.GraphNeedsMove() {
		ix.publishGraphMove("")
		return false, nil
	}
	moved := ""
	if err := ix.step(context.Background(), func() error {
		ix.GraphMoved, ix.graphMovePending = "", ""
		err := ix.moveGraph()
		moved, ix.graphMovePending = ix.graphMovePending, ""
		return err
	}); err != nil {
		ix.publishGraphMove("")
		return false, err
	}
	ix.publishGraphMove(moved)
	return moved != "", nil
}
