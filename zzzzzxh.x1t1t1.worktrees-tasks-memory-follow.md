---
id: x1t1t1
title: "Worktrees: tasks and memory follow the code"
status: todo
type: spike
priority: ""
assignee: []
labels: ["needs-discussion"]
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Agents working in a git worktree see and update the same task board, and the memories and tasks they create there are tagged with the worktree so they can be kept or discarded with it

## Purpose
Agents create worktrees for isolation, then lose sight of the board or create tasks that never reach the main checkout; a discarded experiment should take its tasks and memories with it

## Principle
Tasks stay one shared board (hug/tasks is a separate branch, so every worktree can use the same one); provenance is a tag, not a fork; keeping or discarding a worktree is one explicit command that lists what it would keep or drop

## Relevant details
Today: hug task finds .hug/tasks via the repository root; a linked worktree has a different root and may create its own .hug/tasks worktree or fail. Memory binds by folder/project id (rc- root commit), so worktrees of one repo already share the project scope. Needed: (1) resolve .hug/tasks through git rev-parse --git-common-dir to the main checkout; (2) record origin worktree (path + branch) on tasks and memories created there; (3) hug worktree keep|discard NAME lists and applies

## Depends on

## Acceptance
- [ ] done when task:065wwn is done (hug task ls in a linked worktree shows the main board and writes to it)
- [ ] tasks and memories created in a worktree carry its name
- [ ] a discard command lists, then drops or retires, exactly the worktree's tasks and memories

## Notes
