---
id: ad62aa
title: "cli tests: commit WaitDelay and TestDeduceTwoClonesOneLine fail under load"
state: "done"
type: feature
priority: ""
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzxw"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T23:24"
done_release: "v0.7.71"
---

## Goal
huginn/internal/cli tests pass under machine load as they do alone: name the cause of the 'exec: WaitDelay expired before I/O complete' commit errors and the TestDeduceTwoClonesOneLine failure seen at load 300-700, and close it at the cause

## Purpose
Local suites on this shared machine fail on these, so workers can't trust a red run and CI gating slows down

## Principle
No timeout raised, no retry, no skip; find what holds the pipe open or races, like ybbfsa flake 3 (maintenance repack)

## Relevant details
iwazaru 34r1ft run: 2 git commit WaitDelay errors + TestDeduceTwoClonesOneLine at load 300-700; same tests pass alone (ok huginn/internal/cli 413.676s). tigress saw 'WaitDelay expired before I/O complete' in catchup_test.go:68 before 6ecd08fc. Check whether it still happens on main 477696be

## Depends on

## Acceptance
- [x] Cause named in a note with a log line or reproduction
- [x] A test or amplified reproduction fails first, then passes -count=50 niced under load after the fix
- [x] No timeout raised and no retry added

## Notes


Cause (tigress): gitbranch.runAt (gitbranch.go:185-200) and catchup.go blob() run git with WaitDelay = 2 s and read stdout/stderr through pipes (c.Output). After git exits, os/exec Wait -> awaitGoroutines starts a 2 s wall-clock timer for the pipe-copying goroutines (go/src/os/exec/exec.go awaitGoroutines). When the test process is CPU-starved (niced at load 300-700), those goroutines don't run within 2 s, so a git that exited 0 returns ErrWaitDelay 'exec: WaitDelay expired before I/O complete' and its stdout is lost. On a commit the commit landed, but hug reports failure (34r1ft-full2.txt: '[hug/tasks d05097d] hug task wait xbqnj6 ... exec: WaitDelay expired before I/O complete'). On Locate's rev-parse, openBoard maps any error to 'hug task needs a Git checkout as its working directory' (cli.go:266), which is TestDeduceTwoClonesOneLine 'deduce_test.go:150: [sync]: hug task needs a Git checkout' and TestLintStaleOwnerSessions. Reproduction, main 477696be, gitbranch.Locate in a process with 400 spinning goroutines on GOMAXPROCS=1, 90 s: 'busy=400 Locate calls=282 WaitDelay errors=5 other errors=0', first: 'rev-parse --path-format=absolute --show-toplevel --git-common-dir: exec: WaitDelay expired before I/O complete'. A minimal os/exec probe of the same shape gives 'mode=pipe busy=400 runs=73 errors=4 (git exited 0 in 4) empty stdout 4', and with stdout/stderr to files 'errors=0'. iwazaru's runs (34r1ft HEADs) contain 6ecd08fc, so maintenance is not the cause. A full niced cli run on 477696be passed (load 192-402): the flake needs heavier starvation.
check 1: Cause: gitbranch.runAt and catchup blob() read git's output through pipes with WaitDelay 2 s. After git exits, os/exec gives the pipe-copying goroutines 2 s, and a CPU-starved niced process misses that. A git that exited 0 then returns 'exec: WaitDelay expired before I/O complete' and its output is lost. Reproduction on 477696be (gitbranch.Locate, 400 spinning goroutines, GOMAXPROCS=1): 'busy=400 Locate calls=549 WaitDelay errors=12 other errors=0', first '--version: exec: WaitDelay expired before I/O complete'. TestDeduceTwoClonesOneLine is the same: openBoard maps Locate's error to 'hug task needs a Git checkout' (cli.go:266).
check 3: Diff 477696be..f6834e53 touches no timeout (WaitDelay stays 2 s, NetTimeout unchanged) and adds no retry or skip: git's output goes to temp files (huginn/gitbranch/output.go).
Fix f6834e53 (failing test 9bc7ae14) on branch ad62aa/ci/ad62aa, base 477696be. Fail first: 'output_test.go:47: Locate after a git that exited 0 (2.044945708s): ... exec: WaitDelay expired before I/O complete'; mutation (output() back to pipes) fails the same way at 2.04 s. After: TestGitAnswerOnceGitExits -count=50 niced 'ok huginn/gitbranch 20.118s' (load 97-123). Paired starved amplifier, same load, 240 s: before 'Locate calls=549 WaitDelay errors=12', after 'Locate calls=46 WaitDelay errors=0'. TestDeduceTwoClonesOneLine -count=50 niced: PASS (load 57-104). Full niced cli package: PASS, gitbranch: 'ok 67.101s'. Before the fix, a full niced cli run on 477696be at load 192-402 also passed, so the flake needs heavier starvation than this machine had now; the amplifier is the fail-first evidence. Cost unloaded: git --version 19.6-20.6 ms with files vs 17.7-22.9 ms with pipes. CI 44391237795 running. Report: ~/.jcode/scratch/ad62aa-ev/REPORT.md
check 2: Fail first at f7903264 on main d5c66a1f: 'output_test.go:47: Locate after a git that exited 0 (2.052644792s): ... exec: WaitDelay expired before I/O complete'. Fix e6e3c36b: TestGitAnswerOnceGitExits -count=50 niced 'ok huginn/gitbranch 9.304s'. Amplifier paired at the same load: before 'Locate calls=549 WaitDelay errors=12', after 'Locate calls=46 WaitDelay errors=0'. TestDeduceTwoClonesOneLine -count=50 niced: PASS. Niced suites on the rebased tree: gitbranch 'ok 55.515s', cli 'ok 267.615s'. CI 44396660537 on e6e3c36b: success, 9 of 9 jobs, windows-latest included.
Rebased onto d5c66a1f, as crab asked. Commits: f7903264 (failing test), e6e3c36b (fix + CHANGELOG). Main's catch-up now reads objects with cat-file --batch (objects.go: StdoutPipe read directly, no WaitDelay), so only runAt changed. The earlier CI 44391237795 failure, windows lost-ack 'TempDir RemoveAll cleanup: ... remote.git: being used by another process', came from the old sleep-3 post-receive fixture. 3t4a49 on main already removed it. No fixture edit was needed and upgrade_test.go is untouched. CI 44396660537: success. Report: ~/.jcode/scratch/ad62aa-ev/REPORT.md
landed c923a71 on main 2026-11-18: gitbranch: failing test, a git that exited 0 is its answer though its output is…
landed c25065c on main 2026-11-18: gitbranch: git's output goes to temp files, read once git exits (ad62aa)
shipped in v0.7.71 (c923a71)
## History
- 2026-11-18T20:21 todo -> doing/active (take)
- 2026-11-18T22:10 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 doing/waiting -> done (done, by release v0.7.71: every check holds); was doing_since: 2026-11-18T20:21; waiting_asked_by: tigress; waiting_on: c25065c0 (ci/ad62aa-rebased) to land on main; all 3 checks ticked; close when landed; waiting_since: 2026-11-18T22:10
