---
id: d8yx7y
title: "Rows show acceptance progress as a pie glyph and N/M"
state: "done"
type: feature
priority: "medium"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Every task row in the TUI (overview, list, board) and in hug task ls shows how many acceptance checks are ticked, as a small pie glyph plus N/M before the id, e.g. '◑ 2/4 8bsx4y'. The glyph fills by quarters; colour shows state at a glance.

## Purpose
Jørgen 2026-11-16, from a screenshot of another task board (Statbus backlog TUI): 'a very neat way of showing how many of the acceptance criteria has been checked off'. Today hug rows show status, type, priority and title but not progress; you must open a task to see it.

## Principle
Glyphs: ○ none ticked, ◔ up to a quarter, ◑ half, ◕ three quarters, ● all. Colour: green when all ticked, yellow when partly, dim when none; red only when the task is blocked or has lint findings (red is for problems). A task with no acceptance checks shows a dim '-' in that column. Fixed width so titles stay aligned. NO_COLOR keeps the glyph and N/M. Only shown keys that work; no new keys.

## Relevant details
Reference screenshot: In Progress (10) panel with rows like '◑ 2/4 STATBUS-553 [chore]', '● 2/2 STATBUS-610 [bug]', '○ 0/7 STATBUS-606'. Rows rendered in huginn/internal/tui/render.go; CLI ls in huginn/internal/cli/cli.go.

## Depends on

## Acceptance
- [x] overview, list and board rows show the glyph and N/M for each task with checks (golden)
- [x] glyph steps: 0/4 ○, 1/4 ◔, 2/4 ◑, 3/4 ◕, 4/4 ● ; 1/6 ◔, 1/2 ◑ (test)
- [x] all ticked is green, partly is yellow, none is dim; NO_COLOR keeps glyph and text (test)
- [x] hug task ls shows N/M (test)

## Notes

check 1: c828cfc on origin/main (CI run 43937278863: ubuntu, macos, windows success). Goldens: tmux overview/list/board/chips/grab/sort regenerated in 4 profiles (24 files) plus testdata/overview/spare-rows-140x40.txt; 140x40-no-color/overview.txt row: '> ◑ 1/2 000002   doing   bug     critical Bug task 02: ...'; 80x24 board: '>!○ 0/2 000001   F C ⎘2 Feature task 0…    ◑ 1/2 000002   B C Bug task 02: make…'. Fixture doing tasks are now 1/2. Model test TestRowsShowProgress checks overview, list and board.

check 2: c828cfc on origin/main (CI run 43937278863: ubuntu, macos, windows success). TestPieSteps (huginn/internal/tui/progress_test.go): 0/4 ○, 1/4 ◔, 2/4 ◑, 3/4 ◕, 4/4 ●, 1/6 ◔, 1/2 ◑, 5/6 ◕, 0/1 ○, 1/1 ●.

check 3: c828cfc on origin/main (CI run 43937278863: ubuntu, macos, windows success). TestProgressColours: in truecolor 4/4 renders with status[done] (green #8ac987), 2/4 with the new theme.progress (yellow #e5c07b / light #8a6d00), 0/4 with dim; the three SGR prefixes differ; with NO_COLOR=1 the cell is plain '◑ 2/4' with no colour codes. A task without checks gets a blank cell.

check 4: c828cfc on origin/main (CI run 43937278863: ubuntu, macos, windows success). TestLsShowsCheckProgress (huginn/internal/cli/progress_test.go): hug task ls prints ticked/total as field 4, after type ('1/2' for a task with 1 of 2 ticked, '0/1'); id stays field 1.
