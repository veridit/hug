---
id: jsyksd
title: "TestDoneWithoutAnswerIsCounted fails when a run crosses midnight"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-19
updated: 2026-11-19
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
- [ ] under a fake task.Now of 2026-01-31 23:59:59 the test fails before the fix (note dated the wall clock) and passes after (test, failing first)
- [ ] every reflection test that compares a date reads it from the pinned clock; grep shows no time.Now() date in reflect_test.go
- [ ] CI green on all jobs with the fix

## Notes
