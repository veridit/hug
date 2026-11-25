---
id: 8syncd
title: "Palette hints name each section's job as agents' instructions do"
state: todo
type: feature
priority: "low"
assignee: []
labels: []
group: ""
rank: "0003"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
The : set hints in hug tui name Goal what, Purpose why, Principle how and Details notes, as task.SectionsRule does.

## Purpose
rhino 2026-11-25: palettetree.go:447-448 still read 'Where we aim' and 'Facts a decision hangs on', so a person and an agent read different words.

## Principle
One wording for each section's job; people's hints may be shorter but not different.

## Details
From v0.7.112 (32ym01): task.GoalJob, PurposeJob, PrincipleJob in huginn/internal/task/goal.go. Hints at huginn/internal/tui/palettetree.go:447-448. Part of the 32ym01 trial: if Jørgen changes the sentence, these follow.

## Depends on

## Acceptance
- [ ] The : set hints use the GoalJob, PurposeJob and PrincipleJob words, pinned by a test

## Notes
