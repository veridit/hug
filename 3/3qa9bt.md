---
id: 3qa9bt
title: "TestSettleWaitsOnProgressNotAClock/progressing overshoots on CI"
state: "doing"
type: bug
priority: "high"
assignee: ["giraffe"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-21
updated: 2026-11-21
doing_since: "2026-11-21T14:18"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
The progressing case returns within its 1 s tolerance on a synctest clock every time, or the test states why the overshoot is allowed.

## Purpose
It failed release.yml for v0.7.95 (run 44907060648) and ci/qcq3tm (44720742380); a synctest clock should make it deterministic.

## Principle
A test that fails once gets a real fix, shown failing on today's code first (stress run), never a widened tolerance without a cause.

## Relevant details
Failures, verbatim: run 44907060648 (release v0.7.95, 2026-11-21 13:06Z) 'sync_timeout_output_test.go:88: settleWait returned "" after 1m32.6s, want about 1m30s'; run 44720742380 (ci/qcq3tm, 2026-11-20 10:37Z) 'settleWait returned "" after 1m31.95s, want about 1m30s'. Test: huginn/internal/cli/sync_timeout_output_test.go:49, from 4ebd782a (6zhyq2). Inside synctest.Test, the writer goroutine logs every 100 ms for 3*settleStall then calls release(); settleWait (sync_test.go:150) polls every 50 ms until !bg.Busy() && !bg.Pending(). On a fake clock the extra 2-2.6 s means some wait after release() or in Busy/Pending is a timer (a real or fake sleep or backoff in gitbranch.Background release or Pending), not CPU. Find which, then fix the code or the bound for that reason.

## Depends on

## Acceptance
- [ ] A stress run (go test -run TestSettleWaitsOnProgressNotAClock -count=500, niced) shows the overshoot on main first, with the cause named in a note
- [ ] After the fix, the same stress run passes 500 of 500 and CI is green twice

## Notes


Cause (giraffe, claude-opus-5-5): internal/flock/flock.go:28 and :46: release is f.Close() alone. flock(2) locks belong to the open file description; a parallel test in the same process that starts git (fork then exec) copies the fd table, and until that child execs, its copy keeps the lock. So after the writer's release(), bg.Busy() (Queued's TryAcquire .queue and RunHeld's TryAcquire .run, gitbranch/background.go:145-216) still reports held for a few real ms; each such settleWait poll (cli/sync_test.go:150) advances the synctest fake clock 50 ms at once, so 40-52 polls = the 2-2.6 s overshoot. Busy's own TryAcquire probes are inherited too and look held to the next poll. Stress: go test -run TestSettleWaitsOnProgressNotAClock -count=500, niced, on origin/main c4389611: ok 108.8s, no overshoot alone (no forks in that process); -count=100 with a parallel neighbour exec'ing /usr/bin/true: ok too. Direct probe (Exclusive, release, poll Busy under 8 goroutines exec'ing /usr/bin/true, darwin/arm64, GOMAXPROCS=2 nice 19): 'releases still held after close: 1218 of 2000, worst 108 polls'. Fix: unlock (LOCK_UN / UnlockFileEx) before Close, which frees the description whatever copies a fork holds.
## History
- 2026-11-21T14:18 todo -> doing/active (take)
