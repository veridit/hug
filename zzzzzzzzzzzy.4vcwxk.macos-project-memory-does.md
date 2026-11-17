---
id: 4vcwxk
title: "macOS: a project memory does not reach the other clone within 10 s"
status: todo
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
In TestBackgroundSyncTwoClonesRealBinary, a project memory remembered in clone A shows up in clone B within the test's limit on every OS, as the task note does; the cause of the miss is found and fixed, not the limit raised.

## Purpose
Main CI macOS failed twice the same way: run 44077251884 (893cd50) 'clone B: note after 8.758308208s, memory after -1ns (-1: not within 10s)' and run 44068175565 (c6e0bff) 'note after 9.325147584s, memory after -1ns'. The note arrives (late), the memory never does. 0hkaa5 promised project memories sync in the background within seconds.

## Principle
A wait that never ends is a bug, not a flake: find why the muninn-notes push or fetch does not happen, prove it with logs from .git/hug/sync/, fix it. Raise a limit only if the cause is the runner's speed and the measurement shows it.

## Relevant details
Test: sync_test.go:120. The note at 8.7-9.3 s is also near the 10 s limit; ubuntu and Windows pass. Sync logs: .git/hug/sync/. The memory sync runs on the muninn-notes branch (0hkaa5); the quiet() fix 18203c2 covered the cleanup race, not this.

## Depends on

## Acceptance
- [ ] the cause is named with evidence (sync log lines or a trace) in a note
- [ ] a test that fails before the fix passes after; macOS CI green on three consecutive main runs (run ids)

## Notes
