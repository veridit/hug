---
id: 95ahtk
title: "Task writers wait their turn: no index or temp-file races"
status: todo
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
Any number of hug task writers on one board at the same moment all succeed: they serialise on a per-board lock before touching the index, temp files or the commit, as hug mem does since me51r2.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 3b. Two agents writing at once in Seid raced on the git index and on temp files; Seid now batches writes, one write per ticket per batch, to avoid it. Rule: concurrent writes wait, never fail.

## Principle
Reproduce first with N parallel writers in a test that fails today; then a lock that waits, with no timeout that turns into a failure. Readers never wait for the lock.

## Relevant details
me51r2 (9af390c) fixed this for hug mem. huginn has no file lock today (no flock/LockFile in huginn). Check the cache.sync BEGIN IMMEDIATE fix from 44w1ev too.

## Depends on

## Acceptance
- [ ] a test runs 8 writers (note, check, set) at once on one board, 20 rounds: every write succeeds and is on the branch; it fails on main before the fix
- [ ] readers (ls, show) do not wait on a held writer lock (test)

## Notes
