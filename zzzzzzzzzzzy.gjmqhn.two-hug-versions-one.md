---
id: gjmqhn
title: "Two hug versions on one machine keep their own cache, never wipe each other's"
status: "doing"
type: feature
priority: ""
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T17:30"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Each cache schema version has its own database file (.git/huginn/cache-<schema>.db), so hug binaries of different versions on one machine (a TUI left open on v0.7.41 beside agents on v0.7.42) each open their own cache warm and never drop or rebuild the other's. A cache-N.db no binary has opened for 7 days is removed by the next hug that opens the cache folder.

## Purpose
Observed 2026-11-17 16:10 (wyvern): seid-lang/.git/huginn/cache.db read 14,14,12,14 over 10 s. A hug tui on v0.7.41 (schema 12, pid 227) and the v0.7.42 agents (schema 14) each found the other's schema_version and dropped every table (cache.go Open: version != SchemaVersion -> DROP TABLE ...), so every open was a cold build: the vnyz8z first frame showed '0/0 tasks' on the live board and 2520 tasks on a private copy. Jørgen keeps 3 hug tui open (2 on v0.7.41, 1 older on statbus), and brew upgrades come hourly, so mixed versions are the normal case (crab 16:16).

## Principle
Key by schema, nothing cleverer: one file per SchemaVersion, no migration between them, no shared tables. Bound the leftovers by age (7 days unopened), never by guessing which binary is still running. Measure open time before and after.

## Relevant details
cache.Open is called from huginn/internal/cli/cli.go:244 with filepath.Join(gd, "huginn", "cache.db"). The legacy cache.db: a binary that does not find cache-<schema>.db may adopt cache.db only if its schema_version matches, else leaves it to age out like the others. Age: record last-opened in the file's mtime (touch on Open) or a meta key; removal must skip a file open by a live process if the platform says so (Windows refuses the delete: ignore that error). Crab assigned it to wyvern after np2kk2.

## Depends on

## Acceptance
- [ ] two binaries with different SchemaVersion alternate opens on one board 4 times and neither rebuilds (Stats.Rebuilt false after the first open of each; test)
- [ ] a cache-N.db not opened for 7 days is removed on the next open, a fresher one is kept (test with Chtimes)
- [ ] on Seid, with a v0.7.41 hug tui open beside the new binary, the new binary's warm first frame shows the whole board, numbers in the notes

## Notes


Landed on main e96d6f8 (45cbc0b..e96d6f8; SchemaVersion is 16 on main now). Tests, each failing before the change: TestTwoSchemasAlternateWithoutRebuild (4 rounds of 2 schemas: rebuilt only on each schema's first open; failed 'round 1 schema old: rebuilt true'), TestStaleCacheFilesRemoved (cache.db and cache-12.db 8 days old removed with their -wal/-shm; cache-13.db kept because its WAL is 6 days old; cache-14.db 6 days kept; notes.txt kept), TestOpenMarksCacheUsed. Mutations caught: the shared cache.db name, judging age by the db file alone (ignoring the WAL), no touch on open. -race on cache, cli and board: ok. Check 3, live seid-lang with Jørgen's v0.7.41 hug tui (pid 227) open: .git/huginn now holds cache.db (schema 12, v0.7.41's, 17:42) and cache-16.db (schema 16, 2520 tasks, 17:43) side by side. Warm first frame of the new binary, 5 rounds, load 19-23: 0.291, 0.239, 0.229, 0.254, 0.249 s (median 0.249), quit 0.313, 0.364, 0.563, 0.354, 0.327 s (median 0.354); the frame shows 'HUG TASKS · overview   2520/2520 tasks · ! 5030 findings'. Note: a TUI quit before its refresh lands stores nothing, so the first 6 runs after the switch stayed cold (cache-16.db 110 KB) until a hug task ls built it (41 MB). Cheap, and every agent command builds it, but the first TUI start on a new release may need two opens.
landed e96d6f8 on main 2026-11-17: hug task cache: one database per cache schema (.git/huginn/cache-<N>.db), so two…
## History
- 2026-11-17T17:30 todo -> doing/active (take)
- 2026-11-17T17:43 doing/active -> doing/waiting (wait)
- 2026-11-17T18:00 doing/waiting -> doing/active (resume); was waiting_asked_by: wyvern; waiting_on: main CI for e96d6f8 green on all jobs, then tick 1-3; waiting_since: 2026-11-17T17:43
