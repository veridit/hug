---
id: 6tj2b8
title: "huginn: lint codes, board --wide, init says what it did"
status: "done"
type: feature
priority: ""
assignee: ["Jørgen H. Fjeld"]
labels: []
created: 2026-11-11
updated: 2026-11-11
---

## Goal
the three remaining seid-lang trial findings are fixed

## Depends on

## Acceptance
- [x] huginn lint --json findings carry a short code
- [x] huginn board --wide shows full titles
- [x] huginn init prints branch, worktree path and first commit

## Notes

Done in one commit: (1) every finding has a code derived from its message (FindingCode; text shows [code]; JSON code field); lint --code CODE filters. seid copy: no-acceptance 1158, long-title 827, done-unchecked 432, missing-goal 24, parent-too-deep 20, duplicate-section 10, dependency-cycle 2, none 'other'. (2) board --wide: one section per column, full titles, assignees. (3) init makes an empty first commit and prints 'tasks on branch hug/tasks, worktree ..., head <sha>, remote ...'.
