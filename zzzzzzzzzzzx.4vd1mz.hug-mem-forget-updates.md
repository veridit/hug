---
id: 4vd1mz
title: "hug mem forget updates only what it touched, no full reindex"
state: todo
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-18
updated: 2026-11-18
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
