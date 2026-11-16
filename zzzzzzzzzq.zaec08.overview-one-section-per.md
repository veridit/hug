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
- [x] header counts sum to the task total, and doing names how many of them are blocked (test)
- [x] RECENTLY DONE shows the 3 most recently closed tasks, newest first (test and golden)
- [ ] board view and status filter follow the same one-place rule (test)

## Notes

check 1 (4ee3eb8): TestOverviewOneSectionPerTask: Bravo is doing with an unmet dependency; it is in IN PROGRESS only (not NEXT UP, not BLOCKED), drawn "[id] blocked" there, shown once in the overview; BLOCKED holds only Charlie (todo, unmet dependency); Alpha, Bravo, Charlie, Delta, Golf each in exactly 1 of IN PROGRESS/NEXT UP/BLOCKED. Before, the same board (scratch run on the old code): "Bravo shown 2 times in the overview". Rule: place(t) in tui.go (done, dropped, doing as they are; todo with an unmet dependency is blocked; else todo); boardStatus draws a blocked doing task blocked. Seid read-only (~/Veridit/src/seid-lang, 2507 tasks): tasks in more than one of IN PROGRESS/NEXT UP/BLOCKED: 0; n8zmjw (doing, unmet dependency) is in IN PROGRESS drawn blocked and not in BLOCKED.

check 2 (4ee3eb8): TestHeaderCountsAddUp: header "todo 2  ·  doing 2 (1 blocked)  ·  blocked 1  ·  done 1  ·  dropped 1", sums to 7 = tasks; under a search the places still cover every shown task. Old code on the same board: "todo 3  ·  doing 2  ·  blocked 2  ·  done 1  ·  dropped 1" (9 for 7 tasks). Goldens (60-task fixture): "todo 8  ·  doing 15 (1 blocked)  ·  blocked 7  ·  done 15  ·  dropped 15" = 60, was "todo 15  ·  doing 15  ·  blocked 8  ·  done 15  ·  dropped 15" = 68. Seid now: "todo 1161  ·  doing 9 (1 blocked)  ·  blocked 74  ·  done 1263  ·  dropped 0" = 2507. The ticket expected doing 21 (4 blocked): the board has changed since; grep of status lines there now gives todo 1235, doing 9, done 1262(+1 quoted form), dropped 0, and 1235 = 1161 todo + 74 blocked.

check 3 (4ee3eb8): RECENTLY DONE between BLOCKED and LINT, "[3 of N closed]", newest first. Close time: the updated field is set only at creation, so closed.go reads git: subjects "hug task done|drop ID" (all history) plus -p -U0 diffs where a status line goes from open to done/dropped (last 200 commits on the first read, then only commits since the last head; a note on an already closed task is not a close); falls back to the updated date (imported boards carry one). TestRecentlyDoneNewestFirst: Mike (drop, 4h) Lima (edit status->done, 3h) Kilo (done, 2h), Old (done 1h, later note at 5h) not first; after a new hug task done, November comes first. TestRecentlyDoneShrinksFirst: 1 row at 80x24, 3 at 80x60. TestMergeCloses. Goldens: "RECENTLY DONE · last closed  [3 of 30 closed] · 1–1 shown" at both sizes. Seid: git finds 1 close (23vyy1; the rest came closed via backlog import), so updated dates order them: 2ttpvq (2026-11-15), xy1dwy, p3xtp2. Cost on seid: first read 237-327 ms, same head 15-30 ms, off the UI thread (nfv18p loads).
