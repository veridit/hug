---
id: 44w1ev
title: "Scale: stay fast and safe at 10,000 tasks"
status: todo
type: chore
priority: "medium"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
hug task commands and the TUI stay responsive and never fail on a board of 10,000 tasks, and a release cannot quietly make that worse

## Purpose
Jørgen 2026-11-16: if we launch and people use it for 10,000 tasks, it must not crash. Measured on v0.7.16 with 10,002 synthetic tasks (39 MB of Markdown, machine under load, niced): nothing failed. ls cold (builds the cache) 5.4 s / 187 MB; ls warm 1.4 to 1.6 s / 151 MB; ls --jsonl 1.5 s; next 0.6 s; show 0.7 s; board 0.65 s; lint 3.0 s; note and new (write plus commit) 1.0 s each; ls --epic 1.0 s; cache.db 14.7 MB; TUI first screen 2.4 s, 213 MB, list view and jump to the last row 0.2 s

## Principle
Measure, then fix the largest cost; no speculative rewrites. Costs that grow with the board (reading every task to answer about one) are the ones to remove. Guard with a test so a regression fails CI, as ChildIndex did for ls (19 s to 0.8 s on 2,498 tasks)

## Relevant details
Where the time goes, from the code: every command runs cache.Sync, which walks and stats every .md file (10,002 stats), then List loads every task as JSON from SQLite into memory, even for show/next/note on one task. That is why show costs 0.7 s and 80 MB. Ideas in order of value: (1) skip the walk when the hug/tasks head and the worktree are unchanged since the last sync (one git call instead of 10,002 stats); (2) let single-task commands read one row; (3) ls prints from columns instead of decoding full tasks; (4) lint at 3.0 s re-checks everything: cache findings per file (already stored) and recompute only cross-task checks; (5) TUI: 213 MB is fine, but check the detail view and the palette's value counts (valueCount re-filters per row). Not measured yet: 50,000 tasks, Windows, a cold disk, many small commits in the tasks branch (10,000 commits of history), and import of 10,000. The generator used for this run is in the task notes

## Depends on

## Acceptance
- [ ] a benchmark or test builds a 10,000-task board and fails if ls, next or show exceed a stated budget
- [ ] show, next and note on one task do not load every task (measured time and memory drop, numbers in the notes)
- [ ] an unchanged board skips the per-file walk (measured)
- [ ] the TUI opens a 10,000-task board and filters without a visible stall; numbers in the notes
- [ ] 50,000 tasks measured once and the result recorded, even if not optimised

## Notes
