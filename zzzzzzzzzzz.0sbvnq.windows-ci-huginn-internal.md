---
id: 0sbvnq
title: "Windows CI: huginn/internal/cli tests under 5 minutes"
status: "doing"
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
waiting_on: "CI 44098427171 on ci/0sbvnq-2 (ce87089), then landing slot, then 3 main Windows runs well under 240 s"
waiting_since: "2026-11-17T15:01"
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
- [ ] no test in the package is skipped on Windows that ran there before

## Notes

Measured (no change yet): Windows cli package in the full CI run is 453-514 s (runs 44063552584 469, 44063674567 460, 44065886978 514, 44068680977 453); alone 225 s (run 44068788222, per-test -json): the full run's 4 CPUs are shared with 29 other packages. Every cli test is serial. git spawns: 5622 in the package (wrapper count, macOS), ~45 ms each on Windows. Defender real-time scanning, run 44073995383 (same commit, full go test ./...): cli 359 s on, 269 s off; huginn/internal/tui 197 s on, 91 s off; muninn/internal/app 351 s on, 242 s off.

Step 1 on ci/0sbvnq (8c0fa9c, CI run 44075465280): App.Env per-App environment overlay, so app() needs no t.Setenv; 35 of 46 tests t.Parallel. childMemo locked. Local macOS -race: package 99 s, green.

Run 44075465280 (ci/0sbvnq 8c0fa9c, all jobs green): Windows huginn/internal/cli 275.7 s, under 300 (step-8 run 44074927418 on the same base: 410.7 s). Windows slowest now muninn/internal/app 302.7 s. Second run follows after the rebase on main.

Second run under 300 s: CI 44080439322 (ci/wyvern-stack a7cc707, on 16dxsr c9ea6ed; all jobs green): Windows huginn/internal/cli 265.3 s. First: 44075465280, 275.7 s. Before (same day): 453-514 s. Waits for the landing queue (crab 12:20: wyvern 3rd).

Check 1: Windows huginn/internal/cli under 300 s in two consecutive CI runs: 44075465280 (ci/0sbvnq, 275.7 s) and 44080439322 (ci/wyvern-stack, 265.3 s). Check 2: no test skipped on Windows; the change adds t.Parallel to 35 of 46 tests and no Skip (grep -c 'Skip' unchanged in huginn/internal/cli tests between 893cd50 and a7cc707).

Landed on main d74af5d (04216f3), CI 44087396396. Windows cli on that run: 'ok github.com/veridit/hug/huginn/internal/cli 299.686s', under 300 s by 0.3 s. Earlier runs: 275.7 s (44075465280), 265.3 s (44080439322). The margin is thin: the run also carried sfa8w4's new tests. Check 2 waits for 3 runs on main (crab): main CI 44089952127 is running at d74af5d.


Per-test times on Windows, main d198e01, run 44093916101 (throwaway branch ci/0sbvnq-times2, go test -json ./... as CI does): huginn/internal/cli 289.4 s in the full run; muninn/internal/app 303.7 s; tui 130.8 s. The 19 serial (non-t.Parallel) cli tests sum 159.3 s. The 5 BackgroundSync tests are 107.6 s of that: BurstIsBatched 28.42, OfflineCatchesUp 28.25, ParallelNotesAllLand 23.07, ConflictKeepsCommits 15.87, NoteReachesOtherClone 12.02. They are serial because clones() uses t.Setenv and the package globals Spawn/FetchAfter. Also serial with only app(t): TestMoveCommands 13.41, TestDoneReflectAndReflections 9.41, TestStrikeThroughCLI 8.69, TestBranchWaitRecordsTip 4.83. TestWaitsThroughCLI 9.05 sets cond.LoadAverage, a global. Slowest 5 overall: TestBoardColumns 44.05 (parallel), TestEveryRefCommandAcceptsTaskColon 28.70 (parallel), BurstIsBatched 28.42, OfflineCatchesUp 28.25, TestLifecycleJSONContract 24.18 (parallel). Target from crab: under 240 s with margin; 299.686 s on 44087396396 is not a pass to tick on.
After 752344d (background-sync tests and 6 others parallel, App.Sync per App), Windows timing run 44096841366 (ci/0sbvnq-times3, go test -json ./... as CI): huginn/internal/cli 201.2 s in the full run, was 289.4 s (run 44093916101, main d198e01). The serial tests sum 23.0 s, was 159.3 s; the slowest serial test is now TestWaitsThroughCLI at 11.19 s (sets cond.LoadAverage). ce87089 adds the git-process cuts (take 19 -> 15, note 10 -> 9, ...), counted by TestGitProcessesPerCommand; its timing run is 44098431763. CI for ci/0sbvnq-2: 44096836969 (752344d), 44098427171 (ce87089).
## History
- 2026-11-17T14:32 doing -> doing/waiting (wait)
- 2026-11-17T14:46 doing/waiting -> doing/active (resume); was waiting_asked_by: wyvern; waiting_on: 3 main Windows runs with huginn/internal/cli under 300 s (landing run 44087396396: 299.686 s); waiting_since: 2026-11-17T14:32
- 2026-11-17T15:01 doing/active -> doing/waiting (wait)
