---
id: 3nhp67
title: "Stage changes clear the old stage's fields: wait/resume leave asked_* behind"
state: "doing"
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T02:26"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Every stage change clears every field of the stage it leaves, so a task never carries two stages' fields, and the lint fix for stage-disagrees is a command that settles it.

## Purpose
s6ff5k on hug's board, 2026-11-18: after hug upgrade moved it to asked (8a18427b), crab ran hug task wait at 02:05 local: wait wrote waiting_on/since/asked_by and left stage asked with its asked_* fields (8366bc00). hatchling's resume (5408358f) removed the waiting_* fields but kept asked_by/for/since/what; its wait (5b8733b9) then set stage waiting. Now the file holds both asked_* and waiting_*, lint says broken [stage-disagrees] 'its asked fields are set, so its stage is asked', and its fix 'hug task set s6ff5k --state doing' does nothing (it is already doing).

## Principle
A stage's fields live only while the task is in that stage; the History line keeps what was there. A command refuses, with a fix, rather than writing a half state. Lint fixes must be commands that settle the finding.

## Relevant details
Code: huginn/internal/task stage transitions (wait, resume, ask, answer, review, design), MoveToAsked, the stage-disagrees lint. Repair s6ff5k with the fixed build: it should end as doing/waiting with only waiting_* fields.

## Depends on

## Acceptance
- [ ] wait on an asked task either refuses with a fix line or moves it to waiting and clears asked_* (History keeps them); same for every pair of stages (table test, fails first on main)
- [ ] resume from asked or waiting clears that stage's fields and only those (test)
- [ ] stage-disagrees' fix is a command that, when run, makes lint report nothing for that finding (test runs the fix)
- [ ] s6ff5k on hug's board ends as doing/waiting with no asked_* fields, done with hug commands

## Notes


landed 5b276f2 on main 2026-11-18: task: one stage at a time; a same-state set settles two stages (3nhp67)
## History
- 2026-11-18T02:26 todo -> doing/active (take)
