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
	return ix.applyWritten(ctx, path, ingest.Written{Before: before, After: after, Changed: changed}, nil, true)
}

// ApplyWritten is ApplyNative for any one write to the native store
// (t0qap9): w.Changed may hold memories the write added, and w.Removed ids
// it dropped. vecs holds document vectors the caller embedded before it took
// the write turn, by content hash (ContentHash(EmbeddingText(title, body)));
// they are stored with the units, as a pass would store them, so the
// neighbour lists come out as a full pass makes them. A unit left without
// a vector is embedded by Finish once the turn is dropped (Owed). A short
// id the database had to extend on a collision is written back to the file
// only by a full pass: ApplyWritten returns ErrNotCurrent then, changing
// nothing.
func (ix *Indexer) ApplyWritten(ctx context.Context, path string, w ingest.Written, vecs map[string][]float32) error {
	if w.After == w.Before && len(w.Changed) == 0 && len(w.Removed) == 0 {
		return nil // the write saved nothing
	}
	return ix.applyWritten(ctx, path, w, vecs, false)
}

func (ix *Indexer) applyWritten(ctx context.Context, path string, w ingest.Written, vecs map[string][]float32, known bool) error {
	// While a pass runs, the write indexes its own memories and leaves the
	// rest of the file to that pass's successor (applyNative), instead of
	// waiting for the pass to end; with no pass running, an unrecorded
	// Before is a full pass's work, as before t0qap9.
	running := ix.passRunning()
	if err := ix.step(ctx, func() error {
		_, err := ix.applyNative(path, w, vecs, known, running)
		return err
	}); err != nil {
		return err
	}
	ix.owed.Store(true)
	if ix.DB.HasTurn(ctx) {
		return nil // App.WriteTurn's release runs Finish
	}
	// Never wait for a running pass: its holder looks for queued work
	// after it lets the pass lock go (drain), so it does this write's.
	if _, err := ix.settle(ctx, true); err != nil && !errors.Is(err, ErrBusy) {
		return err
	}
	return nil
}

// applyNative writes the units one write to the native store changed,
// inside the caller's step, and queues the index work they make (their
// concept deltas and neighbour lists), as a pass's writes do. known: every
// changed memory must already be a unit (a forget changes only those). It
// reports whether a unit it wrote still lacks a vector.
//
// When the index recorded the stat the write started from (w.Before), the
// source's stat becomes w.After, so the next pass skips the file. When it
// did not, some other change is unindexed: ErrNotCurrent, changing
// nothing, unless stale is set (a pass is running, maybe reading this file
// right now): then the write's own memories are still written, but the
// stat is left alone, so the next pass reads the file (t0qap9: a remember
// during a full reindex neither waits for it nor runs one).
func (ix *Indexer) applyNative(path string, w ingest.Written, vecs map[string][]float32, known, stale bool) (bool, error) {
	if ix.DB.GetMeta("source_recipe") != SourceRecipe {
		return false, ErrNotCurrent
	}
	var id, mtime, size int64
	if err := ix.DB.QueryRow(`SELECT id, mtime_ns, size FROM sources WHERE kind='native' AND path=?`, path).Scan(&id, &mtime, &size); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNotCurrent
		}
		return false, err
	}
	current := mtime == w.Before.Mtime && size == w.Before.Size && w.Before.Mtime >= 0
	if !current && !stale {
		return false, ErrNotCurrent
	}
	var st Stats
	var dirty []int64
	for _, m := range w.Changed {
		u := prepare(nativeCandidate(m))
		var oldID int64
		var ident string
		err := ix.DB.QueryRow(`SELECT id, ident FROM units WHERE source_id=? AND key=?`, id, m.ID).Scan(&oldID, &ident)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if known && current {
				// The index never saw this memory although it recorded the
				// file: not the state a targeted forget assumes.
				return false, ErrNotCurrent
			}
		case err != nil:
			return false, err
		default:
			if ident == u.ident {
				continue
			}
			u.had, u.oldID, u.oldIdent = true, oldID, ident
		}
		uid, err := ix.writeUnit(id, u, &st)
		if err != nil {
			return false, err
		}
		dirty = append(dirty, uid)
		var sid string
		if err := ix.DB.QueryRow(`SELECT sid FROM units WHERE id=?`, uid).Scan(&sid); err != nil {
			return false, err
		}
		if sid != m.SID {
			return false, ErrNotCurrent // a pass writes the extended id back
		}
	}
	for _, key := range w.Removed {
		var uid int64
		err := ix.DB.QueryRow(`SELECT id FROM units WHERE source_id=? AND key=?`, id, key).Scan(&uid)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return false, err
		}
		if err := ix.removeUnit(uid, unitWrite{}, &st); err != nil {
			return false, err
		}
	}
	missing, err := ix.putGiven(dirty, vecs)
	if err != nil {
		return false, err
	}
	if !current {
		return missing, nil
	}
	return missing, ix.DB.SetSourceStat(id, w.After.Mtime, w.After.Size)
}

// putGiven stores the given vectors of dirty units that lack one under the
// current pipeline, and reports whether an active one is still without.
func (ix *Indexer) putGiven(dirty []int64, vecs map[string][]float32) (bool, error) {
	if ix.Embedder == nil || len(dirty) == 0 {
		return false, nil
	}
	pipeline := PipelineHash(ix.Embedder.Model())
	if ix.DB.GetMeta("pipeline_hash") != pipeline {
		return true, nil // a pipeline switch is a pass's work
	}
	missing := false
	for _, u := range dirty {
		var hash string
		var active, have int
		if err := ix.DB.QueryRow(`SELECT content_hash, active, EXISTS(SELECT 1 FROM embeddings e WHERE e.content_hash=units.content_hash AND e.pipeline_hash=?) FROM units WHERE id=?`, pipeline, u).Scan(&hash, &active, &have); err != nil {
			return false, err
		}
		if have == 1 {
			continue
		}
		v, ok := vecs[hash]
		if !ok || v == nil {
			missing = missing || active == 1
			continue
		}
		if err := ix.DB.PutEmbedding(hash, pipeline, len(v), embed.EncodeF32(v), embed.Quantize(v), embed.Sketch(v)); err != nil {
			return false, err
		}
	}
	return missing, nil
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

// Run performs one incremental pass over every configured source (t0qap9).
// No lock is held while anything is computed: each source file is read,
// parsed and reconciled with no lock, and its changes are written in short
// write steps (steps.go), each one write turn and one transaction of at
// most about stepBudget; the writes queue the index work they make
// (index_deltas, index_pending). Then finish does that work under the pass
// lock only: the concept graph update, the embedder, the neighbour lists,
// each computed with no lock and written in short steps. A memory write
// from another process gets the turn between any two steps.
//
// Called with ctx carrying the write turn (a write flow's own pass), Run
// writes the sources only and leaves the rest owed (Owed): Finish does it
// once the caller has dropped the turn (App.WriteTurn's release does). The
// index then comes out as one whole pass would make it: the same work, in
// the same order.
func (ix *Indexer) Run(ctx context.Context, o Options) (Stats, error) {
	ix.mu.Lock()
	ix.Passes++
	ix.mu.Unlock()
	held := ix.DB.HasTurn(ctx)
	var release func()
	if !held {
		r, ok, err := ix.passLock(ix.NoWait)
		if err != nil {
			return Stats{}, err
		}
		if !ok {
			return Stats{}, ErrBusy
		}
		release = r
	}
	t0 := time.Now()
	var st Stats
	lap := t0
	phase := func(name string) {
		now := time.Now()
		st.Phases = append(st.Phases, Phase{name, now.Sub(lap)})
		lap = now
	}
	if err := ix.sources(ctx, o, &st, phase); err != nil {
		if release != nil {
			release()
		}
		return st, err
	}
	if held {
		ix.owed.Store(true)
		st.Elapsed = time.Since(t0)
		return st, nil
	}
	err := ix.drain(ctx, &st, phase, release, ix.NoWait)
	st.Elapsed = time.Since(t0)
	return st, err
}

// Owed reports whether a write under a held write turn left work for Finish.
func (ix *Indexer) Owed() bool { return ix.owed.Load() }

// Finish does what a write under a held write turn left: the concept graph
// update, the embedder, the neighbour lists. ctx must not carry the turn
// (see store.WithoutTurn). It never queues behind another pass: one that
// holds the pass lock looks for queued work again after it lets go
// (drain), so ErrBusy leaves the work to it, or to the next pass.
func (ix *Indexer) Finish(ctx context.Context) (Stats, error) {
	if !ix.owed.Swap(false) {
		return Stats{}, nil
	}
	return ix.settle(ctx, true)
}

// settle does the queued index work now under the pass lock (noWait: or
// returns ErrBusy at once if it is taken).
func (ix *Indexer) settle(ctx context.Context, noWait bool) (Stats, error) {
	ix.owed.Store(false)
	release, ok, err := ix.passLock(noWait)
	if err != nil {
		return Stats{}, err
	}
	if !ok {
		return Stats{}, ErrBusy
	}
	var st Stats
	t0 := time.Now()
	lap := t0
	err = ix.drain(ctx, &st, func(name string) {
		now := time.Now()
		st.Phases = append(st.Phases, Phase{name, now.Sub(lap)})
		lap = now
	}, release, noWait)
	st.Elapsed = time.Since(t0)
	return st, err
}

// drain runs finish under the pass lock the caller took, lets the lock go,
// and looks for queued work once more: a write flow that queued work while
// the lock was held found it taken (Finish, ErrBusy) and left that work to
// this holder. If there is some, drain takes the lock again without
// waiting; a holder that has it now does the same check after it.
func (ix *Indexer) drain(ctx context.Context, st *Stats, phase func(string), release func(), noWait bool) error {
	for i := 0; ; i++ {
		err := ix.finish(ctx, st, phase)
		release()
		if err != nil || i >= 8 || !ix.workPending() {
			return err
		}
		r, ok, err := ix.passLock(true)
		if err != nil || !ok {
			return err
		}
		release = r
	}
}

// finish does the queued index work under the pass lock: the concept
// graph update (communityStep), the embedder (embedSteps), the neighbour
// lists (neighborStep). A write flow that wrote meanwhile queued its work,
// so finish looks again before it returns, a few rounds at most; drain and
// the next pass do any rest. A capped pass (MaxEmbed, hooks) embeds in its
// first round only.
func (ix *Indexer) finish(ctx context.Context, st *Stats, phase func(string)) error {
	for round := 0; round < 3; round++ {
		if round > 0 {
			embedded := st.Embedded
			if ix.MaxEmbed == 0 {
				if err := ix.embedSteps(ctx, st); err != nil {
					return err
				}
			}
			if st.Embedded == embedded && !ix.workPending() {
				break
			}
		}
		had, err := ix.communityStep(ctx)
		if err != nil {
			return err
		}
		if round == 0 {
			if had {
				phase("communities")
			}
			if err := ix.embedSteps(ctx, st); err != nil {
				return err
			}
			phase("embed")
		}
		if ix.neighborsPending() {
			if err := ix.neighborStep(ctx); err != nil {
				return err
			}
		}
		if round == 0 {
			phase("neighbors")
		}
	}
	return nil
}

// SourceRecipe versions how sources are turned into units (keys, scopes,
// metadata). Bumping it makes the next pass re-read every source once even
// if its (mtime, size) is unchanged. Text that did not change keeps its
// content hash, so nothing is re-embedded.
//
//	2: per-instance jcode scopes ("jcode@<instance>[:<project>]")
const SourceRecipe = "2"

// sources writes every source's changes, pruned sources and the anchor
// backfill, in short write steps, and queues the index work they make.
func (ix *Indexer) sources(ctx context.Context, o Options, st *Stats, phase func(string)) error {
	// A store whose graph is still the gob moves to the tables in the first
	// pass, even one with nothing else to do (4vd1mz, crab 04:44).
	if ix.GraphNeedsMove() || ix.DB.GetMeta("source_recipe") != SourceRecipe {
		moved := ""
		if err := ix.step(ctx, func() error {
			ix.GraphMoved, ix.graphMovePending = "", ""
			defer func() { moved, ix.graphMovePending = ix.graphMovePending, "" }()
			if err := ix.moveGraph(); err != nil {
				return fmt.Errorf("move concept graph: %w", err)
			}
			if ix.DB.GetMeta("source_recipe") != SourceRecipe {
				if _, err := ix.DB.Exec(`UPDATE sources SET mtime_ns=-1, size=-1`); err != nil {
					return err
				}
				return ix.DB.SetMeta("source_recipe", SourceRecipe)
			}
			return nil
		}); err != nil {
			return err
		}
		ix.publishGraphMove(moved)
	}
	ctx = context.WithValue(ctx, passKey{}, true)
	for _, j := range o.Jcode {
		if _, err := os.Stat(j.Dir); err == nil {
			if _, _, err := ix.IndexJcode(ctx, j.Dir, j.Prefix, st); err != nil {
				return err
			}
		}
	}
	if o.NativePath != "" {
		if _, err := os.Stat(o.NativePath); err == nil {
			if _, _, err := ix.IndexNative(ctx, o.NativePath, st); err != nil {
				return err
			}
		}
	}
	if len(o.NotesRoots) > 0 {
		if _, _, err := ix.IndexMarkdown(ctx, MarkdownFiles(o.NotesRoots), st); err != nil {
			return err
		}
	}
	if err := ix.prunedSources(ctx, st); err != nil {
		return err
	}
	if ix.DB.GetMeta("anchor_recipe") != anchor.Recipe {
		if err := ix.backfillAnchors(ctx, st); err != nil {
			return err
		}
	}
	phase("sources")
	return nil
}

// backfillAnchors is BackfillAnchors in short write steps, one unit at a
// time (anchors are a few regexps per unit).
func (ix *Indexer) backfillAnchors(ctx context.Context, st *Stats) error {
	t := time.Now()
	rows, err := ix.DB.Query(`SELECT id FROM units ORDER BY id`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	n := 0
	if err := ix.batches(ctx, len(ids), func(i int) error {
		var title, body string
		var active int
		err := ix.DB.QueryRow(`SELECT title, body, active FROM units WHERE id=?`, ids[i]).Scan(&title, &body, &active)
		if errors.Is(err, sql.ErrNoRows) {
			return nil // deleted since: its anchors went with it
		}
		if err != nil {
			return err
		}
		var anchors []string
		if active == 1 {
			anchors = anchor.Memory(title + " " + body)
			n++
		}
		return ix.DB.SetUnitAnchors(ids[i], anchors)
	}, func() error { return ix.DB.SetMeta("anchor_recipe", anchor.Recipe) }); err != nil {
		return err
	}
	st.AnchorsRecorded = n
	if OnBackfill != nil {
		OnBackfill(n, time.Since(t))
	}
	return nil
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

// prunedSources removes units of sources that no longer exist on disk, in
// short write steps; the source row goes in the last one.
func (ix *Indexer) prunedSources(ctx context.Context, st *Stats) error {
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
		existing, err := ix.DB.UnitHashes(s.id)
		if err != nil {
			return err
		}
		ws := plan(existing, nil, st)
		for i := range ws {
			ws[i].keepEdges = true
		}
		raced := false
		if err := ix.batches(ctx, len(ws), func(i int) error {
			_, r, err := ix.writePlanned(s.id, ws[i], st)
			raced = raced || r
			return err
		}, func() error {
			if raced {
				return nil
			}
			_, err := ix.DB.Exec(`DELETE FROM sources WHERE id=?`, s.id)
			return err
		}); err != nil {
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

// UpdateNeighbors maintains the kNN graph incrementally and writes the
// lists it changed straight away, in the caller's transaction or none
// (tests). A pass plans and writes in steps instead (neighborStep).
func (ix *Indexer) UpdateNeighbors(dirty, removed []int64) error {
	lists, err := ix.planNeighbors(dirty, removed)
	if err != nil {
		return err
	}
	for _, l := range lists {
		if err := ix.writeList(l.id, l.nb); err != nil {
			return err
		}
	}
	return nil
}

// newList is one neighbour list a plan replaces (nb empty: deleted).
type newList struct {
	id int64
	nb []nbr
}

// planNeighbors works out the kNN graph's incremental update with no lock
// held, from one read of the vectors and the stored lists (t0qap9): each
// dirty unit gets a fresh list (one pass over the vectors); every other
// unit only gains the dirty unit if it beats its current k-th neighbour;
// units that lost a removed neighbour are refilled. Nothing else is
// recomputed. It returns the lists that change, by unit id; a unit gone or
// without a vector gets an empty one. Written in order, they turn the
// stored lists into what the one-transaction update made, statement by
// statement, before t0qap9.
func (ix *Indexer) planNeighbors(dirty, removed []int64) ([]newList, error) {
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
		return nil, nil
	}
	vecs, err := ix.activeVectors()
	if err != nil {
		return nil, err
	}
	byID := make(map[int64][]byte, len(vecs))
	for _, v := range vecs {
		byID[v.id] = v.q
	}
	// Every dirty unit (changed text, forgotten, revived) and every removed
	// unit is detached: units that referenced it lose that edge and are
	// refilled from scratch, because the old similarity is stale or the
	// target is no longer active. Dirty units that are still active are then
	// offered back to every other list below. Refilled and detached lists
	// are recomputed whole, so only the others are kept from the stored
	// lists, which are read once (1.3M rows on Jørgen's store, under 0.8 s).
	detached := make(map[int64]bool, len(dirty)+len(removed))
	for _, r := range removed {
		detached[r] = true
	}
	for _, d := range dirty {
		detached[d] = true
	}
	refill := map[int64]bool{}
	stored := map[int64][]nbr{}
	{
		rows, err := ix.DB.Query(`SELECT unit_id, nbr_id, sim FROM neighbors`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var u int64
			var n nbr
			if err := rows.Scan(&u, &n.id, &n.sim); err != nil {
				rows.Close()
				return nil, err
			}
			if detached[n.id] && !detached[u] {
				refill[u] = true
			}
			stored[u] = append(stored[u], n)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	dirtySet := map[int64]bool{}
	for _, d := range dirty {
		dirtySet[d] = true
	}
	// Fresh lists for dirty and refill units, computed in parallel (pure CPU:
	// one int8 dot product per vector).
	var need []int64
	need = append(need, dirty...)
	refills := make([]int64, 0, len(refill))
	for u := range refill {
		if !dirtySet[u] {
			refills = append(refills, u)
		}
	}
	slices.Sort(refills)
	need = append(need, refills...)
	out := make([]newList, len(need), len(need)+len(removed))
	parallel(len(need), func(i int) {
		out[i].id = need[i]
		if q, ok := byID[need[i]]; ok {
			out[i].nb = topK(q, need[i], vecs)
		}
	})
	inOut := make(map[int64]bool, len(need))
	for _, id := range need {
		inOut[id] = true
	}
	for _, r := range removed {
		if !inOut[r] {
			out = append(out, newList{id: r}) // its own list goes with it
			inOut[r] = true
		}
	}
	// Offer each active dirty unit to every other list. Lists recomputed above
	// already considered every vector. The lists live in memory: one SQL
	// lookup per (dirty unit x list) made a 3,691-unit import run ~74M
	// queries (42 min); now each touched list is read once and written once,
	// and the dot products are split across cores by target list (each
	// worker owns a disjoint set of lists, so no locking).
	var offers []vecRow
	for _, d := range dirty {
		if q, ok := byID[d]; ok {
			offers = append(offers, vecRow{d, q})
		}
	}
	if len(offers) == 0 {
		// Only forgotten (inactive) units: nothing to offer, every list that
		// referenced them was refilled above (4vd1mz).
		return out, nil
	}
	type list struct {
		nb      []nbr
		changed bool
	}
	lists := make([]list, len(vecs))
	for i, v := range vecs {
		lists[i].nb = stored[v.id]
	}
	parallel(len(vecs), func(i int) {
		v := vecs[i]
		if dirtySet[v.id] || refill[v.id] {
			return
		}
		l := &lists[i]
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
	for i, v := range vecs {
		if lists[i].changed && !inOut[v.id] {
			out = append(out, newList{id: v.id, nb: lists[i].nb})
		}
	}
	return out, nil
}

// writeLists writes a plan's lists in short write steps; last runs in the
// step that writes the final one.
func (ix *Indexer) writeLists(ctx context.Context, lists []newList, last func() error) error {
	return ix.batches(ctx, len(lists), func(i int) error {
		return ix.writeList(lists[i].id, lists[i].nb)
	}, last)
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
