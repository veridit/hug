---
id: nch9d4
title: "Sort picker offers default board order explicitly"
state: "done"
type: feature
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzsv"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T17:58"
---

## Goal
Make the default ordering visible and directly selectable in the sort picker, clearing every explicit sort field.

## Purpose
Jorgen testing v0.7.61 could not find how to return from priority sorting to the default order.

## Principle
Default means board rank, the manually arranged order. Offer a clearly named default option rather than implying tasks are unordered. Preserve existing sort choices.

## Relevant details
User screenshot 2026-11-18: filter line says priority descending while fresh picker draft says rank, with no default row. Current implementation openSort clears draft, so Enter on untouched picker restores rank but this is undiscoverable. Explain the distinction between current order and draft.

## Depends on

## Acceptance
- [x] Sort picker shows an explicit selectable default board order (rank) option.
- [x] Choosing default from a multi-field sort clears every sort key and restores board rank order, with a regression test.
- [x] Picker text does not misleadingly present an unapplied rank draft as the current priority ordering.

## Notes


Coordinator crab hands this to a swarm engineer at 2026-11-18 11:18Z; no code changes made by crab.
Letter r for rank: free (fields are p t s u d g a n, c is column), and rank is the word the filter line, ': sort rank' and the move refusal already use. Picker row: '  r  rank      board order, the default: moves work only in it'. r applies at once (as c does), clears draft and sort. Branch nch9d4, worktree ~/.jcode/scratch/nch9d4.
check 1: commit 6aee8e30 on branch nch9d4. Picker now lists '  r  rank      board order, the default: moves work only in it' (testdata/header/pickers.txt). TestSortPickerRankRowRestoresBoardOrder failed first at nch9d4_test.go:24 'the sort picker has no rank row', passes after.
check 2: TestSortPickerRankRowRestoresBoardOrder applies p t u (3 keys, rows differ from rank), then s r and s n a r: prompt closed, sort and draft empty, row IDs equal the rank order, filter line 'sort: rank'. Failed first (no rank row), passes at 6aee8e30.
check 3: picker lines 1-2 are now 'now: priority ↓' and 'new: nothing picked yet, Enter or r gives rank' (was 'sort: rank'). TestSortPickerShowsCurrentAndDraft failed first at nch9d4_test.go:48 'at rank: "sort: rank" / ...', passes. Full suite: 0 FAIL lines, tui ok 100.491s.
CI run 44295351540 on ci/nch9d4 (6aee8e30): completed success, all 6 jobs. Report: /Users/jhf/.jcode/scratch/nch9d4/REPORT.md
Jørgen 11:50Z: with no column marked, the 'c current column' row is not selectable and must be drawn fully dimmed (letter and name, not just the hint); normal when a column is marked. Added to macaque's scope.
crab review 11:51Z: diff read (sort.go r row, now:/new: lines, 2 failing-first tests TestSortPickerRankRowRestoresBoardOrder, TestSortPickerShowsCurrentAndDraft). Exact SHA 6aee8e30 CI 44295351540 all 6 jobs success. Fast-forwarded to main 6aee8e30. Open: Jørgen's 11:50Z dimmed current-column row (follow-up commit by macaque), and his look after the next release. The golden env var question was a false alarm: AGENTS.md already says HUG_UPDATE_GOLDEN.
landed 6aee8e3 on main 2026-11-18: TUI sort picker: r rank row, now/new lines (nch9d4)
Follow-up (Jørgen 11:50Z): with no column marked the whole 'c  current column' row is dim, rendered as styles.dim of '  c  current column mark one with H/L first'. Pressing c keeps the picker open and shows 'sort: c orders by the marked column; mark one with H/L first'. With a column marked the row is unchanged ('  c  current column (NAME) ' + dim 'same as o'). TestSortPickerColumnRowDimWithoutColumn failed first at nch9d4_test.go:86 (got undimmed letter and name, only the hint dim) and passes now. TUI package: 'ok  github.com/veridit/hug/huginn/internal/tui 115.898s'. Commit 65ef4d07 on branch nch9d4-dimc from origin/main 6aee8e30, CI run 44298750806 on ci/nch9d4-dimc.
landed 65ef4d0 on main 2026-11-18: TUI sort picker: dim the whole c row with no column marked (nch9d4); in v0.7.62 tag
shipped in v0.7.62 (6aee8e3)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T12:15 todo -> doing/active (take)
- 2026-11-18T12:17 doing/active -> todo/ready (give back; was assignee: crab); was doing_since: 2026-11-18T12:15
- 2026-11-18T12:18 todo/ready -> doing/active (take)
- 2026-11-18T12:50 doing/active -> doing/waiting (wait)
- 2026-11-18T17:58 doing/waiting -> done (done); was doing_since: 2026-11-18T12:18; waiting_asked_by: macaque; waiting_on: crab review; waiting_since: 2026-11-18T12:50
