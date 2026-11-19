---
id: 3dzmh5
title: "Drop internal/fsnotify once fsnotify releases the fix for issue 788"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzz7"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
hug imports github.com/fsnotify/fsnotify again at a released version that fixes issue 788, and internal/fsnotify is gone.

## Purpose
mq5ger carries fsnotify v1.10.1 plus PR 789 in internal/fsnotify because v1.10.1's kqueue Close double-closes descriptors (https://github.com/fsnotify/fsnotify/issues/788). A carried copy misses upstream fixes; go back to the module as soon as a release has the fix.

## Principle
Only on a fsnotify release that contains PR 789 or another fix for issue 788, checked by running hug's close tests against it; no pinning to an unreleased commit.

## Relevant details
Issue: https://github.com/fsnotify/fsnotify/issues/788. Fix: https://github.com/fsnotify/fsnotify/pull/789 (open, head 7d79a83e). Releases: https://github.com/fsnotify/fsnotify/releases. Copy and its deviations: internal/fsnotify/README.md, upstream.patch, upstream-diff.sh. Importers: huginn/internal/tui/watch.go, huginn/internal/tui/watch_test.go, muninn/internal/daemon/watch.go. Related: issue 795 (remove closes the fd before dropping the table entry).

## Depends on

## Acceptance
- [ ] a fsnotify release containing the fix for issue 788 is named, with its release URL
- [ ] go.mod requires that release, the three importers use github.com/fsnotify/fsnotify, internal/fsnotify is deleted
- [ ] TestCloseWhileFilesGoClosesNothingElse and TestRemoveWatchedDirReturns, moved beside the TUI watch against the module, pass on macOS 10 runs of 10

## Notes
