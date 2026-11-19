---
id: awht8w
title: "deliver test: hook ack misses under load"
state: "doing"
type: feature
priority: ""
assignee: ["iwazaru"]
labels: []
group: ""
rank: "zzzzzzzzzzyn"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T22:20"
stage: "waiting"
model: "claude-opus-5-5"
branch: "byj479"
worktree: "/Users/jhf/.jcode/scratch/byj479"
waiting_on: "ci/awht8w-rebased b3b18675 lands on main (crab accepted 00:42); close when landed"
waiting_since: "2026-11-19T01:42"
waiting_asked_by: "iwazaru"
---

## Goal
TestRequestReachesCoordinator passes under machine load as it does alone: name why crab's hook did not acknowledge within 20 s (deliver_test.go:194) at load 95-115, and close it at the cause

## Purpose
A red local cli run on this shared machine must mean a real failure; this flake in the delivery path makes workers rerun and distrust the suite

## Principle
Cause first; no timeout raised, no retry, no skip; a fail-first reproduction under forced starvation, as ad62aa did

## Relevant details
iwazaru byj479 full niced run on fcd1adb1 (scratch/byj479-full.txt): '--- FAIL: TestRequestReachesCoordinator (24.51s) deliver_test.go:194: crab's hook did not acknowledge', load 95-115; alone 2/2 ok (1.786s, 2.072s). Sibling of ad62aa (WaitDelay under starvation). The hook sends on every return path of Acknowledge, so a timeout means it ran >20 s or blocked.

## Depends on

## Acceptance
- [x] Cause named in a note with a log line or reproduction
- [x] A test or amplified reproduction fails first, then passes -count=50 niced under load after the fix
- [x] No timeout raised and no retry added

## Notes


Cause: the test's 20 s wall clock, not the hook. Acknowledge sends on acked on every return path (nil, error, acked), so the only way to 'did not acknowledge' is a hook slower than 20 s. Its critical path after Notify is git processes: the request's own add+commit under the board turn, then the hook's open (rev-parse, remote, diff), Acknowledge's refresh (diff) and its add+commit: 7-8 git processes in a row. At load 95-500 a niced git process took up to 14 s (traced, full niced cli run at load 137-266: 'ZZGIT 14.069s git remote', 'ZZGIT 10.913s git diff', 47 commits mean 1.85 s max 7 s). No lock is held past its write: the turn waits traced at most 0.531 s; no deadlock, no lost send. Amplified reproduction on clean c186f78f: git on PATH replaced by 'sleep N; exec /usr/bin/git' (scratch/awht8w-ev/slowgit/git): N=5 passes (hook 30.9 s after delivery started, 20.7 s after the command returned), N=7: 'deliver_test.go:194: crab s hook did not acknowledge' / '--- FAIL: TestRequestReachesCoordinator (314.89s)'. Trace: scratch/awht8w-ev/slow-5.zz, pkg-full1.txt, loop-1.zz. The production hook is fire-and-forget (turn_start/post_tool_feedback): a slow ack only delays the receipt line.
check 1: Cause: the test's 20 s wall clock. Acknowledge sends on acked on every return path, so 'did not acknowledge' means a hook slower than 20 s, not a lost or stuck one. Delivery to ack is ~8 git processes in a row; traced at load 137-500 one niced git took up to 14 s ('ZZGIT 14.069s git remote', 'ZZGIT 10.913s git diff'; commits mean 1.85 s max 7 s); the hook's board-turn wait was at most 0.531 s. Reproduced on clean c186f78f with every git started 7 s late (scratch/awht8w-ev/slowgit/git): 'deliver_test.go:194: crab's hook did not acknowledge' '--- FAIL: TestRequestReachesCoordinator (314.89s)'; at 5 s late it passes. Evidence: scratch/awht8w-ev/slow-clean-7.txt, slow-5.zz, pkg-full1.txt.
check 3: 251fd2f0 removes the 20 s select from TestRequestReachesCoordinator and TestDeliveryInstructionsMatchCode and blocks on the hook's answer; no timeout constant raised or added (no time.After remains in either test), no retry, no skip; hug's own code unchanged (diff: deliver_test.go, delivery_instructions_test.go, CHANGELOG.md only).
check 2: Fail first on clean c186f78f with every git started 7 s late (git on PATH = 'sleep 7; exec /usr/bin/git', scratch/awht8w-ev/slowgit/git), niced GOMAXPROCS=2 at load 510: '    deliver_test.go:194: crab's hook did not acknowledge' '--- FAIL: TestRequestReachesCoordinator (314.89s)' (slow-clean-7.txt). Same amplifier after fix 251fd2f0, load 468-488: '--- PASS: TestRequestReachesCoordinator (499.93s)' (slow-fixed-7.txt). -count=50 niced (GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19) of TestRequestReachesCoordinator and TestDeliveryInstructionsMatchCode under load 405-492: 'ok  	github.com/veridit/hug/huginn/internal/cli	1130.241s' (count50.txt). Rebased onto main 6bf6a928 as d487f583 (no conflict, test files unchanged by the rebase).
Ready: awht8w = ci/awht8w at d487f583 on main 6bf6a928; CI 44429609615 completed success 8/8. Report: ~/.jcode/scratch/awht8w-REPORT.md. Earlier CI 44428150586 on 251fd2f0 (c186f78f base) failed only TestTmuxGolden palette drift (ubuntu), not in this diff; gone after rebase.
landed b3b1867 on main 2026-11-19: cli tests: wait on the coordinator's hook, not a 20 s clock (awht8w)
## History
- 2026-11-18T22:20 todo -> doing/active (take)
- 2026-11-19T01:42 doing/active -> doing/waiting (wait)
