---
id: fcf4fg
title: "Sync recovers from an interrupted rebase on the tasks worktree"
status: todo
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
When the tasks worktree holds a rebase hug itself started (rebase-merge, possibly with only an autostash), the next hug write finishes or aborts it, re-applies the autostash, and goes on, logging one line about the repair. A Git operation hug did not start still stops with the exact fix line.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 3a. On 2026-11-16 night a sync interrupted mid-rebase in Seid left every agent's hug writes failing until someone ran git rebase --quit and committed by hand. Rule: concurrent writes wait, never fail; fixes never need an editor.

## Principle
Repair only what hug can prove is its own (a marker written before hug's rebase), never a user's rebase. A test that reproduces the stuck state fails first.

## Relevant details
gitbranch/gitbranch.go:598 refuses on rebase-merge/rebase-apply/MERGE_HEAD/... with 'worktree busy' and fix 'rebase --abort'. Seid's case: rebase-merge held only an autostash.

## Depends on

## Acceptance
- [ ] a test kills a sync mid-rebase (rebase-merge with autostash left), then hug task note succeeds, the note and the autostashed change are both on the branch
- [ ] a rebase hug did not start still refuses with the fix line (test)

## Notes
