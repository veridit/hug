---
id: 4vxqp4
title: "Board lint: a task the message only mentions is not the session's task"
status: "doing"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "tpnd3n"
doing_since: "2026-11-17T15:53"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A missing-section finding goes to a session only for a task it owns or acted on, not one its message merely mentions (an example, a dependency, a task it forbids)

## Purpose
After tpnd3n, 5 of the 12 false board lines on the 6 replay sets are such findings at session start: s6ff5k to hog and dromedary told not to start it, vk9td0 to kitten and poodle, n8zmjw to evergreen told not to take it (muninn/docs/BOARD4.md, tpnd3n)

## Principle
Measured by replay with the blind judge on the same sets and fresh data; a message mention may still count for other findings; no change ships without a CI-clear gain

## Relevant details
In tpnd3n's arm 6 lines on message-only tasks were judged right (2 return, 4 start): a rule that drops all message mentions loses them too. Data: scratch/board4/x325pp (tpnd3 arm, judgments3.json).

## Depends on

## Acceptance
- [ ] Replay (board4, blind judge): false lines on the 6 sets fall below tpnd3n's 12 and right lines stay at or above 101
- [ ] Fresh sessions after the change keep their right lint lines

## Notes

crab 14:53: test the start-of-session rule (a task only mentioned in the brief gets no lint line) on its own first: it is 5 of the 9; the other 4 (own/noted task at a wrong moment) after.


## History
- 2026-11-17T15:53 todo -> doing/active (take)
