---
id: khq1fq
title: "A timing test fails a release on a shared runner"
state: "doing"
type: feature
priority: "high"
assignee: ["boar"]
labels: []
group: ""
rank: "0000008"
created: 2026-11-25
updated: 2026-11-25
doing_since: "2026-11-25T15:23"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [x] CI still runs TestBigTaskKeys on macOS, shown by a ci.yml log line

## Notes


Branch khq1fq b3bc627e (ci/khq1fq run 45649108400). internal/timing.Budget(t) skips under HUG_SKIP_TIMING=1; 13 wall-clock budget tests call it; release.yml Test step sets it and prints 'wall-clock budget tests skipped: N'; ci.yml never sets it and prints 'wall-clock budget TestX: judged here'. Failing-first: b3b6dd74 repocheck failed 13 functions + release.yml step. Local HUG_SKIP_TIMING=1 go test -v ./...: exit 0, 'wall-clock budget tests skipped: 13'; plain: exit 0, 13 'judged here'. Mutants 11/11 killed (M11 Budget-after-gate survived first, killed after b3bc627e).
check 2: CI 45649108400 on b3bc627e, job test (macos-latest), step Test: '1tjq11_test.go:142: wall-clock budget TestBigTaskKeys: judged here (HUG_SKIP_TIMING is not 1)' then 'ok  github.com/veridit/hug/huginn/internal/tui 143.331s'; all 8 jobs success. macOS judged 13 budget tests, ubuntu 12 (-race skips TestBigTaskKeys).
Check 1 open until the next tag: the release.yml run's Test step must print 13 lines '...: skipped under HUG_SKIP_TIMING=1 (release.yml); CI judges it', including 'wall-clock budget TestBigTaskKeys', and 'wall-clock budget tests skipped: 13'. Simulated locally with HUG_SKIP_TIMING=1 go test -v ./...: exit 0, 'wall-clock budget tests skipped: 13'. TestReleaseSkipsWallClockBudgets (internal/repocheck) fails if release.yml's go test step loses HUG_SKIP_TIMING=1.
## History
- 2026-11-25T15:23 todo -> doing/active (take)
