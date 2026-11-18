---
id: 4vd1mz
title: "hug mem forget updates only what it touched, no full reindex"
state: "todo"
type: bug
priority: "high"
assignee: []
labels: []
created: 2026-11-18
updated: 2026-11-18
stage: "ready"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
One hug mem forget on Jørgen's store finishes well under a second: it marks the memory inactive and updates only that memory's index entries and neighbours, with no full index pass before or after.

## Purpose
Measured by hatchling 2026-11-18 (66ex4y, file:forget-time.txt, forget-pprof.txt): every forget reindexes twice (run/main.go:376 before, ForgetMany after the write); one forget takes 2.91-3.20 s wall and 948 MB max RSS on 13,685 memories, while a no-change hug mem index takes 0.48 s. Post-write Reindex 1.42 s (UpdateNeighbors 0.58, IndexNative 0.29, loadGraph 0.20), memories.json load+save 0.33 s. Jørgen wants sub-second commands; the daily check alone runs about 25 forgets.

## Principle
Do the work the change needs and nothing more: a forget touches one memory, so it updates that memory's entries. Recall results stay identical to a full reindex (no drift). Tests count work, never time: no time budgets.

## Relevant details
Code: muninn run/main.go:376 (pre-forget reindex), ForgetMany (post-write Reindex), UpdateNeighbors, IndexNative, loadGraph. Batch forgets (forget a b c) also do one bounded update, not one per id.

## Depends on

## Acceptance
- [ ] A test counts full index passes: hug mem forget ID does 0 (fails first on main, which does 2)
- [ ] After N forgets the index equals a full reindex of the same store, byte for byte or by recall results on a fixed query set (test)
- [ ] On a copy of Jørgen's store, one forget's wall time and max RSS before and after, 4 runs each, in a note (expect well under 1 s)

## Notes

crab 02:52 local: moved from hatchling to wyvern (hatchling is on t01kqm; wyvern owns speed). hatchling's measurements on 66ex4y stay the baseline.


wyvern 03:41: work in progress parked as commit 80e31e6a on local branch wy/4vd1mz (worktree ~/.jcode/scratch/hug-0sbvnq-t4). It adds ingest.ForgetManyChanged, Indexer.ApplyNative (targeted update), App.FreshFor, and an early exit in UpdateNeighbors when there are no offers. On the store copy (30,794 units) the index stays consistent: the pass after the forget skips the source. But wall time is not yet sub-second: 1.95-3.6 s at load 50-120, and the profile shows refill topK over every vector plus loadGraph/saveGraph of the 28 MB gob as most of it. Paused for a0p5kt (crab: land first). Also waiting on hatchling (tn8bmn touches the same forget path).
wyvern 04:26: fails-first: on main 0a2f06ae TestForgetRunsNoIndexPass fails 'hug mem forget ID ran 2 index passes, want 0' (also batch 2, after-unindexed-write 2 want 1). Unrolled DotQ added: TestDotQMatchesPlainLoop bit-identical over dims 0-1030; mutation (drop x[7]*y[7]) fails 'dim 8: DotQ 4.1263275, plain 4.7158027'. Paired (pair3.txt, load 38-47): main 3.38-3.95 s / 953-960 MB, new 2.38-2.80 s / 838-871 MB. Remaining cost is the concept graph gob (30.7 MB, 83,245 vertices, 5,027,834 directed edges) decoded+encoded whole: same binary with the graph update skipped gives 1.56-1.68 s / 353-367 MB vs 2.60-3.50 s / 845-865 MB (bound.sh). Next: per-vertex SQLite graph storage, Louvain loads only touched vertices. Floor: hug mem show 0.29-0.42 s. WIP on wy/4vd1mz 2 commits. Parked for mmm8c8 (crab: flaky settle test first).
crab 04:25 approved per-vertex graph storage with conditions: (1) a test that communities and K/M after migration equal the gob's, plus a recall4 replay on real messages byte-identical before/after; (2) migration through hug upgrade (preview, then --apply), graph_state dropped after, no dual path; (3) profile what remains of the 1.56 s without the graph and name it in a note before pushing.
parked for mmm8c8 (crab 04:25: flaky settle test first while 4vd1mz is far from <1 s); WIP on wy/4vd1mz 41b0fc50
crab 04:44 revised condition (2): the graph is derived (an index), so it converts on first use. The first writer that finds graph_state converts it under the index lock, logs one line (hook.log and hug mem stats: 'concept graph moved to tables, N vertices, X s') and removes graph_state. No dual path, no refusal, no hug upgrade step. If conversion takes >1 s, hug mem index and hug upgrade may do it first, but a hook must still cope. Sorted frontier approved. Recall: (1) a recall4 replay byte-identical across the migration itself; (2) for the order change, two replays of main show main's own spread, and the sorted order must fall inside it. Facts on the store copy (pristine.db): communities table == gob Community for all 83,245 concepts (0 differ, 0 missing); every edge weight is integral; max degree 9,244; M 6,926,923; 82,882 of 83,245 communities have SumIn 0; scanning the communities table takes 40 ms. Design: graph_vertex(concept_id, community, k), fully loaded; graph_adj(concept_id, packed blob), loaded per vertex on use; graph_comms(community_id, sum_tot, sum_in), fully loaded; M in meta. The communities table search reads keeps main's write rule unchanged.
wyvern 05:32, WIP 3e553e42 (wy/4vd1mz): graph tables, moved on first use, Louvain in id order, bounded topK (TestTopKMatchesFullSort compares it with the full sort, ties included). On the store copy the move takes 2.0 s and 620 MB, once, inside the first index pass ('concept graph moved to tables, 83245 vertices, 2.0 s' in hook.log). Adjacency packs to 10.7 MB (the gob was 30.7 MB). Paired forgets, pair5.txt, load 12-16: cold cache, main 3.45-3.66 s / 952-960 MB vs new 1.55-1.73 s / 356-369 MB; warm cache (db read once first), main 2.39-2.63 s vs new 0.91-1.01 s / 356-367 MB. Remaining cold wall time per step (TEMP lap timers, 3 runs, now removed): project lookup by git subprocess run 3 times (withApp Bound 105-121 ms incl. open, view.binding 72-80 ms, Refresh 73-84 ms); memories.json load+save 232-251 ms; graph 109-172 ms; neighbour update 745-1197 ms, of which activeVectors is most (the qvec read cold 635 ms vs warm 68 ms: qvec sits after the 3 KB float vec in each embeddings row, so it lives on overflow pages) and topK+write 28-147 ms; readback reload of memories.json 120-123 ms. Next: qvecs in a narrow table (as sketches are), readback skipped when the file stat equals the write's own After, project lookup once per command.
wyvern 05:43, branch wy/4vd1mz 0f4f485a (rebased on e36ecca4), pair7.txt: 4 paired forgets on the 30,794-unit store copy (load 7.9-9.2). hug-main on the gob store vs this branch on the moved store (pristine-moved2.db, graph and qvecs moved once). Cold page cache: main 3.34-3.83 s / 952-959 MB, new 0.81-0.95 s / 214-238 MB. Warm: main 2.39-2.59 s, new 0.60-0.72 s / 206-226 MB. Every forget printed 'forgot ID', and the next index found scanned=0. Changes since 05:32: qvecs narrow table (cold read of all active qvecs 635 -> ~430 ms in sqlite3; backfilled once at open, 1.95 s on the copy); the forget readback skips the reload when the file stat under its lock equals the stat its own save left (TestForgetReadsStoreOnce: 1 load, main 2; TestForgetUndoneByAnotherWriterFails still fails the forget); project.Find kept 2 s per folder within a process (TestFindAgainStartsNoGit: 0 git processes on a repeat, main 2). TestPlacementLooksUpTheProjectOnce now asserts at most 1 lookup (0 when Bound found it first). The recall4 replay (main x2 vs moved) is running.
wyvern 05:44, mutations (mutate.txt, mutate2.txt; each applied to the 0f4f485a tree, then restored). Each new test fails with its mutation: (1) Louvain frontier in Go map order (as main): TestCommunitiesAreDeterministic fails 10 of 10 ('run 0: two indexes of one store have different communities'). Candidate order alone back in map order: passes, because the sorted frontier already decides ties there (kept sorted anyway, same cost). (2) topK tie order flipped: TestTopKMatchesFullSort 'n=5 query 3'. (3) Every vertex loaded up front: TestForgetLoadsOnlyTouchedVertices 'one forget loaded 1536 of 1536 vertices' (with the fix: 10 of 1536 for a 10-concept memory). (4) Move writes SumTot+1: TestGobGraphMovesUnchanged 'communities or their totals changed in the move'. (5) Readback always reloads: TestForgetReadsStoreOnce 'loaded the store 2 times, want 1'. (6) Readback trusts without the stat: TestForgetUndoneByAnotherWriterFails 'forget reported success it does not have'. (7) Find keeps failures: TestFindAgainStartsNoGit 'first Find after git init: {...} <nil>, 0 git processes'. (8) Find memo off: 'Find 2 again: ...'.
Parked for 6ep7y2 (crab 04:54: Windows queue stall first). State: branch wy/4vd1mz a751c26e (WIP, not for main): graph move wired into hug mem upgrade --apply (preview prints a line, exit unchanged), hug mem index prints the move line, hug mem stats graph_moved; tests TestMemUpgradeMovesConceptGraph, TestMemIndexSaysTheGraphMoved; 5 mutations each fail (scratch 4vd1mz/mutate3.txt). hook.log path verified: config.DBPath() = DataDir()/muninn.db, and the run test reads DataDir()/hook.log. Recall replay, development split (200 messages, 105 with hits, 220 hits): A (main), B (main), C (branch after move) byte-identical after removing latency_ms and response.elapsed, sha 790ab703ecf0 for all three (norm.sh). Heldout split still running (bg 929662zp55). Sorted-order spread check still to design: main runs Louvain only on deltas.
## History
- 2026-11-18T02:53 todo -> doing/active (take)
- 2026-11-18T03:41 doing/active -> todo/ready (give back; was assignee: wyvern); was doing_since: 2026-11-18T02:53
- 2026-11-18T04:12 todo/ready -> doing/active (take)
- 2026-11-18T04:26 doing/active -> todo/ready (give back; was assignee: wyvern); was doing_since: 2026-11-18T04:12
- 2026-11-18T04:38 todo/ready -> doing/active (take)
- 2026-11-18T06:31 doing/active -> todo/ready (give back; was assignee: wyvern); was doing_since: 2026-11-18T04:38
