---
id: ec6z6j
title: "Send one request for a selection of tasks"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
Let Jorgen mark multiple tasks on the board and give one instruction applying to that exact selection.

## Purpose
He wants to select tasks with the same structural issue and ask an agent to fix them without leaving the board or repeating a request for each task.

## Principle
Selection is distinct from keyboard focus and acceptance checks. Preserve exact target IDs and instruction. Do not report a request handled until its selected scope is accounted for.

## Relevant details
User feedback 2026-11-18: marking task rxr4z8 is unfinished; single-task requests hbgybn work. Need a batch request over an arbitrary marked set, not the group field. Storage, agent delivery and partial-resolution presentation need design before implementation.

## Depends on
- rxr4z8

## Acceptance
- [ ] A board mock shows marking several tasks, selected count, and one request action for that exact set, reviewed by Jorgen.
- [ ] Submitting one request preserves its exact text and selected IDs, visible to the agent without asking the user to repeat it per task.
- [ ] User can see pending versus handled targets without a partial fix silently closing the whole request.

## Notes
