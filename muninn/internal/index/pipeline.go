package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/veridit/hug/muninn/internal/anchor"
	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/store"
)

// unitConcepts returns a unit's concept ids.
func (ix *Indexer) unitConcepts(unitID int64) ([]int, error) {
	rows, err := ix.DB.Query(`SELECT concept_id FROM unit_concepts WHERE unit_id=? ORDER BY concept_id`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var c int
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// setUnitConcepts replaces a unit's concept membership, creating concepts.
func (ix *Indexer) setUnitConcepts(unitID int64, terms []string) ([]int, error) {
	if _, err := ix.DB.Exec(`DELETE FROM unit_concepts WHERE unit_id=?`, unitID); err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(terms))
	for _, t := range terms {
		if _, err := ix.DB.Exec(`INSERT OR IGNORE INTO concepts(term) VALUES(?)`, t); err != nil {
			return nil, err
		}
		var id int
		if err := ix.DB.QueryRow(`SELECT id FROM concepts WHERE term=?`, t).Scan(&id); err != nil {
			return nil, err
		}
		if _, err := ix.DB.Exec(`INSERT OR IGNORE INTO unit_concepts(unit_id, concept_id) VALUES(?,?)`, unitID, id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

// refreshUnitCommunities recomputes per-community concept counts of units.
func (ix *Indexer) refreshUnitCommunities(units map[int64]bool) error {
	ids := make([]string, 0, len(units))
	for u := range units {
		ids = append(ids, strconv.FormatInt(u, 10))
	}
	for start := 0; start < len(ids); start += 500 {
		list := strings.Join(ids[start:min(start+500, len(ids))], ",")
		if _, err := ix.DB.Exec(`DELETE FROM unit_communities WHERE unit_id IN (` + list + `)`); err != nil {
			return err
		}
		if _, err := ix.DB.Exec(`INSERT INTO unit_communities(community_id, unit_id, n)
		  SELECT c.community_id, uc.unit_id, COUNT(*) FROM unit_concepts uc
		  JOIN communities c ON c.concept_id=uc.concept_id
		  WHERE uc.unit_id IN (` + list + `) GROUP BY 1, 2`); err != nil {
			return err
		}
	}
	return nil
}

func dirtyUnits(ids []int64) map[int64]bool {
	m := make(map[int64]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// applyDeltas runs DF-Louvain on the pass's concept deltas and persists the
// moved communities; dirty units get their per-community counts refreshed.
func (ix *Indexer) applyDeltas(dirty []int64) error {
	g, err := ix.loadGraph()
	if err != nil {
		return err
	}
	g.Update(ix.deltas)
	ix.GraphLoads += g.Loads
	return ix.saveGraph(g, dirtyUnits(dirty))
}

// ErrNotCurrent means the index did not record the source file a targeted
// update starts from; the caller runs a full pass instead.
var ErrNotCurrent = errors.New("index: source not current")

// ApplyNative updates the index for memories one write changed in the
// native store at path, and nothing else (4vd1mz): their units, the concept
// graph's delta, and the neighbour lists they were in. before is the file's
// stat the write started from and after its stat once saved, both taken
// under the store lock. When the index had not recorded before, some other
// change is unindexed and ApplyNative returns ErrNotCurrent without
// changing anything. Recall afterwards equals a full pass on the same file:
// each unit goes through the same reconciliation a pass uses, and the
// source's stat becomes after, so the next pass skips the file.
//
// The caller holds the write turn and no transaction.
func (ix *Indexer) ApplyNative(ctx context.Context, path string, before, after ingest.FileStat, changed []ingest.NativeMemory) error {
	moved := ""
	if err := ix.step(ctx, func() error {
		ix.GraphMoved, ix.graphMovePending = "", ""
		err := ix.applyNative(path, before, after, changed)
		moved, ix.graphMovePending = ix.graphMovePending, ""
		return err
	}); err != nil {
		return err
	}
	ix.publishGraphMove(moved)
	return nil
}

func (ix *Indexer) applyNative(path string, before, after ingest.FileStat, changed []ingest.NativeMemory) error {
	if ix.DB.GetMeta("source_recipe") != SourceRecipe {
		return ErrNotCurrent
	}
	var id, mtime, size int64
	if err := ix.DB.QueryRow(`SELECT id, mtime_ns, size FROM sources WHERE kind='native' AND path=?`, path).Scan(&id, &mtime, &size); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotCurrent
		}
		return err
	}
	if mtime != before.Mtime || size != before.Size || before.Mtime < 0 {
		return ErrNotCurrent
	}
	ix.deltas = nil
	var st Stats
	var dirty []int64
	for _, m := range changed {
		var uid, ident string
		err := ix.DB.QueryRow(`SELECT id, ident FROM units WHERE source_id=? AND key=?`, id, m.ID).Scan(&uid, &ident)
		if errors.Is(err, sql.ErrNoRows) {
			// The index never saw this memory although it recorded the
			// file: not the state a targeted update assumes.
			return ErrNotCurrent
		}
		if err != nil {
			return err
		}
		u, ok, err := ix.syncCandidate(id, nativeCandidate(m), [2]string{uid, ident}, true, &st)
		if err != nil {
			return err
		}
		if ok {
			dirty = append(dirty, u)
		}
	}
	if len(ix.deltas) > 0 {
		if err := ix.applyDeltas(dirty); err != nil {
			return err
		}
	}
	// No text changed, so nothing needs a vector; neighbours lose the
	// forgotten units exactly as a pass would detach them.
	if err := ix.UpdateNeighbors(dirty, nil); err != nil {
		return err
	}
	return ix.DB.SetSourceStat(id, after.Mtime, after.Size)
}

// JcodeSource is one jcode instance's memory directory and its scope prefix.
type JcodeSource struct {
	Dir    string // <JCODE_HOME>/memory
	Prefix string // "jcode@<instance>"
}

// Options controls one indexing pass.
type Options struct {
	Jcode      []JcodeSource
	NativePath string
	NotesRoots []string
}

// ErrBusy is returned with NoWait when another process holds the write lock,
// the write turn, or the pass lock.
var ErrBusy = errors.New("index busy: another muninn process is writing")

// Run performs one incremental pass over every configured source, as short
// write steps (steps.go): the sources step (read, reconcile, communities,
// queue neighbour work) in one transaction, as a pass always did, because
// per-statement fsyncs dominated cold indexing (profiled at 92% in
// syscalls); then the embedder, holding no lock, each batch's vectors
// written in its own step; then the neighbour step.
//
// Called with ctx carrying the write turn (a write flow's own pass), Run
// does the sources step only and leaves the rest owed (Owed): Finish does
// it once the caller has dropped the turn (App.WriteTurn's release does),
// so no embed runs under the turn. The neighbour lists then come out as
// one whole pass would make them: the same work, in the same order.
func (ix *Indexer) Run(ctx context.Context, o Options) (Stats, error) {
	ix.mu.Lock()
	ix.Passes++
	ix.mu.Unlock()
	held := ix.DB.HasTurn(ctx)
	if !held {
		release, ok, err := ix.passLock(ix.NoWait)
		if err != nil {
			return Stats{}, err
		}
		if !ok {
			return Stats{}, ErrBusy
		}
		defer release()
	}
	t0 := time.Now()
	var st Stats
	lap := t0
	phase := func(name string) {
		now := time.Now()
		st.Phases = append(st.Phases, Phase{name, now.Sub(lap)})
		lap = now
	}
	moved := ""
	if err := ix.step(ctx, func() error {
		ix.GraphMoved, ix.graphMovePending = "", ""
		err := ix.sourcesStep(ctx, o, &st, phase)
		moved, ix.graphMovePending = ix.graphMovePending, ""
		return err
	}); err != nil {
		return st, err
	}
	ix.publishGraphMove(moved)
	if held {
		ix.owed.Store(true)
		st.Elapsed = time.Since(t0)
		return st, nil
	}
	err := ix.finish(ctx, &st, phase)
	st.Elapsed = time.Since(t0)
	return st, err
}

// Owed reports whether a pass under a held write turn left work for Finish.
func (ix *Indexer) Owed() bool { return ix.owed.Load() }

// Finish does what a pass under a held write turn left: the embedder with
// no lock, then the neighbour step. ctx must not carry the turn (see
// store.WithoutTurn). It never queues behind another pass: one that holds
// the pass lock looks for queued work again before it lets go (finish), so
// ErrBusy leaves the work to it, or to the next pass.
func (ix *Indexer) Finish(ctx context.Context) (Stats, error) {
	if !ix.owed.Swap(false) {
		return Stats{}, nil
	}
	release, ok, err := ix.passLock(true)
	if err != nil {
		return Stats{}, err
	}
	if !ok {
		return Stats{}, ErrBusy
	}
	defer release()
	var st Stats
	t0 := time.Now()
	lap := t0
	err = ix.finish(ctx, &st, func(name string) {
		now := time.Now()
		st.Phases = append(st.Phases, Phase{name, now.Sub(lap)})
		lap = now
	})
	st.Elapsed = time.Since(t0)
	return st, err
}

// finish embeds what lacks a vector and does the queued neighbour work,
// under the pass lock. A write flow that wrote meanwhile found the pass
// lock taken (Finish, ErrBusy), so its work is looked for again before the
// lock is let go, a few rounds at most; the next pass does any rest. A
// capped pass (MaxEmbed, hooks) embeds in its first round only.
func (ix *Indexer) finish(ctx context.Context, st *Stats, phase func(string)) error {
	for round := 0; round < 3; round++ {
		if round == 0 || ix.MaxEmbed == 0 {
			embedded := st.Embedded
			if err := ix.embedSteps(ctx, st); err != nil {
				return err
			}
			if round == 0 {
				phase("embed")
			} else if st.Embedded == embedded && !ix.neighborsPending() {
				break
			}
		}
		if !ix.neighborsPending() {
			break
		}
		if err := ix.step(ctx, ix.neighborStep); err != nil {
			return err
		}
	}
	phase("neighbors")
	return nil
}

// SourceRecipe versions how sources are turned into units (keys, scopes,
// metadata). Bumping it makes the next pass re-read every source once even
// if its (mtime, size) is unchanged. Text that did not change keeps its
// content hash, so nothing is re-embedded.
//
//	2: per-instance jcode scopes ("jcode@<instance>[:<project>]")
const SourceRecipe = "2"

// sourcesStep is the pass's first write step: sources, pruned sources,
// anchors, communities, and the neighbour work they make, queued.
func (ix *Indexer) sourcesStep(ctx context.Context, o Options, st *Stats, phase func(string)) error {
	// A store whose graph is still the gob moves to the tables in the first
	// pass, even one with nothing else to do (4vd1mz, crab 04:44).
	if err := ix.moveGraph(); err != nil {
		return fmt.Errorf("move concept graph: %w", err)
	}
	if ix.DB.GetMeta("source_recipe") != SourceRecipe {
		if _, err := ix.DB.Exec(`UPDATE sources SET mtime_ns=-1, size=-1`); err != nil {
			return err
		}
		if err := ix.DB.SetMeta("source_recipe", SourceRecipe); err != nil {
			return err
		}
	}
	ix.deltas = nil
	var dirty, removed []int64
	collect := func(d, r []int64, err error) error {
		dirty, removed = append(dirty, d...), append(removed, r...)
		return err
	}
	for _, j := range o.Jcode {
		if _, err := os.Stat(j.Dir); err == nil {
			if err := collect(ix.IndexJcode(ctx, j.Dir, j.Prefix, st)); err != nil {
				return err
			}
		}
	}
	if o.NativePath != "" {
		if _, err := os.Stat(o.NativePath); err == nil {
			if err := collect(ix.IndexNative(ctx, o.NativePath, st)); err != nil {
				return err
			}
		}
	}
	if len(o.NotesRoots) > 0 {
		if err := collect(ix.IndexMarkdown(ctx, MarkdownFiles(o.NotesRoots), st)); err != nil {
			return err
		}
	}
	if err := ix.prunedSources(st, o); err != nil {
		return err
	}
	if ix.DB.GetMeta("anchor_recipe") != anchor.Recipe {
		t := time.Now()
		n, err := BackfillAnchors(ix.DB)
		if err != nil {
			return err
		}
		st.AnchorsRecorded = n
		if OnBackfill != nil {
			OnBackfill(n, time.Since(t))
		}
	}
	phase("sources")
	// Communities: DF-Louvain on the frontier of what changed.
	if len(ix.deltas) > 0 {
		if err := ix.applyDeltas(dirty); err != nil {
			return err
		}
		phase("communities")
	}
	return ix.queueNeighbors(dirty, removed)
}

// OnBackfill, when set, hears every anchor backfill (units, time taken):
// the hug binary writes it to hook.log, so a pass that landed in a hook is
// visible (n7jbmx).
var OnBackfill func(units int, took time.Duration)

// AnchorsPending reports whether the next index pass recomputes every
// memory's anchors (a store from before anchors, or a new anchor.Recipe).
func AnchorsPending(db *store.DB) bool {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM units WHERE active=1`).Scan(&n)
	return n > 0 && db.GetMeta("anchor_recipe") != anchor.Recipe
}

// BackfillAnchors recomputes every active unit's anchors: once for a store
// indexed before anchors existed, and again when anchor.Recipe changes. The
// caller holds the transaction (Run does). It returns the units read.
func BackfillAnchors(db *store.DB) (int, error) {
	rows, err := db.Query(`SELECT id, title, body FROM units WHERE active=1`)
	if err != nil {
		return 0, err
	}
	type unit struct {
		id          int64
		title, body string
	}
	var us []unit
	for rows.Next() {
		var u unit
		if err := rows.Scan(&u.id, &u.title, &u.body); err != nil {
			rows.Close()
			return 0, err
		}
		us = append(us, u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	// Run holds the pass's transaction on the one pooled connection.
	if _, err := db.Exec(`DELETE FROM unit_anchors`); err != nil {
		return 0, err
	}
	for _, u := range us {
		for _, a := range anchor.Memory(u.title + " " + u.body) {
			if _, err := db.Exec(`INSERT OR IGNORE INTO unit_anchors(unit_id, anchor) VALUES(?,?)`, u.id, a); err != nil {
				return 0, err
			}
		}
	}
	return len(us), db.SetMeta("anchor_recipe", anchor.Recipe)
}

// prunedSources removes units of sources that no longer exist on disk.
func (ix *Indexer) prunedSources(st *Stats, o Options) error {
	rows, err := ix.DB.Query(`SELECT id, path FROM sources`)
	if err != nil {
		return err
	}
	type src struct {
		id   int64
		path string
	}
	var gone []src
	for rows.Next() {
		var s src
		if err := rows.Scan(&s.id, &s.path); err != nil {
			rows.Close()
			return err
		}
		if _, err := os.Stat(s.path); os.IsNotExist(err) {
			gone = append(gone, s)
		}
	}
	rows.Close()
	for _, s := range gone {
		if _, _, err := ix.SyncSource(s.id, nil, st); err != nil {
			return err
		}
		if _, err := ix.DB.Exec(`DELETE FROM sources WHERE id=?`, s.id); err != nil {
			return err
		}
	}
	return nil
}

// MarkdownFiles expands roots (files or directories) into .md files,
// skipping hidden and vendor directories.
func MarkdownFiles(roots []string) []string {
	var out []string
	for _, r := range roots {
		fi, err := os.Stat(r)
		if err != nil {
			continue
		}
		if !fi.IsDir() {
			out = append(out, r)
			continue
		}
		filepath.WalkDir(r, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			name := d.Name()
			if d.IsDir() && p != r && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "_build") {
				return filepath.SkipDir
			}
			if !d.IsDir() && strings.HasSuffix(name, ".md") {
				out = append(out, p)
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}

// NeighborK is the neighbour-list length.
const NeighborK = 16

type vecRow struct {
	id int64
	q  []byte
}

func (ix *Indexer) activeVectors() ([]vecRow, error) {
	pipeline := ix.DB.GetMeta("pipeline_hash")
	rows, err := ix.DB.Query(`SELECT u.id, e.qvec FROM units u JOIN qvecs e
	  ON e.content_hash=u.content_hash AND e.pipeline_hash=? WHERE u.active=1`, pipeline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []vecRow
	for rows.Next() {
		var r vecRow
		if err := rows.Scan(&r.id, &r.q); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type nbr struct {
	id  int64
	sim float32
}

// UpdateNeighbors maintains the kNN graph incrementally: each dirty unit gets
// a fresh list (one pass over the vectors); every other unit only gains the
// dirty unit if it beats its current k-th neighbour; units that lost a removed
// neighbour are refilled. Nothing else is recomputed.
func (ix *Indexer) UpdateNeighbors(dirty, removed []int64) error {
	// Source reconciliation and newly embedded vectors can nominate the same
	// unit more than once. A frontier is a set, never a multiset of offers.
	unique := func(ids []int64) []int64 {
		seen := map[int64]bool{}
		out := make([]int64, 0, len(ids))
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
		return out
	}
	dirty, removed = unique(dirty), unique(removed)
	if len(dirty) == 0 && len(removed) == 0 {
		return nil
	}
	vecs, err := ix.activeVectors()
	if err != nil {
		return err
	}
	byID := make(map[int64][]byte, len(vecs))
	for _, v := range vecs {
		byID[v.id] = v.q
	}
	// Every dirty unit (changed text, forgotten, revived) and every removed
	// unit is first detached: units that referenced it lose that edge and are
	// refilled from scratch, because the old similarity is stale or the target
	// is no longer active. Dirty units that are still active are then offered
	// back to every other list below.
	refill := map[int64]bool{}
	detach := append(append([]int64(nil), removed...), dirty...)
	for _, r := range detach {
		rows, err := ix.DB.Query(`SELECT unit_id FROM neighbors WHERE nbr_id=?`, r)
		if err != nil {
			return err
		}
		for rows.Next() {
			var u int64
			rows.Scan(&u)
			refill[u] = true
		}
		rows.Close()
		if _, err := ix.DB.Exec(`DELETE FROM neighbors WHERE unit_id=? OR nbr_id=?`, r, r); err != nil {
			return err
		}
	}
	recompute := func(id int64) error {
		q, ok := byID[id]
		if !ok {
			_, err := ix.DB.Exec(`DELETE FROM neighbors WHERE unit_id=?`, id)
			return err
		}
		return ix.writeList(id, topK(q, id, vecs))
	}
	dirtySet := map[int64]bool{}
	for _, d := range dirty {
		dirtySet[d] = true
	}
	// Fresh lists for dirty and refill units, computed in parallel (pure CPU:
	// one int8 dot product per vector).
	var need []int64
	for _, d := range dirty {
		need = append(need, d)
	}
	for u := range refill {
		if !dirtySet[u] {
			need = append(need, u)
		}
	}
	fresh := make([][]nbr, len(need))
	parallel(len(need), func(i int) {
		if q, ok := byID[need[i]]; ok {
			fresh[i] = topK(q, need[i], vecs)
		}
	})
	for i, id := range need {
		if _, ok := byID[id]; !ok {
			if err := recompute(id); err != nil {
				return err
			}
			continue
		}
		if err := ix.writeList(id, fresh[i]); err != nil {
			return err
		}
	}
	// Offer each active dirty unit to every other list. Lists recomputed above
	// already considered every vector. The lists live in memory for the pass:
	// one SQL lookup per (dirty unit x list) made a 3,691-unit import run
	// ~74M queries (42 min); now each touched list is read once and written
	// once, and the dot products are split across cores by target list (each
	// worker owns a disjoint set of lists, so no locking).
	type list struct {
		nb      []nbr
		changed bool
	}
	lists := make(map[int64]*list, len(vecs))
	for _, v := range vecs {
		lists[v.id] = &list{}
	}
	var offers []vecRow
	for _, d := range dirty {
		if q, ok := byID[d]; ok {
			offers = append(offers, vecRow{d, q})
		}
	}
	if len(offers) == 0 {
		// Only forgotten (inactive) units: nothing to offer, every list that
		// referenced them was refilled above (4vd1mz).
		return nil
	}
	{
		rows, err := ix.DB.Query(`SELECT unit_id, nbr_id, sim FROM neighbors`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var u int64
			var n nbr
			if err := rows.Scan(&u, &n.id, &n.sim); err != nil {
				rows.Close()
				return err
			}
			if l := lists[u]; l != nil {
				l.nb = append(l.nb, n)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	parallel(len(vecs), func(i int) {
		v := vecs[i]
		if dirtySet[v.id] || refill[v.id] {
			return
		}
		l := lists[v.id]
		for _, d := range offers {
			sim := embed.DotQ(d.q, v.q)
			// An offered unit may already be in this list (it was re-embedded,
			// or a partial pass was interrupted): update in place, never add a
			// second edge to the same neighbour.
			if j := slices.IndexFunc(l.nb, func(n nbr) bool { return n.id == d.id }); j >= 0 {
				if l.nb[j].sim != sim {
					l.nb[j].sim = sim
					l.changed = true
				}
				continue
			}
			if len(l.nb) >= NeighborK {
				worst := 0
				for j := range l.nb {
					if l.nb[j].sim < l.nb[worst].sim || (l.nb[j].sim == l.nb[worst].sim && l.nb[j].id > l.nb[worst].id) {
						worst = j
					}
				}
				if sim < l.nb[worst].sim || (sim == l.nb[worst].sim && d.id >= l.nb[worst].id) {
					continue
				}
				l.nb[worst] = nbr{d.id, sim}
			} else {
				l.nb = append(l.nb, nbr{d.id, sim})
			}
			l.changed = true
		}
	})
	for id, l := range lists {
		if l.changed {
			if err := ix.writeList(id, l.nb); err != nil {
				return err
			}
		}
	}
	return nil
}

// topK is q's NeighborK nearest vectors, by similarity then id. It keeps
// the best NeighborK seen so far instead of sorting every vector (4vd1mz: a
// forget refills ~65 lists over 30,000 vectors; sorting each was most of the
// neighbour update). Same order, same ties, so the same list.
func topK(q []byte, id int64, vecs []vecRow) []nbr {
	better := func(a, b nbr) bool { return a.sim > b.sim || (a.sim == b.sim && a.id < b.id) }
	best := make([]nbr, 0, NeighborK+1)
	for _, v := range vecs {
		if v.id == id {
			continue
		}
		n := nbr{v.id, embed.DotQ(q, v.q)}
		if len(best) == NeighborK && !better(n, best[NeighborK-1]) {
			continue
		}
		// Insert in order; drop the worst past NeighborK.
		i := len(best)
		best = append(best, n)
		for i > 0 && better(n, best[i-1]) {
			best[i] = best[i-1]
			i--
		}
		best[i] = n
		if len(best) > NeighborK {
			best = best[:NeighborK]
		}
	}
	return best
}

// writeList replaces one unit's stored neighbour list. Duplicate neighbour
// ids (from any caller) collapse to the highest similarity.
func (ix *Indexer) writeList(id int64, nb []nbr) error {
	// Keep the storage boundary replay-safe too, including duplicate IDs in
	// a batched caller. Sort and cap after deduplication, not before it.
	byID := map[int64]float32{}
	for _, n := range nb {
		if n.id == id {
			continue
		}
		if old, ok := byID[n.id]; !ok || n.sim > old {
			byID[n.id] = n.sim
		}
	}
	nb = make([]nbr, 0, len(byID))
	for nid, sim := range byID {
		nb = append(nb, nbr{nid, sim})
	}
	sort.Slice(nb, func(i, j int) bool {
		if nb[i].sim == nb[j].sim {
			return nb[i].id < nb[j].id
		}
		return nb[i].sim > nb[j].sim
	})
	if len(nb) > NeighborK {
		nb = nb[:NeighborK]
	}
	if _, err := ix.DB.Exec(`SAVEPOINT neighbor_list`); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			ix.DB.Exec(`ROLLBACK TO neighbor_list`)
			ix.DB.Exec(`RELEASE neighbor_list`)
		}
	}()
	if _, err := ix.DB.Exec(`DELETE FROM neighbors WHERE unit_id=?`, id); err != nil {
		return err
	}
	seen := make(map[int64]int, len(nb))
	uniq := nb[:0:0]
	for _, b := range nb {
		if j, ok := seen[b.id]; ok {
			if b.sim > uniq[j].sim {
				uniq[j].sim = b.sim
			}
			continue
		}
		seen[b.id] = len(uniq)
		uniq = append(uniq, b)
	}
	nb = uniq
	if len(nb) == 0 {
		_, err := ix.DB.Exec(`RELEASE neighbor_list`)
		committed = err == nil
		return err
	}
	var sb strings.Builder
	args := make([]any, 0, 3*len(nb))
	sb.WriteString(`INSERT INTO neighbors(unit_id, nbr_id, sim) VALUES `)
	for i, b := range nb {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?,?)")
		args = append(args, id, b.id, b.sim)
	}
	sb.WriteString(` ON CONFLICT(unit_id,nbr_id) DO UPDATE SET sim=excluded.sim`)
	if _, err := ix.DB.Exec(sb.String(), args...); err != nil {
		return err
	}
	_, err := ix.DB.Exec(`RELEASE neighbor_list`)
	committed = err == nil
	return err
}

// parallel runs f(0..n-1) on all cores.
func parallel(n int, f func(i int)) {
	workers := min(runtime.GOMAXPROCS(0), n)
	if workers <= 1 {
		for i := 0; i < n; i++ {
			f(i)
		}
		return
	}
	var next atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= n {
					return
				}
				f(i)
			}
		}()
	}
	wg.Wait()
}
