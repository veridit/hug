---
id: gjmqhn
title: "Two hug versions on one machine keep their own cache, never wipe each other's"
status: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
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

- np2kk2
## Acceptance
- [ ] two binaries with different SchemaVersion alternate opens on one board 4 times and neither rebuilds (Stats.Rebuilt false after the first open of each; test)
- [ ] a cache-N.db not opened for 7 days is removed on the next open, a fresher one is kept (test with Chtimes)
- [ ] on Seid, with a v0.7.41 hug tui open beside the new binary, the new binary's warm first frame shows the whole board, numbers in the notes

## Notes
