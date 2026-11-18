---
id: bbxdd7
title: "Pending requests and answers reach the coordinator's next session"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
A request or answer that could not be delivered (coordinator offline) is handed to the coordinator's next jcode session when it starts, once, through the same inbox and receipt states; and the board cue does not repeat a request already delivered.

## Purpose
gmzd87 delivers only to a live session. Jørgen writes in tickets whenever he likes; an item written while the coordinator is down must still arrive, without a poller.

## Principle
By design: the session-start hook drains the inbox for that coordinator name; one item, one delivery; acknowledged only on actual receipt.

## Relevant details
From mizaru's gmzd87 open questions 2 and 3 (2026-11-18). Inbox: <JCODE_HOME>/hug-inbox/<session>/; turn_start hook already runs hug's Acknowledge. Board cue: muninn/internal/recall/boardcue.go (duckling's area).

## Depends on
- gmzd87

## Acceptance
- [ ] A request written with no live coordinator session shows 'not delivered', and the coordinator's next session start receives it once; a test fails first.
- [ ] The board cue does not name a request that was delivered (test).

## Notes

piglet 15:08: crab assigned this to piglet, stacked on gmzd87 (19a5fc77). hug task take refuses ('bbxdd7 is blocked by gmzd87'); the dependency is real for landing, so it stays, and piglet works on branch bbxdd7 from origin/gmzd87 without the take.
