---
id: d2n0h7
title: "Background sync stalls once in ~30 CI runs (ParallelNotesAllLand)"
state: todo
type: bug
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
TestBackgroundSyncParallelNotesAllLand never stalls: a background pass that is queued always runs and settles.

## Purpose
CI 44156864971 stalled at sync_test.go:87 (settled() timed out) once in about 30 main runs; a hang is a bug, fixed with a test.

## Principle
Wait on the pass's own marker, never a longer clock. Do not widen timeouts.

## Relevant details
Seen: CI 44156864971, sync_test.go:87. Not reproduced locally. settled() dumps queued, busy, pending, turn held and the pass log on failure.

## Depends on

## Acceptance
- [ ] the stall is reproduced or explained from a failing run's dump, and a test fails before the fix and passes after

## Notes

kestrel: not reproduced in 10+12+4 runs (cli package -race loop, 10 runs of which 2 failed on unrelated git segfaults: run 9 'TestLintHidesClosedTasks: git add: signal: segmentation fault', run 12 'TestEightWritersMixedCommandsAllLand ... diff --cached --name-only: signal: segmentation fault'; 12 targeted -race runs of TestBackgroundSync*; 4 more targeted). Diagnostics on failure in place: settled() now prints 'queued %v, busy %v, pending %v, turn held %v (log ...)' plus the pass log. Left open for the next CI failure's dump.

landed 839e593 on main 2026-11-17: test(cli): a stalled background pass says which wait it is in and what the pass…
