---
id: 0sbvnq
title: "Windows CI: huginn/internal/cli tests under 5 minutes"
state: "doing"
type: feature
priority: ""
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
stage: "waiting"
waiting_on: "check 2: 2 more green main runs with Windows huginn/internal/cli under 240 s and no new skip; run 1 is 44118189933 (195.497 s)"
waiting_since: "2026-11-17T16:58"
waiting_asked_by: "wyvern"
---

## Goal
go test ./huginn/internal/cli/ on the windows-latest runner finishes in under 300 s (it took 469 s in run 44063552584 after the benchmark left, 570-612 s before), with every test still run on Windows

## Purpose
The package came within seconds of go test's 10-minute limit and crossed it several times (runs 44056677278, 44057678727, 44059530090, 44060790405); every push waits on the slowest job, and a timeout hides which test failed

## Principle
Measure per test first, then remove waiting, not coverage: share setup, run independent tests in parallel, cut sleeps that wait for real time. No test skipped on Windows to make the number; no absolute timing assertions

## Relevant details
Windows costs about 1.9 s per hug command (run 43943367365). Background-sync tests run the real 1.5 s settle. Get per-test times from go test -v -json on the runner

## Depends on

## Acceptance
- [x] huginn/internal/cli on windows-latest under 300 s in two consecutive CI runs, run ids in the notes
- [x] no test in the package is skipped on Windows that ran there before

## Notes

Measured (no change yet): Windows cli package in the full CI run is 453-514 s (runs 44063552584 469, 44063674567 460, 44065886978 514, 44068680977 453); alone 225 s (run 44068788222, per-test -json): the full run's 4 CPUs are shared with 29 other packages. Every cli test is serial. git spawns: 5622 in the package (wrapper count, macOS), ~45 ms each on Windows. Defender real-time scanning, run 44073995383 (same commit, full go test ./...): cli 359 s on, 269 s off; huginn/internal/tui 197 s on, 91 s off; muninn/internal/app 351 s on, 242 s off.

Step 1 on ci/0sbvnq (8c0fa9c, CI run 44075465280): App.Env per-App environment overlay, so app() needs no t.Setenv; 35 of 46 tests t.Parallel. childMemo locked. Local macOS -race: package 99 s, green.

Run 44075465280 (ci/0sbvnq 8c0fa9c, all jobs green): Windows huginn/internal/cli 275.7 s, under 300 (step-8 run 44074927418 on the same base: 410.7 s). Windows slowest now muninn/internal/app 302.7 s. Second run follows after the rebase on main.

Second run under 300 s: CI 44080439322 (ci/wyvern-stack a7cc707, on 16dxsr c9ea6ed; all jobs green): Windows huginn/internal/cli 265.3 s. First: 44075465280, 275.7 s. Before (same day): 453-514 s. Waits for the landing queue (crab 12:20: wyvern 3rd).

Check 1: Windows huginn/internal/cli under 300 s in two consecutive CI runs: 44075465280 (ci/0sbvnq, 275.7 s) and 44080439322 (ci/wyvern-stack, 265.3 s). Check 2: no test skipped on Windows; the change adds t.Parallel to 35 of 46 tests and no Skip (grep -c 'Skip' unchanged in huginn/internal/cli tests between 893cd50 and a7cc707).

Landed on main d74af5d (04216f3), CI 44087396396. Windows cli on that run: 'ok github.com/veridit/hug/huginn/internal/cli 299.686s', under 300 s by 0.3 s. Earlier runs: 275.7 s (44075465280), 265.3 s (44080439322). The margin is thin: the run also carried sfa8w4's new tests. Check 2 waits for 3 runs on main (crab): main CI 44089952127 is running at d74af5d.


Per-test times on Windows, main d198e01, run 44093916101 (throwaway branch ci/0sbvnq-times2, go test -json ./... as CI does): huginn/internal/cli 289.4 s in the full run; muninn/internal/app 303.7 s; tui 130.8 s. The 19 serial (non-t.Parallel) cli tests sum 159.3 s. The 5 BackgroundSync tests are 107.6 s of that: BurstIsBatched 28.42, OfflineCatchesUp 28.25, ParallelNotesAllLand 23.07, ConflictKeepsCommits 15.87, NoteReachesOtherClone 12.02. They are serial because clones() uses t.Setenv and the package globals Spawn/FetchAfter. Also serial with only app(t): TestMoveCommands 13.41, TestDoneReflectAndReflections 9.41, TestStrikeThroughCLI 8.69, TestBranchWaitRecordsTip 4.83. TestWaitsThroughCLI 9.05 sets cond.LoadAverage, a global. Slowest 5 overall: TestBoardColumns 44.05 (parallel), TestEveryRefCommandAcceptsTaskColon 28.70 (parallel), BurstIsBatched 28.42, OfflineCatchesUp 28.25, TestLifecycleJSONContract 24.18 (parallel). Target from crab: under 240 s with margin; 299.686 s on 44087396396 is not a pass to tick on.
After 752344d (background-sync tests and 6 others parallel, App.Sync per App), Windows timing run 44096841366 (ci/0sbvnq-times3, go test -json ./... as CI): huginn/internal/cli 201.2 s in the full run, was 289.4 s (run 44093916101, main d198e01). The serial tests sum 23.0 s, was 159.3 s; the slowest serial test is now TestWaitsThroughCLI at 11.19 s (sets cond.LoadAverage). ce87089 adds the git-process cuts (take 19 -> 15, note 10 -> 9, ...), counted by TestGitProcessesPerCommand; its timing run is 44098431763. CI for ci/0sbvnq-2: 44096836969 (752344d), 44098427171 (ce87089).
Main run 1 after landing 647acc7 and 9c2b0b9: main d661273, CI 44118189933 green on all 5 jobs (secrets, bench, ubuntu, macos, windows). Windows job 134162049557: 'ok github.com/veridit/hug/huginn/internal/cli 195.497s' (target under 240 s); tui 125.809s. Check 2 waits for 2 more main runs on Windows.
Regressed on main. Windows huginn/internal/cli on the 20 green main runs after run 1 (44118189933, 195.5 s), newest first, by job id: 264.3, 363.9, 433.1, 473.9, 262.1, 450.6, 334.4, 366.8, 400.6, 331.1, 306.7, 347.3, 432.4, 335.8, 226.8, 271.7, 233.8, 274.5, 244.9, 245.7 s (job 134335316392 = b20d191 run 44166882932 ... job 134183814458 = run 44121274410). 2 of 20 under 240 s. Next (crab 20:26): per-test -v run alone on a throwaway ci/ branch, then fix the top setup costs with counting tests.
Per-test Windows times (throwaway ci/0sbvnq-times4, run 44169401377 on main 3bc5d72, branch deleted): cli alone 230.687 s, in the full run 335.014 s. Top alone: TestEightWritersMixedCommandsAllLand 89.14, TestTUIWriteAndCLINoteAtOnce 40.85, BackgroundSyncOfflineCatchesUp 32.14, EveryRefCommandAcceptsTaskColon 30.14, BurstIsBatched 29.37. 65 of 84 tests parallel; the 19 serial sum 54 s, the largest TestDeduceTwoClonesOneLine 15.58 s, though clones() sets nothing process-wide. git processes in the macOS cli suite (logging wrapper on PATH): 11,531, was 5,622 at the start of 0sbvnq; TestEightWritersMixedCommandsAllLand 1,704 of them. Per write: 9 git processes (note, check, set alike). Fix 537a385 on ci/0sbvnq-3 (CI 44177792785): HEAD and the old-layout refs from the files (gitbranch.RefAt, fastHead), add --verbose instead of a second diff --cached, 5 tests parallel. TestGitProcessesPerCommand: reads 5 -> 3, writes 9 -> 6, take 15 -> 11. Suite spawns 11,531 -> 8,820.
Landed 09b85e3 on main (aa4d9ec..09b85e3; -race gitbranch 21.3 s, board, cli 212.2 s, muninn project/app, cmd/hug ok). Windows effect measured paired on one runner (run 44180555871, cli alone, base 3bc5d72 / fix 537a385 alternating): base 246.909 s, fix 246.119 s, base 242.777 s, fix 227.874 s. That is at most 3-15 s of 245, not a pass. (fix-2 also failed TestSetupFilesStepRemembered, ':188 remembered files check: []', the 3mavpt flake also seen on main runs 44172854884 and 44176187867; reported to calf.) Where a write's time goes on the Windows runner (probe run 44184315433, 4 CPUs, git 2.56.0.windows.4, 40 calls each): git --version 25 ms, rev-parse 28, worktree list 29, diff --cached 29, status 32-45, raw write+add+commit 104-106 ms; hug task show 89-92, ls 89, open+close board 88-90, note 290-346, set 278-451. CPU profile of 60 notes: 99.3% runtime.cgocall (Windows syscalls), Repository.runAt 33% cum, Commit 18%. The two writer tests are the cost: TestEightWritersMixedCommandsAllLand runs 160 writes under one board turn, so writes x ~300 ms = 48 s at least. Paired: 96.9/88.2 (base), 101.0/66.8 (fix). TestTUIWriteAndCLINoteAtOnce: 46.1/45.1, 37.8/36.0. Next is crab's call: fewer rounds in the 8-writer test (its 95ahtk check names 20) or more per-write cuts (each git process is ~30 ms of ~300).
landed 09b85e3 on main 2026-11-17: Fewer git processes per hug command: HEAD and old-layout refs from the files, no…
shipped in v0.7.49 (09b85e3)
Check 2, no test skipped on Windows that ran there before: on 893cd50 (start of 0sbvnq) huginn/internal/cli tests have no Windows skip. On main 3 tests skip on Windows, each added with its skip in the same commit, so none ran on Windows before: TestGitProcessesPerCommand (9c2b0b9, wrapper is a POSIX sh script), TestSetupOffersForbidScratch (95079e0, the fix is a POSIX shell command), TestSetupReadsNoBacklogFileAfterImport (e971d24, POSIX permissions). None of the three exists at 893cd50 (git grep: 0 each). 09b85e3 and the next change add no skip.
Not done yet: main CI 44194274440 (98444d7) hit go test's 10-minute limit on Windows cli (600.088 s; TestSetupInterruptedSaysSo running 6m11s in importer.BacklogTaskDirs over 500 files, kestrel's log /Users/jhf/.jcode/scratch/kestrel-ci-44194274440.txt), so the goal (under 300 s on main) does not hold today. Next change 7592dc81 on ci/0sbvnq-4, CI 44204301242: writes 6 -> 4 git processes (worktree list from the files, no diff --cached before a whole-board commit), 8-writer test 3 rounds (95ahtk note has the mutation evidence). Local -race: tui timed out at 1h under load 400-527 (unmeasured whether mine; the CI ubuntu -race job decides), muninn/project ok after its count 4 -> 3.
## History
- 2026-11-17T14:32 doing -> doing/waiting (wait)
- 2026-11-17T14:46 doing/waiting -> doing/active (resume); was waiting_asked_by: wyvern; waiting_on: 3 main Windows runs with huginn/internal/cli under 300 s (landing run 44087396396: 299.686 s); waiting_since: 2026-11-17T14:32
- 2026-11-17T15:01 doing/active -> doing/waiting (wait)
