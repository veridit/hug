---
id: bg1rd8
title: "A waiting task whose checks all hold is told to finish"
state: "done"
type: bug
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzznv"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:54"
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
- [x] A doing task in waiting with every check ticked and every child done gets a lint finding naming hug task done ID; a test fails first on main.
- [ ] ~~The board cue reaches its owner, or the coordinator when the owner is gone, with that finding (test).~~ (not relevant: split to follow-up agh2h7: boardcue.go is in duckling's 4vxqp4, agreed 12:15Z, piglet, 2026-11-18)
- [x] A waiting task with an open check is unchanged (test).

## Notes


commit 59fdd2c on branch bg1rd8 (stacked on gpyqq9 d177829), ci/bg1rd8 run 44306358102. check 1: TestWaitingTaskWhoseChecksAllHold (huginn/internal/task/questions_test.go) fails before: 'got <nil> want "every check holds, but doing/waiting on landing slot for 25 hours: 2 of 2 checks are ticked or struck and 2 of 2 children are done; wyvern: is it done?", fix hug task done ID'; TestLintAsksAWaitingTaskWithEveryCheckToFinish (cli) fails before: 'no findings'. Both pass; code checks-all-hold, broken, a question code; only after the wait is IdleAfter (2h) old, not with an open dep/board condition, not for ask --after waits. check 3: same test, waiting with check 2 open (just parked and 25h later) and with 2 open children gives no question. Local niced suite: ok huginn/internal/task 11.376s, ok huginn/internal/cli 163.963s, ok muninn/internal/recall 0.564s, all huginn+cmd/hug ok. check 2 (board cue) split to agh2h7 (depends on bg1rd8), agreed with duckling: duckling's 4vxqp4 owns boardcue.go now.
CI green: run 44306358102 (59fdd2c, branch head): secrets, bench, test ubuntu/macos/windows, test-windows-cli all success.
crab review 13:02Z: ChecksHoldFindings read (waiting only, every check settled, every child done/dropped, done would pass, not ask --after, wait >= IdleAfter). 2 h threshold accepted: parking a finished task briefly for CI or review is normal. Exact SHA 59fdd2c9 CI 44306358102 all 6 green. Lands right after gpyqq9 (it builds on StageSettles).
Landed on main via 5a8b14da (stack bg1rd8 f7d607e4, qpp30y 3d74229a, 2k68eg 7fafab16, vk9td0 5a8b14da; CI 44328375780 exact SHA 6/6, ubuntu passed on rerun after an unrelated muninn/internal/project flake, see ybbfsa). Ships in the next release.
landed f7d607e on main 2026-11-18: lint: a waiting task whose checks all hold is asked to finish (bg1rd8)
shipped in v0.7.65 (f7d607e)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T13:22 todo -> doing/active (take)
- 2026-11-18T13:42 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T13:22; waiting_asked_by: piglet; waiting_on: crab review; waiting_since: 2026-11-18T13:42
