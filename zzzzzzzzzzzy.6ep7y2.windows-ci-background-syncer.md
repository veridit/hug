---
id: 6ep7y2
title: "Windows CI: a background syncer held the queue 30 s and no pass ran"
state: "doing"
type: feature
priority: ""
assignee: ["wyvern"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T06:31"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When a test's settled() times out, the pass log and the failure say which lock each syncer waited on and for how long, and the cause of TestBackgroundSyncBurstIsBatched's stall on Windows (CI 44237364634) is found and fixed, with a counting test that fails first.

## Purpose
A stuck queue means commits that never push; on CI it fails the windows-cli job (83 s). The current diagnostics (queued/busy/pending/turn) cannot tell a syncer waiting on the run lock from one settling or a pass that never ends.

## Principle
Measure before changing: add diagnostics, reproduce on a ci/ branch on Windows, then fix with a counting test. Do not paper over it by turning off throttling in tests.

## Relevant details
CI 44237364634 (a13d7c74), test-windows-cli: sync_test.go:223 'background pass still running after 30 s: queued true, busy true, pending false, turn held false'. Pass log: two passes at 04:31:24 and 04:31:28 (the task new), then none during the 20 notes or the 30 s after. queued true = a syncer past takeQueue (it holds the queue): in settle.wait (bounded by SettleMax 10 s) or in flock.Acquire(run). So a pass holding run did not end in ~90 s, while fetch/push have NetTimeout 60 s and log on timeout. takeQueue (mmm8c8, 4336e2b0) cannot spin: two TryAcquire and one Acquire(.probe), which a prober holds only across a non-blocking TryAcquire. Candidates (unmeasured): the pass at below-normal priority (b6jgpn, a201b2e6) starved by the parallel test load; local git without a deadline; deducePass. Earlier, different shapes: 44230245084 (pending true, ubuntu), 44156864971 (ubuntu). 8 of 8 local runs on macOS pass.

## Depends on

## Acceptance
- [ ] settled's failure reports the run lock held or free, and the pass log has a line when a syncer takes the queue and when it takes the run lock, with pid and time
- [ ] the Windows stall is reproduced on a ci/ branch with those diagnostics, and the cause is named on this ticket with the log lines
- [ ] a counting test that fails first on the cause, and passes with the fix; mutation check noted
- [ ] main CI green on all jobs with the fix (run id)

## Notes


## History
- 2026-11-18T06:31 todo -> doing/active (take)
