---
id: atex3d
title: "A waiting task's stage line leads with the date: 'waiting until 2026-11-24 · on…"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzli"
created: 2026-11-24
updated: 2026-11-24
---

## Goal
A waiting task's stage line leads with the date: 'waiting until 2026-11-24 · on WHAT · since 11-18', with no second 'until' and no 'on the date' wording; the same in hug tui's header, its list row stage cell ('until 11-24') and hug task show (golden, test)

## Purpose
Jørgen 2026-11-18 02:28 local, on 47sxn2 ('a week of use: has he had to remind an agent to update a ticket? look again 2026-11-24'), which hug upgrade put in his ASKED today: 'waiting for time to pass is a sensible state and when the time has passed the system brings it back... it's not waiting on me... maybe it should be a native thing.' A question that can only be answered later is not his to address now.

## Principle
Time is a thing a task waits on, like CI; a person is asked only when they can answer. hug wakes the task on its own: no agent has to remember. A date reads in the person's local time. No new stage: --until is a wait with a date, --after is an ask that starts as that wait.

## Details
A doing task can wait for time to pass: hug task wait ID --until DATE "why" parks it in WAITING until DATE, then it returns to its owner, first in hug task next; hug task ask ID --to NAME --after DATE "question" parks it in WAITING until DATE, then the question appears in that person's ASKED. Nothing about it shows in ASKED before the date.

Today: hug task dep --add 'when date YYYY-MM-DD' holds a todo task out of next, but a doing task has no timed wait. Fields: waiting_until (and, for --after, the parked question with its asked_for/asked_check, moved to asked_* at the date). Waking happens on any hug read (ls, next, tui refresh) once the date has passed, written as one commit with a History line; the TUI WAITING row shows 'until 24 Nov'. 3nhp67's rule holds: the wait's fields are cleared when it wakes. MCP task_update wait/ask take until/after. Also: hug upgrade turns a wait or ask whose text says 'look again DATE' or 'on DATE' into this form, listed in the preview.

Promoted from item 5 of task:z2vmte.

## Depends on

## Acceptance
- [ ] A waiting task's stage line leads with the date: 'waiting until 2026-11-24 · on WHAT · since 11-18', with no second 'until' and no 'on the date' wording; the same in hug tui's header, its list row stage cell ('until 11-24') and hug task show (golden, test)

## Notes
