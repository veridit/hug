---
id: 065wwn
title: "hug task ls in a linked worktree shows the main board and writes to it"
status: todo
type: "bug"
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
hug task ls in a linked worktree shows the main board and writes to it

Promoted from item 1 of task:x1t1t1.

## Depends on

## Acceptance
- [ ] hug task ls in a linked worktree shows the main board and writes to it

## Notes

2026-11-16 repro: in a linked worktree (git worktree add --detach <dir> <rev>), hug task ls fails: 'worktree add <dir>/.hug/tasks hug/tasks: ... fatal: 'hug/tasks' is already used by worktree at '/Users/jhf/Projects/hug/.hug/tasks'; exit status 128' and its fix: line repeats the same failing git worktree add. Wanted: every linked worktree uses the one hug/tasks checkout of the main worktree (git rev-parse --git-common-dir), reads and writes the same board; no second checkout. MCP and TUI too. Tagging by worktree stays in x1t1t1 (needs discussion with Jørgen).
