---
id: 9xfnwr
title: "u inside a task goes back up to the list"
status: todo
type: feature
priority: "high"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
In a task's detail view, u goes back up one level to where you came from (the list, board or overview), with the same task selected, exactly like Esc does today.

## Purpose
Jørgen 2026-11-16: 'When I'm inside a task, I constantly try to use the letter u to go up again.' His hands expect u for up; today u does nothing in the detail view (it is only a field letter inside the : sort builder).

## Principle
u means up everywhere it is shown: detail to list, and from a followed link back to the task you came from. Esc keeps working as before. u in the sort builder stays the 'updated' field (a different mode, shown on screen). The key bar shows 'u up' in the detail view. Only keys that work are shown.

## Relevant details
Keys: huginn/internal/tui/keys.go; sort letters in sort.go:27. Check vim users' expectations: u is undo in vim, but hug has no undo, so no clash today; if undo ever comes it takes another key.

## Depends on

## Acceptance
- [ ] in detail, u returns to the previous view with the same task selected (test, from overview, list and board)
- [ ] after following a link (Enter on a dependency), u returns to the task you came from (test)
- [ ] the detail key bar shows 'u up' (golden)
- [ ] u inside : sort still adds the updated field (test)

## Notes
