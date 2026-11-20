---
id: jsyksd
title: "TestDoneWithoutAnswerIsCounted fails when a run crosses midnight"
state: "done"
type: feature
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzztj"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T02:01"
done_release: "v0.7.74"
---

## Goal
The reflection note's date comes from the board clock (task.Now), and the test pins that clock, so TestDoneWithoutAnswerIsCounted passes whatever the wall time, midnight included.

## Purpose
CI 44424953831 attempt 1 (ci/reflections a0939668, an AGENTS.md-only change), job test-windows-cli (rest), at 2026-11-19T00:02:15Z: '--- FAIL: TestDoneWithoutAnswerIsCounted (3.58s)' with 'reflect_test.go:157: plain done:' and the task file holding 'Reflection (unanswered, 2026-11-19, rabbit)' and 'done_at: "2026-11-19T00:00"'. Attempt 2 of the same commit passed. The test computes today := time.Now().Format("2006-01-02") before the done; huginn/internal/cli/cli.go:1257 passes its own time.Now() to SetReflection. A run that starts before midnight and dones after it compares two different days.

## Principle
Pin the clock, never widen the check. The note's date uses task.Now, the board clock the other moves already use (done_at), so one fake clock governs both. The test is not parallel while it sets task.Now (as TestTimedWaitCommands). Tests fail first.

## Relevant details
Fix: cli.go:1257 time.Now() -> task.Now(); the test sets task.Now to a fixed 23:59:59 (a date that is not today, e.g. 2026-01-31) and reads today from task.Now. Grep the other reflection tests (reflect_test.go: TestDoneReflectAndReflections, TestReflectionsByInstruction) for the same time.Now() date pattern and pin them too. Reported by piglet 2026-11-19; crab asked for it to be filed.

## Depends on

## Acceptance
- [x] under a fake task.Now of 2026-01-31 23:59:59 the test fails before the fix (note dated the wall clock) and passes after (test, failing first)
- [x] every reflection test that compares a date reads it from the pinned clock; grep shows no time.Now() date in reflect_test.go
- [x] CI green on all jobs with the fix

## Notes


Fail first (/Users/jhf/.jcode/scratch/jsyksd-failfirst.txt), clock pinned at 2026-01-31 23:59:59.9 before the fix: '--- FAIL: TestDoneWithoutAnswerIsCounted (1.06s) reflect_test.go:157: plain done: ... Reflection (unanswered, 2026-11-19, rabbit)'; also TestDoneReflectAndReflections (reflect_test.go:38), TestLifecycleJSONContract (cli_test.go:100), TestStrikeThroughCLI (strike_test.go:21), TestStrikeOutCheck (task strike_test.go:32). After (jsyksd-after.txt): all 5 '--- PASS'. Grep of time.Now() in cli and task *_test.go: the 5 date comparisons (reflect_test.go:38,154, strike_test.go:21, cli_test.go:100, task/strike_test.go:26) all now read the pinned clock; the rest are durations/mtimes (sync_test, setupbacklog_test, scale_test, deliver_test, bench_scale_test), not dates. Not changed: Store.New's created/updated still use time.Now (no test compares them; moving them would shift TUI fixtures).
check 1: Under a fake task.Now of 2026-01-31 23:59:59.9 (pinClock, cli/clock_test.go) TestDoneWithoutAnswerIsCounted failed before the fix: 'reflect_test.go:157: plain done: ... Reflection (unanswered, 2026-11-19, rabbit)' (jsyksd-failfirst.txt); after cli.go:1257 time.Now() -> task.Now(): '--- PASS: TestDoneWithoutAnswerIsCounted (3.76s)' (jsyksd-after.txt). No sleep. 59f7d3d5.
check 2: Every date comparison under huginn/internal/cli and task tests reads the pinned clock: reflect_test.go (both tests), cli_test.go TestLifecycleJSONContract, strike_test.go TestStrikeThroughCLI, task/strike_test.go TestStrikeOutCheck (Store.Strike now dates with task.Now); all 5 failed first, all 5 pass. grep 'time.Now().Format("2006-01-02")' in cli/task *_test.go: 0 lines. Remaining time.Now() in those tests are durations or mtimes, not dates. Local: 'ok  github.com/veridit/hug/huginn/internal/task 13.591s', 'ok  github.com/veridit/hug/huginn/internal/cli 285.890s' (-short).
check 3: CI 44430077651 on 59f7d3d5 (ci/jsyksd): success 8/8 (secrets, bench, test ubuntu/macos/windows, test-windows-cli sync/rest, shards).
landed a592a11 on main 2026-11-19: done's reflection and strike's note dated by the board clock (jsyksd)
shipped in v0.7.74 (a592a11)
## History
- 2026-11-19T01:26 todo -> doing/active (take)
- 2026-11-19T01:48 doing/active -> doing/waiting (wait)
- 2026-11-19T02:01 doing/waiting -> done (done, by release v0.7.74: every check holds); was doing_since: 2026-11-19T01:26; waiting_asked_by: macaque; waiting_on: crab review; waiting_since: 2026-11-19T01:48
