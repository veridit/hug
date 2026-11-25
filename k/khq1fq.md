---
id: khq1fq
title: "A timing test fails a release on a shared runner"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "0000008"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
release.yml never fails on a wall-clock budget once CI has passed the same SHA.

## Purpose
The tag was pushed and brew stayed on v0.7.112 until a rerun. A release must rebuild what CI already proved, not re-roll timing dice.

## Principle
Timing budgets gate in CI, where a miss can be retried before landing. The release job builds and publishes the tested SHA; it does not re-judge speed. Do not raise the budget to hide the miss.

## Details
v0.7.113 release.yml run 45627554531 failed: TestBigTaskKeys 'tab, Notes open' p50 53.8 ms, budget 50 ms (1tjq11_test.go:23), after CI 45625116019 passed 8/8 on the same code. release.yml:22 runs plain 'go test ./...'. ci.yml:50 runs -race on ubuntu, where 1tjq11_test.go:139 skips timing, so timing gates only on macOS CI. Fix by a skip flag for timing tests in release.yml (e.g. HUG_SKIP_TIMING=1) or by release.yml checking that CI passed on the SHA instead of rerunning tests.

## Depends on

## Acceptance
- [ ] release.yml skips wall-clock budget tests, shown by its log on the next release
- [ ] CI still runs TestBigTaskKeys on macOS, shown by a ci.yml log line

## Notes
