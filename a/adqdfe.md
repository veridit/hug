---
id: adqdfe
title: "TUI reload waits on git start under load: test 5 s wait trips"
state: "done"
type: feature
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzx9"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "adqdfe"
worktree: "/Users/jhf/.jcode/scratch/adqdfe"
done_at: "2026-11-19T01:33"
done_release: "v0.7.74"
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
- [x] the failure reproduced under forced load with the verbatim line, and its cause named with file:line
- [x] fixed at the cause: 0 failures in the same forced-load run, no timeout raised, no retry, no skip

## Notes


check 1: Reproduced: TestStarvedChangeBeforeWatch on 7f14e3e1 (STARVE=8 GOMAXPROCS=4, 1000): 4 FAIL, each 'taskdirs_test.go:109: no reload after the watch came up' at 5.21-5.82 s (mq5ger-starve-final-raw.txt). Deterministic: git that sleeps 6 s at start (PATH shim) on origin/main 356fb240: '--- FAIL: TestChangeBeforeWatchIsUpIsSeen (23.14s) taskdirs_test.go:109: no reload after the watch came up'. Fail-first unit: TestReloadDoesNotWaitOnGit (git behind a gate): 'adqdfe_test.go:59: the reload waited on git: no board while git could not start'. Cause: loadSnapshot calls closeLog.times before returning (huginn/internal/tui/tui.go:628), which forks git rev-parse and git log (closed.go:35, :46); the stack mq5ger-stacks-final.txt:2291-2297 has the reload goroutine in os/exec under closeLog.times.
check 2: Fixed at the cause in 87691f85 (ci/adqdfe, on watchflake 02151406): loadSnapshot takes closeLog.latest() (no git); the loadedMsg batches readCloses (git off the UI thread) and closesMsg applies the newest times read; refresh (the TUI's own writes) still reads first. Same forced-load run (STARVE=8 GOMAXPROCS=4 nice -p=1 -count=1000, load ~500): 1000 PASS, 0 FAIL, 'ok  github.com/veridit/hug/huginn/internal/tui 469.955s' (adqdfe-starve-fixed.txt). Was 996/4 on 7f14e3e1. Git sleeping 6 s at start: TestChangeBeforeWatchIsUpIsSeen PASS (was FAIL). TestReloadDoesNotWaitOnGit passes (failed first). No timeout raised, no retry, no skip. CI 44427265935 success 8/8: ubuntu 'go test -race ./...' 'ok  github.com/veridit/hug/huginn/internal/tui 299.278s', macos 'ok ... tui 147.354s'.
Reflection (open, 2026-11-19, macaque): The handover said 'Reproduce it first with kikazaru's probe', which held; the gap was that the stack file named the cause but the 4 in 1000 rate made fail-first slow, so a PATH git that sleeps 6 s and a gated git seam made it deterministic.
landed bf170d4 on main 2026-11-19: tui: a reload does not wait on git for close times (adqdfe)
shipped in v0.7.74 (bf170d4)
## History
- 2026-11-19T00:45 todo -> doing/active (take)
- 2026-11-19T01:24 doing/active -> doing/waiting (wait)
- 2026-11-19T01:33 doing/waiting -> done (done); was doing_since: 2026-11-19T00:45; waiting_asked_by: macaque; waiting_on: crab review; waiting_since: 2026-11-19T01:24
