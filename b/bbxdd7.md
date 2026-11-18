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

piglet 15:29: commit 2a22416d on branch bbxdd7 (stacked on gmzd87 19a5fc77), pushed with ci/bbxdd7, CI run 44324710425. Failing first: TestPendingReachesCoordinatorsNextSession 'crab's next session start delivered [], err <nil>; want [d68ekg request 1]', TestPendingOnlyForItsCoordinator, TestBoardCueSkipsDeliveredRequest 'the cue names request 1, already delivered to crab'. Outputs: ~/.jcode/scratch/lint-gaps2/bbxdd7-failfirst.txt and bbxdd7-check2-failfirst.txt. Full niced suite: 34 ok, 0 FAIL (lint-gaps2/bbxdd7-suite.txt). Check 2 is filtered in muninn/run/board.go (boardTasksFor + board.DeliveredTo), so boardcue.go is untouched; git merge-tree with origin/agh2h7 reports no conflict.
