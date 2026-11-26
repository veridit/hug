---
id: t0qap9
title: "The start-up reindex writes in small transactions, never one of minutes"
state: "doing"
type: bug
priority: "high"
assignee: ["hedgehog"]
labels: []
group: ""
rank: "000001q"
created: 2026-11-25
updated: 2026-11-25
doing_since: "2026-11-26T01:59"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug's background reindex commits in batches of a few seconds each, so no hug holds the memory write lock for minutes.

## Purpose
hippo 2026-11-25 (vrx1ja): every hug mcp starts a background reindex in one write transaction; a full pass held the write lock 409.9 s on a store copy, which is what other hugs waited on.

## Principle
Short transactions; each batch is safe to stop between, so a crash or a stop loses at most one batch.

## Details
From vrx1ja: the reindex runs inside one write transaction (muninn index pipeline). Related: n7jbmx's hook p95 699 ms at load 116, and xhfte0 (SQLITE_BUSY on concurrent deletes despite 30 s wait). Fix: commit every N units or every ~2 s; measure the lock hold time before and after on a store copy.

## Depends on

## Acceptance
- [ ] On a store copy, a full reindex never holds the write lock longer than 5 s at a time (measured, numbers noted)
- [ ] A memory write from another process during a full reindex succeeds within 5 s (test)

## Notes


hedgehog: baseline measured on store copy (vrx1ja clone, hug v0.7.116): one write-lock hold 297.5 s (lockprobe n_stretches 1), index 4m57s, neighbors=4m30s. Designing batched commits now.
## History
- 2026-11-26T01:59 todo -> doing/active (take)
