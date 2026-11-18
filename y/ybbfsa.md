---
id: ybbfsa
title: "Three test flakes seen on 2026-11-18 stop recurring"
state: "doing"
type: bug
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzt"
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


Flake 2 cause (gitbranch TestUpgradeFailuresKeepOriginalLayout TempDir cleanup 'remote.git/objects: directory not empty'): lost-ack and unknown-* kill the push at NetTimeout=1s while the remote's post-receive sleeps 3s. The killed push leaves git-receive-pack running (probe: push returned after 1.008s 'signal: killed'; ps still shows git-receive-pack, /bin/sh hooks/post-receive, sleep 3). After the hook ends, receive-pack starts 'git maintenance run --auto --quiet --detach' (trace2: child_start 3.063841s after hook exit), which creates objects/maintenance.lock 3100-3172 ms after the push started (3 of 3 runs), while t.TempDir removes the folder. gc.auto=0 does not stop it: receive-pack still spawns the child (1 child with gc.auto=0, 1 with no config, 0 with maintenance.auto=false). Env config (GIT_CONFIG_COUNT) does not reach receive-pack (commit 0 children, push 1). Fix per crab's ruling (fixtures only): quietRemote() sets maintenance.auto=false on both upgrade_test remotes. Fail-first: TestFixtureRemoteStartsNoMaintenance with quietRemote setting true: 'upgrade_test.go:223: quietRemote: receive-pack started maintenance 2 times' FAIL; with false: PASS.
Flake 4 cause (muninn/internal/project TestBranchSyncFromLinkedWorktreeUsesMainCheckout/main_first, CI 44328375780 'could not open .git/worktrees/linked/locked for writing'): gitInit's root commit starts 'git maintenance run --auto --detach' (trace2: 2 per commit), and on git 2.54 auto maintenance runs worktree-prune ('git worktree prune --expire 3.months.ago'), which deletes a .git/worktrees/NAME folder that has no gitdir yet: exactly the state the test's next 'git worktree add' is in between its mkdir and writing 'locked'. Probe: half-made .git/worktrees/linked removed by a commit with default config, kept with maintenance.auto=false (0 children). A background prune raced against worktree add reproduced the exact CI error 1 in 200 earlier. Fix (fixtures only): gitInit sets maintenance.auto=false before its first commit. Fail-first: TestGitInitCommitPrunesNoWorktree with the config set true: 'project_test.go:61: the commit started maintenance 2 times' and 'project_test.go:64: a commit pruned a worktree being added' FAIL; with false: PASS.
## History
- 2026-11-18T15:56 todo -> doing/active (take)
