---
id: bqfrda
title: "detach leaves a citation that lint reports as missing-file twice"
state: "doing"
type: bug
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzzw"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T14:03"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug task detach ID FILE removes the file and leaves no dangling citation, or marks it (lost); lint never reports the same missing file twice.

## Purpose
piglet 2026-11-18 while working gpyqq9: after detach, the file: citation in the notes stays and lint reports missing-file twice (also on main).

## Principle
A command must not create a lint finding it could have avoided; one fact, one finding.

## Relevant details
Seen on piglet's gpyqq9 branch tests and on main. Related: the ask gate sees only a task's own findings, so missing-file (board lint) cannot block an ask; keep that as is unless it matters.

## Depends on

## Acceptance
- [ ] After hug task detach, lint shows no missing-file for that file, or one finding with a runnable fix; a test fails first on main.
- [ ] A genuinely missing cited file is reported exactly once (test).

## Notes


## History
- 2026-11-18T14:03 todo -> doing/active (take)
