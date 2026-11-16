---
id: m8shdj
title: "hug task writes wait their turn too, and forget updates only what changed"
status: todo
type: bug
priority: "medium"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Many agents writing the task board at once all succeed, waiting their turn like hug mem writes (me51r2); and a single hug mem forget no longer runs a full index pass

## Purpose
Jørgen 2026-11-16: concurrent agents are the normal case; every hug write waits, then succeeds. me51r2 fixed hug mem writes; hug task writes still use git push as a lock (gitbranch ErrRaced) and a 5 s SQLite busy_timeout in huginn/internal/cache/cache.go:70. Also from me51r2: each forget runs a full index pass (32 s cold on the real store), so 20 queued forgets took 36 to 66 s each.

## Principle
Same model as me51r2: one cross-process write turn with a visible wait notice after a few seconds, failing only after a holder keeps it far beyond a normal write, naming the holder, with a fix: line. Forget marks the unit inactive and updates only the touched rows instead of reindexing everything. Capture's short writes join the same turn (me51r2 open question 4).

## Relevant details
me51r2 report: tmp/lanes/me51r2.md. Turn implementation: muninn/internal/store/turn.go. Task writes: huginn/gitbranch (Update, push lock), huginn/internal/cache.

## Depends on

## Acceptance
- [ ] 20 parallel hug task note commands on one board all succeed (test)
- [ ] a single hug mem forget on the real-store copy finishes in under 2 s warm (measured)
- [ ] capture's database writes take the write turn (test)

## Notes
