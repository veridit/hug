---
id: ce3dmq
title: "Windows: muninn app tests hit the 10m timeout; placement starts 4 git per lookup"
state: "doing"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T19:42"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
stage: "active"
---

## Goal
The muninn/internal/app package finishes well inside go test's 10 minute default on the Windows runner: placement looks up the session's project once per call, and a lookup in a main worktree starts 2 git processes, not 4.

## Purpose
Main 9f3e38f, run 44142356523: windows 'panic: test timed out after 10m0s' in TestPlacesSessionInventory/client_project, the stuck goroutine in exec.LookPath starting git (project.find -> gitbranch.MainRoot). Not a deadlock: the package takes 290-501 s on windows (8 green runs) vs 79 s macOS, 133 s ubuntu; this runner was slow everywhere (cli 515 s vs 331-432 s). Locally the package starts 1656 git processes, 1180 of them rev-parse --show-toplevel.

## Principle
Count git processes, do not time them (as TestGitProcessesPerCommand). Same scopes and Notes as before; no cache that outlives a call.

## Relevant details
app/places.go: Placement calls placementScopes, then PlaceForScope -> Places -> placementScopes again (2 project.Find per call). project/project.go find: rev-parse --show-toplevel, rev-list --max-parents=0, then gitbranch.MainRoot (rev-parse --path-format + worktree list) only to set Notes; in a main worktree .git is a directory and top is the main root.

## Depends on

## Acceptance
- [x] a project lookup in a main worktree starts 2 git processes (test fails before: 4); a linked worktree still gets the main worktree's Notes
- [x] Placement looks up the project once (test fails before: twice)
- [ ] windows CI: muninn/internal/app time on main falls from the 290-501 s range, run id in a note

## Notes


hatchling 18:54: landed debb45a on main (63231d4..debb45a, fast-forward); main CI 44152740444. Cause of run 44142356523: no deadlock. The stuck goroutine is in exec.LookPath starting git (project.find -> gitbranch.MainRoot). muninn/internal/app takes 290-501 s on windows across 8 green main runs (44132567751..44144753956), vs macOS 79 s and ubuntu 133 s; the failing runner was slow everywhere (cli 515 s vs 331-432 s). Locally (git counted with a PATH wrapper) the package started 1656 git processes, 1180 of them rev-parse --show-toplevel. Fix: (1) main worktree (.git is a dir) is its own main root: Find 4 -> 2 git; (2) no .git above dir and no GIT_DIR: 0 git (was a failing rev-parse; 639 calls were the test home's plain ~/.jcode/work); (3) Placement derives aliases once (was twice via PlaceForScope). Package now 422 git processes, 68 s -> 30 s locally. Tests fail before: TestFindInMainWorktreeStartsTwoGit 'Find in a main worktree starts 4 git processes, want 2'; TestPlacementLooksUpTheProjectOnce 'Placement("here") looks the project up 2 times, want 1 (8 git processes)'. Linked worktree still gets the main checkout's Notes (same test, 4 git). Local: go test -race ./muninn/... ./cmd/hug green (niced).
landed debb45a on main 2026-11-17: hug mem: a project lookup starts 2 git processes in a main checkout and none in…
hatchling 19:12: main CI 44152740444 on debb45a: ubuntu, windows, secrets, bench success; macOS cancelled for runner capacity ('not acquired by Runner of type hosted'), one rerun started. Windows muninn/internal/app 309.225 s (8 runs before: 290-501 s, median ~380 s), cli 334 s. So the git cut gives at most ~20% on Windows: git processes are not the main cost there. Diagnostic branch ci/ce3dmq-timing (9249f1e, never main) runs go test -v ./muninn/internal/app alone on 3 OSes for per-test times.
hatchling 19:31: the real cost was found with per-test times on windows (diagnostic branch ci/ce3dmq-timing, run 44155839304, go test -v ./muninn/internal/app alone; branch deleted): TestCaptureEightCandidatesOnTwentyThousandUnits 116.59 s of the package's 162.470 s, next 5.36 s; inside it the timed capture is elapsed=474.7ms, so 116 s was the 20000-unit seed, one autocommit each. Seed now in one BEGIN IMMEDIATE/COMMIT (pool has 1 connection). Run 44156932321 on the fix: windows test 6.06 s, package 57.003 s (was 162.470 s); macOS 3.17 s / 19.995 s; ubuntu 4.21 s / 19.902 s. Landed f24080c (fb66000..f24080c), main CI 44159339484. Before it, debb45a's git cut alone: windows package 309.225 s in the full suite (44152740444) vs 290-501 s before.
landed f24080c on main 2026-11-17: hug mem test: seed the 20k-unit capture timing test in one transaction (ce3dmq)
## History
- 2026-11-17T19:42 todo -> doing/active (take)
- 2026-11-17T20:21 doing -> doing/waiting (wait)
- 2026-11-17T20:31 doing/waiting -> doing/active (resume); was waiting_asked_by: hatchling; waiting_on: windows timing run on ci/ce3dmq-timing (seed in one transaction), then land; waiting_since: 2026-11-17T20:21
- 2026-11-17T20:31 doing/active -> doing/waiting (wait)
- 2026-11-17T20:40 doing/waiting -> doing/active (resume); was waiting_asked_by: hatchling; waiting_on: main CI 44159339484: windows muninn/internal/app time in the full suite; waiting_since: 2026-11-17T20:31
