---
id: z2vmte
title: "Wait for a date: wait --until and ask --after, woken by hug"
state: "doing"
type: feature
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T03:38"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A doing task can wait for time to pass: hug task wait ID --until DATE "why" parks it in WAITING until DATE, then it returns to its owner, first in hug task next; hug task ask ID --to NAME --after DATE "question" parks it in WAITING until DATE, then the question appears in that person's ASKED. Nothing about it shows in ASKED before the date.

## Purpose
Jørgen 2026-11-18 02:28 local, on 47sxn2 ('a week of use: has he had to remind an agent to update a ticket? look again 2026-11-24'), which hug upgrade put in his ASKED today: 'waiting for time to pass is a sensible state and when the time has passed the system brings it back... it's not waiting on me... maybe it should be a native thing.' A question that can only be answered later is not his to address now.

## Principle
Time is a thing a task waits on, like CI; a person is asked only when they can answer. hug wakes the task on its own: no agent has to remember. A date reads in the person's local time. No new stage: --until is a wait with a date, --after is an ask that starts as that wait.

## Relevant details
Today: hug task dep --add 'when date YYYY-MM-DD' holds a todo task out of next, but a doing task has no timed wait. Fields: waiting_until (and, for --after, the parked question with its asked_for/asked_check, moved to asked_* at the date). Waking happens on any hug read (ls, next, tui refresh) once the date has passed, written as one commit with a History line; the TUI WAITING row shows 'until 24 Nov'. 3nhp67's rule holds: the wait's fields are cleared when it wakes. MCP task_update wait/ask take until/after. Also: hug upgrade turns a wait or ask whose text says 'look again DATE' or 'on DATE' into this form, listed in the preview.

## Depends on

## Acceptance
- [x] wait --until DATE: WAITING with 'until DATE' before; on the first hug read after DATE it is doing/active and first in its owner's hug task next, with one History line (test with a fixed clock, fails first)
- [ ] ask --after DATE: not in the person's ASKED before DATE; after it, in ASKED with the question verbatim and its check (test with a fixed clock)
- [ ] 47sxn2 on hug's board becomes ask --to jhf --after 2026-11-24 and leaves Jørgen's ASKED today (done with hug commands)
- [ ] Jørgen sees 47sxn2 come back into his ASKED on 2026-11-24 without anyone touching it

## Notes


## History
- 2026-11-18T03:38 todo -> doing/active (take)
