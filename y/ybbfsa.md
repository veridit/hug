---
id: ybbfsa
title: "Three test flakes seen on 2026-11-18 stop recurring"
state: "doing"
type: bug
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T15:56"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
The three flakes seen today stop recurring, each fixed at its cause or quarantined with a named reason: (1) TestBackgroundSyncParallelNotesAllLand on ubuntu CI 44316906824 attempt 1 (d2n0h7 is marked done, so it came back); (2) a huginn/gitbranch TempDir cleanup error in a niced local full suite (duckling, rk5q7r); (3) TestCatchUpAcrossLayoutMove with 'git commit: error: invalid object ... for a.9d14y6.first-look.md' (mizaru, gmzd87 rebase).

## Purpose
Every flake costs a CI rerun or a false alarm in the landing queue, and today the queue is the bottleneck (5+ runs queued at once).

## Principle
Find the cause, do not raise timeouts or add retries. A test that cannot be made deterministic says why in a comment and the task. Bounded and niced local runs only (GOMAXPROCS=2, nice -n 19).

## Relevant details
Logs: gh run view 44316906824 --log-failed (attempt 1). TempDir cleanup: likely a background git process still writing when t.TempDir removes the folder. Layout move: w40cmk's catch-up test (huginn/internal/cli). tigress's 8023b9f1 trace2 diagnostics (landing with jj7jpz) list git processes that never exited.

## Depends on

## Acceptance
- [ ] Each of the 3 has its cause named with evidence in a note (log line or a reproduction).
- [ ] Each fixed one passes -count=50 locally (niced) and a test that reproduces it failed first where possible.
- [ ] No timeout raised and no retry added.
- [ ] Fourth flake: TestBranchSyncFromLinkedWorktreeUsesMainCheckout/main_first (muninn/internal/project) on ubuntu CI 44328375780 attempt 1: 'git worktree add --detach .../linked HEAD: fatal: could not open .git/worktrees/linked/locked for writing: No such file or directory'. Cause named and fixed like the other three.

## Notes


## History
- 2026-11-18T15:56 todo -> doing/active (take)
