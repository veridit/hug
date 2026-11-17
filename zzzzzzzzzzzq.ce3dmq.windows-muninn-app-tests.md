---
id: ce3dmq
title: "Windows: muninn app tests hit the 10m timeout; placement starts 4 git per lookup"
status: "doing"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T19:42"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] a project lookup in a main worktree starts 2 git processes (test fails before: 4); a linked worktree still gets the main worktree's Notes
- [ ] Placement looks up the project once (test fails before: twice)
- [ ] windows CI: muninn/internal/app time on main falls from the 290-501 s range, run id in a note

## Notes


## History
- 2026-11-17T19:42 todo -> doing/active (take)
