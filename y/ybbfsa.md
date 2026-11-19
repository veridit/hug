---
id: ybbfsa
title: "Three test flakes seen on 2026-11-18 stop recurring"
state: "done"
type: bug
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzxl"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:21"
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
- [x] Each of the 3 has its cause named with evidence in a note (log line or a reproduction).
- [x] Each fixed one passes -count=50 locally (niced) and a test that reproduces it failed first where possible.
- [x] No timeout raised and no retry added.
- [x] Fourth flake: TestBranchSyncFromLinkedWorktreeUsesMainCheckout/main_first (muninn/internal/project) on ubuntu CI 44328375780 attempt 1: 'git worktree add --detach .../linked HEAD: fatal: could not open .git/worktrees/linked/locked for writing: No such file or directory'. Cause named and fixed like the other three.

## Notes


Flake 2 cause (gitbranch TestUpgradeFailuresKeepOriginalLayout TempDir cleanup 'remote.git/objects: directory not empty'): lost-ack and unknown-* kill the push at NetTimeout=1s while the remote's post-receive sleeps 3s. The killed push leaves git-receive-pack running (probe: push returned after 1.008s 'signal: killed'; ps still shows git-receive-pack, /bin/sh hooks/post-receive, sleep 3). After the hook ends, receive-pack starts 'git maintenance run --auto --quiet --detach' (trace2: child_start 3.063841s after hook exit), which creates objects/maintenance.lock 3100-3172 ms after the push started (3 of 3 runs), while t.TempDir removes the folder. gc.auto=0 does not stop it: receive-pack still spawns the child (1 child with gc.auto=0, 1 with no config, 0 with maintenance.auto=false). Env config (GIT_CONFIG_COUNT) does not reach receive-pack (commit 0 children, push 1). Fix per crab's ruling (fixtures only): quietRemote() sets maintenance.auto=false on both upgrade_test remotes. Fail-first: TestFixtureRemoteStartsNoMaintenance with quietRemote setting true: 'upgrade_test.go:223: quietRemote: receive-pack started maintenance 2 times' FAIL; with false: PASS.
Flake 4 cause (muninn/internal/project TestBranchSyncFromLinkedWorktreeUsesMainCheckout/main_first, CI 44328375780 'could not open .git/worktrees/linked/locked for writing'): gitInit's root commit starts 'git maintenance run --auto --detach' (trace2: 2 per commit), and on git 2.54 auto maintenance runs worktree-prune ('git worktree prune --expire 3.months.ago'), which deletes a .git/worktrees/NAME folder that has no gitdir yet: exactly the state the test's next 'git worktree add' is in between its mkdir and writing 'locked'. Probe: half-made .git/worktrees/linked removed by a commit with default config, kept with maintenance.auto=false (0 children). A background prune raced against worktree add reproduced the exact CI error 1 in 200 earlier. Fix (fixtures only): gitInit sets maintenance.auto=false before its first commit. Fail-first: TestGitInitCommitPrunesNoWorktree with the config set true: 'project_test.go:61: the commit started maintenance 2 times' and 'project_test.go:64: a commit pruned a worktree being added' FAIL; with false: PASS.
crab 17:48: fifth flake: huginn/gitbranch TestUpgradeFailuresKeepOriginalLayout failed once in iwazaru's full 196 s package run, passed alone (56.985s); no LocalOnly involved.
Bars (16:14-16:51Z, niced GOMAXPROCS=2, machine load avg 224-423): f4 TestBranchSyncFromLinkedWorktreeUsesMainCheckout+TestGitInitCommitPrunesNoWorktree -count=50: 'ok muninn/internal/project 149.216s', 0 FAIL. f1 TestOpenDuringRebase -count=50: 'ok huginn/gitbranch 33.989s', 0 FAIL; TestBackgroundSyncParallelNotesAllLand -count=25: 'ok huginn/internal/cli 188.779s', 0 FAIL (second 25 running). f2: original symptom ('TempDir RemoveAll cleanup' / 'directory not empty') 0 times in 125 runs (base 30 + fix 25+25+50 partial + base A/B 25). BUT under this load lost-ack/unknown-after fail on NetTimeout=1s ('the remote did not answer within 1s', 'ack-marker: no such file'): fix arm 6/25 (f2-count25b.log) and 0/25 (f2-count25a.log), base arm (origin/main upgrade_test.go) 6/25 (f2-base-count25.log): the same rate without my change, so a separate load-sensitivity of the 1s fixture timeout, not flake 2. Not fixed here (no timeout raises): asking crab.
Committed 4c1317a0 on origin/main 0b3c0e96 (v0.7.66), pushed ybbfsa + ci/ybbfsa; CI 44353109208 queued. Files: huginn/gitbranch/gitbranch.go (+rebasing, Open returns r while rebase-merge/head-name names the branch, ~line 457), gitbranch_test.go (TestOpenDuringRebase), upgrade_test.go (quietRemote + TestFixtureRemoteStartsNoMaintenance), muninn/internal/project/project_test.go (gitInit maintenance.auto=false + TestGitInitCommitPrunesNoWorktree), CHANGELOG. Rebased tip: go test -count=1 huginn/gitbranch 'ok 71.736s', muninn/internal/project 'ok 7.511s' (load avg 229). f1 TestBackgroundSyncParallelNotesAllLand -count=25+25: 'ok 188.779s' and 'ok 171.773s', 0 FAIL. Flake 3 (TestCatchUpAcrossLayoutMove invalid object) NOT fixed here: not reproduced (0/40 alone, 0/1 full traced package), handed to tigress on branch ybbfsa-f3 (16:48Z). Load-only failure of lost-ack/unknown-* at NetTimeout=1s filed as 3t4a49 (crab 16:54).
check 3: No timeout raised and no retry added: diff 4c1317a0 vs origin/main touches no NetTimeout, sleep, deadline or retry; fixes are config (maintenance.auto=false) in two test helpers and a rebase-state check in Open.
check 4: Flake 4 cause: commit's detached auto maintenance worktree-prune deletes a half-made .git/worktrees/linked (fail-first TestGitInitCommitPrunesNoWorktree: 'a commit pruned a worktree being added'); fixed in gitInit (maintenance.auto=false); -count=50 'ok muninn/internal/project 149.216s' 0 FAIL at load avg 417. Commit 4c1317a0.
landed 582afc2 on main 2026-11-18: Three test flakes fixed at their cause (ybbfsa)
shipped in v0.7.68 (582afc2)
Flake 3 (tigress): cause is git's detached auto maintenance repacking the test clone's object store while the test's own git reads and writes it. The commit's cache-tree check (cache-tree.c:451-460, odb_has_object RECHECK_PACKED) then misses a blob. It is closed by 6ecd08fc (0n4rnx, maintenance.auto=false for every git the cli test binary starts, plus quietRemote), which is on main in 8c26ec51. No new code. Fail-first on a 0b3c0e96 binary, with maintenance.geometric-repack.auto=-1 forcing a repack on every auto run, -count=12 -parallel=4: 2 FAIL, 'catchup_test.go:89: ... fatal: bad revision 6a569ee8ecdb7e18583348cfede7a026402eee4c^{tree}' and 'catchup_test.go:68: ... exec: WaitDelay expired before I/O complete'. Same amplifier on an 8c26ec51 binary: 9+8+34 = 51 PASS, 0 FAIL (f3probe/fix{a,b,c}.log). Unamplified on the old binary: 20 PASS 0 FAIL, so the original needs load. Plain-git probes (concurrent repack vs add/commit, clone plus detached maintenance, linked worktree board): 0 errors in 946 commits. hug runs no gc/prune/repack itself. CI 44363087479 on 8c26ec51: success. Report: ~/.jcode/scratch/ybbfsa-f3-report/REPORT.md
check 1: flake 3 (tigress, ~/.jcode/scratch/ybbfsa-f3-report/REPORT.md): git detached auto maintenance repacks the test clone mid-read; forced repack on 0b3c0e96 binary: 2 FAIL 'fatal: bad revision ...^{tree}'. Flakes 1/2/4 named earlier.
check 2: flake 3 fixed by 6ecd08fc (maintenance.auto=false; main 8c26ec51, CI 44363087479): same amplifier 51 PASS 0 FAIL at load 345-677, failed first on 0b3c0e96. Flakes 1/2/4 released in v0.7.68.
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T15:56 todo -> doing/active (take)
- 2026-11-18T17:57 doing/active -> doing/waiting (wait)
- 2026-11-18T20:21 doing/waiting -> done (done); was doing_since: 2026-11-18T15:56; waiting_asked_by: duckling; waiting_on: CI 44353109208 on 4c1317a0 (flakes 1, 2, 4), and flake 3 on tigress's branch ybbfsa-f3; checks 1 and 2 tick when flake 3's cause and fix land; waiting_since: 2026-11-18T17:57
