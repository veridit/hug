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
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "after the release: on Statbus's board, IN PROGRESS should read '· all active' in its heading with no STAGE column, and WAITING/ASKED likewise for their shared values; the titles get the room. Does it read clearly?"
asked_for: "jhf"
asked_since: "2026-11-18T06:18"
asked_by: "nautilus"
asked_check: "5"
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
- [x] A section whose tasks all share a stage draws no STAGE column and its heading ends '· all <stage>' in that stage's colour; the title gets the width (golden at 80 and 140, Jørgen's 13-active IN PROGRESS shape)
- [x] The same holds for type, priority and owner; two uniform columns read '· all active · all feature' (golden)
- [x] The decision uses every task in the section after filters, not the rows on screen: scrolling a 28-task section whose hidden rows differ keeps the column (test)
- [x] H/L skip a hidden column and o never sorts by one (key test)
- [ ] Jørgen confirms on his board after release

## Notes

Jørgen 00:12 local: 'same for' the WAITING section (all 18 rows 'waiting'). With 41hhgn's ASKED split, both ASKED and WAITING are uniform in stage, so neither draws STAGE.


landed 8b169235..52ff46f9 (nautilus). CI run 44242499589 green on all 6 jobs (ubuntu with tmux goldens, macos, windows, windows-cli, bench, secrets); -race on tui ok (114.1s). The previous CI run's macos/windows-cli failure (TestUpgradeTouchesMachineOnlyWithMachine) was main's own (run 44239718907 on 4247303e), fixed on main by 50747eaf before the rebase. Tests: huginn/internal/tui/same_test.go, all four fail first on main 4247303e (shims): TestSharedColumnMovesIntoHeading 'at 80 the heading reads "> IN PROGRESS · assigned to  [13]"', TestEverySharedColumnMoves 'shared {stage: kind: priority: owner:}', TestSharedColumnIsDecidedOnTheWholeSection 'filtered to features, shared {...}', TestMarkerSkipsSharedColumns 'layout {status:true kind:true priority:true ...}, want TYPE and STAGE hidden'. Check 1: 13 active features in IN PROGRESS at 80 and 140: heading '[13] · all active · all feature', no STAGE or TYPE cell on the rows, PRIORITY kept (it differs), the title = width less the cells still drawn; 'active' in the doing colour and 'feature' in the type colour (raw render); a section of one keeps its columns; golden testdata/overview/same-columns.txt. Check 2: four shared columns read '· all active · all feature · all medium · all ann' and none is drawn; the list says '· all feature · all medium' after TITLE in its header, STAGE kept when stages differ. Check 3: 28 doing tasks with the 28th a bug: TYPE stays drawn and no 'all feature' for all 28 j steps; filtered to type feature, the section is all feature. Check 4: L over 8 presses never stops on hidden TYPE or STAGE and reaches PRIORITY; a marker left on TYPE stands on priority, o sorts by priority, the hint and sc name priority. Also: stageBoard, longBoard and the owner-branch golden updated: WAITING/NEXT UP in the stage fixture share feature/high, so their rows lose those cells; the jmx overview goldens show '· all ready · all feature · all critical' etc.
landed 52ff46f on main 2026-11-18: tui: a column every task of a section shares moves into its heading ('· all…
shipped in v0.7.57 (52ff46f)
## History
- 2026-11-18T05:47 todo -> doing/active (take)
- 2026-11-18T06:18 doing/active -> doing/asked (ask)
