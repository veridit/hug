---
id: ht3cdx
title: "Filter, sort and creating a task keep the view you are in"
state: "doing"
type: "bug"
priority: "high"
assignee: ["hamster"]
labels: []
group: ""
rank: "zzzzzzzzzzz1"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-21T15:34"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] ~~Verify Filter and sort change the view~~ (not relevant: replaced by checks 2-6, crab, 2026-11-20)
- [x] In the overview and on the board, the filter picker's Enter and ': todo' / ': state X' apply the filter and leave the view unchanged (tmux test, fails on main first)
- [x] In the overview and on the board, a sort from the s picker leaves the view unchanged (tmux test)
- [ ] Creating a task with S from the overview opens it over the overview, and Esc returns to the overview with the new task selected (tmux test, fails on main first)
- [ ] # jump and ': goto ID' open the task over the current view, and Esc returns to that view (tmux test)
- [ ] Jørgen applies a filter, a sort and creates a task from the overview, and stays in the overview each time

## Requests
- 2026-11-20T11:18 request 1 by jhf:
  And as you see from the screenshots, the first is before I create this ticket, and then after I created this ticket, I suddenly is in the list view. I think I should just keep the view I'm in. The same thing happens if I apply filter or if I apply sort, and now apparently if I create a ticket.
  resolved 2026-11-20T11:18 by crab:
    Turned into a high-priority bug. Title, Goal, Purpose and Principle now say: filter, sort and creating a task keep the view you are in; only choosing a view changes it. Details list the 3 places on main that force the list view (filter picker Enter, the : filter commands, and openTask, which S-create, # and goto use). Checks 2-5 are tmux tests that must fail on main first (filter, sort, create, jump); check 6 is yours to try after release. Your screenshots stay attached as before/after.

## Notes

file:screen-1117.ans (12719 bytes, sha256:b3ce9a88) screen capture from hug tui

file:screen-1117-2.ans (29394 bytes, sha256:a5354c13) screen capture from hug tui


Rulings (crab 2026-11-21 15:25Z, on hamster's proposal): (1) a section or column a filter leaves empty reads '· None with FILTER', e.g. 'NEXT UP · rank order  [0] · None with state:dropped'. (2) When the filter hides the selected task, focus stays in its section if that section still has rows, else moves to the first section with rows in the same view. A task opened with # or goto that the view or filter leaves out still opens, and Esc shows the mcvfbz 'hidden by the filter' footer with no stand-in row. Sort: on v0.7.96 a sort from the s picker never switched view (tmux repro, overview and board, 120x30). Jørgen may have seen the list after a filter that came before the sort. The tmux test holds the sort check anyway.
file:after-filter-1-applied.ans (2745 bytes, sha256:4d2aac3b) after (hamster, branch ht3cdx): overview, then f s todo Enter: still the overview, empty sections read 'None with state:todo'
file:after-create-2-esc.ans (3531 bytes, sha256:cb64d387) after (hamster, branch ht3cdx): overview, S c 'Foxtrot task' opened as 'overview › NEXT UP › ID', then Esc: the overview with the new task selected
check 2: TestTmuxActionsKeepTheView/{overview,board}/2_filter (ht3cdx_tmux_test.go:134-156) and TestFilterFromATaskKeepsTheView (ht3cdx_test.go:13). On v0.7.96: 'overview: after the filter picker's Enter the view is not the overview: HUG · repo · list 3/5 tasks'. Pass on branch ht3cdx 556473cc, CI 44920278685 success (all 8 jobs)
check 3: TestTmuxActionsKeepTheView/{overview,board}/3_sort (ht3cdx_tmux_test.go:157-162): s p Enter, then s r, keep the view. Passed on v0.7.96 too: the s picker never switched view (repro in a note). Pass on branch ht3cdx 556473cc, CI 44920278685 success (all 8 jobs)
## History
- 2026-11-20T11:18 request 1 by jhf
- 2026-11-20T11:18 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:18 request 1 acknowledged by crab
- 2026-11-20T11:18 request 1 in progress by crab: hug task set ht3cdx
- 2026-11-20T11:18 resolved request 1 by crab
- 2026-11-21T15:34 todo -> doing/active (take)
