---
id: xfv9pw
title: "Search and command prompts recall earlier entries with Up/Down"
state: "doing"
type: feature
priority: ""
assignee: ["engineer"]
labels: []
group: ""
rank: "zzzzzzzzzzzh"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T12:47"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In hug tui, the / search prompt and the : command prompt keep a history: Up recalls the previous entry, Down the next, like a shell; Enter on a recalled entry runs it again.

## Purpose
Jørgen 2026-11-18: after searching, finding a task and acting on it, returning to the same task meant typing the search again; Up in the empty / prompt did nothing.

## Principle
Shell-like and predictable: newest first, no duplicates in a row, an edited recalled entry becomes a new entry. History persists across hug tui sessions per board, stored outside the task branch (never committed).

## Relevant details
No history exists today in find.go, palette.go or filters.go (no 'history' anywhere in the TUI). Up/Down in the prompt are free; in the list they move rows, so history keys apply only while a prompt is open. Store location: the per-user state dir hug already uses, keyed by board path.

## Depends on

## Acceptance
- [x] In the / prompt, Up recalls the last search, repeated Up goes further back, Down returns, and Enter runs it; a test fails first on main.
- [x] The : prompt has its own history with the same keys (test).
- [ ] History survives quitting and reopening hug tui on the same board, is per board, and is never written into .hug/tasks (test).

## Notes


Failing-first on origin/main 4e776695 (branch xfv9pw, worktree ~/.jcode/scratch/xfv9pw): TestSearchHistoryUpDown 'history_test.go:19: Up in the empty / prompt: prompt "/" value "" query "", want the last search'; TestCommandHistoryUpDown 'history_test.go:68: Up in the empty : prompt: prompt ":" value "", want list'; TestFindRecallsSearches 'history_test.go:105: Up in an empty find: prompt "find" value "", 0 hits'; TestTUIHistorySurvivesReopen without the hooks.go wiring: 'tuihistory_test.go:63: history file: open .../.git/huginn/tui-history.json: no such file or directory'. All 4 pass with the change.
Check 1: TestSearchHistoryUpDown (huginn/internal/tui/history_test.go) passes on d262da43: Up in empty / recalls 'Task 07', Up again 'Zebra' (repeat kept once), Up at oldest stays, Down forward, Down past newest = empty prompt with 60 rows, Enter on recalled Zebra selects it, edited Zebra->Zeb becomes new entry, Esc-left search recalled, typed text keeps Down for rows.
Check 2: TestCommandHistoryUpDown passes on d262da43: : history list/board separate from / history (Zebra), Up/Down/Enter as /. Help rows '↑ empty  recall earlier commands/searches' in palette, / find and find-in-page contexts; golden testdata/help/contexts.txt +3 lines reviewed.
## History
- 2026-11-18T12:47 todo -> doing/active (take)
