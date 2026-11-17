---
id: n2hq8p
title: "One ticket in doing per agent; a second take says why or refuses"
status: todo
type: feature
priority: "medium"
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
hug task take refuses when the taker already has a ticket in doing whose stage is active, naming it and offering the fixes: finish it (hug task done), park it (hug task wait ID "on what"), or put it back (hug task set ID --status todo with a note). Tickets in waiting, review or depends do not count. The board's doing count then means work in hand.

## Purpose
Epic 47sxn2. hatchling held 5 tickets in doing while working on one; Jørgen read them as stuck.

## Principle
Doing means someone is on it now. The refusal is the fix, not a lint later. Humans (--model human) get the same rule.

## Relevant details
Needs 6ephqk's stages (active, waiting, review, depends) and the wait command. Owner identity: model + --as name from take (t709he, x1t1t1).

## Depends on

## Acceptance
- [ ] a second take by the same agent with one active ticket is refused with the three fix lines (test)
- [ ] a take is allowed when the earlier ticket is waiting, review or depends (test)
- [ ] MCP task_take returns the same refusal (test)

## Notes
