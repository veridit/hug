---
id: xv39b9
title: "Head() hides why an unborn-branch check failed: rev-parse error shown instead"
state: "doing"
type: bug
priority: ""
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T23:43"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When gitbranch Head() cannot tell an unborn branch from a broken HEAD, the error names the git call that actually failed (symbolic-ref or show-ref, with its exit status), so 'Needed a single revision' is never shown for a branch that is merely unborn.

## Purpose
kestrel stress run 2026-11-17 (scratch/flake3/out-1.txt), TestTakeRemindsTodosAndShowsTask at load ~150: '[new Take me ...]: rev-parse --verify HEAD: fatal: Needed a single revision / exit status 128 / fix: git -C .../.hug/tasks rev-parse --verify HEAD'. The board was freshly inited (unborn hug/tasks). Same run window: another git call died with 'remote: signal: segmentation fault'. So a fallback git call most likely crashed and Head() returned the first, misleading error.

## Principle
No retry loops that hide real faults. Report the failing call verbatim; the fix: line re-runs that call. Healthy git keeps today's behaviour (unborn -> empty head).

## Relevant details
huginn/gitbranch/gitbranch.go Head() (~line 473): rev-parse fails; then symbolic-ref HEAD, then show-ref --verify --quiet refs/heads/BRANCH must exit 1. Any other outcome (error from symbolic-ref, show-ref exit 128 or signal) falls through to 'return h, e' with the rev-parse error. Fix in mind: if the fallback call fails for a reason other than 'ref absent', return that call's error (Op names it, Fix re-runs it). A user hits this only when git itself fails mid-check (crash, lock, I/O), so it is small.

## Depends on

## Acceptance
- [ ] a test with a git wrapper that crashes show-ref (exit 139) on an unborn branch gets an error naming show-ref, not 'Needed a single revision'
- [ ] healthy unborn branch: Head() still returns empty and no error (existing tests pass)

## Notes


kestrel: landed a201b2e..98444d7. Check 1: TestHeadNamesTheCallThatCrashed (a PATH git wrapper kills show-ref, then symbolic-ref, with SIGSEGV on an unborn hug/tasks) failed on main: 'the error does not name show-ref first: rev-parse --verify HEAD: fatal: Needed a single revision exit status 128'; now the error starts with the crashed call and says 'segmentation fault', fix re-runs it. gitbranch.Crashed(err): killed by a signal, or on Windows an NTSTATUS exit >= 0xC0000000. Check 2: TestHeadUnbornIsEmpty, plus -race gitbranch ok 22.578s, cli ok 158.120s.
## History
- 2026-11-17T23:43 todo -> doing/active (take)
