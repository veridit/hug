package index

import (
	"context"
	"fmt"
	"os"
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

	deltas []UnitDelta // concept-set changes collected during a write step
	// mu is the step mutex: the one pooled connection carries one write
	// transaction, and a write step's per-pass state (deltas, the graph
	// move receipt) is touched under it.
	mu sync.Mutex
	// owed is set when a pass under a held write turn left embedding and
	// neighbour work for Finish (t0qap9).
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

// SyncSource reconciles one source with its current candidates. Units whose
// content hash is unchanged are not touched at all: no write, no re-embed, no
// graph work. Only dirty units flow onward (seid search's incremental rule).
func (ix *Indexer) SyncSource(sourceID int64, cands []Candidate, st *Stats) (dirty []int64, removed []int64, err error) {
	existing, err := ix.DB.UnitHashes(sourceID)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	for _, c := range cands {
		seen[c.Key] = true
		old, had := existing[c.Key]
		id, changed, err := ix.syncCandidate(sourceID, c, old, had, st)
		if err != nil {
			return nil, nil, err
		}
		if changed {
			dirty = append(dirty, id)
		}
	}
	for key, old := range existing {
		if seen[key] {
			continue
		}
		id, _ := strconv.ParseInt(old[0], 10, 64)
		oldConcepts, err := ix.unitConcepts(id)
		if err != nil {
			return nil, nil, err
		}
		ix.deltas = append(ix.deltas, UnitDelta{Old: oldConcepts})
		if err := ix.DB.DeleteUnit(id); err != nil {
			return nil, nil, err
		}
		removed = append(removed, id)
		st.Removed++
	}
	return dirty, removed, nil
}

// syncCandidate reconciles one candidate with its stored unit (old = (id,
// ident) when had). It reports the unit id and whether it changed; an
// unchanged unit is not touched.
func (ix *Indexer) syncCandidate(sourceID int64, c Candidate, old [2]string, had bool, st *Stats) (int64, bool, error) {
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
	if had && old[1] == ident {
		st.Unchanged++
		return 0, false, nil
	}
	var oldConcepts []int
	if had {
		oldID, _ := strconv.ParseInt(old[0], 10, 64)
		var err error
		if oldConcepts, err = ix.unitConcepts(oldID); err != nil {
			return 0, false, err
		}
	}
	id, err := ix.DB.UpsertUnit(store.Unit{
		SourceID: sourceID, SID: c.SID, Origin: c.Origin, Key: c.Key, ContentHash: ContentHash(text), Ident: ident, Kind: c.Kind,
		Title: c.Title, Body: c.Body, Category: c.Category, Type: c.Type, Holds: c.Holds, Trigger: c.Trigger, Tags: c.Tags, Scope: c.Scope,
		Active: c.Active, Updated: c.Updated,
	})
	if err != nil {
		return 0, false, err
	}
	var newConcepts []int
	if c.Active {
		if newConcepts, err = ix.setUnitConcepts(id, Concepts(c.Title+" "+c.Body+" "+strings.Join(c.Tags, " "))); err != nil {
			return 0, false, err
		}
	} else if _, err = ix.setUnitConcepts(id, nil); err != nil {
		return 0, false, err
	}
	var anchors []string
	if c.Active {
		anchors = anchor.Memory(c.Title + " " + c.Body)
	}
	if err := ix.DB.SetUnitAnchors(id, anchors); err != nil {
		return 0, false, err
	}
	ix.deltas = append(ix.deltas, UnitDelta{Old: oldConcepts, New: newConcepts})
	if had {
		st.Changed++
	} else {
		st.Added++
	}
	return id, true, nil
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
		d, r, err := ix.indexFile(path, "jcode", st, func() ([]Candidate, error) {
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
	dirty, removed, err := ix.indexFile(path, "native", st, func() ([]Candidate, error) {
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
		if err := ix.DB.QueryRow(`SELECT u.sid FROM units u JOIN sources s ON s.id=u.source_id WHERE s.path=? AND u.key=?`, path, loaded[i].ID).Scan(&sid); err != nil {
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
		d, r, err := ix.indexFile(path, "markdown", st, func() ([]Candidate, error) {
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

// indexFile skips a file whose (mtime, size) is unchanged; otherwise it parses
// and reconciles it.
func (ix *Indexer) indexFile(path, kind string, st *Stats, parse func() ([]Candidate, error)) ([]int64, []int64, error) {
	mtime, size, err := statOf(path)
	if err != nil {
		return nil, nil, err
	}
	id, oldM, oldS, err := ix.DB.Source(kind, path)
	if err != nil {
		return nil, nil, err
	}
	if oldM == mtime && oldS == size {
		st.SourcesSkipped++
		return nil, nil, nil
	}
	st.SourcesScanned++
	cands, err := parse()
	if err != nil {
		return nil, nil, err
	}
	dirty, removed, err := ix.SyncSource(id, cands, st)
	if err != nil {
		return nil, nil, err
	}
	return dirty, removed, ix.DB.SetSourceStat(id, mtime, size)
}
