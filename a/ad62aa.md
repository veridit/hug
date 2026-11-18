---
id: ad62aa
title: "cli tests: commit WaitDelay and TestDeduceTwoClonesOneLine fail under load"
state: "doing"
type: feature
priority: ""
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzzs"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T20:21"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] Cause named in a note with a log line or reproduction
- [ ] A test or amplified reproduction fails first, then passes -count=50 niced under load after the fix
- [ ] No timeout raised and no retry added

## Notes


Cause (tigress): gitbranch.runAt (gitbranch.go:185-200) and catchup.go blob() run git with WaitDelay = 2 s and read stdout/stderr through pipes (c.Output). After git exits, os/exec Wait -> awaitGoroutines starts a 2 s wall-clock timer for the pipe-copying goroutines (go/src/os/exec/exec.go awaitGoroutines). When the test process is CPU-starved (niced at load 300-700), those goroutines don't run within 2 s, so a git that exited 0 returns ErrWaitDelay 'exec: WaitDelay expired before I/O complete' and its stdout is lost. On a commit the commit landed, but hug reports failure (34r1ft-full2.txt: '[hug/tasks d05097d] hug task wait xbqnj6 ... exec: WaitDelay expired before I/O complete'). On Locate's rev-parse, openBoard maps any error to 'hug task needs a Git checkout as its working directory' (cli.go:266), which is TestDeduceTwoClonesOneLine 'deduce_test.go:150: [sync]: hug task needs a Git checkout' and TestLintStaleOwnerSessions. Reproduction, main 477696be, gitbranch.Locate in a process with 400 spinning goroutines on GOMAXPROCS=1, 90 s: 'busy=400 Locate calls=282 WaitDelay errors=5 other errors=0', first: 'rev-parse --path-format=absolute --show-toplevel --git-common-dir: exec: WaitDelay expired before I/O complete'. A minimal os/exec probe of the same shape gives 'mode=pipe busy=400 runs=73 errors=4 (git exited 0 in 4) empty stdout 4', and with stdout/stderr to files 'errors=0'. iwazaru's runs (34r1ft HEADs) contain 6ecd08fc, so maintenance is not the cause. A full niced cli run on 477696be passed (load 192-402): the flake needs heavier starvation.
## History
- 2026-11-18T20:21 todo -> doing/active (take)
