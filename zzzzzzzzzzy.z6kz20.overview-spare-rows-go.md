---
id: z6kz20
title: "Overview: spare rows go to IN PROGRESS first, then NEXT UP"
status: "doing"
type: feature
priority: "high"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
The overview never leaves the screen half empty: rows not needed by the focused section go first to IN PROGRESS (up to all its tasks), then to NEXT UP, then the rest; every section still shows at least min(count, 3); the focused section still grows to everything it needs

## Purpose
Jørgen 2026-11-16 on v0.7.28: with IN PROGRESS focused and 2 tasks, NEXT UP showed 3 of 32 rows and two thirds of the screen stayed empty. 'In progress grow automatically with first priority and second priority the upcoming, if there is space on the screen.'

## Principle
Row budget order: 1) every section its minimum, min(count, 3), empty sections one line; 2) the focused section up to all its tasks; 3) IN PROGRESS up to all its tasks; 4) NEXT UP up to all; 5) BLOCKED, LINT, RECENTLY DONE. Nothing scrolls while spare rows remain blank. Same rule on the board where it applies.

## Relevant details
Code: overview row split in huginn/internal/tui/render.go (overviewRows), changed by h9zghj in v0.7.28.

## Depends on

## Acceptance
- [x] with IN PROGRESS focused holding 2 tasks and NEXT UP 32, on a 140x40 terminal NEXT UP fills every spare row (golden)
- [ ] with a small section focused, IN PROGRESS shows all its tasks before NEXT UP grows past 3 (test)
- [ ] no blank rows remain below the last section while any section has hidden rows (test)

## Notes

commit 8368c53. CI run 43931868360 on ci/zgr01t (60d3f4b = 82273c1 version corner + 8368c53 z6kz20 + 60d3f4b zgr01t, ff onto bb665ea v0.7.31, now origin/main): test (ubuntu-latest) success, test (macos-latest) success, test (windows-latest) success. overviewRows (render.go) gives rows the focused section does not need to spareOrder = IN PROGRESS, NEXT UP, BLOCKED, RECENTLY DONE, LINT. Check 1 golden testdata/overview/spare-rows-140x40.txt (ids masked ######): '> IN PROGRESS · assigned to  [2]', 2 rows, then 'NEXT UP · rank order  [32] · 1–21 shown' with 21 rows; TestOverviewSpareRowsGolden asserts rows[IN PROGRESS]=2 and rows[NEXT UP]=budget-2=21.

Check 2: TestOverviewSpareRowsInProgressFirst: BLOCKED (empty) focused, 12 doing + 40 todo: IN PROGRESS 12, NEXT UP budget-12; 40 doing + 40 todo: NEXT UP 3, IN PROGRESS budget-3 (drawn rows match via shownRows). PASS locally and in CI.
