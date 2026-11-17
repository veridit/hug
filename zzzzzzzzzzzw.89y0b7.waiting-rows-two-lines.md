---
id: 89y0b7
title: "Waiting rows: two lines, the title on top, what it waits on below"
status: "doing"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T16:11"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A row whose stage waits on something (waiting, review, design, depends) takes two lines in every list, overview section and board: line 1 is the task as usual (pie, id, type, priority, title, owner); line 2 hangs under the title with └─ and says what it waits on, the reason directly after the colon, and since when: '└─ waiting on you: approve the design ▸ bxqtkt.md   since 14:42'. The person is named, never initials: 'on you' for the TUI user, 'on wyvern', 'on CI'. An attachment named in the wait shows as ▸ name. Active and ready rows stay one line.

## Purpose
Jørgen 2026-11-17 14:55-14:58: the one-line waiting row hides the title, so he cannot tell what he is asked about; the colon floats away from its text; 'jhf' is just text, so he cannot tell it is for him; the file it names is a path he cannot open. He approved the two-line design at 14:58 ('let's try that'). Mock attached to 3jtfw9: waiting-rows.md.

## Principle
The title always shows; the reason is what a waiting row adds, not what replaces it. Text that reads as a sentence: 'waiting on you: ...'. Two lines only where there is something to say. The 102bc0 rule still holds per line: each line fits the width, cut in a fixed order.

## Relevant details
Replaces the 10:56 'stage: reason replaces the title' rule for waiting rows. 'you' needs hug to know the TUI user: hug.toml people = [...], hug setup sets HUG_USER once, lint flags a --on naming no listed person or agent (part of this ticket). Row code: huginn/internal/tui render.go, stage.go forYou(). Section heights count two-line rows. Moving with j/k moves by task, not by line.

## Depends on

## Acceptance
- [ ] a waiting, review or design row renders as two lines with └─ and the reason after the colon; active and ready rows stay one line (goldens at 80 and 140)
- [ ] 'on you' when the wait names the TUI user, else the name; hug setup sets HUG_USER from hug.toml people, and lint flags a --on naming no listed person or agent (test)
- [ ] j/k move by task over two-line rows, and section heights and scrolling count lines correctly (test)
- [ ] Jørgen sees the hug board with the new rows and says it reads clearly

## Notes

file:waiting-rows.md (1583 bytes, sha256:5108b054) approved mock (Jørgen 14:58)


## History
- 2026-11-17T16:11 todo -> doing/active (take)
