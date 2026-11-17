---
id: r5r8c1
title: "take refused by a dependency says 'taken by  (id)': name the blocker"
status: todo
type: bug
priority: "medium"
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
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
