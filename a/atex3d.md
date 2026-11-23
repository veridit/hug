---
id: atex3d
title: "A dated wait's stage line leads with the date"
state: "doing"
type: feature
priority: "high"
assignee: ["elephant"]
labels: []
group: ""
rank: "zzzzzzzzzzli"
created: 2026-11-24
updated: 2026-11-24
doing_since: "2026-11-24T00:02"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A timed wait reads 'waiting until 2026-11-24 · on WHAT · since 11-18' in hug tui, the list and hug task show, with no doubled 'until'.

## Purpose
Jørgen 2026-11-18 02:28 local, on 47sxn2 ('a week of use: has he had to remind an agent to update a ticket? look again 2026-11-24'), which hug upgrade put in his ASKED today: 'waiting for time to pass is a sensible state and when the time has passed the system brings it back... it's not waiting on me... maybe it should be a native thing.' A question that can only be answered later is not his to address now.

## Principle
Time is a thing a task waits on, like CI; a person is asked only when they can answer. hug wakes the task on its own: no agent has to remember. A date reads in the person's local time. No new stage: --until is a wait with a date, --after is an ask that starts as that wait.

## Details
From z2vmte check 5 (jhf request 2026-11-20 13:05). Today on main: timedwait.go:163 writes waiting_on 'the date DATE, then ask WHO', and cli.go:2419/2443, store.go:867 and waitrow.go:154/180 append ' · until DATE', so a line reads 'waiting on the date 2026-11-24, then ask jhf since ... · until 2026-11-24'. The list row stage cell should say 'until 11-24'. Golden and test.

## Depends on

## Acceptance
- [ ] A waiting task's stage line leads with the date: 'waiting until 2026-11-24 · on WHAT · since 11-18', with no second 'until' and no 'on the date' wording; the same in hug tui's header, its list row stage cell ('until 11-24') and hug task show (golden, test)

## Notes


## History
- 2026-11-24T00:02 todo -> doing/active (take)
