---
id: 16dxsr
title: "Depend on a group or epic, and on a stated condition"
status: todo
type: feature
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
A task can wait on a whole group or epic (blocked until every task in it is done or dropped) and on a condition hug can check (load below N, a branch merged, a release published, a date reached). Blocked shows the open condition; next skips the task until it holds.

## Purpose
Jørgen 2026-11-16, seeing a coordinator's todo list full of 'Blocked: after phase 1 landings' entries: dependencies belong on the board, between tickets and groups, not in an agent's todo list; and some waits are conditions, not tickets ('blocked on load: load below something').

## Principle
Depends on accepts a task, an epic or group (all children done), or a condition line with a checkable predicate. hug evaluates conditions when it reads the board; an unknown or unevaluable condition counts as unmet and lint names it with a fix: line. No agent-side polling: next simply skips blocked tasks. Today Blocked() (huginn/internal/task/query.go:220) only resolves single tasks; a dep on an epic waits for the epic's own status, and groups have no dependency at all.

## Relevant details
Condition kinds to start: load below N (5-min average), branch merged into main, tag/release exists, date reached. Same vocabulary as ez50rb's memory conditions, so share one evaluator.

## Depends on

## Acceptance
- [ ] a task depending on a group is blocked until every task in the group is done or dropped (test)
- [ ] a task depending on an epic is blocked until all its children are closed (test)
- [ ] a condition 'load below 120' blocks next while the 5-min load is above 120 and frees it when below (test with an injected load)
- [ ] hug task show and the TUI print the unmet condition; lint flags an unknown condition with a fix: line (test)

## Notes
