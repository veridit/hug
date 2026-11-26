---
id: t0qap9
title: "No hug holds a memory lock while computing: index work runs outside both locks"
state: "todo"
type: bug
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "0000003"
created: 2026-11-25
updated: 2026-11-25
stage: "ready"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
parent: "9taa9d"
---

## Goal
Memory writes never wait on index work: compute runs outside the write turn and SQLite lock; each lock hold is seconds.

## Purpose
hedgehog 2026-11-26 on a store copy: a full reindex held the SQLite write lock 297.5 s in one stretch (4m30 of it neighbour lists), and holds hug's write turn the whole pass, so remember, forget and capture wait up to 20 min.

## Principle
Compute without locks, write in short steps. Keep the neighbour lists and topic groups exactly as today (same results); change only when and how they are written. Jørgen 2026-11-26: hold the locks short and fix all the other findings.

## Details
Findings to fix (hedgehog note above): (1) App.Reindex holds the write turn for the whole pass, embed included (app/app.go:164-191); (2) Ollama embed calls run inside the transaction (EmbedDirty); (3) a remember runs a full Reindex after its write (binding.go:778, 839); (4) UpdateNeighbors does per-unit SQL inside the pass (pipeline.go:528-705); (5) Run wraps the whole pass in one BEGIN IMMEDIATE (pipeline.go:203-230). Community detection (DF-Louvain) needs all of a pass's concept deltas at once: compute in memory, then write. Once the index lives in its own database (sibling task under b8mk9s), the full rebuild builds beside and swaps; this task makes the incremental path short-locked in either layout.

## Depends on

## Acceptance
- [ ] ~~On a store copy, a full reindex never holds the write lock longer than 5 s at a time (measured, numbers noted)~~ (not relevant: replaced by check 3, which also covers the write turn, crab, 2026-11-26)
- [ ] ~~A memory write from another process during a full reindex succeeds within 5 s (test)~~ (not relevant: replaced by check 4, crab, 2026-11-26)
- [ ] On a store copy, a full reindex never holds the SQLite write lock or the write turn longer than 5 s at a time (lockprobe, numbers noted)
- [ ] A remember from another process during a full reindex succeeds within 5 s (test)
- [ ] No embed call runs inside a write transaction or under the write turn (test)
- [ ] A remember no longer runs a full reindex; it indexes only what it wrote (test)
- [ ] Neighbour lists and topic groups after a reindex equal today's on the same store (test)

## Notes


hedgehog: baseline measured on store copy (vrx1ja clone, hug v0.7.116): one write-lock hold 297.5 s (lockprobe n_stretches 1), index 4m57s, neighbors=4m30s. Designing batched commits now.
hedgehog, stopped on crab's word (2026-11-26 07:26Z, Jørgen is reconsidering the design). No code was written or committed. Worktree /Users/jhf/.jcode/scratch/wt-t0qap9 (branch t0qap9, at origin/main 29ced89c v0.7.116) is clean and left as is.

MEASURED (store copy: APFS clone of vrx1ja-store at a new path, so the native source is new: every native unit was added and the old source pruned; hug v0.7.116 hug-base; GOMAXPROCS=4 nice 19; load about 24). Tools: /Users/jhf/.jcode/scratch/t0qap9/run-full.sh BIN TAG and lockprobe.py (tries BEGIN IMMEDIATE with no wait every 50 ms, records the stretches someone else held the lock).
- `hug mem index`: 'sources scanned=1 skipped=0 | units added=22064 changed=0 removed=0 unchanged=0 | embedded=0 reused=38649 | 4m57.198s sources=21.61s communities=5.137s embed=423ms neighbors=4m30.027s', wall 299.797 s.
- lockprobe: one stretch, longest_s 297.499, n_stretches 1 (probe-base.json). The whole pass is one write-lock hold.
- Store sizes: 80,973 active units; qvecs 38,654 rows (about 30 MB, all on the current pipeline_hash); neighbors 1,295,568 rows; graph_vertex 87,210 rows (12 MB adj); concepts and communities 87,210 each; unit_communities 604k; unit_concepts 2.3M.
- Read costs on the copy: every full-table select used by a pass (active vectors, all neighbors, communities) under 0.8 s; DELETE FROM neighbors about 0.68 s.
NOT MEASURED: the CPU vs SQL split inside neighbors=4m30s. No profile was taken. What follows on it is from reading the code, not timing.

WHERE THE ONE TRANSACTION IS: muninn/internal/index/pipeline.go Run (lines 202-242): BEGIN IMMEDIATE, ix.run, COMMIT; store.Open caps the pool at one connection (store.go:297 SetMaxOpenConns(1)), so the BEGIN brackets every statement of the pass. The comment says why: per-statement fsyncs were 92% of cold indexing. Inside it, in order: moveGraph (gob to tables, once), source_recipe reset, IndexJcode/IndexNative/IndexMarkdown (UpsertUnit, setUnitConcepts, SetUnitAnchors per changed unit), prunedSources, BackfillAnchors when anchor_recipe changed (DELETE FROM unit_anchors then re-insert for every active unit, pipeline.go:392-426), applyDeltas (DF-Louvain, saveGraph), EmbedDirty (Ollama calls INSIDE the transaction, embed.BatchSize 32 per call; MaxEmbed caps it, 8 in ensureFresh main.go:288, 1 in capture.go:123), then UpdateNeighbors.

UpdateNeighbors (pipeline.go:528-705), for this run's frontier (22,064 dirty units plus the pruned source's removed units), from the code:
- SQL, per unit: detach loop does one SELECT unit_id FROM neighbors WHERE nbr_id=? and one DELETE ... WHERE unit_id=? OR nbr_id=? for every dirty and removed unit (lines 561-575).
- CPU: topK over all 38.6k vectors for every dirty and refill unit, parallel across cores (lines 599-604).
- SQL, per list: writeList for each recomputed list: SAVEPOINT, DELETE, one multi-row INSERT, RELEASE (lines 739-812).
- SQL then CPU: reads all 1.3M neighbor rows into memory (line 642), then offers every active dirty unit to every other list: about 38.6k lists x 22k offers DotQ, plus a linear slices.IndexFunc per offer (lines 662-696). This is the likely bulk; unverified.
- SQL: writeList again for every changed list.
- pipeline switch (new model or prompt): DELETE FROM neighbors and every active unit becomes dirty (pipeline.go:350-367).

WHO READS THE NEIGHBORS TABLE:
- search.go:584 Searcher.RelatedIn: SELECT nbr_id, sim FROM neighbors WHERE unit_id=? ORDER BY sim DESC (related memories).
- tidy/tidy.go:211 ReadGraph: SELECT unit_id,nbr_id,sim FROM neighbors ORDER BY unit_id,sim DESC,nbr_id, in a tx.
- app/stats.go:57: count for hug mem stats.
- index itself (UpdateNeighbors, above); scopetest/fixture.go:157 inserts rows in tests.

THE WRITE TURN, A SECOND LOCK: App.Reindex (app/app.go:164-191) takes the flock write turn (store/turn.go, <db>.write.lock plus <db>.writer holder JSON) for the whole pass, embed included, then reindexMu, then the SQLite lock. Every memory write flow waits on that turn: RememberPlaced (binding.go:669), forget (memories.go:44 and applyNative :193), rescope, tidy, correction, capture (capture.go:249/578/749), capture_audit. Waiters give up only after TurnStuckAfter = 20 min. So batching SQLite commits alone does not let a remember in: the turn must also be released between batches, or not held for the compute.
Also: remember itself runs a full Reindex after its write (binding.go:778, 839), so a remember on a store with pending work runs the whole pass under its own turn. ensureFresh/NoWait passes skip with ErrBusy instead of queueing (TryWriteTurn, then busy_timeout=0 BEGIN).

OTHER WRITE TRANSACTIONS FOUND:
- store.Open migrations under BEGIN IMMEDIATE (6gtv7a; read-first since v0.7.116, gives up after MigrateWait 5 s with ErrBusy).
- ApplyNative (pipeline.go:121-138), one tx per forget, scoped to that forget's units.
- MoveGraph (graphstore.go:322-344), one tx, once per store.
- capture_batch.go:225 and memories.go:110: txs that only UPDATE units SET active=0 and insert suppressed / capture_reinforcements rows.
- shortids.go:207 sid migration inside Open's tx.
None of these was timed.

STATE OF THE DESIGN WHEN STOPPED (not started in code): per-stage bounded commits; embedding outside any tx; neighbor work resumable through dirty markers; release turn and SQLite lock between batches. Open questions the redesign should settle: DF-Louvain applies all deltas at once, so batching sources changes community results unless deltas are kept until one graph update; per-batch reloads of vectors and neighbors cost under 0.8 s each on this store.
Test patterns to reuse: index/busy_test.go, knn_test.go (bruteNeighbors, neighborChurn), partial_neighbors_test.go, run/concurrent_writes_test.go (runCLI), testutil.FakeEmbedder; speed tests must call timing.Budget(t) (internal/repocheck/timing_test.go).
Checks 1 and 2 are not ticked.
## History
- 2026-11-26T01:59 todo -> doing/active (take)
- 2026-11-26T09:17 doing/active -> todo/ready (give back; was assignee: hedgehog); was doing_since: 2026-11-26T01:59
