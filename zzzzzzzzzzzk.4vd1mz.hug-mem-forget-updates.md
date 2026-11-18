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
## History
- 2026-11-18T02:53 todo -> doing/active (take)
- 2026-11-18T03:41 doing/active -> todo/ready (give back; was assignee: wyvern); was doing_since: 2026-11-18T02:53
- 2026-11-18T04:12 todo/ready -> doing/active (take)
- 2026-11-18T04:26 doing/active -> todo/ready (give back; was assignee: wyvern); was doing_since: 2026-11-18T04:12
