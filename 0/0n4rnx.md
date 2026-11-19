---
id: 0n4rnx
title: "Windows CLI tests run at 450 of the 600 s cap"
state: "done"
type: feature
priority: "high"
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzxq"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T21:00"
done_release: "v0.7.70"
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
- [x] A table of the 15 slowest cli tests on windows-latest, before and after
- [x] huginn/internal/cli on windows-latest under 300 s in 3 consecutive CI runs

## Notes


BEFORE (windows-latest, CI 44353649746, origin/main 0b3c0e96 + measure-only step 'go test -json -timeout 20m ./huginn/internal/cli/', artifact wincli-json, parsed by scripts/testtimes on branch ci/0n4rnx-measure): 'package huginn/internal/cli: pass in 458.8 s; 177 top-level tests (139 parallel), 1286.5 s summed, 50.5 s in serial tests'. Top 15 (s): 1 TestCatchUpAcrossLayoutMove 64.64; 2 TestCatchUpRemoteOldLayoutIsTranslatedAtomically 63.32; 3 TestCatchUpKeepsBothSidesValueInNote 57.49; 4 TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade 36.66; 5 TestOrdinarySyncScalarFieldsKeepBoardAndReportLocal 34.20; 6 TestOrdinarySyncUnreadableTaskKeepsLocalCommits 28.58; 7 TestRequestRoutesThroughNext 27.47; 8 TestOrdinarySyncDisjointScalarLinesNeverFormHybrid 26.17; 9 TestBackgroundSyncOfflineCatchesUp 25.88; 10 TestRequestCommands 23.84; 11 TestBackgroundSyncBurstIsBatched 22.71; 12 TestCatchUpWhenBothClonesMoved 22.55; 13 TestReadOnlyCloneFastForwardsAcrossLayoutMove 22.37; 14 TestBackgroundSyncParallelNotesAllLand 22.21; 15 TestBackgroundSyncConflictKeepsCommits 22.06. Full top 30: ~/.jcode/scratch/0n4rnx-win-before.md. Local (macOS, niced, load 139-225): 293.7 s, 521.5 s summed. Where the time goes (local trace2, TestOrdinarySyncScalarFieldsKeepBoardAndReportLocal): 485 git processes, of which 37 'maintenance run --auto' (on Windows these run in the foreground: daemonize is ENOSYS), 32 'config --get core.sshCommand' (netEnv, one per network command), 29 'rev-parse --show-toplevel --git-common-dir'. Fixture phases (local): clones 6.1 s, oldLayoutBoard 5.1 s, upgrade --apply 3.3 s, each sync 2.4-3.2 s.
crab 18:29 local: second timeout in 1h: CI 44356197911 (b2232191, ybbfsa, no cli change) 'panic: test timed out after 10m0s', running: TestSyncMergesBothSidesAppends/an_answer_against_a_note (28s), TestOrdinarySyncLegacyLayoutDoesNotImplicitlyUpgrade (16s). Log /Users/jhf/.jcode/scratch/wincli-44356197911.log. This now blocks landing; priority high.
CUT 2 plan (fixture reuse, test-only, crab ok 17:35Z). clones() is built once per test process and copied for each test; the catch-up pair (clones + oldLayoutBoard + push + pull, 117 git processes locally) is built once too. Each copy then fixes its own remote url and runs worktree repair, 4 git calls. Bar before landing: (1) go test ./huginn/internal/cli/ passes locally, niced; (2) the 44 clones/catchup callers pass 3x locally (-count=3 -run the affected tests); (3) a guard fails the test if any copied gitdir or url still names the template folder; (4) the windows-cli package time on ci/0n4rnx drops below cut 1's run 44360178493. Counts per arm: git processes for TestCatchUpAcrossLayoutMove by trace2, before vs after.
landed on main 8c26ec51 (CI 44363087479 6/6, windows-cli 549.357s)
landed 6ecd08f on main 2026-11-18: test(cli): no git maintenance in cli test gits (0n4rnx)
REGRESSION WINDOW (main CI, test-windows-cli package line, 60 main runs read, table ~/.jcode/scratch/0n4rnx-maintimes.tsv): 41 runs from 0c3d3924 to 7f9f7bd8 (v0.7.60) 133-273 s; first run after is ead1be4c 403.7 s, then 356-507 s on all 11 green runs. 7f9f7bd8..ead1be4c is the w40cmk layout-3 merge (21 commits: catch-up, ordinary sync, layout tests). SPLIT (local macOS, niced, goroutine samples every 100 ms on the test goroutine, cut 1 tree): FirstParentRetainsMergeResolutionOnce 35.7 s wall = exec 31.2 / sleep 2.2 (settled) / other 0.3; BoardColumns 20.1 = exec 18.1 / other 0.8; OrdinarySyncScalarFields 24.6 = exec 21.0 / sleep 1.6 / other 0.4; BackgroundSyncOfflineCatchesUp 29.6 = exec 10.9 / sleep 16.4 (settled waiting a 5 s ssh hang) / other 0.7; RequestRoutesThroughNext 2.7 = exec 1.8. Exec = waiting on a child process. Windows split runs: ci/0n4rnx-measure (main+cut1) and ci/0n4rnx-measure-old (7f9f7bd8).
shipped in v0.7.68 (6ecd08f)
crab 20:14 local, ruling changed: -parallel 4 no gain on windows (m-p4 CI 44370619880: default 403.662/391.412 s, -parallel 4 401.022/386.655 s). The cap blocks all lanes (4tycwd 44373015352 x2, z77pem 44371620428, 3t4a49 44370325830, jrj1vq-c2 44372296874) and the growth is new coverage (w40cmk window), not a hang. So the windows-cli JOB is sharded in two by test name, 10m cap each, with a CI step proving the union equals go test -list and no overlap; every test still runs on Windows. The replay change follows as its own commit.
Shard a74ba787 (ci/0n4rnx-shard, on main 477696be): ci.yml only + 1 CHANGELOG line. test-windows-cli is a 2-shard matrix: sync = -run '^(TestCatchUp|TestOrdinarySync|TestBackgroundSync|TestReadOnlyClone|TestSyncMerges|TestSyncBothSides|TestWriteEndsInterruptedSyncRebase|TestDeduce|TestSettled|TestUpgrade)', rest = -skip same. New job test-windows-cli-shards diffs ran-sync + ran-rest against go test -list on Windows. Local list (darwin): 188 tests, sync 44, rest 144, union = all, overlap 0. Summed test time from CI 44370638232 artifacts: sync 577 s, rest 933 s (sums of parallel tests, not wall). Bar set before the run: both shards pass with package time < 600 s each and the shards job passes; the 300 s target is judged per shard over 3 runs.
Replay 3520ad61 (ci/0n4rnx-replay, on 8c26ec51). Fail-first on 8c26ec51: TestCatchUpProcessesPerCommit 'catch-up of 12 commits started 221 git processes, want at most 80'; TestCatchUpStopLeavesNoReader 'no cat-file --batch reader started'. New code: 66. TestForeignRebaseStillRefuses: ok alone on 8c26ec51 (5.193s) and 3/3 ok on replay code; its failure in the full suite was at its own b.fetch() (rebase_recover_test.go:132), which reaches none of the 3 changed files. Bar before CI: every job green; cli Windows time is reported, not judged, since this run has no shard.
Shard a74ba787 CI 44379205716 success, all 9 jobs. test-windows-cli (sync): 'ok github.com/veridit/hug/huginn/internal/cli 211.348s', 44 top-level tests. (rest): '196.195s', 144 tests. test-windows-cli-shards: '188 all.sorted / 44 ran-sync.sorted / 144 ran-rest.sorted / all 188 tests ran, each in one shard'. Run 1 of 3 under 300 s per shard. Earlier 'failure' runs 44370619880, 44368801244, 44368819470: only test (macos-latest) cancelled, every other job success.
landed a74ba78 on main 2026-11-18: ci: Windows runs the huginn/internal/cli tests in two shards on two runners…
Replay 3520ad61 CI 44379591295 success, all 6 jobs. windows test-windows-cli (unsharded, base 8c26ec51): 'ok github.com/veridit/hug/huginn/internal/cli 445.515s'. gitbranch: ubuntu -race 26.774s, windows 172.428s, macos 66.498s. One run only; single-runner cli on this base ranged 387-403 s (44370619880), so no time gain is claimed. The solid evidence is the count, 221 -> 66 processes for 12 commits.
Cut 2 (fixture templates, 030d0fe4 on branch 0n4rnx-cut2) DROPPED, crab 2026-11-18 19:47. Its only case was time. Its one Windows number, m-c2 CI 44370638232 (with lever a), was 536.8 s for the package against 387-403 s single-runner on the same base (44370619880), so no gain is shown. The only count for it is -826 git processes (5%). The shard (a74ba787) made the time case moot. Not landed; the branch stays as record. Lever (a) fbee8e5b lands separately (a test that hangs until its cap is a defect). Shard runs under 300 s: (1) ci 44379205716 sync 211.348 s / rest 196.195 s; (2) main 44382063226 sync 237.316 s / rest 175.752 s; (3) rerun 44379205716 attempt 2, pending.
Check 1 plan: before = CI 44360222667 (ci/0n4rnx-measure 5322c325, single runner, go test -json, 593.9 s). After = ci/0n4rnx-measure-after (main a74ba787 + upload of each shard's -v cli.log, measure only, never lands): per-test '--- PASS: TestX (Ns)' times from both shards. Table: 15 slowest top-level tests before, with their after times.
shipped in v0.7.69 (a74ba78)
Check 1 table (top-level test Elapsed, Windows). Before: CI 44360222667, ci/0n4rnx-measure 5322c325, one runner, go test -json, package 593.924 s, 179 tests. After: CI 44384341239, ci/0n4rnx-measure-after 38e2e389 = main a74ba787 + cli.log upload (measure only), shard sync 'ok ... 207.655s', rest 'ok ... 176.987s', 188 tests. The 15 slowest before -> after (all in shard sync): WhenBothClonesMoved 51.75->22.13; StopsOnUnreadableFileAndRestores 50.88->14.70; ReadOnlyCloneFastForwardsAcrossLayoutMove 41.21->19.28; LayoutCommitAlsoKeepsItsNote 38.31->19.34; BackgroundSyncSameTaskAppendsConvergeAfterUpgrade 31.72->25.71; OrdinarySyncScalarFieldsKeepBoardAndReportLocal 30.04->24.65; BackgroundSyncOfflineCatchesUp 30.01->22.53 (lever a not on main yet); RemoteOldLayoutIsTranslatedAtomically 29.31->22.61; AcrossLayoutMove 27.94->21.40; DisjointScalarLinesNeverFormHybrid 27.10->22.71; BackgroundSyncConflictKeepsCommits 26.75->18.08; BurstIsBatched 24.41->19.31; ParallelNotesAllLand 24.41->15.00; OrdinarySyncSameTaskAppendsConvergeAfterUpgrade 24.43->20.51; UnreadableTaskKeepsLocalCommits 23.93->19.14. Cause: the shard halves the tests contending on one runner; no per-test cost change is claimed. Full table: scratch 0n4rnx-table.tsv.
check 1: CI 44360222667 (before, 593.924 s) vs CI 44384341239 (after, shards 207.655/176.987 s): 15 slowest before -> after, all 15 dropped, max after 25.71 s; table in note
check 2: 3 consecutive runs with the shard (a74ba787), each shard under 300 s: CI 44379205716 sync 211.348/rest 196.195; main CI 44382063226 sync 237.316/rest 175.752; CI 44379205716 attempt 2 sync 273.553/rest 215.834. Also measure CI 44384341239: 207.655/176.987. Shard guard 'all 188 tests ran, each in one shard' in all. Margin: worst 273.553 s, 26 s under the bar.
Reflection (unanswered, 2026-11-18, crab)
Replay + lever (a) rebased onto crab's base 8dc479b2 (green, CI 44385513689): 5ef6bc8b replay, 8bf24c02 lever (a); CHANGELOG conflict resolved by adding both lines under Unreleased. ci/0n4rnx-replay CI 44386571890 success, all 8 jobs. Shards: sync 195.655 s / rest 207.895 s (base 8dc479b2: 218.652 / 198.602, one run each, no gain claimed); guard 'all 196 tests ran, each in one shard'.
landed 5ef6bc8 on main 2026-11-18: gitbranch: a catch-up replay reads objects through one cat-file --batch (0n4rnx)
landed 8bf24c0 on main 2026-11-18: cli test: the offline test's remote hangs until the test releases it, not 5 s…
shipped in v0.7.70 (5ef6bc8)
## History
- 2026-11-18T17:58 todo -> doing/active (take)
- 2026-11-18T21:00 doing/active -> done (done); was doing_since: 2026-11-18T17:58
