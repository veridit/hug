---
id: 0n4rnx
title: "Windows CLI tests run at 450 of the 600 s cap"
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
huginn/internal/cli finishes on windows-latest with clear margin under go test's 10m timeout, by making slow tests cheaper, not by raising the cap

## Purpose
Green runs take 443-453 s (CI 44344774710, 44347035097). Run 44350205436 hit 'panic: test timed out after 10m0s' with no hung test (running tests 3-7 s old), so a slow runner fails exact-SHA landing

## Principle
Measure per-test time first (go test -json on windows); cut process spawns and sleeps in the slowest tests; do not raise -timeout and do not split the package just to dodge the cap

## Relevant details
Log: /Users/jhf/.jcode/scratch/wincli-44350205436.log. Running at the panic: TestBackgroundSyncParallelNotesAllLand, TestTakeAsAndModelRecordsBoth, TestTakeUsesJcodeSessionName, TestTakeWithoutModelFailsWithFix.

## Depends on

## Acceptance
- [ ] A table of the 15 slowest cli tests on windows-latest, before and after
- [ ] huginn/internal/cli on windows-latest under 300 s in 3 consecutive CI runs

## Notes
