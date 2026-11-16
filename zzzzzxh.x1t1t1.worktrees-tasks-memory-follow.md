---
id: x1t1t1
title: "Worktrees: record branch and worktree on take; branch as a condition"
status: todo
type: spike
priority: ""
assignee: []
labels: ["needs-discussion"]
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Every git worktree already shares one board and one memory store (065wwn). On top of that, hug records which branch and worktree work happens in: take stores branch and worktree on the task next to the model; memories captured in a worktree store the branch. A merged or deleted branch becomes a condition: lint flags a doing task on it, and a memory tied to it goes to a dream item. Nothing is deleted automatically and there is no discard command.

## Purpose
Agents create worktrees for isolation, then lose sight of the board or create tasks that never reach the main checkout; a discarded experiment should take its tasks and memories with it

## Principle
Decided with Jørgen 2026-11-16: detect when hug can, require it when hug cannot. The CLI detects branch and worktree from the current directory (git rev-parse); --branch/--worktree override. Where hug cannot know (MCP server shared between sessions, no cwd), task_take requires them (or a cwd to detect from) and fails with a fix: line, like the model. The branch is what matters; the worktree folder is only where to look now. Show prints 'taken by: NAME · MODEL · BRANCH'; the TUI shows the branch on in-progress rows when it is not the main branch. Ties into ez50rb: 'holds while BRANCH is unmerged'.

## Relevant details
Today: hug task finds .hug/tasks via the repository root; a linked worktree has a different root and may create its own .hug/tasks worktree or fail. Memory binds by folder/project id (rc- root commit), so worktrees of one repo already share the project scope. Needed: (1) resolve .hug/tasks through git rev-parse --git-common-dir to the main checkout; (2) record origin worktree (path + branch) on tasks and memories created there; (3) hug worktree keep|discard NAME lists and applies

## Depends on

## Acceptance
- [x] done when task:065wwn is done (hug task ls in a linked worktree shows the main board and writes to it)
- [ ] hug task take in a linked worktree on branch feat/x records branch feat/x and the worktree folder on the task, without flags (test)
- [ ] MCP task_take without a cwd and without branch fails with a fix: line; with them it records both (test)
- [ ] show, ls --json and the TUI detail show the branch; in-progress rows show it when it is not the main branch (test and golden)
- [ ] memories captured or remembered in a worktree record its branch (test)
- [ ] lint flags a doing task whose branch is merged into main or deleted, with fix: hug task done ID (test)

## Notes
