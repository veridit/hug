---
id: xfv9pw
title: "Search and command prompts recall earlier entries with Up/Down"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzh"
created: 2026-11-18
updated: 2026-11-18
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
- [ ] In the / prompt, Up recalls the last search, repeated Up goes further back, Down returns, and Enter runs it; a test fails first on main.
- [ ] The : prompt has its own history with the same keys (test).
- [ ] History survives quitting and reopening hug tui on the same board, is per board, and is never written into .hug/tasks (test).

## Notes
