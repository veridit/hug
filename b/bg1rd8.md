---
id: bg1rd8
title: "A waiting task whose checks all hold is told to finish"
state: "doing"
type: bug
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzzl"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T13:22"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When every acceptance check of a doing task is ticked (and every child is done), hug tells its owner and the coordinator to finish it, whatever its stage, instead of letting a stale waiting stage hide it.

## Purpose
Jørgen on 44w1ev 2026-11-18: all 5 checks ticked, all 3 children done, but it sat in waiting for a day on a long-landed 'landing slot'; nothing told any agent to close it. rt80x4 had the same shape (all 3 ticks, waiting on a handoff).

## Principle
By design, not doctrine: the board says what is left. All checks holding is a fact hug can see; it must surface as a finding with a runnable fix, not depend on someone noticing.

## Relevant details
questions.go noteQuestion silences shipped-not-done when stageSaysWhy (waiting/asked/review); an all-ticked waiting task therefore raises nothing. The board cue (muninn/internal/recall/boardcue.go) already says 'every check of X holds: close it' for active tasks only. Coordinate with gpyqq9 (ask refused by shipped-not-done).

## Depends on

## Acceptance
- [ ] A doing task in waiting with every check ticked and every child done gets a lint finding naming hug task done ID; a test fails first on main.
- [ ] The board cue reaches its owner, or the coordinator when the owner is gone, with that finding (test).
- [ ] A waiting task with an open check is unchanged (test).

## Notes


commit 59fdd2c on branch bg1rd8 (stacked on gpyqq9 d177829), ci/bg1rd8 run 44306358102. check 1: TestWaitingTaskWhoseChecksAllHold (huginn/internal/task/questions_test.go) fails before: 'got <nil> want "every check holds, but doing/waiting on landing slot for 25 hours: 2 of 2 checks are ticked or struck and 2 of 2 children are done; wyvern: is it done?", fix hug task done ID'; TestLintAsksAWaitingTaskWithEveryCheckToFinish (cli) fails before: 'no findings'. Both pass; code checks-all-hold, broken, a question code; only after the wait is IdleAfter (2h) old, not with an open dep/board condition, not for ask --after waits. check 3: same test, waiting with check 2 open (just parked and 25h later) and with 2 open children gives no question. Local niced suite: ok huginn/internal/task 11.376s, ok huginn/internal/cli 163.963s, ok muninn/internal/recall 0.564s, all huginn+cmd/hug ok. check 2 (board cue) split to agh2h7 (depends on bg1rd8), agreed with duckling: duckling's 4vxqp4 owns boardcue.go now.
## History
- 2026-11-18T13:22 todo -> doing/active (take)
