---
id: 0sbvnq
title: "Windows CI: huginn/internal/cli tests under 5 minutes"
status: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
go test ./huginn/internal/cli/ on the windows-latest runner finishes in under 300 s (it took 469 s in run 44063552584 after the benchmark left, 570-612 s before), with every test still run on Windows

## Purpose
The package came within seconds of go test's 10-minute limit and crossed it several times (runs 44056677278, 44057678727, 44059530090, 44060790405); every push waits on the slowest job, and a timeout hides which test failed

## Principle
Measure per test first, then remove waiting, not coverage: share setup, run independent tests in parallel, cut sleeps that wait for real time. No test skipped on Windows to make the number; no absolute timing assertions

## Relevant details
Windows costs about 1.9 s per hug command (run 43943367365). Background-sync tests run the real 1.5 s settle. Get per-test times from go test -v -json on the runner

## Depends on

## Acceptance
- [ ] huginn/internal/cli on windows-latest under 300 s in two consecutive CI runs, run ids in the notes
- [ ] no test in the package is skipped on Windows that ran there before

## Notes
