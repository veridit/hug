package index

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/veridit/hug/internal/flock"
	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/store"
)

// An index pass is a series of short write steps (t0qap9). Each step holds
// hug's write turn and one SQLite write transaction for its writes only;
// the embedder is called between steps, holding neither. A pass used to be
// one transaction under one turn: on a 80,000-memory store that was one
// write-lock hold of 297.5 s, and every remember, forget and capture waited.
//
// Lock order: the pass lock (one pass at a time per database, so two
// processes do not do the same work twice), then per step the write turn,
// then the step mutex (the one pooled connection carries one transaction),
// then BEGIN IMMEDIATE. A step never waits for the pass lock, so a flow
// that holds the turn can run steps while another process's pass waits
// between its own.

// passLocks is one in-process mutex per database path: flock is per open
// file, so a second pass in this process would block on its own lock.
var passLocks sync.Map

// passLock takes the pass lock: wait, or with noWait report ok=false at once.
func (ix *Indexer) passLock(noWait bool) (release func(), ok bool, err error) {
	v, _ := passLocks.LoadOrStore(ix.DB.Path, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	if noWait {
		if !mu.TryLock() {
			return nil, false, nil
		}
		unlock, held, err := flock.TryAcquire(ix.DB.Path + ".index.lock")
		if err != nil || !held {
			mu.Unlock()
			return nil, false, err
		}
		return func() { unlock(); mu.Unlock() }, true, nil
	}
	mu.Lock()
	unlock, err := flock.Acquire(ix.DB.Path + ".index.lock")
	if err != nil {
		mu.Unlock()
		return nil, false, err
	}
	return func() { unlock(); mu.Unlock() }, true, nil
}

// passRunning reports whether a pass holds the pass lock now, in this
// process or another. It only tries the lock, so it may run under the turn.
func (ix *Indexer) passRunning() bool {
	release, ok, err := ix.passLock(true)
	if err != nil {
		return false
	}
	if ok {
		release()
	}
	return !ok
}

// step runs f as one write step: the write turn (unless ctx carries it),
// the step mutex, one BEGIN IMMEDIATE, f, COMMIT. With NoWait a held turn
// or write lock is ErrBusy at once.
func (ix *Indexer) step(ctx context.Context, f func() error) error {
	if !ix.DB.HasTurn(ctx) {
		var release func()
		if ix.NoWait {
			c, r, ok, err := ix.DB.TryWriteTurn(ctx)
			if err != nil {
				return err
			}
			if !ok {
				return ErrBusy
			}
			ctx, release = c, r
		} else {
			c, r, err := ix.DB.WriteTurn(ctx)
			if err != nil {
				return err
			}
			ctx, release = c, r
		}
		defer release()
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if ix.NoWait {
		// busy_timeout applies per statement; drop it for the lock attempt,
		// then restore store.Open's value.
		ix.DB.Exec(`PRAGMA busy_timeout=0`)
		_, err := ix.DB.Exec(`BEGIN IMMEDIATE`)
		ix.DB.Exec(`PRAGMA busy_timeout=` + strconv.Itoa(store.BusyTimeout))
		if err != nil {
			if strings.Contains(err.Error(), "SQLITE_BUSY") || strings.Contains(err.Error(), "locked") {
				return ErrBusy
			}
			return err
		}
	} else if _, err := ix.DB.Exec(`BEGIN IMMEDIATE`); err != nil {
		return err
	}
	if err := f(); err != nil {
		ix.DB.Exec(`ROLLBACK`)
		return err
	}
	if _, err := ix.DB.Exec(`COMMIT`); err != nil {
		ix.DB.Exec(`ROLLBACK`)
		return err
	}
	return nil
}

// stepBudget bounds one write step of a batched write: a step stops taking
// items once it has run this long, commits, and lets the turn go, so a
// memory write from another process gets in between (t0qap9 check 4).
var stepBudget = 250 * time.Millisecond

// stepGap is how long a batched write waits between two of its steps
// without the turn, so a writer blocked on the turn is woken and takes it
// before the next step asks again: flock does not queue its waiters.
var stepGap = 5 * time.Millisecond

// batches runs each(0..n-1) in as few write steps as stepBudget allows and
// last, when set, in the step that runs the final item (or alone, n = 0).
// A step that fails rolls back its own items only: the caller's queued work
// and unrecorded stats make the next pass redo the rest.
func (ix *Indexer) batches(ctx context.Context, n int, each func(i int) error, last func() error) error {
	i := 0
	for {
		done := false
		if err := ix.step(ctx, func() error {
			t := time.Now()
			for i < n {
				if err := each(i); err != nil {
					return err
				}
				i++
				if i < n && time.Since(t) > stepBudget {
					return nil
				}
			}
			done = true
			if last != nil {
				return last()
			}
			return nil
		}); err != nil {
			return err
		}
		if done {
			return nil
		}
		if !ix.DB.HasTurn(ctx) {
			time.Sleep(stepGap)
		}
	}
}

// queueDelta records a unit's concept-set change for the communities step,
// inside the caller's step.
func (ix *Indexer) queueDelta(unitID int64, old, new []int) error {
	// An empty delta changes no edge, but its unit's community counts are
	// refreshed with the rest, as a one-transaction pass did.
	_, err := ix.DB.Exec(`INSERT INTO index_deltas(unit_id, old, new) VALUES(?,?,?)`, unitID, joinIDs(old), joinIDs(new))
	return err
}

// queueList records that a unit's neighbour list is recomputed (removed:
// the unit is gone and its edges are detached), inside the caller's step.
func (ix *Indexer) queueList(unitID int64, removed bool) error {
	r := 0
	if removed {
		r = 1
	}
	_, err := ix.DB.Exec(`INSERT INTO index_pending(unit_id, removed) VALUES(?,?)`, unitID, r)
	return err
}

func joinIDs(ids []int) string {
	var sb strings.Builder
	for i, id := range ids {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.Itoa(id))
	}
	return sb.String()
}

func splitIDs(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

// deltasPending reports queued concept-set changes (a read, no lock).
func (ix *Indexer) deltasPending() bool {
	var one int
	return ix.DB.QueryRow(`SELECT 1 FROM index_deltas LIMIT 1`).Scan(&one) == nil
}

// communityStep applies the queued concept deltas to the concept graph as
// one DF-Louvain update, the update a one-transaction pass made: read and
// computed with no lock, then saved in one short step that also clears the
// deltas it applied. It reports whether there were any.
//
// Only this step writes the graph, under the pass lock, so nothing of this
// hug's changes it between the read and the save. A hug from before
// t0qap9 still updates the graph itself; when meta graph_m (the graph's
// total weight, which every update rewrites) changed meanwhile, the update
// is redone inside the step from the graph as it is then. Edge weights are
// sums of unit deltas, so applying ours after its own leaves them right.
func (ix *Indexer) communityStep(ctx context.Context) (bool, error) {
	rows, err := ix.DB.Query(`SELECT seq, unit_id, old, new FROM index_deltas ORDER BY seq`)
	if err != nil {
		return false, err
	}
	var deltas []UnitDelta
	units := map[int64]bool{}
	var last int64 = -1
	for rows.Next() {
		var seq, id int64
		var old, new string
		if err := rows.Scan(&seq, &id, &old, &new); err != nil {
			rows.Close()
			return false, err
		}
		deltas = append(deltas, UnitDelta{Old: splitIDs(old), New: splitIDs(new)})
		units[id] = true
		last = seq
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	if last < 0 {
		return false, nil
	}
	if ix.GraphNeedsMove() {
		if _, err := ix.MoveGraph(); err != nil {
			return true, err
		}
	}
	m0 := ix.DB.GetMeta("graph_m")
	g, err := ix.loadGraph()
	if err != nil {
		return true, err
	}
	g.Update(deltas)
	if g.err != nil {
		return true, g.err
	}
	loads := g.Loads
	err = ix.step(ctx, func() error {
		if ix.DB.GetMeta("graph_m") != m0 {
			g2, err := ix.loadGraph()
			if err != nil {
				return err
			}
			g2.Update(deltas)
			g, loads = g2, loads+g2.Loads
		}
		if err := ix.saveGraph(g, units); err != nil {
			return err
		}
		_, err := ix.DB.Exec(`DELETE FROM index_deltas WHERE seq<=?`, last)
		return err
	})
	if err == nil {
		ix.mu.Lock()
		ix.GraphLoads += loads
		ix.mu.Unlock()
	}
	return true, err
}

// neighborStep does the queued neighbour work: the lists are computed with
// no lock (planNeighbors), then written in short steps; the last one
// clears the work it did.
func (ix *Indexer) neighborStep(ctx context.Context) error {
	rows, err := ix.DB.Query(`SELECT seq, unit_id, removed FROM index_pending ORDER BY seq`)
	if err != nil {
		return err
	}
	var dirty, removed []int64
	var last int64 = -1
	for rows.Next() {
		var seq, id int64
		var r int
		if err := rows.Scan(&seq, &id, &r); err != nil {
			rows.Close()
			return err
		}
		if r == 1 {
			removed = append(removed, id)
		} else {
			dirty = append(dirty, id)
		}
		last = seq
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if last < 0 {
		return nil
	}
	lists, err := ix.planNeighbors(dirty, removed)
	if err != nil {
		return err
	}
	return ix.writeLists(ctx, lists, func() error {
		_, err := ix.DB.Exec(`DELETE FROM index_pending WHERE seq<=?`, last)
		return err
	})
}

// neighborsPending reports queued neighbour work (a read, no lock).
func (ix *Indexer) neighborsPending() bool {
	var one int
	return ix.DB.QueryRow(`SELECT 1 FROM index_pending LIMIT 1`).Scan(&one) == nil
}

// workPending reports queued community or neighbour work.
func (ix *Indexer) workPending() bool { return ix.deltasPending() || ix.neighborsPending() }

// embedJob is one text to embed and the content hash its vector is stored under.
type embedJob struct{ hash, text string }

// embedSteps embeds units that lack a vector under the current pipeline.
// Identical text across units shares one vector (content-addressed). What
// to embed is read with no lock, the embedder runs with no lock, and each
// batch's vectors are written in one step that also queues the neighbour
// work they make. The step that completes a new pipeline switches to it.
func (ix *Indexer) embedSteps(ctx context.Context, st *Stats) error {
	if ix.Embedder == nil {
		return nil
	}
	pipeline := PipelineHash(ix.Embedder.Model())
	rows, err := ix.DB.Query(`SELECT DISTINCT u.content_hash, u.title, u.body FROM units u
	  LEFT JOIN embeddings e ON e.content_hash=u.content_hash AND e.pipeline_hash=?
	  WHERE e.content_hash IS NULL`, pipeline)
	if err != nil {
		return err
	}
	var jobs []embedJob
	for rows.Next() {
		var h, t, b string
		if err := rows.Scan(&h, &t, &b); err != nil {
			rows.Close()
			return err
		}
		jobs = append(jobs, embedJob{h, EmbeddingText(t, b)})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var total int
	ix.DB.QueryRow(`SELECT COUNT(DISTINCT content_hash) FROM units`).Scan(&total)
	st.EmbedReused = total - len(jobs)
	// A hook must answer in milliseconds: it embeds at most MaxEmbed texts and
	// leaves the rest (e.g. a full re-embed after a model change) to
	// `hug mem index` or `hug mem serve`. Search keeps using the previous
	// pipeline until every vector of the new one exists.
	pending := len(jobs)
	if ix.MaxEmbed > 0 && len(jobs) > ix.MaxEmbed {
		jobs = jobs[:ix.MaxEmbed]
	}
	for start := 0; start < len(jobs); start += embed.BatchSize {
		end := min(start+embed.BatchSize, len(jobs))
		texts := make([]string, 0, end-start)
		for _, j := range jobs[start:end] {
			texts = append(texts, j.text)
		}
		vecs, err := ix.Embedder.Embed(ctx, embed.Documents(ix.Embedder, texts))
		if err != nil {
			st.EmbedErr = err
			return nil // degrade to keyword-only; never fail indexing
		}
		batch := jobs[start:end]
		if err := ix.step(ctx, func() error { return ix.putVectors(pipeline, batch, vecs) }); err != nil {
			return err
		}
		st.Embedded += len(vecs)
	}
	if st.Embedded < pending {
		st.EmbedPending = pending - st.Embedded
		return nil
	}
	prefix := embed.PromptsFor(ix.Embedder.Model()).Query
	if ix.DB.GetMeta("pipeline_hash") == pipeline && ix.DB.GetMeta("pipeline_query_prefix") == prefix {
		return nil
	}
	return ix.step(ctx, func() error {
		if ix.DB.GetMeta("pipeline_hash") != pipeline {
			// New model or prompt recipe: every stored similarity is stale,
			// so every neighbour list is rebuilt once.
			for _, q := range []string{`DELETE FROM neighbors`, `DELETE FROM index_pending`,
				`INSERT INTO index_pending(unit_id, removed) SELECT id, 0 FROM units WHERE active=1 ORDER BY id`} {
				if _, err := ix.DB.Exec(q); err != nil {
					return err
				}
			}
		}
		if err := ix.DB.SetMeta("pipeline_query_prefix", prefix); err != nil {
			return err
		}
		return ix.DB.SetMeta("pipeline_hash", pipeline)
	})
}

// putVectors stores one embedded batch. Once the pipeline is current, a new
// vector is a graph delta for every active unit with that text, even if no
// source text changed in this pass (a capped earlier pass checkpointed the
// source before its vector existed). Staging vectors of a pipeline not yet
// current queue nothing: the switch rebuilds every list.
func (ix *Indexer) putVectors(pipeline string, jobs []embedJob, vecs [][]float32) error {
	current := ix.DB.GetMeta("pipeline_hash") == pipeline
	for i, v := range vecs {
		if err := ix.DB.PutEmbedding(jobs[i].hash, pipeline, len(v),
			embed.EncodeF32(v), embed.Quantize(v), embed.Sketch(v)); err != nil {
			return err
		}
		if current {
			if _, err := ix.DB.Exec(`INSERT INTO index_pending(unit_id, removed)
			  SELECT id, 0 FROM units WHERE active=1 AND content_hash=? ORDER BY id`, jobs[i].hash); err != nil {
				return err
			}
		}
	}
	return nil
}
