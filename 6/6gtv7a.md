---
id: 6gtv7a
title: "Memory store opens without a long write: no schema write when nothing is missing"
state: todo
type: bug
priority: "high"
assignee: []
labels: []
group: ""
rank: "000001h"
created: 2026-11-25
updated: 2026-11-25
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
- [ ] With nothing missing, store.Open runs no write statement (test counts writes)
- [ ] With a table to add and the database write-locked by another process, a second opener does not wait behind the first's init lock (test with two openers)

## Notes
