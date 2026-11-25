---
id: ta6fy6
title: "Release v0.7.99: H and L stop on no column past either end"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzy8"
created: 2026-11-22
updated: 2026-11-22
stage: "active"
doing_since: "2026-11-25T13:00"
doing_with: "hktdgs"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Jørgen tries v0.7.99 and says whether H and L stepping onto no column past either end feels right.

## Purpose
One release ticket per release. v0.7.99 = main a5910763, landed 1d864738 on exact-SHA CI 8/8 (run 44960662438).

## Principle
Ask one thing; the frame is on hktdgs.

## Relevant details
Shipped: hktdgs check 2 (failing-first test in overview, list and board; 7 mutants killed; 5 help goldens changed, one line each). An order set with o stays; o and the sort picker's c say to mark a column first while none is marked.

## Depends on

## Acceptance
- [x] Jørgen steps past both ends with H and L in v0.7.99 and says it reads right, or what to change

## Questions
- 2026-11-22T00:32 question 1 by crab to jhf (check 1):
  v0.7.99 is installed. On the board, press L until you pass the last column: the marker goes away and the footer says 'H/L pick a column', as when the board opens. One more L marks the first column; H from the first column does the same the other way. Does that feel right, or what should change?
  answered 2026-11-25T10:23 by jhf:
    Yes, this is approved and working the way it is supposed to. The caveat is what I responded in another ticket that I would like to always see the columns. And since the selection is clearly visible, subtly, I don't need to hide the columns themselves, just because none of them are selected.

## Notes

crab 23:33Z: shipped v0.7.99 (main a5910763, landed 1d864738, CI run 44960662438 8/8). release.yml success; brew-verified: Cellar/hug/0.7.99, hug version = v0.7.99.


check 1: Jørgen 2026-11-25: 'approved and working the way it is supposed to'; his caveat (always show the columns) is hktdgs check 4
crab 2026-11-25: approved by Jørgen; open only for child hktdgs (always show the columns).
## History
- 2026-11-22T00:32 todo -> todo/design (ask)
- 2026-11-25T10:23 answered design (check 1), by jhf, for crab (todo/design -> todo/ready):
  > v0.7.99 is installed. On the board, press L until you pass the last column: the marker goes away and the footer says 'H/L pick a column', as when the board opens. One more L marks the first column; H from the first column does the same the other way. Does that feel right, or what should change?
  Yes, this is approved and working the way it is supposed to. The caveat is what I responded in another ticket that I would like to always see the columns. And since the selection is clearly visible, subtly, I don't need to hide the columns themselves, just because none of them are selected.
- 2026-11-25T10:23 answer 1 delivered to crab (session_crab_1795558631109_253787e0f5d2bbbb)
- 2026-11-25T13:00 todo/ready -> doing/active (take with hktdgs)
- 2026-11-25T13:00 crab read the answer: hug task take ta6fy6
