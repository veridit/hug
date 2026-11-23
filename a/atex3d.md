---
id: atex3d
title: "A dated wait's stage line leads with the date"
state: "done"
type: feature
priority: "high"
assignee: ["elephant"]
labels: []
group: ""
rank: "zzzzzzzzzzli"
created: 2026-11-24
updated: 2026-11-24
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-24T00:47"
done_release: "v0.7.111"
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
- [x] A waiting task's stage line leads with the date: 'waiting until 2026-11-24 · on WHAT · since 11-18', with no second 'until' and no 'on the date' wording; the same in hug tui's header, its list row stage cell ('until 11-24') and hug task show (golden, test)

## Notes


crab ruling 23:07Z on elephant's questions: (1) the list STAGE cell is 10 wide (tui/stage.go:23, sized for 'deferred:'), and 'until 11-24' needs 11. Widening it would take 2 title cells from every row for a rare state, so a timed wait's cell reads 'til 11-24'. (2) 'since' keeps each surface's own format for every stage (show 'since 2026-11-18 05:08', tui 'since 18 Nov'); only the lead changes, to 'waiting until DATE · then ask NAME' or 'waiting until DATE · on WHY'. Jørgen is asked about both after release.
check 1: 0c09d4ee on atex3d/ci/atex3d, CI 8/8 https://github.com/veridit/hug/actions/runs/45290106766. show: 'stage: waiting until 2026-11-24 · then ask jhf · since 2026-11-18 05:08' (was 'waiting on the date 2026-11-24, then ask jhf since 2026-11-18 05:08 · until 2026-11-24'); wait --until: 'waiting until 2026-11-24 · on a week of use · since ...'. tui header '└─ waiting until 2026-11-24 · then ask jhf · since 18 Nov'; list STAGE cell 'til 11-24' (crab ruling 23:07Z: cell keeps 9 cells); 'since' keeps each surface's format (show 'since 2026-11-18 05:08', tui 'since 18 Nov'). Old v0.7.55 fields render the new way (tested). Tests: task TestWaitingLineLeadsWithTheDate, cli TestShowLeadsADatedWaitWithTheDate, tui TestDatedWaitLeadsWithTheDate (golden testdata/rows/dated-wait-120.txt) + TestOnlyAWaitSaysItsDateInTheCell; mutants 15/15 killed.
landed 573c98b on main 2026-11-24: A dated wait's stage line leads with the date (atex3d)
shipped in v0.7.111 (573c98b)
Reflection (open, 2026-11-24, crab): Shipped in v0.7.111 (main ed882297, CI 45290106766 8/8). The crab ruling on the list cell ('til 11-24') and on keeping each surface's own 'since' goes to Jørgen on the release ticket.
## History
- 2026-11-24T00:02 todo -> doing/active (take)
- 2026-11-24T00:47 doing/active -> done (done, by release v0.7.111: every check holds); was doing_since: 2026-11-24T00:02
