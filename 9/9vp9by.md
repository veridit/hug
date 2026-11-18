---
id: 9vp9by
title: "Memory layout migration commit runs no repo hooks"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
muninn's layout migration (project/branch.go migrationGit rm/commit/restore of .muninn/memories.jsonl) commits without running the code repository's hooks, so a commit-msg gate (Seid's Model: gate, jr55s8) or a hook that leaves a background process holding stdout can neither refuse nor hang it

## Purpose
In Seid the commit-msg hook refuses a commit without a Model: trailer, so the migration would fail; a hook's background job holding git's stdout would block CombinedOutput forever (WaitDelay 0). Same lesson as the 2026-11-09 seid-lang incident that made runAt pass core.hooksPath=/dev/null

## Principle
hug's own commits never run the host project's hooks; reuse gitbranch's runner semantics, no timeout raised, no retry

## Relevant details
Audit 2026-11-18 (tigress, origin/main 8a45b0eb): migrationGit at muninn/internal/project/branch.go:159-166 runs exec.Command + CombinedOutput with no core.hooksPath override; callers :135-139. runAt does it at huginn/gitbranch/gitbranch.go:170. Tests: a repo whose commit-msg hook exits 1, and one whose hook leaves 'sleep' in the background holding stdout; fail first on main, niced.

## Depends on

## Acceptance
- [ ] A test with a commit-msg hook that exits 1 fails on main and passes after the fix
- [ ] A test with a hook leaving a background sleep holding stdout fails (hangs past a bound) on main and completes after the fix
- [ ] No timeout raised, no retry; niced -count=50 of the new tests passes

## Notes
