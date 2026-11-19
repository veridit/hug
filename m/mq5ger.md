---
id: mq5ger
title: "tui watch test misses a change under load"
state: "doing"
type: feature
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzzj"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T22:22"
stage: "waiting"
model: "claude-opus-5-5"
branch: "kikazaru/watchflake"
worktree: "/Users/jhf/.jcode/scratch/watchflake"
waiting_on: "crab: check 4 on stack CI of 6bf6a928"
waiting_since: "2026-11-19T01:24"
waiting_asked_by: "macaque"
---

## Goal
TestChangeBeforeWatchIsUpIsSeen passes on every run, loaded or not, with no raised timeout, retry or skip: its cause found, shown failing first under forced starvation, and fixed.

## Purpose
It failed on macOS CI 44384875316 ('TempDir RemoveAll cleanup: unlinkat .../b/bhfm8r.md: bad file descriptor') and under load for iwazaru; it is the last known flake in huginn/internal/tui, and a red CI job on an unrelated change costs every lane a rerun.

## Principle
Find the cause like tigress on ad62aa: reproduce under forced starvation first, then fix the cause. No timeout raised, no retry, no skip.

## Relevant details
Test: huginn/internal/tui/taskdirs_test.go:83 (np2kk2 check 2). The CI failure is in TempDir cleanup, not the test's assertion: the watch is closed by release() with 'go w.close()' (start.go:136), so RemoveAll can run while fsnotify's kqueue backend (one fd per watched file on macOS) still closes its fds. Reported by crab 21:21Z.

## Depends on

## Acceptance
- [x] a test or harness shows the failure first under forced starvation on origin/main, with the verbatim failure line
- [x] the cause is named with file:line evidence
- [x] the fix passes the same starvation run 0 failures in N runs, with no timeout raised, no retry and no skip
- [ ] full tui suite green locally (non-short) and CI green

## Notes


Fail first, stock fsnotify v1.10.1 (origin/main 8a45b0eb plus probe only): TestStrayClose (closes a watch while its files are deleted, holds files open meanwhile), STRAY=300 -count=6: 3 of 6 runs FAIL, verbatim: 'zz_stray_test.go:90: 3 descriptors closed underneath in 300 rounds; first: round 34: unlinkat /Users/jhf/.jcode/scratch/TestStrayClose3079977529/001/34/g0/t15.md: bad file descriptor' (the CI line's shape) and 'first: round 205: fd 36 closed underneath: fstat bad file descriptor ino 0'. The real test under STARVE=8 GOMAXPROCS=4 on stock, 2x1500: '--- FAIL: TestStarvedChangeBeforeWatch (0.01s) taskdirs_test.go:84: close /Users/jhf/.jcode/scratch/TestStarvedChangeBeforeWatch3502987801/001/h/.huginn-809769911: bad file descriptor' (store.New's temp file lost its fd to a watcher's stray close). Logs: $JCODE_SCRATCH_DIR/mq5ger-stray-stock.txt, mq5ger-starve-stock-b.txt.
Cause: fsnotify v1.10.1 kqueue (macOS/BSD). Close() (backend_kqueue.go:244-271) snapshots the watches and closes each fd at :267 'unix.Close(info.wd)'; concurrently readEvents (:440) on a file's Remove event calls w.remove (:501) which closes the same fd at :320. remove()'s isClosed check (:305) runs before Close's snapshot is done with that fd, so both close it. The second close hits whatever reused the number: RemoveAll's dir fd (unlinkat EBADF) or store.New's temp file. Proven with an fd-tracking copy of fsnotify (closes logged per fd): 'MQ5GER fd 10 closed again at remove', 'fsnotify double closes=3' in 300 rounds (mq5ger-amp-instr.txt). hug side: release() 'go w.close()' (huginn/internal/tui/start.go:141) lets the test's TempDir cleanup delete watched files during Close; watch.run 'defer fw.Close()' (watch.go:198). Upstream: fsnotify issue 788, open PR 789 (unreleased); since v1.10.0 (#740). Same root cause filed by plumb #560 and muxreveal #49.
crab 23:02 conditions, at 7f14e3e1: (1) internal/fsnotify/LICENSE byte-identical to fsnotify v1.10.1's (cmp). (2) README names v1.10.1 commit 76b01a6e, PR 789 head 7d79a83e and 3 deviations: Close's v1.10.1 snapshot loop removed (PR removes main's remove2 loop), remove2 comment hunk has no counterpart, and remove's inner w.Remove -> w.remove(name,false) (PR's opMu hung it on v1.10.1: 'panic: test timed out after 1m0s TestRemoveWatchedDirReturns (1m0s)', passes after). (3) upstream-diff.sh + upstream.patch, held by TestUpstreamDiff (--- PASS). (4) drop task 3dzmh5 filed. (5) 'no reload after the watch came up' split out as adqdfe (stack: forkExec under closeLog.times, closed.go:35/46 <- tui.go:619 <- tui.go:371); not tigress's hfkt1n, which is a torn tmux capture. CI 44415523937 green at b6242d0d (before the lock fix); new CI on 7f14e3e1 running.
crab 00:00: kikazaru's session is gone (reaped; Unknown target). State handed over: branch kikazaru/watchflake = ci/watchflake 7f14e3e1. CI 44417801741 FAILED on ubuntu only: TestUpstreamDiff upstream_test.go:19 exit status 1, the patch's hunk headers differ on GNU diff (@@ -164,6 +172,23 @@ expected, @@ -166,6 +174,23 @@ got). Final starve run on 7f14e3e1 (file /Users/jhf/.jcode/scratch/mq5ger-starve-final.txt): PASS 996 FAIL 4, each FAIL at ~5.2-5.8 s, i.e. the test's 5 s reload wait: likely adqdfe (reload waits on git fork), not the fsnotify double close. Non-short tui suite was killed at 561 s (signal: terminated), not a test failure. Next owner: macaque, together with adqdfe (same test).
macaque 00:05: TestUpstreamDiff now compares code: upstream-diff.sh --check applies upstream.patch (git apply) to a fresh v1.10.1 tree from the module cache and requires cmp-equality for every carried file. 02151406 on kikazaru/watchflake + ci/watchflake. Checked: GNU's header '@@ -166,6 +174,23 @@' in the patch passes the new check and fails the old one ('upstream-diff: the copy differs from upstream.patch'); a one-line drift in Close fails the new one ('v1.10.1 plus upstream.patch differs from the copy in: backend_kqueue.go'). Final starve on 7f14e3e1 (mq5ger-starve-final-raw.txt): all 4 FAILs are 'taskdirs_test.go:109: no reload after the watch came up', 0 'bad file descriptor'; stack mq5ger-stacks-final.txt:2291 is git under closeLog.times (closed.go:35/46 <- tui.go:628 <- tui.go:380) = adqdfe.
check 3: 0 double-close failures; the 4 remaining are adqdfe. Final starve on 7f14e3e1 (STARVE=8 GOMAXPROCS=4, 1000 runs, mq5ger-starve-final-raw.txt): 'PASS 996 FAIL 4', every FAIL 'taskdirs_test.go:109: no reload after the watch came up' at 5.21/5.49/5.69/5.82 s, 0 'bad file descriptor'/'unlinkat' lines; the stack (mq5ger-stacks-final.txt:2291) is the reload in git under closeLog.times, adqdfe's cause. TestCloseWhileFilesGoClosesNothingElse: 10 of 10 with the fix (kikazaru). No timeout raised, no retry, no skip.
CI 44426076160 on 02151406 (ci/watchflake): success 8/8 (secrets, bench, test ubuntu/macos/windows, test-windows-cli sync/rest, shards); ubuntu 'ok  github.com/veridit/hug/internal/fsnotify 1.942s' (was TestUpstreamDiff FAIL on 44417801741). Check 4's local non-short tui suite is still open (kikazaru's run was killed at 561 s by the tool cap).
landed e0c211b on main 2026-11-19: fsnotify: carry v1.10.1 in internal/fsnotify; failing test for its kqueue double…
landed 0558a34 on main 2026-11-19: fsnotify kqueue: Close waits for the reader to release the watches (fsnotify PR…
landed c09d5eb on main 2026-11-19: internal/fsnotify: remove's inner call no longer re-takes opMu; reproducible…
landed 6bf6a92 on main 2026-11-19: internal/fsnotify: TestUpstreamDiff compares code, not diff's hunk layout…
## History
- 2026-11-18T22:22 todo -> doing/active (take)
- 2026-11-19T01:24 doing/active -> doing/waiting (wait)
