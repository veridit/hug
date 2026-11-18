---
id: 1bbe96
title: "A column that is the same on every row of a section moves into its header"
state: "doing"
type: feature
priority: ""
assignee: ["nautilus"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T05:47"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In the overview and list, when every task in a section has the same value in a column (all 'active' in STAGE, all 'feature' in TYPE), that column is not drawn for the section and the value is said once in the section heading: 'IN PROGRESS · assigned to [13] · all active'. The freed width goes to the title. A section of one task keeps its columns.

## Purpose
Jørgen 00:11 local, on Statbus's IN PROGRESS section where all 13 rows read 'active': 'when they're all active, shouldn't that be in the section header saying this is state active? And we can save the display of the whole column.' One fact in one place.

## Principle
Said once, where it is true for all. Decided on every task in the section after filters, not only the rows on screen, so scrolling never makes a column appear or vanish. Columns are never hidden silently: the heading names the value. Subtle colours: the value keeps its own colour in the heading.

## Relevant details
Applies to every row column (stage, type, priority, owner), not only stage (crab: same reason for each). Each section draws its own column header row today, so sections may differ in columns. The H/L column marker skips a hidden column, and o cannot sort by one; if the marker sat on it, it moves to the next drawn column. The board view is left as is (its columns are states). Code: huginn/internal/tui/render.go (overviewText, listText, taskRowMark, the column header row). Builds on header-spec.md's bands (e3sevr). Owner: nautilus, after ahejza.

## Depends on

## Acceptance
- [ ] A section whose tasks all share a stage draws no STAGE column and its heading ends '· all <stage>' in that stage's colour; the title gets the width (golden at 80 and 140, Jørgen's 13-active IN PROGRESS shape)
- [ ] The same holds for type, priority and owner; two uniform columns read '· all active · all feature' (golden)
- [ ] The decision uses every task in the section after filters, not the rows on screen: scrolling a 28-task section whose hidden rows differ keeps the column (test)
- [ ] H/L skip a hidden column and o never sorts by one (key test)
- [ ] Jørgen confirms on his board after release

## Notes

Jørgen 00:12 local: 'same for' the WAITING section (all 18 rows 'waiting'). With 41hhgn's ASKED split, both ASKED and WAITING are uniform in stage, so neither draws STAGE.


## History
- 2026-11-18T05:47 todo -> doing/active (take)
