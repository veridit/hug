---
id: r5r8c1
title: "take refused by a dependency says 'taken by  (id)': name the blocker"
state: "doing"
type: bug
priority: "medium"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T19:09"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "crab's order: 3mavpt first; branch kestrel/r5r8c1 (ae8e5dd) then rebases onto the state rename"
waiting_since: "2026-11-17T20:17"
waiting_asked_by: "calf"
waiting_for: "crab"
---

## Goal
hug task take on a task with an unmet dependency exits 3 with 'blocked by ID title (status)' for each open dependency and a fix: line (hug task take ID-of-the-blocker or hug task dep ... --remove), never 'taken by  (id)' with an empty name.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 3c.

## Principle
Every error says what is wrong and ends with a fix: line an agent can run.

## Relevant details
cli.go:950 builds 'taken by '+t.Value+' ('+t.ID+')' for Kind 'taken'; the dependency path seems to reach it with an empty Value. take.go:54 is the assignee case.

## Depends on

## Acceptance
- [ ] a test takes a task whose dependency is open: exit 3, the output names the dependency id and title, and a fix: line

## Notes


## History
- 2026-11-17T19:09 todo -> doing/active (take)
- 2026-11-17T20:17 doing/active -> doing/waiting (wait)
