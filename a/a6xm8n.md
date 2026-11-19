---
id: a6xm8n
title: "Hook budget cuts a call mid-flight and logs the run as failed"
state: "doing"
type: feature
priority: ""
assignee: ["cactus"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T05:55"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A hook capture stops before the call it cannot finish: no extraction call starts unless the remaining hook budget covers that call's expected latency, so a run that ran out of time ends as 'deferred to next hook (budget)', not as an error with a paid call thrown away.

## Purpose
capture_runs 8467 (2026-11-19T04:29Z, v0.7.76): 13 calls succeeded (2.4-5.2 s each), call 14 was cancelled by the context deadline with 0 completion tokens and the run was recorded as 'extraction timeout or cancellation: context deadline exceeded'. 8 such deadline errors since 2026-11-12.

## Principle
Keep the hook's time bound; only change when the last call starts. No added paid call.

## Relevant details
muninn/internal/app/capture.go: the loop checks 'time.Until(deadline) < 3*time.Second' before a chunk, but calls take up to 5.2 s in 8467 (latency_ms 2422-5215). Use a margin from observed call latency (e.g. the max of this run's calls so far) instead of a fixed 3 s.

## Depends on

## Acceptance
- [x] a test with a fake extractor taking 5 s and a hook deadline 4 s away: the run ends with skipped 'remaining messages deferred to next hook (budget)' and no error, and no call is started (fails before)
- [ ] capture_runs on the live store shows no 'context deadline exceeded' on turn_end hooks for a day after release (count in a note)

## Notes

crab 04:55Z, second instance: capture_runs 8468 (04:39:46Z) processed 855 msgs, 88 actions over 13 calls, then call 13 got 0 completion tokens in 417 ms and the run was recorded 'invalid extraction response (body redacted)' (error_class parse): the deadline cut the response body mid-read, so a budget cut shows up as a parse error too. Both 8467 and 8468 used extract_ms ~50 s + plan_ms ~112 s; the per-chunk check at capture.go (3 s left) and the provider reserve (2 s, capture.go:396-400) do not cover a 2-6 s call. Acceptance should also cover: a call cancelled by the hook budget is never reported as a parse error.


cactus: fail-first c1832e85, fix c2b0bbe5 on 1a49540d; pushed a6xm8n and ci/a6xm8n, CI run 44464516250. plan_ms (~112-116 s) is inside the run budget (RunTimeout = 4 x 45 s = 180 s; extract 48-52 s + plan 112-116 s + apply 12 s = 176 s in 8467/8468); it is summed per-candidate capturePlan (memoryMatches) and per-chunk search. The gate now measures the per-chunk search (slowSearch) and per-chunk planning+save (slowAfter) and needs both before a chunk.
check 1: TestCaptureHookDefersWhenBudgetCannotCoverCall (muninn/internal/app/capture_budget_test.go): 5 s extractor, 4 s hook deadline: skipped 'remaining messages deferred to next hook (budget)', err nil, 0 calls started, audit error empty; on 1a49540d it failed with 'extraction timeout or cancellation: context deadline exceeded' after 5.05 s (fail-first c1832e85). Mid-body cut: TestCaptureBudgetCutMidBodyIsTimeoutNotParse + extract TestDeadlineMidBodyIsTimeout give class timeout (was parse). Fix c2b0bbe5; CI run 44464516250 success on ubuntu, macos, windows, bench, windows-cli.
## History
- 2026-11-19T05:55 todo -> doing/active (take)
