---
id: wdkqzx
title: "Lint asks no question about a task that already waits on a person"
state: "doing"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T23:22"
stage: "active"
model: "claude-opus-5-5/high"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
The at3qep questions (landed-unchecked, shipped-not-done, todo-has-commit, doing-idle, branch-gone) are not raised on a task whose stage already waits on a named person (waiting with waiting_for, design, review), since its owner has said what is left. So a task waiting on Jørgen shows in WAITING only, never also in LINT. Other findings (broken files, missing sections) still show.

## Purpose
Jørgen 2026-11-17 23:20 local, on the v0.7.48 overview: 6ephqk (8/9, waiting on him for check 9) is in WAITING and in LINT at once: 'waiting for me is not a lint issue.' Its lint line: 'broken [shipped-not-done] shipped, but doing: release v0.7.45 contains commit f4ee48b naming 6ephqk (its note says so); nautilus: is it done?' The owner had already answered that by waiting on jhf.

## Principle
One place per fact: a question already asked of a person is not asked again by lint. Recomputed from state, nothing stored. A test fails first.

## Relevant details
huginn/internal/task/questions.go noteQuestion and IdleFindings: skip when the stage waits on a person (waiting_for set, or design/review). A plain wait with no person (waiting on CI) still gets the questions, since nobody answers it.

## Depends on

## Acceptance
- [ ] a doing task with a 'shipped in' note and waiting_for: jhf gives no shipped-not-done; the same task active does (test, fails on main)
- [ ] the overview shows 6ephqk in WAITING and not in LINT (golden or test)
- [ ] shipped-not-done never offers a fix that hug refuses: with open checks its fix names the first open check ('hug task check ID N', or strike it), not 'hug task done ID' (test: 8/9 settled, active, gives 'fix: hug task check ID 9')

## Notes


crab 23:48 local, from Jørgen's 'is this lint truly clear?': besides the waiting case, the fix line 'hug task done 6ephqk' is wrong on its own. Check 9 is open, so hug task done refuses with unchecked-acceptance (verified on a scratch board). A fix: line must be one hug accepts.
## History
- 2026-11-17T23:22 todo -> doing/active (take)
