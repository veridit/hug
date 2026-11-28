package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/veridit/hug/muninn/internal/anchor"
	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/shortid"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/trigger"
)

// Stats summarises one indexing pass.
type Stats struct {
	SourcesScanned, SourcesSkipped     int
	Added, Changed, Removed, Unchanged int
	Embedded, EmbedReused              int
	EmbedPending                       int // left for a later pass (MaxEmbed)
	AnchorsRecorded                    int // units whose anchors the one-time backfill wrote (n7jbmx)
	Elapsed                            time.Duration
	Phases                             []Phase
	EmbedErr                           error
}

// Phase is one timed step of an indexing pass.
type Phase struct {
	Name string
	Dur  time.Duration
}

func (s Stats) String() string {
	out := fmt.Sprintf("sources scanned=%d skipped=%d | units added=%d changed=%d removed=%d unchanged=%d | embedded=%d reused=%d | %s",
		s.SourcesScanned, s.SourcesSkipped, s.Added, s.Changed, s.Removed, s.Unchanged, s.Embedded, s.EmbedReused, s.Elapsed.Round(time.Millisecond))
	if s.AnchorsRecorded > 0 {
		out += fmt.Sprintf(" | anchors recorded for %d memories (once)", s.AnchorsRecorded)
	}
	if s.EmbedPending > 0 {
		out += fmt.Sprintf(" | %d still to embed (run `hug mem index`)", s.EmbedPending)
	}
	if s.Elapsed > time.Second {
		for _, p := range s.Phases {
			out += fmt.Sprintf(" %s=%s", p.Name, p.Dur.Round(time.Millisecond))
		}
	}
	return out
}

// Indexer applies sources to the store incrementally.
type Indexer struct {
	DB       *store.DB
	Embedder embed.Embedder // nil = keyword-only
	// MaxEmbed caps embeddings per pass (0 = no cap); hooks set it.
	MaxEmbed int
	// NoWait makes Run return ErrBusy instead of waiting for another
	// process's write transaction (hooks).
	NoWait bool

	// mu is the step mutex: the one pooled connection carries one write
	// transaction, and a write step's per-pass state (the graph move
	// receipt) is touched under it.
	mu sync.Mutex
	// owed is set when a write under a held write turn left index work
	// (communities, embedding, neighbour lists) for Finish (t0qap9).
	owed atomic.Bool

	// Passes counts full index passes (Run) on this Indexer, so tests can
	// prove a command did none (4vd1mz).
	Passes int
	// GraphLoads counts concept adjacencies read from storage (4vd1mz);
	// GraphMoved is set when this indexer moved a gob graph to the tables.
	GraphLoads       int
	GraphMoved       string
	graphMovePending string
	// graphVisitOrder is package-private, per-instance and nil in production.
	// Tests may inject a common visitation schedule into independently built
	// graphs without changing production recall ordering.
	graphVisitOrder func([]int)
}

// Candidate is a unit as produced by a source, before storage.
type Candidate struct {
	Key, Kind, Title, Body, Category, Scope string
	Type                                    string // memory type, see store.Unit.Type
	Holds                                   string // condition, see store.Unit.Holds
	Trigger                                 string // when-then trigger, see store.Unit.Trigger
	SID, Origin                             string
	Tags                                    []string
	Active                                  bool
	Updated                                 time.Time
}

// unitWrite is one unit change a source makes, computed with no lock held
// (prepare, plan) and written in a short step (writeUnit, removeUnit).
type unitWrite struct {
	c      Candidate
	key    string
	remove bool
	// keepEdges: a removed unit's neighbour edges are not detached and the
	// lists that named it are not refilled. A pruned source's units went
	// that way before t0qap9 (its removals never reached the neighbour
	// update), and t0qap9 keeps the results.
	keepEdges bool
	// noQueue: write the unit only, queue no index work (SyncSource).
	noQueue  bool
	had      bool   // the unit existed when the plan read the source's units
	oldID    int64  // its id then
	oldIdent string // its ident then
	text     string // EmbeddingText of the candidate
	ident    string
	terms    []string // concepts; none for an inactive unit
	anchors  []string
}

// prepare computes everything a candidate's unit write needs that is not
// in the database: its ident, concepts and anchors. Pure CPU.
func prepare(c Candidate) unitWrite {
	text := EmbeddingText(c.Title, c.Body)
	ident := ContentHash(text + "\x00" + c.Category + "\x00" + c.Scope + "\x00" +
		strconv.FormatBool(c.Active) + "\x00" + c.Origin + "\x00" + strings.Join(c.Tags, " "))
	if c.Type != "" {
		// Only typed memories hash the type, so existing idents (and with
		// them every untyped unit) stay unchanged by this field.
		ident = ContentHash(ident + "\x00type:" + c.Type)
	}
	if c.Holds != "" {
		// Same rule for the condition (ez50rb): no condition, no change.
		ident = ContentHash(ident + "\x00holds:" + c.Holds)
	}
	if c.Trigger != "" {
		// Same rule for the trigger (jrj1vq): no trigger, no change.
		ident = ContentHash(ident + "\x00trigger:" + c.Trigger)
	}
	w := unitWrite{c: c, key: c.Key, text: text, ident: ident}
	if c.Active {
		w.terms = Concepts(c.Title + " " + c.Body + " " + strings.Join(c.Tags, " "))
		w.anchors = anchor.Memory(c.Title + " " + c.Body)
	}
	return w
}

// plan is the unit writes that reconcile a source's stored units (existing:
// key -> (id, ident), read before the candidates were) with its current
// candidates: changed and new units, in candidate order, then units gone
// from the source, by id. Units whose ident is unchanged are not touched at
// all: no write, no re-embed, no graph work (seid search's incremental
// rule). A key given twice keeps its last candidate.
func plan(existing map[string][2]string, cands []Candidate, st *Stats) []unitWrite {
	last := make(map[string]int, len(cands))
	for i, c := range cands {
		last[c.Key] = i
	}
	var ws, gone []unitWrite
	for i, c := range cands {
		if last[c.Key] != i {
			continue
		}
		old, had := existing[c.Key]
		w := prepare(c)
		if had && old[1] == w.ident {
			st.Unchanged++
			continue
		}
		w.had = had
		if had {
			w.oldID, _ = strconv.ParseInt(old[0], 10, 64)
			w.oldIdent = old[1]
		}
		ws = append(ws, w)
	}
	for key, old := range existing {
		if _, ok := last[key]; ok {
			continue
		}
		id, _ := strconv.ParseInt(old[0], 10, 64)
		gone = append(gone, unitWrite{key: key, remove: true, had: true, oldID: id, oldIdent: old[1]})
	}
	sort.Slice(gone, func(i, j int) bool { return gone[i].oldID < gone[j].oldID })
	return append(ws, gone...)
}

// writeUnit stores one planned unit, inside the caller's step: the unit,
// its concepts and anchors, and the work it makes for the index (its
// concept delta and its neighbour list, queued). It returns the unit id.
func (ix *Indexer) writeUnit(sourceID int64, w unitWrite, st *Stats) (int64, error) {
	c := w.c
	var oldConcepts []int
	if w.had {
		var err error
		if oldConcepts, err = ix.unitConcepts(w.oldID); err != nil {
			return 0, err
		}
	}
	id, err := ix.DB.UpsertUnit(store.Unit{
		SourceID: sourceID, SID: c.SID, Origin: c.Origin, Key: c.Key, ContentHash: ContentHash(w.text), Ident: w.ident, Kind: c.Kind,
		Title: c.Title, Body: c.Body, Category: c.Category, Type: c.Type, Holds: c.Holds, Trigger: c.Trigger, Tags: c.Tags, Scope: c.Scope,
		Active: c.Active, Updated: c.Updated,
	})
	if err != nil {
		return 0, err
	}
	newConcepts, err := ix.setUnitConcepts(id, w.terms)
	if err != nil {
		return 0, err
	}
	if !c.Active {
		newConcepts = nil
	}
	if err := ix.DB.SetUnitAnchors(id, w.anchors); err != nil {
		return 0, err
	}
	if !w.noQueue {
		if err := ix.queueDelta(id, oldConcepts, newConcepts); err != nil {
			return 0, err
		}
		if err := ix.queueList(id, false); err != nil {
			return 0, err
		}
	}
	if w.had {
		st.Changed++
	} else {
		st.Added++
	}
	return id, nil
}

// removeUnit deletes a unit gone from its source, inside the caller's step,
// and queues its concept delta and the detaching of its neighbour edges.
func (ix *Indexer) removeUnit(id int64, w unitWrite, st *Stats) error {
	oldConcepts, err := ix.unitConcepts(id)
	if err != nil {
		return err
	}
	if !w.noQueue {
		if err := ix.queueDelta(id, oldConcepts, nil); err != nil {
			return err
		}
	}
	if err := ix.DB.DeleteUnit(id); err != nil {
		return err
	}
	st.Removed++
	if w.keepEdges || w.noQueue {
		return nil
	}
	return ix.queueList(id, true)
}

// writePlanned writes one planned change unless another writer changed the
// unit since the plan read it (raced): then it writes nothing, and the
// caller leaves the source's stat unrecorded so the next pass reads it
// again.
func (ix *Indexer) writePlanned(sourceID int64, w unitWrite, st *Stats) (id int64, raced bool, err error) {
	var curID int64
	var curIdent string
	err = ix.DB.QueryRow(`SELECT id, ident FROM units WHERE source_id=? AND key=?`, sourceID, w.key).Scan(&curID, &curIdent)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, false, err
	}
	if exists != w.had || (exists && (curID != w.oldID || curIdent != w.oldIdent)) {
		return 0, true, nil
	}
	if w.remove {
		return curID, false, ix.removeUnit(curID, w, st)
	}
	id, err = ix.writeUnit(sourceID, w, st)
	return id, false, err
}

// SyncSource reconciles one source's units with its current candidates
// inside the caller's transaction (or none): units, concepts and anchors
// only. It queues no index work (concept graph, neighbour lists); the
// caller does that with the ids it returns, or not at all (fixtures).
func (ix *Indexer) SyncSource(sourceID int64, cands []Candidate, st *Stats) (dirty []int64, removed []int64, err error) {
	existing, err := ix.DB.UnitHashes(sourceID)
	if err != nil {
		return nil, nil, err
	}
	for _, w := range plan(existing, cands, st) {
		w.noQueue = true
		id, _, err := ix.writePlanned(sourceID, w, st)
		if err != nil {
			return nil, nil, err
		}
		if w.remove {
			removed = append(removed, id)
		} else {
			dirty = append(dirty, id)
		}
	}
	return dirty, removed, nil
}

// nativeCandidate is how IndexNative turns a stored memory into a unit.
func nativeCandidate(m ingest.NativeMemory) Candidate {
	origin := m.Origin
	if strings.HasPrefix(m.ID, "cap_repair_") {
		origin = shortid.Identity(m.ID, m.Origin)
	}
	return Candidate{Key: m.ID, SID: m.SID, Origin: origin, Kind: "memory", Title: m.Title, Body: m.Content, Category: m.Category,
		Type: m.Type, Holds: m.Holds, Trigger: trigger.Encode(m.Trigger), Tags: m.Tags, Scope: m.Scope, Active: m.Active, Updated: m.Updated}
}

// statOf returns a file's (mtime ns, size).
func statOf(path string) (int64, int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	return fi.ModTime().UnixNano(), fi.Size(), nil
}

// IndexJcode indexes the jcode memory store under dir with scope prefix
// (e.g. "jcode@brovs").
func (ix *Indexer) IndexJcode(ctx context.Context, dir, prefix string, st *Stats) ([]int64, []int64, error) {
	files, err := ingest.JcodeFiles(dir, prefix)
	if err != nil {
		return nil, nil, err
	}
	var dirtyAll, removedAll []int64
	for path, scope := range files {
		d, r, err := ix.indexFile(ctx, path, "jcode", st, func() ([]Candidate, error) {
			mems, err := ingest.ReadJcodeFile(path)
			if err != nil {
				return nil, err
			}
			out := make([]Candidate, 0, len(mems))
			for _, m := range mems {
				out = append(out, Candidate{Key: m.Key, Kind: "memory", Body: m.Content, Category: m.Category,
					Tags: m.Tags, Scope: scope, Active: m.Active, Updated: m.Updated})
			}
			return out, nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		dirtyAll, removedAll = append(dirtyAll, d...), append(removedAll, r...)
	}
	return dirtyAll, removedAll, nil
}

// IndexNative indexes muninn's own memory store.
func (ix *Indexer) IndexNative(ctx context.Context, path string, st *Stats) ([]int64, []int64, error) {
	var loaded []ingest.NativeMemory
	dirty, removed, err := ix.indexFile(ctx, path, "native", st, func() ([]Candidate, error) {
		mems, err := ingest.LoadNative(path)
		if err != nil {
			return nil, err
		}
		loaded = mems
		out := make([]Candidate, 0, len(mems))
		for _, m := range mems {
			out = append(out, nativeCandidate(m))
		}
		return out, nil
	})
	if err != nil {
		return dirty, removed, err
	}
	// The database sees all sources, so its collision extension is authoritative.
	changed := false
	for i := range loaded {
		var sid string
		err := ix.DB.QueryRow(`SELECT u.sid FROM units u JOIN sources s ON s.id=u.source_id WHERE s.path=? AND u.key=?`, path, loaded[i].ID).Scan(&sid)
		if errors.Is(err, sql.ErrNoRows) {
			continue // another writer dropped it since the read: not ours to name
		}
		if err != nil {
			return dirty, removed, err
		}
		if sid != loaded[i].SID {
			loaded[i].SID = sid
			changed = true
		}
	}
	if changed {
		// Write back only the SIDs, onto the file as it is now: another process
		// may have written since it was read.
		sids := map[string]string{}
		for _, m := range loaded {
			sids[m.ID] = m.SID
		}
		err = ingest.UpdateNative(path, func(cur []ingest.NativeMemory) ([]ingest.NativeMemory, bool, error) {
			save := false
			for i := range cur {
				if sid, ok := sids[cur[i].ID]; ok && cur[i].SID != sid {
					cur[i].SID = sid
					save = true
				}
			}
			return cur, save, nil
		})
	}
	return dirty, removed, err
}

// IndexMarkdown indexes markdown files under roots, split by heading.
func (ix *Indexer) IndexMarkdown(ctx context.Context, files []string, st *Stats) ([]int64, []int64, error) {
	var dirtyAll, removedAll []int64
	for _, path := range files {
		d, r, err := ix.indexFile(ctx, path, "markdown", st, func() ([]Candidate, error) {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			secs := ingest.SplitMarkdown(path, string(b))
			out := make([]Candidate, 0, len(secs))
			for _, s := range secs {
				out = append(out, Candidate{Key: "L" + strconv.Itoa(s.Line) + ":" + s.Heading, Kind: "section",
					Title: s.Heading, Body: s.Content, Scope: "notes", Active: true})
			}
			return out, nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		dirtyAll, removedAll = append(dirtyAll, d...), append(removedAll, r...)
	}
	return dirtyAll, removedAll, nil
}

// passKey marks a pass's context: only a pass's source writes queue index
// work. IndexJcode, IndexNative and IndexMarkdown called directly write
// units only, as they always did.
type passKey struct{}

// indexFile skips a file whose (mtime, size) is unchanged; otherwise it
// reads the source's units, parses the file and works out the changes with
// no lock held, then writes them in short steps (t0qap9). A unit another
// writer changed meanwhile is left to it (writePlanned), and then the
// file's stat is not recorded, so the next pass reads the file again.
func (ix *Indexer) indexFile(ctx context.Context, path, kind string, st *Stats, parse func() ([]Candidate, error)) ([]int64, []int64, error) {
	mtime, size, err := statOf(path)
	if err != nil {
		return nil, nil, err
	}
	var id, oldM, oldS int64
	err = ix.DB.QueryRow(`SELECT id, mtime_ns, size FROM sources WHERE path=?`, path).Scan(&id, &oldM, &oldS)
	if errors.Is(err, sql.ErrNoRows) {
		err = ix.step(ctx, func() error {
			var e error
			id, oldM, oldS, e = ix.DB.Source(kind, path)
			return e
		})
	}
	if err != nil {
		return nil, nil, err
	}
	if oldM == mtime && oldS == size {
		st.SourcesSkipped++
		return nil, nil, nil
	}
	st.SourcesScanned++
	existing, err := ix.DB.UnitHashes(id)
	if err != nil {
		return nil, nil, err
	}
	cands, err := parse()
	if err != nil {
		return nil, nil, err
	}
	ws := plan(existing, cands, st)
	if ctx.Value(passKey{}) == nil {
		for i := range ws {
			ws[i].noQueue = true
		}
	}
	var dirty, removed []int64
	raced := false
	err = ix.batches(ctx, len(ws), func(i int) error {
		uid, r, err := ix.writePlanned(id, ws[i], st)
		switch {
		case err != nil:
			return err
		case r:
			raced = true
		case ws[i].remove:
			removed = append(removed, uid)
		default:
			dirty = append(dirty, uid)
		}
		return nil
	}, func() error {
		if raced {
			return nil
		}
		return ix.DB.SetSourceStat(id, mtime, size)
	})
	return dirty, removed, err
}
