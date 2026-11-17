---
id: zd3x24
title: "Flaky on Windows: capture test never sees the child persist its batch"
status: todo
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
TestCaptureKilledAfterSaveBeforeCheckpoint passes reliably on Windows: the cause of 'child never persisted native batch' is found and fixed, with a test that fails before the fix, and no wait raised to hide it.

## Purpose
Main CI run 44092559068 on d198e01, windows: 'capture_batch_test.go:200: child never persisted native batch' (10.54s), FAIL muninn/internal/app 321.218s. The same commit was green on ci/6ephqk-rows (run 44090100603), so it is not caused by 6ephqk. It blocked the v0.7.41 release.

## Principle
A flake is a bug in the test or in hug; find which. No retries, no longer waits, unless measurement shows runner speed is the cause.

## Relevant details
muninn/internal/app/capture_batch_test.go:200. The test kills a child after save and before checkpoint. Windows file sharing (see kbc99j: os.Open without FILE_SHARE_DELETE, rename via MoveFileEx) and process kill semantics differ; check whether the child's write is visible to the parent, and whether the 10 s wait races the child start.

## Depends on

## Acceptance
- [ ] the cause is named with evidence (log lines or a probe run on Windows CI, run id)
- [ ] a test that fails before the fix passes after; Windows -count=20 of that test green (run id)

## Notes
