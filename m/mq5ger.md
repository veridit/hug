---
id: mq5ger
title: "tui watch test misses a change under load"
state: "doing"
type: feature
priority: ""
assignee: ["kikazaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzt"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T22:22"
stage: "active"
model: "claude-opus-5-5"
branch: "kikazaru/watchflake"
worktree: "/Users/jhf/.jcode/scratch/watchflake"
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
- [ ] a test or harness shows the failure first under forced starvation on origin/main, with the verbatim failure line
- [ ] the cause is named with file:line evidence
- [ ] the fix passes the same starvation run 0 failures in N runs, with no timeout raised, no retry and no skip
- [ ] full tui suite green locally (non-short) and CI green

## Notes


Fail first, stock fsnotify v1.10.1 (origin/main 8a45b0eb plus probe only): TestStrayClose (closes a watch while its files are deleted, holds files open meanwhile), STRAY=300 -count=6: 3 of 6 runs FAIL, verbatim: 'zz_stray_test.go:90: 3 descriptors closed underneath in 300 rounds; first: round 34: unlinkat /Users/jhf/.jcode/scratch/TestStrayClose3079977529/001/34/g0/t15.md: bad file descriptor' (the CI line's shape) and 'first: round 205: fd 36 closed underneath: fstat bad file descriptor ino 0'. The real test under STARVE=8 GOMAXPROCS=4 on stock, 2x1500: '--- FAIL: TestStarvedChangeBeforeWatch (0.01s) taskdirs_test.go:84: close /Users/jhf/.jcode/scratch/TestStarvedChangeBeforeWatch3502987801/001/h/.huginn-809769911: bad file descriptor' (store.New's temp file lost its fd to a watcher's stray close). Logs: $JCODE_SCRATCH_DIR/mq5ger-stray-stock.txt, mq5ger-starve-stock-b.txt.
## History
- 2026-11-18T22:22 todo -> doing/active (take)
