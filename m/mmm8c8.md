---
id: mmm8c8
title: "Flaky TestSettledWaitsForSlowChild since the 100 ms settle"
state: "done"
type: bug
priority: "high"
assignee: ["wyvern"]
labels: []
group: ""
rank: "zzzzzzzzzzm1"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T04:46"
done_release: "v0.7.55"
---

## Goal
TestSettledWaitsForSlowChild passes on every run on every OS, because the code it tests waits for the pass to push by a signal, not by time; the test counts events, never sleeps.

## Purpose
nautilus 2026-11-18 04:25 local: macOS CI failed once with huginn/internal/cli sync_test.go:184 'settled returned before the pass pushed', then passed on rerun. 0sbvnq (1c40426d, wyvern) cut the sync tests' settle from 1.5 s to 100 ms; a test that depends on a settle time is a timing test. Jørgen's rule: never skip a test or raise a limit to hide a flake.

## Principle
Fix the cause: settled must return only after the pass it waits for has pushed, by a signal the pass writes. Do not raise the settle back, do not retry the test, do not skip it.

## Relevant details
Test: huginn/internal/cli/sync_test.go:184 TestSettledWaitsForSlowChild; settle from HUG_TEST_SETTLE (0sbvnq). Prove the fix with a counting stress run (-count=200 under load, or a forced child delay via HUG_TEST_CHILD_DELAY) that fails first on main.

## Depends on

## Acceptance
- [x] A stress run that fails first on main (N failures of M runs, quoted) passes M of M after the fix
- [x] No sleep or settle time decides the test's outcome (code review note naming the signal)

## Notes


Cause: Queued() probes by TryAcquire+release of the queue lock; a syncer's Run TryAcquire in that window failed and it exited without a pass. Path in the CI failure: child held 6 s > pendingFor 5 s, so settled's own polls (Busy->Queued) fell through to the lock probe and met the child. Fix 4336e2b0: probe holds .probe while it holds the queue; a refused Run waits on .probe and retries once (refused again = a real syncer). Fails first: TestProbeNeverCostsASyncerItsTurn (300 Runs vs a looping Queued) on main: 10 of 10 runs fail, 4-6 of 300 lost, e.g. 'background_test.go:40: 5 of 300 syncers gave up their turn to a probe and ran no pass'. Fixed: -count=10 ok (0 of 3000). -race gitbranch 20.3 s, cli 136.3 s ok. Check 2 signal: the outcome is decided by the .queue/.probe locks and the pass's own push (test reads the bare remote's head), no settle time decides it: settled's 30 s is a failure deadline only. CI 44231734696 pending. Also removed a stray '||||||| parent of c9ad1b2e' line from CHANGELOG.md (from 27643306).
landed 4336e2b on main 2026-11-18: background sync: a queue check never costs a syncer its pass (mmm8c8)
Main CI 44231734696 (4336e2b0) green: ubuntu, macos, windows, test-windows-cli, bench, secrets all success; TestProbeNeverCostsASyncerItsTurn and TestSettledWaitsForSlowChild pass on every OS.
Reflection (dismissed, 2026-11-18, wyvern): The flake looked like a settle-time problem (0sbvnq had just cut it to 100 ms), but the cause was a lost wakeup: a read-only probe briefly took the same lock the syncer uses to decide whether it is redundant. Lesson: a try-lock used both as a query and as an ownership claim can't tell an observer from an owner; give observers their own lock. The counting stress test found it in 0.2 s, where the timed test took a CI run to hit it once.
shipped in v0.7.55 (4336e2b)
Reflection 1 dismissed (tb0dra check 5 triage, crab 2026-11-19): G10, a lesson about one's own method, fixed in its own ticket; no instruction misled.
## History
- 2026-11-18T04:26 todo -> doing/active (take)
- 2026-11-18T04:38 doing/active -> doing/waiting (wait)
- 2026-11-18T04:46 doing/waiting -> done (done); was doing_since: 2026-11-18T04:26; waiting_asked_by: wyvern; waiting_on: main CI 44231734696 (4336e2b0); waiting_since: 2026-11-18T04:38
