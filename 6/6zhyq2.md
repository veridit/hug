---
id: 6zhyq2
title: "TestViewKeysBlinkAfterV flakes on CI timing (497 ms vs 530 ms)"
state: "done"
type: bug
priority: ""
assignee: ["bird"]
labels: []
group: ""
rank: "zzzzzzzzzzx4"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T01:15"
done_release: "v0.7.86"
parent: "ddpxs8"
---

## Goal
The view-line blink tests pass on CI regardless of tick timing.

## Purpose
maple 2026-11-19 (yp3c09 CI 44520969132): TestViewKeysBlinkAfterV failed once measuring 497 ms against the expected 530 ms; the test is from a82j3n/r80svs, not yp3c09.

## Principle
Drive the blink by sending cursor blink messages, never by wall-clock sleeps; no retries to hide a flake.

## Relevant details
huginn/internal/tui/a82j3n_test.go, r80svs_test.go: find any time.Sleep or real tick in the blink tests and replace with explicit blink msgs. Same check for TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade (Windows, huginn/internal/cli), seen once in yp3c09 CI 44522707343 attempt 1.

## Depends on

## Acceptance
- [x] The blink tests use no wall-clock waits (grep in a note) and pass 50 runs in a row niced (-count=50)
- [x] TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade failure on Windows is reproduced or explained in a note

## Notes


Check 1 evidence (branch 6zhyq2, blink commit 3342fe76 on 57fce63b). BEFORE, grep -n 'realTick\|time.Since\|time.Now\|BlinkSpeed = time.Millisecond' over a82j3n/r80svs/911j70/qpp30y tests at 57fce63b: a82j3n_test.go:52 'func realTick(t *testing.T, cmd tea.Cmd) (tea.Msg, time.Duration)', :54 'start := time.Now()', :56 'took := time.Since(start)', :111 and :122 'realTick(t, cmd)', r80svs_test.go:140 'msg, took := realTick(t, cmd)', 911j70_test.go:179 and qpp30y_test.go:112 'c.BlinkSpeed = time.Millisecond' (a real 1 ms timer). AFTER: realTick is gone; the two real-tick tests run inside synctest.Test and bubbleTick asserts exactly 530ms on the bubble's clock (a82j3n_test.go:61 'took != 530*time.Millisecond'); vBlinkTick/blinkTick use 'c.BlinkSpeed = 0' (deadline already past, no timer). Mutation: vCursor.BlinkSpeed=497ms in tui.go -> 'a82j3n_test.go:120: the blink tick came after 497ms of the bubble's clock, want 530ms' (reverted). -count=50: 'GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=50 -run ^(TestViewKeysBlinkAfterV|TestViewLineAfterVBlinksTheChoices|TestNoCursorCellAfterView|TestViewKeysSteadyAfterSecondKey|TestPickedVIsUnderlined|TestViewBlinkGolden|TestViewLineNoColorKeepsCues|TestViewCursorBlinksAfterV|TestViewCursorGolden|TestBoxCursorBlinksUnderline|TestBoxGolden)$ ./huginn/internal/tui/' at 3342fe76 -> 'ok  	github.com/veridit/hug/huginn/internal/tui	42.264s'; again with -v: 'ok  	github.com/veridit/hug/huginn/internal/tui	41.344s', 50 PASS lines for TestViewKeysBlinkAfterV, 50 for TestViewLineAfterVBlinksTheChoices, 550 top-level PASS, 0 FAIL.
Check 2: EXPLAINED and fixed (commit 4ebd782a). Cause: a slow Windows runner, not a hang. settled() failed any pass still running at a fixed 30 s. Evidence from 'gh run view 44522707343 --attempt 1 --log-failed' (job 135555232846): 'sync_semantic_test.go:166: background pass still running after 30 s: queued false, busy true, pending false, turn held false, run held true'. The pass (pid 1360) took the run lock at 12:24:09.905, logged 'board turn taken; integrating' at 12:24:11.571, and trace2 shows it was still starting git: cat-file --batch at 12:24:24.567 (its object reader, kept open for the replay by design, objects.go) and 'git -C ...hug-catchup-832927163 add --all -- .' at 12:24:39.453760Z. The 30 s window began after the note returned, about 12:24:09.8, so the deadline fell about 12:24:39.8: the pass started a new git 0.35 s before it. It was replaying (resolveDiverged -> replay in a private worktree) because A and B both appended to the same task after the upgrade. The runner was slow across the shard: per-test durations in attempt 1 vs a later green full log (ci/6zhyq2 44638912154): TestUpgradeMovesBoardToLayout3 93.76 vs 11.86 s (7.91x), TestOrdinarySyncLegacyLayoutDoesNotImplicitlyUpgrade 113.23 vs 14.60 (7.76x), TestOrdinarySyncUnreadableTaskKeepsLocalCommits 145.42 vs 23.03 (6.31x), this test 110.30 vs 32.54 (3.39x). Shard total 414.519s vs 162.8-322.0 s across 16 green runs. Attempts 2 and 3 of the same run passed (210.215s). LOCAL REPRO at c7f69d12 (before the fix): a PATH git wrapper that sleeps 1 s only when GIT_TRACE2_EVENT is set (only the background child has it) -> '--- FAIL: TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade (66.57s)' with 'background pass still running after 30 s: ... turn held true, run held true', last phase 'board turn taken; integrating', 'git never exited: [... cat-file --batch]'. FIX: settled waits while the pass log or the child's trace2 folder grows. It fails after 30 s with neither ('no progress for 30s'), or after 3 min in all ('still going after 3m0s'). Same repro after the fix: 'ok  	github.com/veridit/hug/huginn/internal/cli	67.348s'. New TestSettleWaitsOnProgressNotAClock (synctest clock, 3 cases: progressing waited for, silent fails at 30s, capped fails at 1m). Mutation 'last = p' (progress ignored = old behaviour) -> 'settleWait = "no progress for 30s (30s in all)" after 30s, want ""' and '... want "still going after 1m0s"'. Full packages niced at 4ebd782a: 'ok  	github.com/veridit/hug/huginn/internal/cli	184.131s', 'ok  	github.com/veridit/hug/huginn/internal/tui	147.218s', 'ok  	github.com/veridit/hug/internal/env	0.451s'. No new HUG_* variable.
Side finding, not fixed here (outside this ticket): first CI on ci/6zhyq2, run 44638912154 at c7f69d12 (tui tests and CHANGELOG only), failed test-windows-cli (sync) on another wall-clock bound: 'sync_test.go:294: note took 2.749082s with the remote hanging, 417.7603ms without background sync: it waited on the network', '--- FAIL: TestBackgroundSyncOfflineCatchesUp (40.92s)'. The bound is baseline+max(2s, baseline). The same 0.35-s-type slow-runner margin. Candidate for its own ticket.
CI green: run 44642751056 on ci/6zhyq2 at 597a0b24d3f0aa50493c2129e21f8700b5219f46 (= branch 6zhyq2 tip, on origin/main 57fce63b). Conclusion success, all 8 jobs success: secrets, bench, test (ubuntu/macos/windows-latest), test-windows-cli (sync), test-windows-cli (rest), test-windows-cli-shards. Superseded run 44638912154 (c7f69d12) had already completed (failure, TestBackgroundSyncOfflineCatchesUp, see side note); nothing to cancel.
Landed on main 597a0b24 by fast-forward (CI 44642751056 8/8 green on that exact SHA). Reviewed settledWait: fails at 30 s without progress or 3 min in all, so a hang still fails; no retry. Test-only, so it ships with the next user-facing release.
Reflection (dismissed, 2026-11-20, crab): Nothing misled me; bird's report was grounded and complete.
landed 3342fe7 on main 2026-11-20: tui: blink tests run v's blink on a synctest clock, no wall-clock waits (6zhyq2)
landed c7f69d1 on main 2026-11-20: changelog: blink tests on a synctest clock (6zhyq2)
landed 4ebd782 on main 2026-11-20: cli tests: settled fails a sync pass that stops, not one still running at 30 s…
landed 597a0b2 on main 2026-11-20: changelog: settled waits on progress (6zhyq2)
shipped in v0.7.86 (3342fe7)
## History
- 2026-11-20T00:04 todo -> doing/active (take)
- 2026-11-20T01:15 doing/active -> done (done); was doing_since: 2026-11-20T00:04
