---
id: bg1rd8
title: "A waiting task whose checks all hold is told to finish"
state: "doing"
type: bug
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzzs"
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


## History
- 2026-11-18T13:22 todo -> doing/active (take)
