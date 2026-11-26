---
id: 1qe1xk
title: "Index in its own database, rebuilt beside the old one and swapped in"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-26
updated: 2026-11-26
---

## Goal
The derived index lives in index.db, apart from memories; a full rebuild writes a new file and swaps it in.

## Purpose
Jørgen 2026-11-26: two databases. The index has one writer and can always be rebuilt from the memories, so a rebuild never needs to block a memory write.

## Principle
Memories are the truth; index.db is derived and disposable. A full rebuild builds index.db.new, then one rename; readers keep the old file until then. Incremental updates stay short-locked (t0qap9).

## Details
index.db holds vectors, full-text search, concepts and communities, and neighbour lists. Store copy 2026-11-26 (1 GB muninn.db): capture bookkeeping ~430 MB (capture_runs 145, capture_seen 87+96, capture_backlog 50+55), vectors 213 MB (embeddings 169 + qvecs 44), units 66 MB, neighbors 30+21 MB. Decide where capture bookkeeping lives (with memories or a third file) and say why. A pipeline or embedder switch (dt02yw) becomes a side rebuild.

## Depends on

## Acceptance
- [ ] A full rebuild runs while another process remembers and searches, and neither waits more than 1 s (test)
- [ ] A crash mid-rebuild leaves the old index in use and the next run starts clean (test)
- [ ] Search, related memories and tidy give the same results from index.db as from today's muninn.db on a store copy (test)

## Notes
