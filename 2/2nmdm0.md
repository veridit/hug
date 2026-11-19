---
id: 2nmdm0
title: "rebase-land.sh misses a CHANGELOG entry cut by a conflict"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
Restacking with scratch/rebase-land.sh fails loudly when a CHANGELOG entry from the source branch is missing or shorter after the rebase, as it already does for code diffs.

## Purpose
v0.7.75: 5mmqjk's 5-line entry reached main as its first line only (5b6f0de7), because a conflict resolution kept one line. CODE-DIFF IDENTICAL excludes CHANGELOG, so nothing flagged it; crab found it by reading Unreleased at release time and restored it in 4591627d.

## Principle
Check, don't auto-fix: the script reports the entry and stops; a person resolves it.

## Relevant details
Compare the source branch's added CHANGELOG lines (git diff FROM..SRC -- CHANGELOG.md, + lines) with the landed commit's (BASE..NEW); every added line must appear verbatim.

## Depends on

## Acceptance
- [ ] a test or dry run: a rebase that drops a line of an added CHANGELOG entry makes rebase-land.sh exit non-zero naming the missing line
- [ ] the normal restack prints CHANGELOG IDENTICAL next to CODE-DIFF IDENTICAL

## Notes
