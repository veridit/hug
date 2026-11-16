---
id: 4emme8
title: "CI green on Windows again (lint, auto-refresh and memory worktree tests)"
status: "doing"
type: bug
priority: "high"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
GitHub CI passes on ubuntu, macos and windows for every push to main, so it can be the release gate

## Purpose
CI on main has failed on windows-latest since 6e47fca (v0.7.20) while ubuntu and macos passed; nobody saw it because only the release workflow was watched. Failing on 1133d95: TestLintNextFixesFirstFinding, TestLintHidesClosedTasks (cli), TestChangeOnDiskAppears, TestChangeOnDiskWaitsForPrompt, TestNoReloadWhileIdleOneForABurst, TestChangeOnDiskPollsWithoutWatcher (tui), TestBranchSyncFromLinkedWorktreeUsesMainCheckout (muninn/internal/project).

## Principle
Fix the code or the test for the real Windows behaviour (paths, line endings, file watching, git worktree paths); never skip a test on Windows unless the feature truly cannot exist there, and then say why in the skip. Read the CI log first: gh run view --log-failed.

## Relevant details
Latest failing run: gh run list --workflow ci.yml. Local machine is macOS; reproduce through CI (push to a branch is fine) and the logs. cli package took 285 s on windows.

## Depends on

## Acceptance
- [ ] gh run list --workflow ci.yml shows success on windows-latest for main
- [ ] each of the 7 failing tests passes on windows with the cause named in the commit message

## Notes
