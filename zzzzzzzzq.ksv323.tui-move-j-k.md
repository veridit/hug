---
id: ksv323
title: "TUI move (J/K) must move past the row you see"
status: "doing"
type: bug
priority: "high"
assignee: ["poodle"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Pressing J or K on a task moves it past the next or previous row shown in the section you are looking at, and the screen shows the row in its new place

## Purpose
Jørgen 2026-11-16 pressed J on a1dx7s in the overview's NEXT UP three times. Each time the bar asked to confirm, said 'move a1dx7s', and the row stayed exactly where it was. Three commits were made (a655982, 37ec5ec, dfdc006) that renamed the file zzzzzu -> zzzzzx8 -> zzzzzu -> zzzzzx8, so the board was changed with nothing visible

## Principle
Move relative to what is on screen: the neighbour is the adjacent row of the visible list (after filters, in the focused section), never a hidden task. A move that changes nothing visible must not happen or must say why. The confirm line names the task it will move past

## Relevant details
Cause: huginn/internal/tui/actions.go lines 67-82 pick the neighbour from m.all (every task in rank order), i+1 or i-1. NEXT UP hides epics, taken tasks and done tasks, so the neighbour of a1dx7s (rank zzzzzx8) was the epic d634kr 'Recall quality' (zzzzzx) above and x1t1t1 below; moving after/before a hidden task leaves the visible order unchanged. Fix: take the neighbour from m.indices() (the focused pane's visible rows). Also: the message says 'move ID' for both directions; say 'moved ID below OTHER'. The key bar and help call this 'move down / up in global rank order'; check the wording still fits. Board view: J/K inside a status column should use that column's rows. Check that a filtered list moves relative to the filtered rows. Rank strings grew to 7 characters here; rebalance exists for that (rank.NeedsRebalance at more than 12)

## Depends on

## Acceptance
- [ ] in the overview NEXT UP, J on a row swaps it with the row shown below it, and the screen shows the new order (test with a hidden epic ranked between them)
- [ ] K does the same upward; at the first or last visible row the message says so and no commit is made
- [ ] the confirm line names both tasks, e.g. 'Move a1dx7s below x1t1t1?', and the result message says where it went
- [ ] with a filter active, moves are relative to the filtered rows (test)
- [ ] tests reproduce Jørgen's case: three presses no longer produce commits with no visible change

## Notes

2026-11-16 Jørgen: ranks are sparse so a task can be inserted between two others, and the move must take that into account. Checked: Store.Move -> rankPlan (huginn/internal/task/ordering.go:20) already does the sparse insert correctly: it places the moved task right after (or before) the target in the full order and gives it a rank between its new neighbours, renaming as few files as possible. So the rank arithmetic is sound; the fault is only which target the TUI hands it (the adjacent task in the full list, m.all, which may be hidden). Fix in the TUI: pass the adjacent VISIBLE row as the target. Then 'after x1t1t1' lands a1dx7s between x1t1t1 and the next task in the full order (gem59b, a hidden epic), which shows as below x1t1t1. Second finding for this ticket or 44w1ev: new tasks are appended at the end, so ranks only grow (this board: 52 tasks, ranks up to 9 characters, the last six all 'zzzzzzz…'; rank.NeedsRebalance fires above 12). Appending should leave room (pick the midpoint between the last rank and the end, or rebalance automatically when a rank passes a length), otherwise every board drifts toward 'hug task rebalance'.
