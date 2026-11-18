---
id: fcf4fg
title: "Sync recovers from an interrupted rebase on the tasks worktree"
state: "doing"
type: bug
priority: "high"
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzch"
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-18T15:41"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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


tigress took over from kestrel (session gone) on crab's 14:41 assignment; assignee set to tigress.
Fix 69d21471 on ci/fcf4fg (rebased on origin/main c84d3273), CI run 44347035097 pending. Design: integrate() writes <gitdir>/hug-rebase (the marked head) and holds hug-rebase.lock while git rebase runs; a dead process frees the lock. Writers (under the board turn, before reading), sync, take and the background pass call Repository.Recover: rebase state begun within 1 min of a free marker, from the marked head, on this branch = hug's; full state -> git rebase --abort; autostash-only (Seid) -> HEAD back on the unmoved branch, reset, stash apply; one 'sync: ended an interrupted hug rebase ...' line. Edits the rebase did not make are stashed first. Failing first (Recover stubbed): 'rebase_recover_test.go:84: rebase-merge left after recovery: <nil>' (autostash and pick shapes), CLI 'rebase_recover_test.go:71: note on the stuck board: exit 2'. Mutant removing the ownership checks: TestForeignRebaseStillRefuses fails on both cases (no marker; stale marker). Local -race before rebase: 'ok  	github.com/veridit/hug/huginn/gitbranch	70.514s', 'ok  	github.com/veridit/hug/huginn/internal/cli	281.602s', 'ok  	github.com/veridit/hug/internal/env	6.045s'; after rebase, focused gitbranch 11.140s and cli 38.429s ok, repocheck ok.
## History
- 2026-11-18T15:41 todo -> doing/active (take)
