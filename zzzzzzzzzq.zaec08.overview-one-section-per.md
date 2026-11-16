---
id: zaec08
title: "Overview: one section per task, counts that add up, recently done"
status: "doing"
type: bug
priority: "high"
assignee: ["poodle"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Every task in the overview appears in exactly one section, the header counts add up to the total, and a RECENTLY DONE section shows the last 3 tasks closed

## Purpose
Jørgen 2026-11-16 on seid (2,507 tasks): 'the blocked tasks in progress and the blocked tasks in the block don't really correspond'. Measured: BLOCKED [75] = 71 todo + 4 doing; the 4 doing ones (4f9djb, hh03qs, d7f7pp, n8zmjw) also show in IN PROGRESS drawn as blocked. Header says todo 1223, doing 21, blocked 75, done 1263: blocked overlaps todo and doing, so nothing adds up. He also wants to see what just finished.

## Principle
Decided in kx2ndm: one section per task, in progress wins and is drawn blocked. So BLOCKED holds only todo tasks with unmet dependencies; NEXT UP the unblocked ones; IN PROGRESS every doing task. Header counts are disjoint: todo (unblocked) + blocked (todo with unmet deps) + doing + done + dropped = total, and the blocked doing count is named inside doing (e.g. 'doing 21 (4 blocked)'). RECENTLY DONE: last 3 by close time (the updated date, or the git commit time of hug task done if that is cheap), below BLOCKED and above LINT, shrinks first when space is short. Same rule for the board columns and the status filter.

## Relevant details
Code: huginn/internal/tui/tui.go paneIndices (case 2 match = m.blocked[t.Path] takes doing tasks too), boardStatus (blocked beats doing), render.go header counts. Seid board for checking: ~/Veridit/src/seid-lang (read only).

## Depends on

## Acceptance
- [x] on a board with a doing task that has an unmet dependency, it appears only in IN PROGRESS (drawn blocked), not in BLOCKED (test)
- [ ] header counts sum to the task total, and doing names how many of them are blocked (test)
- [ ] RECENTLY DONE shows the 3 most recently closed tasks, newest first (test and golden)
- [ ] board view and status filter follow the same one-place rule (test)

## Notes

check 1 (4ee3eb8): TestOverviewOneSectionPerTask: Bravo is doing with an unmet dependency; it is in IN PROGRESS only (not NEXT UP, not BLOCKED), drawn "[id] blocked" there, shown once in the overview; BLOCKED holds only Charlie (todo, unmet dependency); Alpha, Bravo, Charlie, Delta, Golf each in exactly 1 of IN PROGRESS/NEXT UP/BLOCKED. Before, the same board (scratch run on the old code): "Bravo shown 2 times in the overview". Rule: place(t) in tui.go (done, dropped, doing as they are; todo with an unmet dependency is blocked; else todo); boardStatus draws a blocked doing task blocked. Seid read-only (~/Veridit/src/seid-lang, 2507 tasks): tasks in more than one of IN PROGRESS/NEXT UP/BLOCKED: 0; n8zmjw (doing, unmet dependency) is in IN PROGRESS drawn blocked and not in BLOCKED.
