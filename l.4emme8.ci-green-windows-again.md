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
- [x] gh run list --workflow ci.yml shows success on windows-latest for main
- [x] each of the 7 failing tests passes on windows with the cause named in the commit message

## Notes

gh run view 43896372629 (branch ci/4emme8, commit 2988e9d, same tree as main 9d56b3e minus CHANGELOG): secrets success / test (ubuntu-latest) success / test (windows-latest) success / test (macos-latest) success. Causes: lint tests exec'd #!/bin/sh ed.sh ('%1 is not a valid Win32 application') -> test binary as VISUAL; watch tests drain() gave the git-backed reload 100ms -> await loadedMsg; worktree list prints C:/Users/runneradmin vs Go C:\Users\RUNNER~1 -> os.SameFile.

2026-11-16 crab: production check asked of vole: does hug code compare worktree path strings (would break on Windows like the test did)? Answer pending in vole's report.

2026-11-16 vole on production path comparisons: none of the C:/ vs RUNNER~1 kind. gitbranch matches checkouts by branch name, never by path string; muninn project.Find/sourceRootAlias compare EvalSymlinks'd paths via filepath.Rel; the TUI watcher builds both sides from EvalSymlinks'd paths. In the failing run every production check before the test's string assert passed.
