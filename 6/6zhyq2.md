---
id: 6zhyq2
title: "TestViewKeysBlinkAfterV flakes on CI timing (497 ms vs 530 ms)"
state: todo
type: bug
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
The view-line blink tests pass on CI regardless of tick timing.

## Purpose
maple 2026-11-19 (yp3c09 CI 44520969132): TestViewKeysBlinkAfterV failed once measuring 497 ms against the expected 530 ms; the test is from a82j3n/r80svs, not yp3c09.

## Principle
Drive the blink by sending cursor blink messages, never by wall-clock sleeps; no retries to hide a flake.

## Relevant details
huginn/internal/tui/a82j3n_test.go, r80svs_test.go: find any time.Sleep or real tick in the blink tests and replace with explicit blink msgs. Same check for TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade (Windows, huginn/internal/cli), seen once in yp3c09 CI 44522707343 attempt 1.

## Depends on

## Acceptance
- [ ] The blink tests use no wall-clock waits (grep in a note) and pass 50 runs in a row niced (-count=50)
- [ ] TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade failure on Windows is reproduced or explained in a note

## Notes
