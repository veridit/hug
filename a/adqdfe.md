---
id: adqdfe
title: "TUI reload waits on git start under load: test 5 s wait trips"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
TestChangeBeforeWatchIsUpIsSeen (and any TUI reload) does not depend on how fast the machine forks git: the reload after the watch comes up lands within the test's 5 s on a loaded machine, or the test waits on the reload itself rather than on time.

## Purpose
Seen once in 3000 starved runs on origin/main 8a45b0eb (mq5ger, not its cause): '--- FAIL: TestStarvedChangeBeforeWatch (11.38s) taskdirs_test.go:109: no reload after the watch came up', load average ~270. A flake on a busy CI runner costs every lane a rerun.

## Principle
Find the cause under forced load first, as mq5ger and ad62aa did; no raised timeout, no retry, no skip. Not tigress's hfkt1n (that is a torn tmux capture after Esc).

## Relevant details
Stack at the failure (file:/Users/jhf/.jcode/scratch/mq5ger-probes/reload-stuck-stack.txt, from mq5ger-stacks-stock.txt): the reload goroutine sits in syscall.forkExec (readlen on the exec pipe) under os/exec.Cmd.Output, called from tui.git (huginn/internal/tui/closed.go:35) <- closeLog.times (closed.go:46) <- loadSnapshot (tui.go:619) <- model.reload (tui.go:371). Every TUI load forks git rev-parse (and git log on a new head) for close times before it returns. Repro: TestStarvedChangeBeforeWatch (STARVE=8 GOMAXPROCS=4, -count=1500) from mq5ger's probes in /Users/jhf/.jcode/scratch/mq5ger-probes/zz_starve_test.go, on a machine already at load ~250.

## Depends on

## Acceptance
- [ ] the failure reproduced under forced load with the verbatim line, and its cause named with file:line
- [ ] fixed at the cause: 0 failures in the same forced-load run, no timeout raised, no retry, no skip

## Notes
