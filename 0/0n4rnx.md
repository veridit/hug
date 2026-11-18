---
id: 0n4rnx
title: "Windows CLI tests run at 450 of the 600 s cap"
state: "doing"
type: feature
priority: "high"
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzx"
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
crab 18:29 local: second timeout in 1h: CI 44356197911 (b2232191, ybbfsa, no cli change) 'panic: test timed out after 10m0s', running: TestSyncMergesBothSidesAppends/an_answer_against_a_note (28s), TestOrdinarySyncLegacyLayoutDoesNotImplicitlyUpgrade (16s). Log /Users/jhf/.jcode/scratch/wincli-44356197911.log. This now blocks landing; priority high.
CUT 2 plan (fixture reuse, test-only, crab ok 17:35Z). clones() is built once per test process and copied for each test; the catch-up pair (clones + oldLayoutBoard + push + pull, 117 git processes locally) is built once too. Each copy then fixes its own remote url and runs worktree repair, 4 git calls. Bar before landing: (1) go test ./huginn/internal/cli/ passes locally, niced; (2) the 44 clones/catchup callers pass 3x locally (-count=3 -run the affected tests); (3) a guard fails the test if any copied gitdir or url still names the template folder; (4) the windows-cli package time on ci/0n4rnx drops below cut 1's run 44360178493. Counts per arm: git processes for TestCatchUpAcrossLayoutMove by trace2, before vs after.
landed on main 8c26ec51 (CI 44363087479 6/6, windows-cli 549.357s)
landed 6ecd08f on main 2026-11-18: test(cli): no git maintenance in cli test gits (0n4rnx)
REGRESSION WINDOW (main CI, test-windows-cli package line, 60 main runs read, table ~/.jcode/scratch/0n4rnx-maintimes.tsv): 41 runs from 0c3d3924 to 7f9f7bd8 (v0.7.60) 133-273 s; first run after is ead1be4c 403.7 s, then 356-507 s on all 11 green runs. 7f9f7bd8..ead1be4c is the w40cmk layout-3 merge (21 commits: catch-up, ordinary sync, layout tests). SPLIT (local macOS, niced, goroutine samples every 100 ms on the test goroutine, cut 1 tree): FirstParentRetainsMergeResolutionOnce 35.7 s wall = exec 31.2 / sleep 2.2 (settled) / other 0.3; BoardColumns 20.1 = exec 18.1 / other 0.8; OrdinarySyncScalarFields 24.6 = exec 21.0 / sleep 1.6 / other 0.4; BackgroundSyncOfflineCatchesUp 29.6 = exec 10.9 / sleep 16.4 (settled waiting a 5 s ssh hang) / other 0.7; RequestRoutesThroughNext 2.7 = exec 1.8. Exec = waiting on a child process. Windows split runs: ci/0n4rnx-measure (main+cut1) and ci/0n4rnx-measure-old (7f9f7bd8).
## History
- 2026-11-18T17:58 todo -> doing/active (take)
