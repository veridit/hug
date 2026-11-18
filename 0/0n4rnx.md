---
id: 0n4rnx
title: "Windows CLI tests run at 450 of the 600 s cap"
state: "doing"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T17:58"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
huginn/internal/cli finishes on windows-latest with clear margin under go test's 10m timeout, by making slow tests cheaper, not by raising the cap

## Purpose
Green runs take 443-453 s (CI 44344774710, 44347035097). Run 44350205436 hit 'panic: test timed out after 10m0s' with no hung test (running tests 3-7 s old), so a slow runner fails exact-SHA landing

## Principle
Measure per-test time first (go test -json on windows); cut process spawns and sleeps in the slowest tests; do not raise -timeout and do not split the package just to dodge the cap

## Relevant details
Log: /Users/jhf/.jcode/scratch/wincli-44350205436.log. Running at the panic: TestBackgroundSyncParallelNotesAllLand, TestTakeAsAndModelRecordsBoth, TestTakeUsesJcodeSessionName, TestTakeWithoutModelFailsWithFix.

## Depends on

## Acceptance
- [ ] A table of the 15 slowest cli tests on windows-latest, before and after
- [ ] huginn/internal/cli on windows-latest under 300 s in 3 consecutive CI runs

## Notes


BEFORE (windows-latest, CI 44353649746, origin/main 0b3c0e96 + measure-only step 'go test -json -timeout 20m ./huginn/internal/cli/', artifact wincli-json, parsed by scripts/testtimes on branch ci/0n4rnx-measure): 'package huginn/internal/cli: pass in 458.8 s; 177 top-level tests (139 parallel), 1286.5 s summed, 50.5 s in serial tests'. Top 15 (s): 1 TestCatchUpAcrossLayoutMove 64.64; 2 TestCatchUpRemoteOldLayoutIsTranslatedAtomically 63.32; 3 TestCatchUpKeepsBothSidesValueInNote 57.49; 4 TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade 36.66; 5 TestOrdinarySyncScalarFieldsKeepBoardAndReportLocal 34.20; 6 TestOrdinarySyncUnreadableTaskKeepsLocalCommits 28.58; 7 TestRequestRoutesThroughNext 27.47; 8 TestOrdinarySyncDisjointScalarLinesNeverFormHybrid 26.17; 9 TestBackgroundSyncOfflineCatchesUp 25.88; 10 TestRequestCommands 23.84; 11 TestBackgroundSyncBurstIsBatched 22.71; 12 TestCatchUpWhenBothClonesMoved 22.55; 13 TestReadOnlyCloneFastForwardsAcrossLayoutMove 22.37; 14 TestBackgroundSyncParallelNotesAllLand 22.21; 15 TestBackgroundSyncConflictKeepsCommits 22.06. Full top 30: ~/.jcode/scratch/0n4rnx-win-before.md. Local (macOS, niced, load 139-225): 293.7 s, 521.5 s summed. Where the time goes (local trace2, TestOrdinarySyncScalarFieldsKeepBoardAndReportLocal): 485 git processes, of which 37 'maintenance run --auto' (on Windows these run in the foreground: daemonize is ENOSYS), 32 'config --get core.sshCommand' (netEnv, one per network command), 29 'rev-parse --show-toplevel --git-common-dir'. Fixture phases (local): clones 6.1 s, oldLayoutBoard 5.1 s, upgrade --apply 3.3 s, each sync 2.4-3.2 s.
## History
- 2026-11-18T17:58 todo -> doing/active (take)
