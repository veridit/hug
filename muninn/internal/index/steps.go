package index

import (
	"context"
	"strconv"
	"strings"
	"sync"

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

// queueNeighbors records neighbour work in index_pending, in order, inside
// the caller's step: dirty lists are recomputed, removed units detached.
func (ix *Indexer) queueNeighbors(dirty, removed []int64) error {
	for flag, ids := range [][]int64{dirty, removed} {
		for start := 0; start < len(ids); start += 300 {
			chunk := ids[start:min(start+300, len(ids))]
			var sb strings.Builder
			args := make([]any, 0, 2*len(chunk))
			sb.WriteString(`INSERT INTO index_pending(unit_id, removed) VALUES `)
			for i, id := range chunk {
				if i > 0 {
					sb.WriteByte(',')
				}
				sb.WriteString("(?,?)")
				args = append(args, id, flag)
			}
			if _, err := ix.DB.Exec(sb.String(), args...); err != nil {
				return err
			}
		}
	}
	return nil
}

// neighborStep does the queued neighbour work and clears what it did.
func (ix *Indexer) neighborStep() error {
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
	if err := ix.UpdateNeighbors(dirty, removed); err != nil {
		return err
	}
	_, err = ix.DB.Exec(`DELETE FROM index_pending WHERE seq<=?`, last)
	return err
}

// neighborsPending reports queued neighbour work (a read, no lock).
func (ix *Indexer) neighborsPending() bool {
	var one int
	return ix.DB.QueryRow(`SELECT 1 FROM index_pending LIMIT 1`).Scan(&one) == nil
}

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
