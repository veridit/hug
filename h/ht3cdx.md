---
id: ht3cdx
title: "Filter, sort and creating a task keep the view you are in"
state: todo
type: "bug"
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
Applying a filter or a sort, or creating a task, leaves you in the view you were in (overview, list, board or lint).

## Purpose
Jumping to the list loses your place and makes the view feel unreliable.

## Principle
Only choosing a view changes the view; every other action works inside the current one.

## Relevant details
Where it happens on main 489684a6 (huginn/internal/tui):
- filters.go:179, the filter picker's Enter: setFilter, then setView("list").
- palette.go:357 and :374, ': todo|doing|...' and ': state X' etc.: setFilter, then setView("list").
- filters.go:241 openTask (used by capture.go:307 when S creates a task, by palette goto and by # jump): setView("list") before opening the task, so Esc returns to the list. The new task should open over the view you were in, and Esc should go back there.
- sort.go applySort itself does not change the view; Jørgen saw it happen after a sort too, so reproduce that path first (from the s picker in the overview and on the board).
- If a filter hides every row of a section, the view shows the section empty with the filter named, rather than switching view.

Jørgen 2026-11-20 10:18Z (request 1, screenshots screen-1117.ans before, screen-1117-2.ans after): 'the first is before I create this ticket, and then after I created this ticket, I suddenly is in the list view. I think I should just keep the view I'm in. The same thing happens if I apply filter or if I apply sort, and now apparently if I create a ticket.'

## Depends on

## Acceptance
- [ ] Verify Filter and sort change the view

## Requests
- 2026-11-20T11:18 request 1 by jhf:
  And as you see from the screenshots, the first is before I create this ticket, and then after I created this ticket, I suddenly is in the list view. I think I should just keep the view I'm in. The same thing happens if I apply filter or if I apply sort, and now apparently if I create a ticket.

## Notes

file:screen-1117.ans (12719 bytes, sha256:b3ce9a88) screen capture from hug tui

file:screen-1117-2.ans (29394 bytes, sha256:a5354c13) screen capture from hug tui


## History
- 2026-11-20T11:18 request 1 by jhf
- 2026-11-20T11:18 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:18 request 1 acknowledged by crab
- 2026-11-20T11:18 request 1 in progress by crab: hug task set ht3cdx
