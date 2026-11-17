---
id: vzghj8
title: "TUI search finds a task by its imported id (STATBUS-402)"
status: "doing"
type: bug
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In hug tui, / search and # jump match a task's aliases (the imported source ids, e.g. STATBUS-402, TASK-056) as well as its id, slug, title and assignee, case-insensitively: typing statbus-402 finds the task whose aliases hold STATBUS-402, and the row shows which alias matched (dim, after the title).

## Purpose
Jørgen 2026-11-17 12:10: agents still cite the old Statbus ids (STATBUS-402), and searching for one in the TUI finds nothing. The CLI already resolves aliases (hug task show STATBUS-402 works); the TUI does not: tui.go:352 searches only slug, title, id and assignee.

## Principle
One meaning for an id everywhere: whatever hug task show accepts, the TUI finds. No new syntax.

## Relevant details
huginn/internal/tui/tui.go:352 builds the search text from t.Slug, t.Title, t.ID, t.Assignee; add t.Aliases. # jump: filters.go:250 resolves via task.Resolve, check its completion list includes aliases. Statbus board ~/ssb/statbus/.hug/tasks: 630 tasks carry aliases like ["STATBUS-326"].

## Depends on

## Acceptance
- [ ] / statbus-402 on a fixture with aliases ["STATBUS-402"] selects that task, and the row shows the matched alias (test + golden)
- [x] # STATBUS-402 jumps to the task (test)
- [ ] on the Statbus board, Jørgen finds a task by an old id he remembers

## Notes

568c3e3 on ci/vzghj8 (on the wyvern stack): / search matches aliases, the row names the alias it matched; # jump already matched them. TestSearchAndJumpFindImportedAlias fails before the fix ('/ statbus-402 found []'); huginn/internal/tui passes with it, goldens unchanged.

Checks 1 and 2 on main d74af5d (25459c1, CI 44087396396 green on all 5 jobs). TestSearchAndJumpFindImportedAlias (huginn/internal/tui/alias_search_test.go): '/ statbus-402' leaves the one task visible and selected, and the row reads 'Statbus import · STATBUS-402'; '/ statbus imp' (a title match) shows no alias; '# STATBUS-402 enter' opens that task. It failed before the fix. Golden for check 1: ci/vzghj8-golden adds testdata/rows/alias-search-120.txt (the whole list screen, ids as ######; fixture gets Purpose/Principle so the screen has 0 findings), CI 44091154613. A mutation dropping the ' · ' separator in render.go fails the test. Check 3 is Jørgen's, on the Statbus board.
