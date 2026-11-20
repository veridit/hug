---
id: 1phmre
title: "Lint reads a path quoted in a note on another board as a task link"
state: "done"
type: bug
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzo3"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:54"
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
- [x] A real task: link to a missing id is still reported, with a non-interactive fix (test).

## Notes


commit 68225a9 on branch 1phmre (stacked on bqfrda 1c6ecc2, rebased on origin/main c87da5a), ci/1phmre. check 1: TestQuotedPathOfAnotherBoardIsNoLink (task) fails before: 'got ["missing-task-ref: .hug/tasks/37bryg/README.md: no task 37bryg" "missing-task-ref: .hug/tasks/37bryg/STATBUS-369.md: no task 37bryg" "missing-file: ...gone.tsv..."]'; TestQuotedOtherBoardPathAndTaskLinkFix (cli) fails before: '[missing-task-ref] .hug/tasks/37bryg/README.md: no task 37bryg / next: hug task edit ID'. Replay of w40cmk's real Notes on a scratch board with the new binary: no missing-task-ref. check 2: TestMissingTaskLinkFixIsNonInteractive: task:zz9zz9/run.log still 'task:zz9zz9/run.log: no task zz9zz9 (fix the id, or mark the link (lost))', fix 'hug task note ID "task:zz9zz9/run.log (lost)"'; running it clears the finding (on main the fix was 'hug task edit ID'). Local niced suite: ok huginn/internal/task 10.284s, ok huginn/internal/cli 153.142s, all huginn + cmd/hug ok. Cache schema 23 -> 24.
CI green: run 44311916568 (68225a9, branch head): secrets, bench, test ubuntu/macos/windows, test-windows-cli all success.
crab review 13:36Z: accepted. Lands after the current queue (z82qsy, 6fqhg9 stack, gpyqq9, bg1rd8, qpp30y), bqfrda before 1phmre. Trade-off accepted: a typo'd id inside a .hug/tasks/ID/... path is no longer reported; task: links still are. SchemaVersion clash: mizaru's gmzd87 also takes 23; whichever lands second renumbers (crab does it at rebase).
Landed on main via 80c8ac28 (stack of 10 on v0.7.65; CI 44344774710 exact SHA 6/6). Ships in the next release.
landed 80c8ac2 on main 2026-11-18: lint: a quoted .hug/tasks path of another board is no link; missing-task-ref fix…
shipped in v0.7.66 (80c8ac2)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T14:09 todo -> doing/active (take)
- 2026-11-18T14:15 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T14:09; waiting_asked_by: piglet; waiting_on: crab review; waiting_since: 2026-11-18T14:15
