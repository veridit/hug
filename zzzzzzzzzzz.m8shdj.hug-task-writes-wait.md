---
id: m8shdj
title: "hug task writes wait their turn too, and forget updates only what changed"
status: "doing"
type: bug
priority: "medium"
assignee: ["dragon"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [x] 20 parallel hug task note commands on one board all succeed (test)
- [x] a single hug mem forget on the real-store copy finishes in under 2 s warm (measured)
- [ ] capture's database writes take the write turn (test)

## Notes

dragon, commit c353b83 (on dragon/0hkaa5, after 0hkaa5). Check 1: TestParallelNotesWithoutRemoteAllLand failed before the fix: '15 of 20 notes in the task file', errors 'fatal: cannot lock ref 'HEAD': is at fe84a8a... but expected b4d7e63...' and 'commit: index already contains staged changes' (the board turn held only with a remote). After: go test -count=3 -run TestParallelNotesWithoutRemoteAllLand: 'ok ... 24.975s' (3 of 3). With a remote: TestBackgroundSyncParallelNotesAllLand PASS (0hkaa5). Check 3: TestCaptureWritesWaitForTheWriteTurn PASS (0.59s): another handle takes the write turn while the fake model answers; capture_seen stays 0 rows until release, then 1. Mutation check: with the turn removed from the save path the test fails 'capture_seen has 1 rows while the turn is held elsewhere'. Capture's capture_lock/bindings statements (captureExec), backlog checkpoint, batch save + checkpoint and SaveAudit (SaveAuditCtx nests in remember's turn) take the turn; never across extract.Run. muninn/internal/app ok 32.5s, run ok 12.3s.

Check 2 measured, real-store copy (tmp/lanes/m8shdj-forget.sh on a fresh copy via me51r2-fresh.sh, 12,723 units, 14.6 MB memories.json, niced, GOMAXPROCS=2): cold first index 'units added=12723 ... 31.645s'; warm, 5 single forgets: real 1,77 / 1,35 / 1,35 / 1,36 / 1,34 s, all under 2 s. Profile of one (1.52 s wall, 1.22 s samples): ForgetMany 0.74s, of which Reindex 0.44s (IndexNative/LoadNative 0.21s: the full 14.6 MB JSON parse) and ingest.ForgetMany 0.30s (load + MarshalIndent save); runtime.madvise 0.34s. So the forget is already incremental in the index (sources skipped unless changed, communities only on deltas); what remains is parsing and re-saving the whole native JSON twice. Under 2 s holds without a change; a further cut (reuse the parsed file in the post-forget pass) is possible but not needed for this check.
