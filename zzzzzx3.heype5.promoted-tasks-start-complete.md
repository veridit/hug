---
id: heype5
title: "Promoted tasks start complete; note text in angle brackets is not lost"
status: "doing"
type: bug
priority: "medium"
assignee: ["poodle"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
A task made by hug task promote lints clean from the start, and a note like '(git worktree add --detach <dir> <rev>)' shows in the TUI exactly as written

## Purpose
Jørgen 2026-11-16 on 065wwn: right after promote it showed 'missing section Purpose, Principle', a finding hug itself created; and the note in the detail view read '(git worktree add --detach  )' because <dir> and <rev> vanished

## Principle
Promote copies the parent's Purpose and Principle into the new task (with 'From task:PARENT.' so it reads as inherited); it never creates a task that fails its own lint. The TUI renders task text as text: markdown is fine, but anything that looks like an HTML tag is shown literally, never dropped.

## Relevant details
Promote: huginn/internal/task (promote). Rendering: huginn/internal/tui/detail.go uses glamour, which treats <dir> as HTML and drops it. Check both CLI show and TUI.

## Depends on

## Acceptance
- [x] hug task promote ID N on a task with Purpose and Principle gives a child that hug task lint reports clean (test)
- [x] a note containing <dir> and <rev> shows both in the TUI detail view (test)

## Notes

819ad49 (tui/detail.go literalAngles: escapes tag-like '<' before glamour; code spans/fences untouched) + ed9545a (task/promote.go: child gets source Purpose, Principle, Relevant details) + 36064c5 CHANGELOG. TestPromoteChildLintsClean: without fix fails 'q.jbspb0.second.md:1: [missing-section] missing section Purpose, Principle'; with fix PASS. TestDetailShowsAngleBracketsLiterally: without fix fails 'note not shown as written: want "(git worktree add --detach <dir> <rev>)"'; with fix PASS. Full suite ok (task 4.039s, tui 47.606s, cmd/hug 0.298s).
