---
id: 6gtv7a
title: "Memory store opens without a long write: no schema write when nothing is missing"
state: "doing"
type: bug
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "000001h"
created: 2026-11-25
updated: 2026-11-25
doing_since: "2026-11-26T00:09"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
store.Open writes the schema only when a table is missing, and never waits 30 s for the database while holding the init lock.

## Purpose
hippo 2026-11-25 (vrx1ja): with a table to add and the write lock held, an open waited 30.3 s holding the init lock, so four openers queued 30/60/90/121 s; hooks and the CLI still have this one-after-another wait.

## Principle
Opening reads first and writes only what is missing; a lock is never held while waiting on another.

## Details
Measured on a store copy (vrx1ja notes): normal open 0.12-0.23 s; init lock held 20 s -> 19.3 s (no timeout on that lock); a table to add + write lock -> 'schema exec 30.318s' then SQLITE_BUSY, 30.4 s. hug mcp now opens memory in the background (v0.7.113), but hooks (recall, capture) and hug mem CLI still open in the foreground. Fix: check sqlite_master before any CREATE; take the init lock only when something is missing; on SQLITE_BUSY release the init lock before waiting.

## Depends on

## Acceptance
- [x] With nothing missing, store.Open runs no write statement (test counts writes)
- [x] With a table to add and the database write-locked by another process, a second opener does not wait behind the first's init lock (test with two openers)

## Notes


Branch 6gtv7a (= ci/6gtv7a) 627b0ba7. Failing-first 60e62c3f on main code: 'TestOpenWritesNothingWhenNothingIsMissing: Open of a fully migrated store ran 61 write statements, want none'; 'TestOpenSkipsTheInitLockWhenNothingIsMissing: waited 3.004s for the init lock'; 'TestOpenersDoNotQueueBehindALockedSchemaUpdate: at 1.5 s an opener held muninn.db.init.lock ... 2 of 2 openers had not returned after 10s'; 'TestOpenGivesUpOnAHeldInitLock: Open waited 10s'. Fix bc18b28c+627b0ba7: Open reads sqlite_master/pragma_table_info/meta (missingSchema); only when something is missing it TryAcquires the init lock, BEGIN IMMEDIATE with busy_timeout 0, adds all in one tx, lets both locks go on SQLITE_BUSY, gives up after MigrateWait 5 s with store.ErrBusy + held by: + fix: line; hug mcp busy() retries on ErrBusy. Store copy (990 MB vrx1ja-store, cp -c), before e51f93d9 / after: plain open 2 ms / 2 ms; write lock held: 2 ms / 1 ms; init lock held 20 s: 18.987 s / 2 ms; holds_ended dropped + write lock 70 s, 2 openers: 30.609 s + 1m1.289 s (SQLITE_BUSY) / 5.009 s + 5.014 s (memory store busy ... fix:); upgrade after release 11 ms, next open 2 ms. 11 mutants, 10 killed (table in report).
check 1: 627b0ba7, CI 45730059023 8/8. muninn/internal/store/open_nowait_test.go TestOpenWritesNothingWhenNothingIsMissing: an authorizer on every connection counts write actions (INSERT/UPDATE/DELETE/CREATE/DROP/ALTER/TRANSACTION); fully migrated store -> 0 (main e51f93d9: 'ran 61 write statements, want none'). Also: open with the init lock held elsewhere 2 ms (main: waited 3.004s at the 3 s bound; store copy 18.987 s for a 20 s hold); TestOpenFillsEveryKindOfGap: missing table/index/trigger/column/old-hug view/backfills/short ids each filled, then next open writes nothing. Mutants 'always run schema exec', 'check only tables in sqlite_master', 'skip column checks' killed.
check 2: 627b0ba7, CI 45730059023 8/8. TestOpenersDoNotQueueBehindALockedSchemaUpdate: holds_ended dropped, another connection holds BEGIN IMMEDIATE throughout, 2 openers: init lock free at 1.5 s, both return at 5.038 s with 'schema: memory store busy: this hug has schema to add and gave up after 5.037s: database is locked (5) (SQLITE_BUSY) / held by: ... / fix: retry in a minute (hug mem stats); hug mcp opens memory again by itself' (main: init lock held at 1.5 s, 2 of 2 not returned after 10s). Bound 10 s = MigrateWait 5 s + CI slack; 5 s fits the recall hook's 8 s budget and the 15 s Claude/Codex hook timeout. Store copy, write lock 70 s, 2 openers: main 30.609 s + 1m1.289 s, now 5.009 s + 5.014 s. Lock released at 1 s: both open at 1.04/1.09 s. hug mcp's busy() retries on store.ErrBusy (TestMCPRetriesAnOpenBehindTheInitLock). Mutants 'hold init lock across waits', 'busy_timeout 30 s in BEGIN', 'MigrateWait 30 s', 'no fix: line', 'mcp ignores ErrBusy' killed.
crab review: landed 627b0ba7 (CI 45730059023, 8/8 jobs success) as main 29ced89c, v0.7.116; release.yml 45731254681 success; brew ../Cellar/hug/0.7.116/bin/hug, 'hug v0.7.116'.
## History
- 2026-11-26T00:09 todo -> doing/active (take)
