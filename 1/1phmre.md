---
id: 1phmre
title: "Lint reads a path quoted in a note on another board as a task link"
state: "doing"
type: bug
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzzn"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T14:09"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
missing-task-ref fires only for a real task: link, not for a .hug/tasks/<id>/... path quoted inside note prose that names another board's task.

## Purpose
w40cmk's note quotes Statbus's '.hug/tasks/37bryg/README.md' from a 2026-11-18 01:24 design discussion; lint reports it as a broken link on hug's board, and the only fix it offers is hug task edit (interactive).

## Principle
A finding must be true and its fix runnable non-interactively. Quoted history is not a link.

## Relevant details
Finding: w/w40cmk.md missing-task-ref '.hug/tasks/37bryg/README.md: no task 37bryg', fix 'hug task edit w40cmk'. 37bryg is a Statbus task.

## Depends on

## Acceptance
- [x] Lint on w40cmk reports no missing-task-ref for the quoted Statbus path; a test with a quoted path in a note fails first on main.
- [ ] A real task: link to a missing id is still reported, with a non-interactive fix (test).

## Notes


commit 68225a9 on branch 1phmre (stacked on bqfrda 1c6ecc2, rebased on origin/main c87da5a), ci/1phmre. check 1: TestQuotedPathOfAnotherBoardIsNoLink (task) fails before: 'got ["missing-task-ref: .hug/tasks/37bryg/README.md: no task 37bryg" "missing-task-ref: .hug/tasks/37bryg/STATBUS-369.md: no task 37bryg" "missing-file: ...gone.tsv..."]'; TestQuotedOtherBoardPathAndTaskLinkFix (cli) fails before: '[missing-task-ref] .hug/tasks/37bryg/README.md: no task 37bryg / next: hug task edit ID'. Replay of w40cmk's real Notes on a scratch board with the new binary: no missing-task-ref. check 2: TestMissingTaskLinkFixIsNonInteractive: task:zz9zz9/run.log still 'task:zz9zz9/run.log: no task zz9zz9 (fix the id, or mark the link (lost))', fix 'hug task note ID "task:zz9zz9/run.log (lost)"'; running it clears the finding (on main the fix was 'hug task edit ID'). Local niced suite: ok huginn/internal/task 10.284s, ok huginn/internal/cli 153.142s, all huginn + cmd/hug ok. Cache schema 23 -> 24.
## History
- 2026-11-18T14:09 todo -> doing/active (take)
