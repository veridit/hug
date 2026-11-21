---
id: 7b9kbc
title: "Board: flag a doing task nobody is working on"
state: todo
type: feature
priority: "medium"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzd"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
hug flags a task left in doing/active when its owner has no live session or no commit or note on it for a day.

## Purpose
Three of crab's tasks sat in doing for hours with nobody building them, and lint said nothing.

## Principle
Deduce from sessions, notes and commits; ask the owner, never move it on its own.

## Relevant details
Found 2026-11-20 12:04Z checking 47sxn2 request 1: 8a0ybb, q7n2zd and 3cqvx5 were doing/active under crab (taken with --with so crab could ask on them) while nothing was being built; wxsyc5 waited two days on a check crab could run at any time. Existing finding doing-idle (questions.go) did not fire for them. Rule to build: doing/active with no note, check or commit naming it in 24 h, or whose owner holds more than one active doing task, gets a finding 'doing but idle: put it back to todo, or wait on what holds it' with fix: hug task set ID --state todo. A wait whose 'on' names the owner itself and an action it can take (e.g. 'crab checks') is flagged after a day too.

## Depends on

## Acceptance
- [ ] A doing/active task with no note, check or commit naming it for 24 h gets a lint finding with a fix: line (test)
- [ ] A wait that names the owner's own action is flagged after a day (test)
- [ ] On hug's board the next day, every flagged task is either worked on or back in todo

## Notes
