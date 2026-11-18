---
id: mq5ger
title: "tui watch test misses a change under load"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-18
updated: 2026-11-18
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
